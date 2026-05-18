import { useState } from 'react';
import { Sparkles, GitBranch, Link2, Timer, BarChart3, Search, RefreshCw, AlertCircle, CheckCircle, XCircle, Wifi, WifiOff, ArrowRight } from 'lucide-react';
import { useAdaptiveMemories, useMemoryVersions, useConnectors, useAdaptiveAnalytics } from '../../hooks/useAdaptiveMemory';

type AdaptiveTab = 'versions' | 'connectors' | 'forget' | 'analytics';

const tabs: { id: AdaptiveTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'versions', label: 'Memory Versions', icon: GitBranch },
  { id: 'connectors', label: 'External Connectors', icon: Link2 },
  { id: 'forget', label: 'Auto-Forget Rules', icon: Timer },
  { id: 'analytics', label: 'Analytics', icon: BarChart3 },
];

export function AdaptiveMemory() {
  const [activeTab, setActiveTab] = useState<AdaptiveTab>('versions');

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-amber-500/20">
            <Sparkles className="w-5 h-5 text-amber-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Adaptive Memory</h2>
            <p className="text-sm text-white/50 mt-0.5">Supermemory — Self-evolving knowledge with version chains, auto-forget & external connectors</p>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="px-6 pt-4 border-b border-white/10">
        <div className="flex items-center gap-1">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2.5 text-sm rounded-t-lg transition-colors ${
                  isActive
                    ? 'bg-amber-500/10 text-amber-400 border-b-2 border-amber-400'
                    : 'text-white/60 hover:text-white/80 hover:bg-white/5'
                }`}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'versions' && <MemoryVersionExplorer />}
        {activeTab === 'connectors' && <ExternalConnectors />}
        {activeTab === 'forget' && <AutoForgetRules />}
        {activeTab === 'analytics' && <AdaptiveAnalyticsView />}
      </div>
    </div>
  );
}

/* ─── Memory Version Explorer ─── */
function MemoryVersionExplorer() {
  const { data: memories, isLoading, isError, refetch } = useAdaptiveMemories();
  const [selectedMemoryId, setSelectedMemoryId] = useState<string | null>(null);

  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  return (
    <div className="grid grid-cols-5 gap-6">
      {/* Memory List */}
      <div className="col-span-2 space-y-3">
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
          <input type="text" placeholder="Search memories..."
            className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-amber-500/50" />
        </div>
        {(memories ?? []).map((memory) => (
          <button key={memory.id} onClick={() => setSelectedMemoryId(memory.id)}
            className={`w-full text-left p-4 rounded-xl border transition-all ${
              selectedMemoryId === memory.id ? 'bg-amber-500/10 border-amber-500/30' : 'bg-[#1a1a1f] border-white/10 hover:border-white/20'
            }`}>
            <div className="flex items-start justify-between mb-2">
              <p className="text-sm text-white line-clamp-2">{memory.content}</p>
              {memory.is_latest && (
                <span className="ml-2 flex-shrink-0 px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">latest</span>
              )}
            </div>
            <div className="flex items-center gap-2 text-xs">
              <span className={`px-2 py-0.5 rounded ${memory.memory_type === 'static' ? 'bg-blue-500/20 text-blue-400' : 'bg-amber-500/20 text-amber-400'}`}>
                {memory.memory_type}
              </span>
              {memory.relation_type && (
                <span className="text-white/30">{memory.relation_type}</span>
              )}
            </div>
          </button>
        ))}
        {(!memories || memories.length === 0) && <EmptyState message="No adaptive memories found" />}
      </div>

      {/* Version Chain Detail */}
      <div className="col-span-3">
        {selectedMemoryId ? (
          <VersionChainView memoryId={selectedMemoryId} />
        ) : (
          <div className="flex flex-col items-center justify-center h-64 text-center">
            <GitBranch className="w-12 h-12 text-white/20 mb-3" />
            <p className="text-white/40 text-sm">Select a memory to view version chain</p>
          </div>
        )}
      </div>
    </div>
  );
}

/* ─── Version Chain View ─── */
function VersionChainView({ memoryId }: { memoryId: string }) {
  const { data: versions, isLoading } = useMemoryVersions(memoryId);
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
      <h3 className="text-lg font-semibold text-white mb-6">Version History</h3>
      <div className="relative pl-6">
        <div className="absolute left-2.5 top-0 bottom-0 w-px bg-amber-500/30" />
        {(versions ?? []).map((version, index) => (
          <div key={version.id} className="relative mb-6">
            <div className={`absolute -left-3.5 w-3 h-3 rounded-full border-2 border-[#1a1a1f] ${
              version.is_latest ? 'bg-amber-500' : 'bg-white/30'
            }`} />
            <div className="ml-4 p-4 bg-white/5 rounded-lg border border-white/10">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-medium text-amber-400">v{version.version_number}</span>
                  {version.is_latest && (
                    <span className="px-1.5 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">current</span>
                  )}
                </div>
                <span className="text-xs text-white/30">{version.created_at}</span>
              </div>
              <p className="text-sm text-white/80">{version.content}</p>
              {version.diff && (
                <div className="mt-3 p-2 bg-black/40 rounded text-xs font-mono text-white/50 border border-white/5">
                  <span className="text-amber-400">diff: </span>{version.diff}
                </div>
              )}
            </div>
            {index < (versions?.length ?? 0) - 1 && (
              <div className="absolute left-2 top-full -translate-x-1/2 py-1">
                <ArrowRight className="w-3 h-3 text-amber-500/50 rotate-90" />
              </div>
            )}
          </div>
        ))}
      </div>
      {(!versions || versions.length === 0) && <EmptyState message="No version history available" />}
    </div>
  );
}

/* ─── External Connectors ─── */
function ExternalConnectors() {
  const { data: connectors, isLoading, isError, refetch } = useConnectors();
  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  const connectorIcons: Record<string, string> = {
    google_drive: '📁',
    gmail: '📧',
    notion: '📝',
    onedrive: '☁️',
    github: '🐙',
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-lg font-semibold text-white">External Data Connectors</h3>
        <button className="px-4 py-2 bg-amber-500/20 text-amber-400 rounded-lg text-sm hover:bg-amber-500/30 transition-colors">
          + Add Connector
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {(connectors ?? []).map((connector) => (
          <div key={connector.id} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 hover:border-white/20 transition-all">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <span className="text-2xl">{connectorIcons[connector.type] ?? '🔗'}</span>
                <div>
                  <h4 className="text-sm font-medium text-white capitalize">{connector.type.replace('_', ' ')}</h4>
                  <p className="text-xs text-white/40 mt-0.5">
                    {connector.last_sync ? `Last sync: ${connector.last_sync}` : 'Never synced'}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="text-sm font-semibold text-white">{connector.document_count}</p>
                  <p className="text-xs text-white/40">documents</p>
                </div>
                <div className="flex items-center gap-1.5">
                  {connector.status === 'Connected' ? (
                    <span className="flex items-center gap-1 px-2.5 py-1 bg-green-500/20 text-green-400 rounded-full text-xs">
                      <Wifi className="w-3 h-3" /> Connected
                    </span>
                  ) : connector.status === 'Error' ? (
                    <span className="flex items-center gap-1 px-2.5 py-1 bg-red-500/20 text-red-400 rounded-full text-xs">
                      <XCircle className="w-3 h-3" /> Error
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 px-2.5 py-1 bg-white/10 text-white/40 rounded-full text-xs">
                      <WifiOff className="w-3 h-3" /> Disconnected
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {connector.status === 'Connected' && (
                    <button className="px-3 py-1.5 bg-white/5 text-white/60 rounded-lg text-xs hover:bg-white/10">Sync Now</button>
                  )}
                  <button className="px-3 py-1.5 bg-white/5 text-white/60 rounded-lg text-xs hover:bg-white/10">
                    {connector.status === 'Disconnected' ? 'Connect' : 'Settings'}
                  </button>
                </div>
              </div>
            </div>
            {connector.error_message && (
              <div className="mt-3 p-2 bg-red-500/10 border border-red-500/20 rounded text-xs text-red-400">
                {connector.error_message}
              </div>
            )}
          </div>
        ))}
      </div>
      {(!connectors || connectors.length === 0) && <EmptyState message="No connectors configured" />}
    </div>
  );
}

/* ─── Auto-Forget Rules ─── */
function AutoForgetRules() {
  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Auto-Forget Configuration</h3>
        <p className="text-sm text-white/50 mb-6">Configure memory decay policies for automatic cleanup.</p>

        <div className="space-y-4">
          {[
            { type: 'Static Memories', duration: '90d', noise: true, resolution: 'keep_latest', color: 'blue' },
            { type: 'Dynamic Memories', duration: '30d', noise: true, resolution: 'keep_both', color: 'amber' },
          ].map((rule) => (
            <div key={rule.type} className="p-5 bg-white/5 rounded-lg border border-white/10">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <Timer className={`w-5 h-5 ${rule.color === 'blue' ? 'text-blue-400' : 'text-amber-400'}`} />
                  <h4 className="text-sm font-medium text-white">{rule.type}</h4>
                </div>
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                  rule.color === 'blue' ? 'bg-blue-500/20 text-blue-400' : 'bg-amber-500/20 text-amber-400'
                }`}>{rule.color === 'blue' ? 'Static' : 'Dynamic'}</span>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <p className="text-xs text-white/40 mb-1">Forget After</p>
                  <div className="flex items-center gap-2">
                    <input type="text" defaultValue={rule.duration}
                      className="w-20 px-3 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white focus:outline-none focus:ring-1 focus:ring-amber-500/50" />
                    <span className="text-xs text-white/30">(e.g., 30d, 90d)</span>
                  </div>
                </div>
                <div>
                  <p className="text-xs text-white/40 mb-1">Noise Filtering</p>
                  <button className={`w-10 h-5 rounded-full relative ${rule.noise ? 'bg-amber-500' : 'bg-white/20'}`}>
                    <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm ${rule.noise ? 'right-0.5' : 'left-0.5'}`} />
                  </button>
                </div>
                <div>
                  <p className="text-xs text-white/40 mb-1">Contradiction Resolution</p>
                  <select defaultValue={rule.resolution}
                    className="px-3 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white focus:outline-none focus:ring-1 focus:ring-amber-500/50">
                    <option value="keep_latest">Keep Latest</option>
                    <option value="keep_both">Keep Both</option>
                    <option value="manual">Manual Review</option>
                  </select>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Save Button */}
      <div className="flex justify-end">
        <button className="px-6 py-2.5 bg-amber-500 text-black font-medium rounded-lg hover:bg-amber-400 transition-colors">
          Save Rules
        </button>
      </div>
    </div>
  );
}

/* ─── Analytics ─── */
function AdaptiveAnalyticsView() {
  const { data: analytics, isLoading } = useAdaptiveAnalytics();
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-4 gap-4">
        {[
          { label: 'Creation Rate', value: `${analytics?.creation_rate ?? 0}/day`, color: 'text-green-400' },
          { label: 'Deletion Rate', value: `${analytics?.deletion_rate ?? 0}/day`, color: 'text-red-400' },
          { label: 'Contradictions', value: String(analytics?.contradiction_count ?? 0), color: 'text-amber-400' },
          { label: 'Connector Syncs', value: String(analytics?.connector_sync_count ?? 0), color: 'text-blue-400' },
        ].map((stat) => (
          <div key={stat.label} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
            <p className="text-xs text-white/40 mb-2">{stat.label}</p>
            <p className={`text-2xl font-semibold ${stat.color}`}>{stat.value}</p>
          </div>
        ))}
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Storage Usage</h3>
        <div className="h-4 bg-white/5 rounded-full overflow-hidden">
          <div className="h-full bg-gradient-to-r from-amber-500 to-orange-500 rounded-full"
            style={{ width: `${Math.min(((analytics?.storage_usage_bytes ?? 0) / (1024 * 1024 * 100)) * 100, 100)}%` }} />
        </div>
        <p className="text-xs text-white/40 mt-2">
          {((analytics?.storage_usage_bytes ?? 0) / (1024 * 1024)).toFixed(1)} MB used
        </p>
      </div>
    </div>
  );
}

/* ─── Shared UI ─── */
function LoadingSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      {[1, 2, 3].map((i) => (
        <div key={i} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5">
          <div className="h-4 bg-white/10 rounded w-1/3 mb-3" />
          <div className="h-3 bg-white/5 rounded w-full mb-2" />
          <div className="h-3 bg-white/5 rounded w-2/3" />
        </div>
      ))}
    </div>
  );
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center h-48 text-center">
      <AlertCircle className="w-10 h-10 text-red-400 mb-3" />
      <p className="text-white/60 mb-3">Failed to load data</p>
      <button onClick={onRetry} className="flex items-center gap-2 px-4 py-2 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30">
        <RefreshCw className="w-4 h-4" /> Retry
      </button>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-32 text-center">
      <Sparkles className="w-8 h-8 text-white/15 mb-2" />
      <p className="text-white/40 text-sm">{message}</p>
    </div>
  );
}
