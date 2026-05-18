import { GitBranch, Play, CheckCircle2, AlertCircle, Clock, RefreshCw, Loader2 } from 'lucide-react';
import { usePipelineJobs, useQueueMetrics } from '../../hooks/usePipelines';

const statusConfig: Record<string, { icon: React.ComponentType<{ className?: string }>; color: string; bg: string }> = {
  Running: { icon: Play, color: 'text-blue-400', bg: 'bg-blue-500/20' },
  Completed: { icon: CheckCircle2, color: 'text-green-400', bg: 'bg-green-500/20' },
  Failed: { icon: AlertCircle, color: 'text-red-400', bg: 'bg-red-500/20' },
  Queued: { icon: Clock, color: 'text-white/40', bg: 'bg-white/10' },
};

const enginePipelineStages: Record<string, string[]> = {
  cognee: ['Add → Cognify → Search Index'],
  graphiti: ['Ingest → Episode → Entity → Edge'],
  zep: ['Message → Working Memory → Graph → Fact'],
  memobase: ['Blob → Buffer → Extract → Merge → Profile → Cache'],
  supermemory: ['Document → Memory → Version → Search Index'],
};

export function PipelinesMonitor() {
  const { data: jobs, isLoading, isError, refetch } = usePipelineJobs('all');
  const { data: queueMetrics } = useQueueMetrics();

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-indigo-500/20">
            <GitBranch className="w-5 h-5 text-indigo-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Pipelines Monitor</h2>
            <p className="text-sm text-white/50 mt-0.5">Per-engine pipeline stages and job queue status</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Engine Pipeline Stages */}
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Pipeline Stages by Engine</h3>
          <div className="space-y-3">
            {Object.entries(enginePipelineStages).map(([engine, stages]) => (
              <div key={engine} className="flex items-center gap-4 p-3 bg-white/5 rounded-lg border border-white/5">
                <span className="text-sm font-medium text-white w-28 capitalize">{engine}</span>
                <span className="text-xs text-white/50 font-mono">{stages[0]}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Active Jobs */}
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white">Active Jobs</h3>
            <button onClick={() => refetch()} className="text-xs text-white/40 hover:text-white/60 flex items-center gap-1">
              <RefreshCw className="w-3.5 h-3.5" /> Refresh
            </button>
          </div>

          {isLoading && (
            <div className="flex items-center justify-center h-24">
              <Loader2 className="w-6 h-6 text-white/30 animate-spin" />
            </div>
          )}

          {isError && (
            <div className="flex flex-col items-center justify-center h-24">
              <AlertCircle className="w-8 h-8 text-red-400 mb-2" />
              <button onClick={() => refetch()} className="text-sm text-blue-400">Retry</button>
            </div>
          )}

          {!isLoading && !isError && (
            <div className="space-y-3">
              {(jobs ?? []).map((job: any) => {
                const cfg = statusConfig[job.status] ?? statusConfig.Queued;
                const Icon = cfg.icon;
                return (
                  <div key={job.id} className="flex items-center justify-between p-4 bg-white/5 rounded-lg border border-white/5 hover:border-white/10 transition-colors">
                    <div className="flex items-center gap-4">
                      <div className={`p-2 rounded-lg ${cfg.bg}`}>
                        <Icon className={`w-4 h-4 ${cfg.color}`} />
                      </div>
                      <div>
                        <h4 className="text-sm font-medium text-white">{job.name || `Job ${job.id}`}</h4>
                        <p className="text-xs text-white/40 mt-0.5">{job.engine} • {job.description || 'Processing...'}</p>
                      </div>
                    </div>
                    <span className={`text-sm font-medium ${cfg.color}`}>{job.status}</span>
                  </div>
                );
              })}
              {(!jobs || jobs.length === 0) && (
                <div className="text-center py-8 text-white/30 text-sm">No active jobs</div>
              )}
            </div>
          )}
        </div>

        {/* Queue Metrics */}
        {queueMetrics && (
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: 'Pending', value: queueMetrics.pending ?? 0, color: 'text-white' },
              { label: 'Processing', value: queueMetrics.processing ?? 0, color: 'text-blue-400' },
              { label: 'Failed (24h)', value: queueMetrics.failed ?? 0, color: 'text-red-400' },
            ].map((m) => (
              <div key={m.label} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
                <p className="text-xs text-white/40 mb-1">{m.label}</p>
                <p className={`text-2xl font-semibold ${m.color}`}>{m.value}</p>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
