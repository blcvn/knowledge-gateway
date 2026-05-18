import { useState } from 'react';
import { Search, Filter, Clock, Database, MessageSquare, Folder, UserCircle, Sparkles, RefreshCcw, History, Tag } from 'lucide-react';
import { useMemorySearch } from '../../hooks/useMemory';
import { MemoryItem } from '../../types/memory';

const memoryTypes = [
  { id: 'all', label: 'All', icon: null, color: 'gray' },
  { id: 'episodic', label: 'Episodic', icon: Clock, color: 'purple' },
  { id: 'semantic', label: 'Semantic', icon: Database, color: 'blue' },
  { id: 'conversational', label: 'Conversational', icon: MessageSquare, color: 'green' },
  { id: 'procedural', label: 'Procedural', icon: Folder, color: 'orange' },
  { id: 'profile', label: 'Profile (Memobase)', icon: UserCircle, color: 'teal' },
  { id: 'adaptive', label: 'Adaptive (Supermem)', icon: Sparkles, color: 'amber' },
];

export function MemoryExplorer() {
  const [activeTab, setActiveTab] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedMemory, setSelectedMemory] = useState<MemoryItem | null>(null);

  const { data: searchResult, isLoading, error, refetch } = useMemorySearch({
    query: searchQuery,
    type: activeTab !== 'all' ? activeTab : undefined,
  });

  const memories = searchResult?.results || [];
  const facets = searchResult?.facets?.byType || {};

  return (
    <div className="flex-1 overflow-hidden flex">
      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0 border-r border-white/10">
        {/* Header */}
        <div className="p-6 border-b border-white/10">
          <h2 className="text-2xl font-semibold text-white">Memory Explorer</h2>
          <p className="text-sm text-white/50 mt-1">Unified search across all memory types</p>
        </div>

        {/* Search & Filters */}
        <div className="p-6 space-y-4 border-b border-white/10">
          <div className="flex gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
              <input
                type="text"
                placeholder="Semantic search, hybrid search, or graph query..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
              />
            </div>
            <button className="flex items-center gap-2 px-4 py-2.5 bg-white/5 border border-white/10 rounded-lg hover:bg-white/10 transition-colors">
              <Filter className="w-4 h-4 text-white/70" />
              <span className="text-sm text-white/70">Filters</span>
            </button>
          </div>

          {/* Memory Type Tabs */}
          <div className="flex items-center gap-2 overflow-x-auto pb-2 scrollbar-hide">
            {memoryTypes.map((type) => {
              const Icon = type.icon;
              const isActive = activeTab === type.id;
              const count = type.id === 'all' ? searchResult?.total || 0 : facets[type.id] || 0;
              
              return (
                <button
                  key={type.id}
                  onClick={() => setActiveTab(type.id)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm transition-colors whitespace-nowrap ${
                    isActive
                      ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                      : 'bg-white/5 text-white/70 border border-white/10 hover:bg-white/10'
                  }`}
                >
                  {Icon && <Icon className="w-4 h-4" />}
                  <span>{type.label}</span>
                  <span className="ml-1 px-2 py-0.5 bg-white/10 rounded-full text-xs">{count}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Results */}
        <div className="flex-1 overflow-y-auto p-6">
          {isLoading ? (
            <div className="space-y-4 animate-pulse">
              {[1, 2, 3].map((i) => (
                <div key={i} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 h-32"></div>
              ))}
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center h-full space-y-4">
              <div className="text-red-500">Failed to load memories.</div>
              <button onClick={() => refetch()} className="flex items-center gap-2 px-4 py-2 bg-white/5 border border-white/10 rounded-lg hover:bg-white/10">
                <RefreshCcw className="w-4 h-4" /> Retry
              </button>
            </div>
          ) : memories.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-white/50">
              <Database className="w-12 h-12 mb-4 opacity-20" />
              <p>No memories found matching your criteria.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {memories.map((memory) => {
                const typeInfo = memoryTypes.find(t => t.id === memory.memoryType) || memoryTypes[0];
                const TypeIcon = typeInfo.icon || Database;
                const typeColor = typeInfo.color;

                return (
                  <div 
                    key={memory.id} 
                    onClick={() => setSelectedMemory(memory)}
                    className={`bg-[#1a1a1f] border rounded-xl p-5 transition-all cursor-pointer ${
                      selectedMemory?.id === memory.id ? 'border-blue-500/50 bg-blue-500/5' : 'border-white/10 hover:border-white/20'
                    }`}
                  >
                    {/* Header */}
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg ${
                          typeColor === 'purple' ? 'bg-purple-500/20' :
                          typeColor === 'blue' ? 'bg-blue-500/20' :
                          typeColor === 'green' ? 'bg-green-500/20' :
                          typeColor === 'orange' ? 'bg-orange-500/20' :
                          typeColor === 'teal' ? 'bg-teal-500/20' :
                          typeColor === 'amber' ? 'bg-amber-500/20' :
                          'bg-gray-500/20'
                        }`}>
                          <TypeIcon className={`w-4 h-4 ${
                            typeColor === 'purple' ? 'text-purple-400' :
                            typeColor === 'blue' ? 'text-blue-400' :
                            typeColor === 'green' ? 'text-green-400' :
                            typeColor === 'orange' ? 'text-orange-400' :
                            typeColor === 'teal' ? 'text-teal-400' :
                            typeColor === 'amber' ? 'text-amber-400' :
                            'text-gray-400'
                          }`} />
                        </div>
                        <div>
                          <h3 className="text-base font-medium text-white">{memory.title}</h3>
                          <div className="flex items-center gap-2 mt-1">
                            <span className="text-xs text-white/50">{memory.engine}</span>
                            {memory.versionChain && (
                              <span className="text-xs bg-amber-500/20 text-amber-400 px-1.5 py-0.5 rounded flex items-center gap-1">
                                <History className="w-3 h-3" /> v{memory.versionChain.length}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-green-400">
                          {Math.round(memory.score * 100)}% confidence
                        </span>
                      </div>
                    </div>

                    {/* Summary */}
                    <p className="text-sm text-white/70 mb-3">{memory.summary}</p>

                    {/* Entities */}
                    <div className="flex flex-wrap items-center gap-2">
                      {memory.entities.map((entity, i) => (
                        <span key={i} className="px-2 py-1 bg-white/5 border border-white/10 rounded text-xs text-white/60">
                          {entity}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Side Inspector */}
      {selectedMemory && (
        <div className="w-80 bg-[#141419] flex flex-col overflow-y-auto">
          <div className="p-6 border-b border-white/10 flex items-center justify-between">
            <h3 className="text-lg font-medium text-white">Inspector</h3>
            <button onClick={() => setSelectedMemory(null)} className="text-white/50 hover:text-white">✕</button>
          </div>
          
          <div className="p-6 space-y-6">
            <div>
              <div className="text-xs text-white/40 uppercase font-medium mb-2">Memory ID</div>
              <div className="text-sm text-white font-mono bg-white/5 p-2 rounded">{selectedMemory.id}</div>
            </div>

            <div>
              <div className="text-xs text-white/40 uppercase font-medium mb-2">Engine Engine</div>
              <div className="flex items-center gap-2">
                <Database className="w-4 h-4 text-blue-400" />
                <span className="text-sm text-white capitalize">{selectedMemory.engine}</span>
              </div>
            </div>

            {selectedMemory.temporalValidity && (
              <div>
                <div className="text-xs text-white/40 uppercase font-medium mb-2">Temporal Validity</div>
                <div className="text-sm text-white">From: {selectedMemory.temporalValidity.from}</div>
                {selectedMemory.temporalValidity.to && (
                  <div className="text-sm text-white">To: {selectedMemory.temporalValidity.to}</div>
                )}
              </div>
            )}

            {selectedMemory.policyTags && selectedMemory.policyTags.length > 0 && (
              <div>
                <div className="text-xs text-white/40 uppercase font-medium mb-2">Policy Tags</div>
                <div className="flex flex-wrap gap-2">
                  {selectedMemory.policyTags.map(tag => (
                    <span key={tag} className="flex items-center gap-1 text-xs bg-red-500/20 text-red-400 px-2 py-1 rounded">
                      <Tag className="w-3 h-3" /> {tag}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {selectedMemory.versionChain && (
              <div>
                <div className="text-xs text-white/40 uppercase font-medium mb-2 flex items-center justify-between">
                  Version History
                  <span className="bg-amber-500/20 text-amber-400 px-1.5 py-0.5 rounded text-[10px]">
                    v{selectedMemory.versionChain.length}
                  </span>
                </div>
                <div className="space-y-2 border-l border-white/10 ml-2 pl-4">
                  {selectedMemory.versionChain.map((v: any, i: number) => (
                    <div key={v.version} className="relative">
                      <div className="absolute -left-[21px] top-1.5 w-2 h-2 rounded-full bg-white/20"></div>
                      <div className="text-sm text-white">v{v.version}</div>
                      <div className="text-xs text-white/50">{v.timestamp}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}
            
            <div>
              <div className="text-xs text-white/40 uppercase font-medium mb-2">Full Content</div>
              <div className="text-sm text-white/70 bg-white/5 p-3 rounded-lg whitespace-pre-wrap">
                {selectedMemory.content}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
