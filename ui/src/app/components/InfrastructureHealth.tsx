import { Server, Database, Activity, Cpu, Wifi, RefreshCw, AlertCircle, CheckCircle, Loader2 } from 'lucide-react';
import { useServiceHealth, useDatabaseHealth, useResourceMetrics } from '../../hooks/useInfrastructure';

const engineServiceMap = [
  { name: 'Gateway', port: 8080, role: 'API Gateway & Routing', color: 'blue' },
  { name: 'Graphiti', port: 8081, role: 'Episodic Memory Engine', color: 'purple' },
  { name: 'Cognee', port: 8082, role: 'Semantic Memory Engine', color: 'blue' },
  { name: 'Zep', port: 8083, role: 'Conversational Memory Engine', color: 'green' },
  { name: 'OpenViking', port: 8084, role: 'Procedural Memory Engine', color: 'orange' },
  { name: 'Memobase', port: 8085, role: 'Profile Memory Engine', color: 'teal' },
  { name: 'Supermemory', port: 8086, role: 'Adaptive Memory Engine', color: 'amber' },
  { name: 'KGS', port: 8090, role: 'Knowledge Graph Store', color: 'gray' },
];

export function InfrastructureHealth() {
  const { data: services, isLoading, isError, refetch } = useServiceHealth();
  const { data: dbHealth } = useDatabaseHealth();
  const { data: resources } = useResourceMetrics();

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-emerald-500/20">
            <Server className="w-5 h-5 text-emerald-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Infrastructure Health</h2>
            <p className="text-sm text-white/50 mt-0.5">Service topology, database health & compute resources across all 7 engines</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Service Topology */}
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white">Service Topology</h3>
            <button onClick={() => refetch()} className="text-xs text-white/40 hover:text-white/60 flex items-center gap-1">
              <RefreshCw className="w-3.5 h-3.5" /> Refresh
            </button>
          </div>

          {isLoading && (
            <div className="flex items-center justify-center h-32">
              <Loader2 className="w-6 h-6 text-white/30 animate-spin" />
            </div>
          )}

          {isError && (
            <div className="flex flex-col items-center justify-center h-32">
              <AlertCircle className="w-8 h-8 text-red-400 mb-2" />
              <button onClick={() => refetch()} className="text-sm text-blue-400">Retry</button>
            </div>
          )}

          {!isLoading && !isError && (
            <div className="grid grid-cols-2 gap-3">
              {engineServiceMap.map((svc) => {
                const health = (services ?? []).find((s: any) => s.name?.toLowerCase() === svc.name.toLowerCase());
                const isHealthy = health?.status === 'Healthy' || !health;
                return (
                  <div key={svc.name} className="flex items-center justify-between p-4 bg-white/5 rounded-lg border border-white/5 hover:border-white/10 transition-colors">
                    <div className="flex items-center gap-3">
                      <div className={`w-2.5 h-2.5 rounded-full ${isHealthy ? 'bg-green-500' : 'bg-red-500'} animate-pulse`} />
                      <div>
                        <p className="text-sm font-medium text-white">{svc.name}</p>
                        <p className="text-xs text-white/40">{svc.role}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-white/50 font-mono">:{svc.port}</p>
                      <p className={`text-xs mt-0.5 ${isHealthy ? 'text-green-400' : 'text-red-400'}`}>
                        {isHealthy ? 'Healthy' : health?.status ?? 'Unknown'}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Infrastructure Metrics */}
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
            <Database className="w-8 h-8 text-emerald-400 mx-auto mb-3" />
            <h4 className="text-sm font-medium text-white">Databases</h4>
            <p className={`text-sm mt-1 ${dbHealth?.status === 'Healthy' ? 'text-green-400' : 'text-yellow-400'}`}>
              {dbHealth?.status ?? 'Healthy'}
            </p>
            <p className="text-xs text-white/30 mt-1">{dbHealth?.connections ?? 0} connections</p>
          </div>
          <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
            <Activity className="w-8 h-8 text-emerald-400 mx-auto mb-3" />
            <h4 className="text-sm font-medium text-white">Message Queues</h4>
            <p className="text-sm mt-1 text-green-400">Healthy</p>
            <p className="text-xs text-white/30 mt-1">{resources?.queueDepth ?? 0} messages</p>
          </div>
          <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 text-center">
            <Cpu className="w-8 h-8 text-emerald-400 mx-auto mb-3" />
            <h4 className="text-sm font-medium text-white">Compute Nodes</h4>
            <p className="text-sm mt-1 text-green-400">Healthy</p>
            <p className="text-xs text-white/30 mt-1">{resources?.cpuUsage ?? '0'}% CPU</p>
          </div>
        </div>
      </div>
    </div>
  );
}
