import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { Network, Search, ZoomIn, ZoomOut, Maximize2, Filter, RotateCcw, Play, Clock, ChevronRight, Database, Layers, Target, RefreshCw, Loader2, AlertCircle, Info } from 'lucide-react';
import { useSubgraph, useTimeline } from '../../hooks/useGraph';
import type { GraphNode, GraphEdge, SubgraphData } from '../../types/graph';

/* ─── Types ─── */
interface GraphCanvas {
  nodePositions: Map<string, { x: number; y: number }>;
  scale: number;
  offset: { x: number; y: number };
}

const NODE_COLORS: Record<string, string> = {
  Entity: '#8b5cf6',
  Person: '#3b82f6',
  Concept: '#10b981',
  Event: '#f59e0b',
  Location: '#ef4444',
  Document: '#6b7280',
};

const EDGE_COLORS: Record<string, string> = {
  KNOWS: '#8b5cf6',
  PARTICIPATED_IN: '#f59e0b',
  RELATED_TO: '#10b981',
  MENTIONS: '#6b7280',
  CAUSED: '#ef4444',
};

/* ─── Rich mock subgraph for dev mode ─── */
const RICH_MOCK_SUBGRAPH: SubgraphData = {
  nodes: [
    { id: 'n1', label: 'Alice Chen', type: 'Person', properties: { role: 'developer', company: 'VNP' } },
    { id: 'n2', label: 'Memory System', type: 'Concept', properties: { domain: 'AI', maturity: 'emerging' } },
    { id: 'n3', label: 'Graphiti Engine', type: 'Entity', properties: { type: 'episodic', version: '1.2' } },
    { id: 'n4', label: 'TypeScript', type: 'Concept', properties: { domain: 'programming' } },
    { id: 'n5', label: 'Design Review', type: 'Event', properties: { date: '2026-05-10', outcome: 'approved' } },
    { id: 'n6', label: 'Bob Kim', type: 'Person', properties: { role: 'architect' } },
    { id: 'n7', label: 'React Query', type: 'Concept', properties: { domain: 'frontend' } },
    { id: 'n8', label: 'VNP Platform', type: 'Entity', properties: { type: 'product' } },
  ],
  edges: [
    { id: 'e1', source: 'n1', target: 'n2', type: 'KNOWS' },
    { id: 'e2', source: 'n1', target: 'n3', type: 'RELATED_TO' },
    { id: 'e3', source: 'n1', target: 'n4', type: 'KNOWS' },
    { id: 'e4', source: 'n1', target: 'n5', type: 'PARTICIPATED_IN' },
    { id: 'e5', source: 'n6', target: 'n5', type: 'PARTICIPATED_IN' },
    { id: 'e6', source: 'n6', target: 'n8', type: 'RELATED_TO' },
    { id: 'e7', source: 'n3', target: 'n8', type: 'RELATED_TO' },
    { id: 'e8', source: 'n2', target: 'n7', type: 'MENTIONS' },
    { id: 'e9', source: 'n1', target: 'n7', type: 'KNOWS' },
  ],
};

/* ─── Layout: force-directed positions (simple circular for now) ─── */
function computeLayout(nodes: GraphNode[]): Map<string, { x: number; y: number }> {
  const positions = new Map<string, { x: number; y: number }>();
  const cx = 500, cy = 300, radius = 220;
  nodes.forEach((node, i) => {
    const angle = (i / nodes.length) * 2 * Math.PI - Math.PI / 2;
    positions.set(node.id, {
      x: cx + radius * Math.cos(angle),
      y: cy + radius * Math.sin(angle),
    });
  });
  return positions;
}

export function GraphStudio() {
  const { data: subgraph, isLoading, error, refetch } = useSubgraph({});
  const { data: timeline } = useTimeline({});

  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [selectedEdge, setSelectedEdge] = useState<GraphEdge | null>(null);
  const [search, setSearch] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [scale, setScale] = useState(1);
  const [query, setQuery] = useState('');

  const displayData = subgraph ?? RICH_MOCK_SUBGRAPH;

  const filteredNodes = useMemo(() => {
    return displayData.nodes.filter(n => {
      const matchSearch = !search || n.label.toLowerCase().includes(search.toLowerCase()) || n.type.toLowerCase().includes(search.toLowerCase());
      const matchType = filterType === 'all' || n.type === filterType;
      return matchSearch && matchType;
    });
  }, [displayData.nodes, search, filterType]);

  const filteredNodeIds = useMemo(() => new Set(filteredNodes.map(n => n.id)), [filteredNodes]);

  const filteredEdges = useMemo(() =>
    displayData.edges.filter(e => filteredNodeIds.has(e.source) && filteredNodeIds.has(e.target)),
    [displayData.edges, filteredNodeIds]
  );

  const positions = useMemo(() => computeLayout(filteredNodes), [filteredNodes]);

  const nodeTypes = useMemo(() => Array.from(new Set(displayData.nodes.map(n => n.type))), [displayData.nodes]);

  const handleZoomIn = () => setScale(s => Math.min(s + 0.1, 2.5));
  const handleZoomOut = () => setScale(s => Math.max(s - 0.1, 0.3));
  const handleReset = () => { setScale(1); setSelectedNode(null); setSelectedEdge(null); };

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center flex-col gap-4">
        <Loader2 className="w-10 h-10 text-purple-400 animate-spin" />
        <p className="text-white/40 text-sm">Loading knowledge graph...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center flex-col gap-4">
        <AlertCircle className="w-10 h-10 text-red-400" />
        <p className="text-white/60">Failed to load graph</p>
        <button onClick={() => refetch()} className="flex items-center gap-2 px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30">
          <RefreshCw className="w-4 h-4" /> Retry
        </button>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-5 border-b border-white/10 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-purple-500/20">
            <Network className="w-5 h-5 text-purple-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Graph Studio</h2>
            <p className="text-sm text-white/50 mt-0.5">
              {filteredNodes.length} nodes · {filteredEdges.length} edges — episodic knowledge graph
            </p>
          </div>
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-white/40" />
            <input
              type="text" value={search} onChange={e => setSearch(e.target.value)}
              placeholder="Search nodes..."
              className="pl-8 pr-3 py-1.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-purple-500/50 w-44"
            />
          </div>
          <select
            value={filterType}
            onChange={e => setFilterType(e.target.value)}
            className="px-3 py-1.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white focus:outline-none"
          >
            <option value="all">All Types</option>
            {nodeTypes.map(t => <option key={t} value={t}>{t}</option>)}
          </select>
          <div className="flex items-center gap-1 bg-white/5 rounded-lg border border-white/10 px-1">
            <button onClick={handleZoomOut} className="p-1.5 text-white/60 hover:text-white"><ZoomOut className="w-4 h-4" /></button>
            <span className="text-xs text-white/40 w-12 text-center">{Math.round(scale * 100)}%</span>
            <button onClick={handleZoomIn} className="p-1.5 text-white/60 hover:text-white"><ZoomIn className="w-4 h-4" /></button>
          </div>
          <button onClick={handleReset} className="p-2 bg-white/5 border border-white/10 rounded-lg text-white/60 hover:text-white">
            <RotateCcw className="w-4 h-4" />
          </button>
          <button onClick={() => refetch()} className="p-2 bg-white/5 border border-white/10 rounded-lg text-white/60 hover:text-white">
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-hidden flex">
        {/* ── Canvas ── */}
        <div className="flex-1 relative bg-[#080808] overflow-hidden">
          <svg className="w-full h-full" style={{ transform: `scale(${scale})`, transformOrigin: 'center center', transition: 'transform 0.15s ease' }}>
            <defs>
              <pattern id="grid-studio" width="40" height="40" patternUnits="userSpaceOnUse">
                <path d="M 40 0 L 0 0 0 40" fill="none" stroke="rgba(255,255,255,0.04)" strokeWidth="0.5" />
              </pattern>
              <marker id="arrowhead" markerWidth="6" markerHeight="4" refX="6" refY="2" orient="auto">
                <polygon points="0 0, 6 2, 0 4" fill="rgba(255,255,255,0.25)" />
              </marker>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid-studio)" />

            {/* Edges */}
            {filteredEdges.map(edge => {
              const from = positions.get(edge.source);
              const to = positions.get(edge.target);
              if (!from || !to) return null;
              const color = EDGE_COLORS[edge.type] ?? '#ffffff30';
              const isSelected = selectedEdge?.id === edge.id;
              const midX = (from.x + to.x) / 2;
              const midY = (from.y + to.y) / 2;
              return (
                <g key={edge.id} onClick={() => { setSelectedEdge(edge); setSelectedNode(null); }} className="cursor-pointer">
                  <line
                    x1={from.x} y1={from.y} x2={to.x} y2={to.y}
                    stroke={isSelected ? color : `${color}66`}
                    strokeWidth={isSelected ? 2.5 : 1.5}
                    markerEnd="url(#arrowhead)"
                  />
                  <text x={midX} y={midY - 6} fill="rgba(255,255,255,0.25)" fontSize="9" textAnchor="middle">
                    {edge.type}
                  </text>
                </g>
              );
            })}

            {/* Nodes */}
            {filteredNodes.map(node => {
              const pos = positions.get(node.id);
              if (!pos) return null;
              const color = NODE_COLORS[node.type] ?? '#6b7280';
              const isSelected = selectedNode?.id === node.id;
              return (
                <g key={node.id} onClick={() => { setSelectedNode(node); setSelectedEdge(null); }} className="cursor-pointer">
                  <circle
                    cx={pos.x} cy={pos.y} r={isSelected ? 32 : 26}
                    fill={`${color}22`}
                    stroke={color}
                    strokeWidth={isSelected ? 3 : 1.5}
                    className="transition-all duration-150"
                  />
                  {isSelected && (
                    <circle cx={pos.x} cy={pos.y} r={40} fill="none" stroke={color} strokeWidth="0.5" strokeDasharray="3 3" opacity={0.4} />
                  )}
                  <text x={pos.x} y={pos.y - 2} textAnchor="middle" fill="white" fontSize="11" fontWeight="500">
                    {node.label.length > 12 ? node.label.slice(0, 12) + '…' : node.label}
                  </text>
                  <text x={pos.x} y={pos.y + 12} textAnchor="middle" fill={`${color}cc`} fontSize="8">
                    {node.type}
                  </text>
                </g>
              );
            })}
          </svg>

          {/* Stats overlay */}
          <div className="absolute top-4 left-4 flex gap-2">
            {[
              { label: 'Nodes', value: filteredNodes.length, icon: Database },
              { label: 'Edges', value: filteredEdges.length, icon: Layers },
            ].map(s => (
              <div key={s.label} className="bg-[#1a1a1f]/90 border border-white/10 rounded-lg px-3 py-2 backdrop-blur-sm flex items-center gap-2">
                <s.icon className="w-3.5 h-3.5 text-white/40" />
                <div>
                  <div className="text-xs text-white/40">{s.label}</div>
                  <div className="text-sm font-semibold text-white">{s.value}</div>
                </div>
              </div>
            ))}
          </div>

          {/* Legend */}
          <div className="absolute bottom-4 left-4 bg-[#1a1a1f]/90 border border-white/10 rounded-lg p-3 backdrop-blur-sm">
            <p className="text-xs text-white/40 mb-2 uppercase tracking-wider">Node Types</p>
            <div className="space-y-1">
              {Object.entries(NODE_COLORS).map(([type, color]) => (
                <div key={type} className="flex items-center gap-2">
                  <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: color }} />
                  <span className="text-xs text-white/50">{type}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* ── Inspector Panel ── */}
        <div className="w-72 flex-shrink-0 border-l border-white/10 overflow-y-auto bg-[#0f0f14] p-4 space-y-4">
          {!selectedNode && !selectedEdge && (
            <div className="text-center py-12">
              <Target className="w-8 h-8 text-white/20 mx-auto mb-2" />
              <p className="text-white/40 text-xs">Click a node or edge to inspect</p>
            </div>
          )}

          {selectedNode && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full" style={{ backgroundColor: NODE_COLORS[selectedNode.type] ?? '#6b7280' }} />
                <h3 className="text-sm font-semibold text-white">{selectedNode.label}</h3>
              </div>
              <div className="space-y-2">
                <InfoRow label="Node ID" value={selectedNode.id} mono />
                <InfoRow label="Type" value={selectedNode.type} />
                {Object.entries(selectedNode.properties ?? {}).map(([k, v]) => (
                  <InfoRow key={k} label={k} value={String(v)} />
                ))}
              </div>

              {/* Connected edges */}
              <div>
                <p className="text-xs font-semibold text-white/40 uppercase tracking-wider mb-2">Connections</p>
                <div className="space-y-1">
                  {displayData.edges
                    .filter(e => e.source === selectedNode.id || e.target === selectedNode.id)
                    .map(edge => {
                      const other = edge.source === selectedNode.id ? edge.target : edge.source;
                      const otherNode = displayData.nodes.find(n => n.id === other);
                      const dir = edge.source === selectedNode.id ? '→' : '←';
                      return (
                        <div key={edge.id} className="flex items-center gap-2 p-2 bg-white/5 rounded text-xs text-white/60">
                          <span className="text-purple-400">{dir}</span>
                          <span className="text-white/40">{edge.type}</span>
                          <span className="text-white/70">{otherNode?.label}</span>
                        </div>
                      );
                    })}
                </div>
              </div>
            </div>
          )}

          {selectedEdge && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <ChevronRight className="w-4 h-4" style={{ color: EDGE_COLORS[selectedEdge.type] ?? '#6b7280' }} />
                <h3 className="text-sm font-semibold text-white">{selectedEdge.type}</h3>
              </div>
              <div className="space-y-2">
                <InfoRow label="Edge ID" value={selectedEdge.id} mono />
                <InfoRow label="Source" value={displayData.nodes.find(n => n.id === selectedEdge.source)?.label ?? selectedEdge.source} />
                <InfoRow label="Target" value={displayData.nodes.find(n => n.id === selectedEdge.target)?.label ?? selectedEdge.target} />
                {Object.entries(selectedEdge.properties ?? {}).map(([k, v]) => (
                  <InfoRow key={k} label={k} value={String(v)} />
                ))}
              </div>
            </div>
          )}

          {/* Cypher Query box */}
          <div className="mt-4">
            <p className="text-xs font-semibold text-white/40 uppercase tracking-wider mb-2">Graph Query (Cypher)</p>
            <textarea
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="MATCH (n:Person)-[:KNOWS]->(m) RETURN n, m LIMIT 25"
              rows={4}
              className="w-full px-3 py-2 bg-black/40 border border-white/10 rounded-lg text-xs font-mono text-green-300/80 placeholder:text-white/20 resize-none focus:outline-none focus:ring-1 focus:ring-purple-500/50"
            />
            <button className="mt-2 w-full flex items-center justify-center gap-2 px-3 py-2 bg-purple-500/20 text-purple-400 rounded-lg text-sm hover:bg-purple-500/30 border border-purple-500/30">
              <Play className="w-3.5 h-3.5" /> Run Query
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-2 py-1.5 border-b border-white/5">
      <span className="text-xs text-white/40 capitalize flex-shrink-0">{label}</span>
      <span className={`text-xs text-right ${mono ? 'font-mono text-purple-300' : 'text-white/70'}`}>{value}</span>
    </div>
  );
}
