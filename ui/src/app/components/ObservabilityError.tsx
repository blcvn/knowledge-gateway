import { useState } from 'react';
import { Activity, Bug, BarChart3, Clock, Search, RefreshCw, AlertCircle, Loader2, ChevronRight } from 'lucide-react';
import { useMetricsDashboard, useTraces, useErrors } from '../../hooks/useObservability';

type ObsTab = 'metrics' | 'tracing' | 'errors';

const tabs: { id: ObsTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'metrics', label: 'System Metrics', icon: BarChart3 },
  { id: 'tracing', label: 'Distributed Tracing', icon: Clock },
  { id: 'errors', label: 'Error Tracking', icon: Bug },
];

export function ObservabilityError() {
  const [activeTab, setActiveTab] = useState<ObsTab>('metrics');

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-blue-500/20">
            <Activity className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Observability & Errors</h2>
            <p className="text-sm text-white/50 mt-0.5">Metrics, distributed tracing, and error tracking across all engines</p>
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
              <button key={tab.id} onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2.5 text-sm rounded-t-lg transition-colors ${
                  isActive ? 'bg-blue-500/10 text-blue-400 border-b-2 border-blue-400' : 'text-white/60 hover:text-white/80 hover:bg-white/5'
                }`}>
                <Icon className="w-4 h-4" />{tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'metrics' && <MetricsDashboard />}
        {activeTab === 'tracing' && <TracingView />}
        {activeTab === 'errors' && <ErrorTracking />}
      </div>
    </div>
  );
}

/* ─── System Metrics ─── */
function MetricsDashboard() {
  const { data: metrics, isLoading } = useMetricsDashboard();
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-4 gap-4">
        {[
          { label: 'Avg Latency', value: `${metrics?.avgLatency ?? 0}ms`, color: metrics?.avgLatency < 200 ? 'text-green-400' : 'text-yellow-400' },
          { label: 'Request Rate', value: `${metrics?.requestRate ?? 0}/s`, color: 'text-blue-400' },
          { label: 'Error Rate', value: `${metrics?.errorRate ?? 0}%`, color: metrics?.errorRate < 1 ? 'text-green-400' : 'text-red-400' },
          { label: 'Uptime', value: `${metrics?.uptime ?? '99.99'}%`, color: 'text-green-400' },
        ].map((stat) => (
          <div key={stat.label} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
            <p className="text-xs text-white/40 mb-2">{stat.label}</p>
            <p className={`text-2xl font-semibold ${stat.color}`}>{stat.value}</p>
          </div>
        ))}
      </div>

      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Engine Response Times</h3>
        <div className="space-y-3">
          {[
            { engine: 'Cognee', latency: 45, color: 'blue' },
            { engine: 'Graphiti', latency: 78, color: 'purple' },
            { engine: 'Zep', latency: 32, color: 'green' },
            { engine: 'OpenViking', latency: 120, color: 'orange' },
            { engine: 'Memobase', latency: 55, color: 'teal' },
            { engine: 'Supermemory', latency: 89, color: 'amber' },
          ].map((e) => (
            <div key={e.engine} className="flex items-center gap-4 p-3 bg-white/5 rounded-lg">
              <span className="text-sm text-white w-28">{e.engine}</span>
              <div className="flex-1 h-2 bg-white/5 rounded-full overflow-hidden">
                <div className={`h-full bg-${e.color}-500 rounded-full`} style={{ width: `${Math.min((e.latency / 200) * 100, 100)}%` }} />
              </div>
              <span className={`text-xs w-16 text-right ${e.latency < 100 ? 'text-green-400' : 'text-yellow-400'}`}>{e.latency}ms</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ─── Tracing View ─── */
function TracingView() {
  const { data: traces, isLoading, isError, refetch } = useTraces({});
  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
        <input type="text" placeholder="Search by trace ID or operation..."
          className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-blue-500/50" />
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-5 gap-4 p-4 bg-white/5 text-xs font-medium text-white/50 uppercase">
          <span>Trace ID</span><span>Operation</span><span>Service</span><span>Duration</span><span>Status</span>
        </div>
        {(traces ?? []).map((trace: any) => (
          <div key={trace.id} className="grid grid-cols-5 gap-4 p-4 border-t border-white/5 text-sm hover:bg-white/5 cursor-pointer group">
            <span className="text-blue-400 font-mono text-xs">{trace.id}</span>
            <span className="text-white/70">{trace.operation}</span>
            <span className="text-white/50">{trace.service}</span>
            <span className={`${trace.duration < 200 ? 'text-green-400' : 'text-yellow-400'}`}>{trace.duration}ms</span>
            <div className="flex items-center justify-between">
              <span className={`text-xs ${trace.status === 'ok' ? 'text-green-400' : 'text-red-400'}`}>{trace.status}</span>
              <ChevronRight className="w-4 h-4 text-white/20 opacity-0 group-hover:opacity-100" />
            </div>
          </div>
        ))}
      </div>
      {(!traces || traces.length === 0) && <EmptyState message="No traces found" />}
    </div>
  );
}

/* ─── Error Tracking ─── */
function ErrorTracking() {
  const { data: errors, isLoading, isError, refetch } = useErrors({});
  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
        <input type="text" placeholder="Search errors..."
          className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-red-500/50" />
      </div>
      <div className="space-y-3">
        {(errors ?? []).map((err: any) => (
          <div key={err.id} className="bg-[#1a1a1f] border border-red-500/20 rounded-xl p-5 hover:border-red-500/30 cursor-pointer">
            <div className="flex items-start justify-between mb-2">
              <div className="flex items-center gap-2">
                <Bug className="w-4 h-4 text-red-400" />
                <h4 className="text-sm font-medium text-white">{err.message}</h4>
              </div>
              <span className="text-xs text-red-400 bg-red-500/20 px-2 py-0.5 rounded">{err.count}x</span>
            </div>
            <pre className="p-3 bg-black/40 rounded text-xs font-mono text-red-300/70 border border-white/5 overflow-x-auto mt-2">
              {err.stack ?? 'No stack trace available'}
            </pre>
            <div className="flex items-center gap-4 mt-3 text-xs text-white/40">
              <span>Service: {err.service}</span>
              <span>Last: {err.lastOccurrence}</span>
            </div>
          </div>
        ))}
      </div>
      {(!errors || errors.length === 0) && <EmptyState message="No errors — all systems operating normally" />}
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
      <button onClick={onRetry} className="flex items-center gap-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg hover:bg-blue-500/30">
        <RefreshCw className="w-4 h-4" /> Retry
      </button>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-32 text-center">
      <Activity className="w-8 h-8 text-white/15 mb-2" />
      <p className="text-white/40 text-sm">{message}</p>
    </div>
  );
}
