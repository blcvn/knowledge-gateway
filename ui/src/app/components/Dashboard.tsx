import { Activity, Database, Zap, AlertTriangle, Users, MonitorPlay, UserCircle, Sparkles, RefreshCcw } from 'lucide-react';
import { AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { useMetrics, useEngineHealth, useThroughput } from '../../hooks/useDashboard';

const memoryTypeData = [
  { type: 'Episodic', count: 145, color: '#8b5cf6' },
  { type: 'Semantic', count: 312, color: '#3b82f6' },
  { type: 'Conversational', count: 198, color: '#10b981' },
  { type: 'Procedural', count: 87, color: '#f97316' },
];

export function Dashboard() {
  const { data: metrics, isLoading: isLoadingMetrics, error: errorMetrics, refetch: refetchMetrics } = useMetrics();
  const { data: engines, isLoading: isLoadingEngines, error: errorEngines, refetch: refetchEngines } = useEngineHealth();
  const { data: throughput, isLoading: isLoadingThroughput, error: errorThroughput, refetch: refetchThroughput } = useThroughput();

  const handleRetry = () => {
    refetchMetrics();
    refetchEngines();
    refetchThroughput();
  };

  const isLoading = isLoadingMetrics || isLoadingEngines || isLoadingThroughput;
  const error = errorMetrics || errorEngines || errorThroughput;

  if (isLoading) {
    return (
      <div className="flex-1 p-6 space-y-6">
        <div>
          <h2 className="text-2xl font-semibold text-white">Platform Overview</h2>
          <p className="text-sm text-white/50 mt-1">Loading dashboard data...</p>
        </div>
        <div className="grid grid-cols-4 gap-4 animate-pulse">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-4 h-28"></div>
          ))}
        </div>
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6 h-80 animate-pulse"></div>
        <div className="grid grid-cols-2 gap-6">
          <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6 h-64 animate-pulse"></div>
          <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6 h-64 animate-pulse"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-6 space-y-4">
        <AlertTriangle className="w-12 h-12 text-red-500" />
        <h2 className="text-xl font-semibold text-white">Failed to load dashboard data</h2>
        <p className="text-sm text-white/50">There was an issue communicating with the gateway API.</p>
        <button onClick={handleRetry} className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">
          <RefreshCcw className="w-4 h-4" />
          Retry
        </button>
      </div>
    );
  }

  const kpiData = [
    { label: 'Active Agents', value: metrics?.activeAgents || 0, icon: Users, color: 'from-blue-500 to-cyan-500' },
    { label: 'Recall Latency (p50)', value: `${metrics?.recallLatencyP50Ms || 0}ms`, icon: Activity, color: 'from-purple-500 to-pink-500' },
    { label: 'Context Savings', value: `${metrics?.contextSavingsPct || 0}%`, icon: Zap, color: 'from-green-500 to-emerald-500' },
    { label: 'Graph Growth', value: metrics?.graphGrowth24h || 0, icon: Database, color: 'from-orange-500 to-red-500' },
    { label: 'Error Rate', value: `${metrics?.errorRatePct || 0}%`, icon: AlertTriangle, color: 'from-red-500 to-rose-500' },
    { label: 'Active Sessions', value: metrics?.activeSessions || 0, icon: MonitorPlay, color: 'from-indigo-500 to-violet-500' },
    { label: 'Active Profiles', value: metrics?.activeProfiles || 0, icon: UserCircle, color: 'from-teal-500 to-cyan-500' },
    { label: 'Memory Versions', value: metrics?.memoryVersions || 0, icon: Sparkles, color: 'from-amber-500 to-yellow-500' },
  ];

  // Simplified chart parsing for mock compatibility
  const memoryFlowData = throughput?.engines 
    ? ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00'].map((time) => {
        // Just mock some time-series based on the throughput static data 
        // In a real scenario, API would return time-series array directly for chart
        const base = (throughput.engines.memobase?.ingestPerSec || 0) * 10;
        return {
          time,
          ingest: base + Math.random() * 50,
          recall: base * 2 + Math.random() * 100,
          embeddings: base * 1.5 + Math.random() * 50,
          profileExtractions: throughput.engines.memobase?.profileExtractionsPerSec || 0,
        };
      })
    : [];

  return (
    <div className="flex-1 overflow-y-auto p-6 space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-semibold text-white">Platform Overview</h2>
        <p className="text-sm text-white/50 mt-1">Enterprise Cognitive Infrastructure Control Plane</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-4 gap-4">
        {kpiData.map((kpi) => {
          const Icon = kpi.icon;
          return (
            <div key={kpi.label} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-4 hover:border-white/20 transition-colors">
              <div className="flex items-start justify-between mb-3">
                <div className={`p-2.5 rounded-lg bg-gradient-to-br ${kpi.color}`}>
                  <Icon className="w-5 h-5 text-white" />
                </div>
              </div>
              <div className="text-2xl font-semibold text-white mb-1">{kpi.value}</div>
              <div className="text-xs text-white/50">{kpi.label}</div>
            </div>
          );
        })}
      </div>

      {/* Memory Flow Visualization */}
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Memory Flow (24h)</h3>
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={memoryFlowData}>
            <defs>
              <linearGradient id="colorIngest" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
              </linearGradient>
              <linearGradient id="colorRecall" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10b981" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
              </linearGradient>
              <linearGradient id="colorEmbeddings" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" />
            <XAxis dataKey="time" stroke="#ffffff50" />
            <YAxis stroke="#ffffff50" />
            <Tooltip
              contentStyle={{ backgroundColor: '#1a1a1f', border: '1px solid #ffffff20', borderRadius: '8px' }}
              labelStyle={{ color: '#fff' }}
            />
            <Area type="monotone" dataKey="ingest" stroke="#3b82f6" fillOpacity={1} fill="url(#colorIngest)" />
            <Area type="monotone" dataKey="recall" stroke="#10b981" fillOpacity={1} fill="url(#colorRecall)" />
            <Area type="monotone" dataKey="embeddings" stroke="#8b5cf6" fillOpacity={1} fill="url(#colorEmbeddings)" />
          </AreaChart>
        </ResponsiveContainer>
        <div className="flex items-center justify-center gap-6 mt-4">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <span className="text-xs text-white/70">Ingest/sec</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="text-xs text-white/70">Recall/sec</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-purple-500"></div>
            <span className="text-xs text-white/70">Embeddings/sec</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-teal-500"></div>
            <span className="text-xs text-white/70">Profile Extract/sec</span>
          </div>
        </div>
      </div>

      {/* Two Column Layout */}
      <div className="grid grid-cols-2 gap-6">
        {/* Engine Health */}
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Engine Health</h3>
          <div className="space-y-3">
            {engines?.map((engine) => (
              <div key={engine.name} className="flex items-center justify-between p-3 bg-white/5 rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                <div className="flex items-center gap-3">
                  <div className={`w-2.5 h-2.5 rounded-full ${
                    engine.status === 'Healthy' ? 'bg-green-500' :
                    engine.status === 'Warning' ? 'bg-yellow-500' :
                    'bg-red-500'
                  }`}></div>
                  <div>
                    <div className="text-sm font-medium text-white capitalize">{engine.name}</div>
                    <div className="text-xs text-white/40">{engine.role}</div>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs text-white/50">
                  <span>{engine.latencyP50Ms}ms</span>
                  <span className="px-2 py-1 bg-white/10 rounded">Q: {engine.queueDepth}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Memory Type Distribution */}
        <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Memory Type Distribution</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={memoryTypeData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" />
              <XAxis dataKey="type" stroke="#ffffff50" />
              <YAxis stroke="#ffffff50" />
              <Tooltip
                contentStyle={{ backgroundColor: '#1a1a1f', border: '1px solid #ffffff20', borderRadius: '8px' }}
                labelStyle={{ color: '#fff' }}
              />
              <Bar dataKey="count" radius={[8, 8, 0, 0]}>
                {memoryTypeData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
