import { useState } from 'react';
import { Shield, FileText, ScrollText, Timer, Search, RefreshCw, AlertCircle, Eye, Trash2 } from 'lucide-react';
import { useTenants, usePolicies, useAuditLogs } from '../../hooks/useGovernance';

type GovTab = 'gdpr' | 'policies' | 'audit' | 'retention';

const tabs: { id: GovTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'gdpr', label: 'GDPR Forget Center', icon: Trash2 },
  { id: 'policies', label: 'OPA Policy Editor', icon: FileText },
  { id: 'audit', label: 'Audit Explorer', icon: ScrollText },
  { id: 'retention', label: 'Data Retention', icon: Timer },
];

export function GovernanceCenter() {
  const [activeTab, setActiveTab] = useState<GovTab>('audit');

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-purple-500/20">
            <Shield className="w-5 h-5 text-purple-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Governance Center</h2>
            <p className="text-sm text-white/50 mt-0.5">GDPR compliance, policy management, audit trail & data retention</p>
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
                  isActive ? 'bg-purple-500/10 text-purple-400 border-b-2 border-purple-400' : 'text-white/60 hover:text-white/80 hover:bg-white/5'
                }`}>
                <Icon className="w-4 h-4" />{tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'gdpr' && <GDPRForgetCenter />}
        {activeTab === 'policies' && <OPAPolicyEditor />}
        {activeTab === 'audit' && <AuditExplorer />}
        {activeTab === 'retention' && <DataRetention />}
      </div>
    </div>
  );
}

/* ─── GDPR Forget Center ─── */
function GDPRForgetCenter() {
  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-2">Right to Erasure</h3>
        <p className="text-sm text-white/50 mb-6">Cascade delete user data and memories across all 6 engines.</p>
        <div className="flex gap-4">
          <input type="text" placeholder="Enter User ID or email..."
            className="flex-1 px-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-red-500/50" />
          <button className="px-6 py-2.5 bg-red-500/20 text-red-400 rounded-lg text-sm hover:bg-red-500/30 border border-red-500/30">
            <Trash2 className="w-4 h-4 inline mr-2" />Request Deletion
          </button>
        </div>
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Deletion Cascade Scope</h3>
        <div className="grid grid-cols-3 gap-4">
          {[
            { engine: 'Graphiti', data: 'Episodes, entities, edges', color: 'purple' },
            { engine: 'Cognee', data: 'Semantic nodes, datasets', color: 'blue' },
            { engine: 'Zep', data: 'Threads, messages, graph', color: 'green' },
            { engine: 'OpenViking', data: 'Procedures, workflows', color: 'orange' },
            { engine: 'Memobase', data: 'Profiles, events, buffers', color: 'teal' },
            { engine: 'Supermemory', data: 'Memories, versions, docs', color: 'amber' },
          ].map((e) => (
            <div key={e.engine} className="p-4 bg-white/5 rounded-lg border border-white/10">
              <p className={`text-sm font-medium text-${e.color}-400`}>{e.engine}</p>
              <p className="text-xs text-white/40 mt-1">{e.data}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ─── OPA Policy Editor ─── */
function OPAPolicyEditor() {
  const { data: policies, isLoading } = usePolicies();
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white">Access Control Policies</h3>
        <button className="px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg text-sm hover:bg-purple-500/30">+ New Policy</button>
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-4 gap-4 p-4 bg-white/5 text-xs font-medium text-white/50 uppercase">
          <span>Policy Name</span><span>Engine Scope</span><span>Action</span><span>Status</span>
        </div>
        {[
          { name: 'memory-read-only', scope: 'All Engines', action: 'Deny write for viewer role', status: 'Active' },
          { name: 'pii-restriction', scope: 'Memobase, Zep', action: 'Mask PII in profile responses', status: 'Active' },
          { name: 'graph-admin-only', scope: 'Graphiti, Cognee', action: 'Restrict graph mutations', status: 'Draft' },
        ].map((policy) => (
          <div key={policy.name} className="grid grid-cols-4 gap-4 p-4 border-t border-white/5 text-sm">
            <span className="text-purple-400 font-mono">{policy.name}</span>
            <span className="text-white/60">{policy.scope}</span>
            <span className="text-white/60">{policy.action}</span>
            <span className={`text-xs px-2 py-1 rounded w-fit ${policy.status === 'Active' ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/40'}`}>
              {policy.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─── Audit Explorer ─── */
function AuditExplorer() {
  const { data: logs, isLoading, isError, refetch } = useAuditLogs({});
  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
        <input type="text" placeholder="Search audit logs..."
          className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500/50" />
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-5 gap-4 p-4 bg-white/5 text-xs font-medium text-white/50 uppercase">
          <span>Timestamp</span><span>Actor</span><span>Action</span><span>Resource</span><span>Status</span>
        </div>
        {(logs ?? []).map((log: any) => (
          <div key={log.id} className="grid grid-cols-5 gap-4 p-4 border-t border-white/5 text-sm">
            <span className="text-white/40 font-mono text-xs">{log.timestamp}</span>
            <span className="text-white/70">{log.actor}</span>
            <span className="text-white/70">{log.action}</span>
            <span className="text-purple-400">{log.resource}</span>
            <span className={`text-xs ${log.status === 'success' ? 'text-green-400' : 'text-red-400'}`}>{log.status}</span>
          </div>
        ))}
      </div>
      {(!logs || logs.length === 0) && <EmptyState message="No audit logs found" />}
    </div>
  );
}

/* ─── Data Retention ─── */
function DataRetention() {
  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Retention Policies by Engine</h3>
        <div className="space-y-4">
          {[
            { engine: 'Graphiti', ttl: '365d', desc: 'Episodic memory retention', color: 'purple' },
            { engine: 'Cognee', ttl: 'Forever', desc: 'Semantic knowledge preservation', color: 'blue' },
            { engine: 'Zep', ttl: '180d', desc: 'Conversation history', color: 'green' },
            { engine: 'OpenViking', ttl: 'Forever', desc: 'Procedural knowledge', color: 'orange' },
            { engine: 'Memobase', ttl: '365d', desc: 'User profile TTL (configurable)', color: 'teal' },
            { engine: 'Supermemory', ttl: '30-90d', desc: 'Auto-forget (forgetAfter rules)', color: 'amber' },
          ].map((r) => (
            <div key={r.engine} className="flex items-center justify-between p-4 bg-white/5 rounded-lg border border-white/10">
              <div className="flex items-center gap-3">
                <div className={`w-2.5 h-2.5 rounded-full bg-${r.color}-500`} />
                <div>
                  <p className="text-sm font-medium text-white">{r.engine}</p>
                  <p className="text-xs text-white/40">{r.desc}</p>
                </div>
              </div>
              <input type="text" defaultValue={r.ttl}
                className="w-24 px-3 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white text-right focus:outline-none focus:ring-1 focus:ring-purple-500/50" />
            </div>
          ))}
        </div>
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
      <button onClick={onRetry} className="flex items-center gap-2 px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30">
        <RefreshCw className="w-4 h-4" /> Retry
      </button>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-32 text-center">
      <Shield className="w-8 h-8 text-white/15 mb-2" />
      <p className="text-white/40 text-sm">{message}</p>
    </div>
  );
}
