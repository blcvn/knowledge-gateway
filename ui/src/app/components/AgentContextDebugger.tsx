import { useState, useCallback, useRef, useEffect } from 'react';
import { Terminal, Settings2, Play, Database, Filter, ChevronDown, ChevronRight, Clock, Layers, Cpu, AlertCircle, CheckCircle, Loader2, Search, XCircle, Copy, Info } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';
import { API_CONFIG } from '../../config/api.config';

/* ─── Types ─── */
interface DebugQueryConfig {
  agentId: string;
  userId: string;
  embeddingModel: string;
  retrievalStrategy: string;
  topK: number;
  engines: string[];
  minScore: number;
}

interface ContextResult {
  engine: string;
  items: ContextItem[];
  totalTokens: number;
  latencyMs: number;
}

interface ContextItem {
  id: string;
  content: string;
  score: number;
  type: string;
  source: string;
  metadata?: Record<string, unknown>;
}

interface DebugResponse {
  query: string;
  results: ContextResult[];
  totalTokens: number;
  totalLatencyMs: number;
  assembledContext: string;
  promptTokenBudget: number;
}

/* ─── Mock debug response ─── */
function buildMockDebugResponse(query: string): DebugResponse {
  return {
    query,
    totalTokens: 1842,
    totalLatencyMs: 147,
    promptTokenBudget: 4096,
    assembledContext: `[SYSTEM CONTEXT — 1842 tokens assembled from 3 engines]\n\n[EPISODIC — Graphiti]\nUser previously discussed preference for dark mode UI and requested TypeScript examples.\n\n[PROFILE — Memobase]\nUser: developer, expertise: fullstack, interests: [react, graphql, ai], goals: [learn_memory_systems]\n\n[SEMANTIC — Cognee]\nMemory system architectures include episodic (event-based), semantic (knowledge-based), and procedural (skill-based) paradigms.`,
    results: [
      {
        engine: 'graphiti',
        latencyMs: 34,
        totalTokens: 620,
        items: [
          { id: 'ep_1', content: 'User preferred dark mode and TypeScript in previous session.', score: 0.92, type: 'episodic', source: 'episode:session_abc' },
          { id: 'ep_2', content: 'User asked about memory system design last week.', score: 0.85, type: 'episodic', source: 'episode:session_def' },
        ],
      },
      {
        engine: 'memobase',
        latencyMs: 45,
        totalTokens: 380,
        items: [
          { id: 'prof_1', content: 'User Profile: fullstack developer interested in AI memory systems.', score: 0.98, type: 'profile', source: 'profile:user_123' },
          { id: 'buf_1', content: 'Recent: asking about Graphiti API integration patterns.', score: 0.88, type: 'buffer', source: 'buffer:session_xyz' },
        ],
      },
      {
        engine: 'cognee',
        latencyMs: 68,
        totalTokens: 842,
        items: [
          { id: 'sem_1', content: 'Memory system architectures: episodic vs semantic vs procedural.', score: 0.76, type: 'semantic', source: 'knowledge:memory_systems' },
          { id: 'sem_2', content: 'React Query patterns for enterprise data fetching with cache invalidation.', score: 0.71, type: 'semantic', source: 'knowledge:react_patterns' },
        ],
      },
    ],
  };
}

const ENGINES = ['graphiti', 'cognee', 'zep', 'openviking', 'memobase', 'supermemory'];
const EMBEDDING_MODELS = [
  'openai/text-embedding-3-small',
  'openai/text-embedding-3-large',
  'Medical_BGE_M3 (Custom)',
  'cohere/embed-multilingual-v3',
];
const RETRIEVAL_STRATEGIES = [
  'hybrid (Graph + Semantic)',
  'semantic_only',
  'graph_traversal_only',
  'profile_enriched',
];

const ENGINE_COLORS: Record<string, string> = {
  graphiti: 'text-purple-400 bg-purple-500/20 border-purple-500/30',
  cognee: 'text-blue-400 bg-blue-500/20 border-blue-500/30',
  zep: 'text-green-400 bg-green-500/20 border-green-500/30',
  openviking: 'text-orange-400 bg-orange-500/20 border-orange-500/30',
  memobase: 'text-teal-400 bg-teal-500/20 border-teal-500/30',
  supermemory: 'text-amber-400 bg-amber-500/20 border-amber-500/30',
};

export function AgentContextDebugger() {
  const [config, setConfig] = useState<DebugQueryConfig>({
    agentId: 'agent-001',
    userId: 'user-123',
    embeddingModel: 'openai/text-embedding-3-small',
    retrievalStrategy: 'hybrid (Graph + Semantic)',
    topK: 5,
    engines: ['graphiti', 'memobase', 'cognee'],
    minScore: 0.7,
  });
  const [query, setQuery] = useState('');
  const [result, setResult] = useState<DebugResponse | null>(null);
  const [expandedEngines, setExpandedEngines] = useState<Set<string>>(new Set(['graphiti', 'memobase']));
  const [activeTab, setActiveTab] = useState<'results' | 'assembled' | 'raw'>('results');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const debugMutation = useMutation({
    mutationFn: (q: string): Promise<DebugResponse> =>
      apiClient.post<DebugResponse>(
        '/v1/console/debugger/context',
        { query: q, config }
      ),
    onSuccess: (data) => {
      setResult(data);
      setActiveTab('results');
    },
  });

  const handleRun = useCallback(() => {
    if (!query.trim()) return;
    debugMutation.mutate(query.trim());
  }, [query, debugMutation]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') handleRun();
  };

  const toggleEngine = (eng: string) => {
    setConfig(prev => ({
      ...prev,
      engines: prev.engines.includes(eng)
        ? prev.engines.filter(e => e !== eng)
        : [...prev.engines, eng],
    }));
  };

  const toggleExpandEngine = (eng: string) => {
    setExpandedEngines(prev => {
      const next = new Set(prev);
      next.has(eng) ? next.delete(eng) : next.add(eng);
      return next;
    });
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).catch(console.error);
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10 flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-fuchsia-500/20">
            <Terminal className="w-5 h-5 text-fuchsia-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Agent Context Debugger</h2>
            <p className="text-sm text-white/50 mt-0.5">Inspect exact memory retrieval — what context gets injected into your LLM prompt</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-hidden flex">
        {/* ── Left Panel: Config ── */}
        <div className="w-80 flex-shrink-0 border-r border-white/10 overflow-y-auto p-5 space-y-5">
          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Agent / User</label>
            <div className="space-y-2">
              <input
                type="text"
                value={config.agentId}
                onChange={e => setConfig(p => ({ ...p, agentId: e.target.value }))}
                placeholder="Agent ID"
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-fuchsia-500/50"
              />
              <input
                type="text"
                value={config.userId}
                onChange={e => setConfig(p => ({ ...p, userId: e.target.value }))}
                placeholder="User ID"
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-fuchsia-500/50"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Embedding Model</label>
            <select
              value={config.embeddingModel}
              onChange={e => setConfig(p => ({ ...p, embeddingModel: e.target.value }))}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white focus:outline-none focus:ring-1 focus:ring-fuchsia-500/50"
            >
              {EMBEDDING_MODELS.map(m => <option key={m} value={m}>{m}</option>)}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Retrieval Strategy</label>
            <select
              value={config.retrievalStrategy}
              onChange={e => setConfig(p => ({ ...p, retrievalStrategy: e.target.value }))}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white focus:outline-none focus:ring-1 focus:ring-fuchsia-500/50"
            >
              {RETRIEVAL_STRATEGIES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Top K Results</label>
            <div className="flex items-center gap-3">
              <input
                type="range" min={1} max={20} value={config.topK}
                onChange={e => setConfig(p => ({ ...p, topK: Number(e.target.value) }))}
                className="flex-1 accent-fuchsia-500"
              />
              <span className="text-sm text-white w-6 text-right">{config.topK}</span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Min Score</label>
            <div className="flex items-center gap-3">
              <input
                type="range" min={0} max={100} value={config.minScore * 100}
                onChange={e => setConfig(p => ({ ...p, minScore: Number(e.target.value) / 100 }))}
                className="flex-1 accent-fuchsia-500"
              />
              <span className="text-sm text-white w-10 text-right">{config.minScore.toFixed(2)}</span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">Engine Selection</label>
            <div className="space-y-2">
              {ENGINES.map(eng => {
                const colors = ENGINE_COLORS[eng] ?? 'text-white/60 bg-white/5 border-white/10';
                const isOn = config.engines.includes(eng);
                return (
                  <button
                    key={eng}
                    onClick={() => toggleEngine(eng)}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg border text-sm transition-all ${
                      isOn ? colors : 'text-white/30 bg-white/5 border-white/5'
                    }`}
                  >
                    <span className="capitalize">{eng}</span>
                    <div className={`w-3.5 h-3.5 rounded-full border-2 flex items-center justify-center ${isOn ? 'border-current' : 'border-white/20'}`}>
                      {isOn && <div className="w-1.5 h-1.5 rounded-full bg-current" />}
                    </div>
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* ── Right Panel: Query + Results ── */}
        <div className="flex-1 overflow-hidden flex flex-col">
          {/* Query Input */}
          <div className="p-5 border-b border-white/10 flex-shrink-0">
            <label className="block text-xs font-semibold text-white/50 uppercase tracking-wider mb-2">
              Test Query / Agent Prompt <span className="text-white/30 normal-case font-normal">(Ctrl+Enter to run)</span>
            </label>
            <div className="flex gap-3">
              <textarea
                ref={textareaRef}
                value={query}
                onChange={e => setQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Enter what the agent is asking — e.g., 'What are the user's current goals and recent activity?'"
                rows={3}
                className="flex-1 px-4 py-3 bg-white/5 border border-white/10 rounded-xl text-sm text-white placeholder:text-white/30 resize-none focus:outline-none focus:ring-2 focus:ring-fuchsia-500/40"
              />
              <button
                onClick={handleRun}
                disabled={!query.trim() || debugMutation.isPending}
                className="px-5 bg-fuchsia-600 hover:bg-fuchsia-500 disabled:bg-white/10 disabled:text-white/30 text-white rounded-xl font-medium flex items-center justify-center gap-2 transition-colors"
              >
                {debugMutation.isPending
                  ? <Loader2 className="w-5 h-5 animate-spin" />
                  : <Play className="w-5 h-5" />
                }
                <span className="text-sm">{debugMutation.isPending ? 'Running...' : 'Run'}</span>
              </button>
            </div>
            {debugMutation.isError && (
              <div className="mt-2 flex items-center gap-2 text-sm text-red-400">
                <XCircle className="w-4 h-4" />
                Failed to debug context. Check console for details.
              </div>
            )}
          </div>

          {/* Results Panel */}
          <div className="flex-1 overflow-y-auto">
            {!result && !debugMutation.isPending && (
              <div className="flex flex-col items-center justify-center h-full text-center px-8">
                <div className="w-16 h-16 rounded-2xl bg-fuchsia-500/10 flex items-center justify-center mb-4">
                  <Database className="w-8 h-8 text-fuchsia-400" />
                </div>
                <p className="text-white/50 text-sm max-w-xs">
                  Run a query to inspect the exact context payload injected into your LLM — per engine, per memory type.
                </p>
              </div>
            )}

            {debugMutation.isPending && (
              <div className="flex flex-col items-center justify-center h-full gap-4">
                <Loader2 className="w-10 h-10 text-fuchsia-400 animate-spin" />
                <p className="text-white/40 text-sm">Retrieving context from {config.engines.length} engines...</p>
              </div>
            )}

            {result && (
              <div className="p-5 space-y-4">
                {/* Summary row */}
                <div className="flex items-center gap-4 flex-wrap">
                  <div className="flex items-center gap-2 px-3 py-1.5 bg-green-500/10 border border-green-500/20 rounded-full text-xs text-green-400">
                    <CheckCircle className="w-3.5 h-3.5" />
                    {result.totalLatencyMs}ms total
                  </div>
                  <div className="flex items-center gap-2 px-3 py-1.5 bg-blue-500/10 border border-blue-500/20 rounded-full text-xs text-blue-400">
                    <Layers className="w-3.5 h-3.5" />
                    {result.totalTokens.toLocaleString()} / {result.promptTokenBudget.toLocaleString()} tokens
                  </div>
                  <div className="flex items-center gap-2 px-3 py-1.5 bg-white/5 border border-white/10 rounded-full text-xs text-white/50">
                    <Database className="w-3.5 h-3.5" />
                    {result.results.reduce((s, r) => s + r.items.length, 0)} items from {result.results.length} engines
                  </div>
                </div>

                {/* Tab selector */}
                <div className="flex gap-1 border-b border-white/10 pb-0">
                  {(['results', 'assembled', 'raw'] as const).map(tab => (
                    <button
                      key={tab}
                      onClick={() => setActiveTab(tab)}
                      className={`px-4 py-2 text-sm transition-colors rounded-t-lg capitalize ${
                        activeTab === tab
                          ? 'text-fuchsia-400 bg-fuchsia-500/10 border-b-2 border-fuchsia-400'
                          : 'text-white/50 hover:text-white/70 hover:bg-white/5'
                      }`}
                    >
                      {tab === 'results' ? 'Engine Results' : tab === 'assembled' ? 'Assembled Context' : 'Raw JSON'}
                    </button>
                  ))}
                </div>

                {/* Results tab */}
                {activeTab === 'results' && (
                  <div className="space-y-3">
                    {result.results.map(r => {
                      const colors = ENGINE_COLORS[r.engine] ?? '';
                      const isExpanded = expandedEngines.has(r.engine);
                      return (
                        <div key={r.engine} className={`border rounded-xl overflow-hidden ${
                          colors.includes('bg-') ? colors.split(' ').find(c => c.startsWith('border')) ?? 'border-white/10' : 'border-white/10'
                        }`}>
                          <button
                            onClick={() => toggleExpandEngine(r.engine)}
                            className="w-full flex items-center justify-between p-4 hover:bg-white/5 transition-colors"
                          >
                            <div className="flex items-center gap-3">
                              {isExpanded ? <ChevronDown className="w-4 h-4 text-white/40" /> : <ChevronRight className="w-4 h-4 text-white/40" />}
                              <span className={`font-medium capitalize text-sm ${colors.split(' ')[0]}`}>{r.engine}</span>
                              <span className="text-xs text-white/30">{r.items.length} items</span>
                            </div>
                            <div className="flex items-center gap-4 text-xs text-white/40">
                              <span className="flex items-center gap-1"><Clock className="w-3.5 h-3.5" />{r.latencyMs}ms</span>
                              <span className="flex items-center gap-1"><Cpu className="w-3.5 h-3.5" />{r.totalTokens} tok</span>
                            </div>
                          </button>
                          {isExpanded && (
                            <div className="border-t border-white/5 divide-y divide-white/5">
                              {r.items.map(item => (
                                <div key={item.id} className="p-4 hover:bg-white/5">
                                  <div className="flex items-start justify-between gap-4 mb-2">
                                    <p className="text-sm text-white/80 flex-1">{item.content}</p>
                                    <span className={`flex-shrink-0 text-xs font-medium px-2 py-0.5 rounded ${
                                      item.score >= 0.9 ? 'bg-green-500/20 text-green-400' :
                                      item.score >= 0.7 ? 'bg-yellow-500/20 text-yellow-400' :
                                      'bg-white/10 text-white/40'
                                    }`}>
                                      {item.score.toFixed(2)}
                                    </span>
                                  </div>
                                  <div className="flex items-center gap-3 text-xs text-white/30">
                                    <span className="px-1.5 py-0.5 bg-white/5 rounded">{item.type}</span>
                                    <span className="font-mono truncate">{item.source}</span>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Assembled Context tab */}
                {activeTab === 'assembled' && (
                  <div className="relative">
                    <button
                      onClick={() => copyToClipboard(result.assembledContext)}
                      className="absolute top-3 right-3 p-2 bg-white/10 hover:bg-white/20 rounded text-white/60 hover:text-white"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                    <pre className="p-5 bg-black/40 rounded-xl border border-white/10 text-xs font-mono text-white/80 whitespace-pre-wrap leading-relaxed overflow-x-auto">
                      {result.assembledContext}
                    </pre>
                    <div className="mt-2 flex items-center gap-2 text-xs text-white/30">
                      <Info className="w-3.5 h-3.5" />
                      Token usage: {result.totalTokens.toLocaleString()} / {result.promptTokenBudget.toLocaleString()} ({Math.round(result.totalTokens/result.promptTokenBudget*100)}% budget used)
                    </div>
                  </div>
                )}

                {/* Raw JSON tab */}
                {activeTab === 'raw' && (
                  <div className="relative">
                    <button
                      onClick={() => copyToClipboard(JSON.stringify(result, null, 2))}
                      className="absolute top-3 right-3 p-2 bg-white/10 hover:bg-white/20 rounded text-white/60 hover:text-white"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                    <pre className="p-5 bg-black/40 rounded-xl border border-white/10 text-xs font-mono text-green-300/80 whitespace-pre overflow-x-auto max-h-[60vh]">
                      {JSON.stringify(result, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
