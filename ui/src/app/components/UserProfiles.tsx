import { useState } from 'react';
import { User, Settings, Droplets, CalendarDays, Eye, Search, ChevronRight, ChevronDown, RefreshCw, AlertCircle, Sparkles } from 'lucide-react';
import { useProfileList, useProfileDetail, useBufferStatus, useUserEvents, useContextAssembly } from '../../hooks/useProfiles';

type ProfileTab = 'explorer' | 'config' | 'buffers' | 'events' | 'context';

const tabs: { id: ProfileTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'explorer', label: 'Profile Explorer', icon: User },
  { id: 'config', label: 'Profile Config', icon: Settings },
  { id: 'buffers', label: 'Buffer Zone', icon: Droplets },
  { id: 'events', label: 'Event Timeline', icon: CalendarDays },
  { id: 'context', label: 'Context Preview', icon: Eye },
];

export function UserProfiles() {
  const [activeTab, setActiveTab] = useState<ProfileTab>('explorer');
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-teal-500/20">
            <User className="w-5 h-5 text-teal-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">User Profiles</h2>
            <p className="text-sm text-white/50 mt-0.5">Memobase-powered structured user knowledge extraction</p>
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
                    ? 'bg-teal-500/10 text-teal-400 border-b-2 border-teal-400'
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
        {activeTab === 'explorer' && <ProfileExplorer selectedUserId={selectedUserId} onSelectUser={setSelectedUserId} />}
        {activeTab === 'config' && <ProfileConfig />}
        {activeTab === 'buffers' && <BufferZoneMonitor userId={selectedUserId} />}
        {activeTab === 'events' && <EventTimeline userId={selectedUserId} />}
        {activeTab === 'context' && <ContextAssemblyPreview userId={selectedUserId} />}
      </div>
    </div>
  );
}

/* ─── Profile Explorer ─── */
function ProfileExplorer({ selectedUserId, onSelectUser }: { selectedUserId: string | null; onSelectUser: (id: string) => void }) {
  const { data: users, isLoading, isError, refetch } = useProfileList();

  if (isLoading) return <LoadingSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} />;

  return (
    <div className="grid grid-cols-3 gap-6">
      {/* User List */}
      <div className="col-span-1 space-y-3">
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
          <input type="text" placeholder="Search users..."
            className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-teal-500/50" />
        </div>
        {(users ?? []).map((user) => (
          <button key={user.user_id} onClick={() => onSelectUser(user.user_id)}
            className={`w-full text-left p-4 rounded-xl border transition-all ${
              selectedUserId === user.user_id ? 'bg-teal-500/10 border-teal-500/30' : 'bg-[#1a1a1f] border-white/10 hover:border-white/20'
            }`}>
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-teal-500/20 flex items-center justify-center">
                <User className="w-4 h-4 text-teal-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-white">{user.user_id}</p>
                <p className="text-xs text-white/40">{user.profiles.length} profile entries</p>
              </div>
            </div>
          </button>
        ))}
        {(!users || users.length === 0) && <EmptyState message="No user profiles yet" />}
      </div>

      {/* Profile Detail */}
      <div className="col-span-2">
        {selectedUserId ? <ProfileDetailView userId={selectedUserId} /> : (
          <div className="flex flex-col items-center justify-center h-64 text-center">
            <User className="w-12 h-12 text-white/20 mb-3" />
            <p className="text-white/40 text-sm">Select a user to view profile</p>
          </div>
        )}
      </div>
    </div>
  );
}

/* ─── Profile Detail Tree ─── */
function ProfileDetailView({ userId }: { userId: string }) {
  const { data: profile, isLoading } = useProfileDetail(userId);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  if (isLoading) return <LoadingSkeleton />;
  if (!profile) return <EmptyState message="Profile not found" />;

  const grouped = (profile.profiles ?? []).reduce<Record<string, typeof profile.profiles>>((acc, p) => {
    if (!acc[p.topic]) acc[p.topic] = [];
    acc[p.topic].push(p);
    return acc;
  }, {});

  const toggle = (topic: string) => setExpanded(prev => {
    const next = new Set(prev);
    next.has(topic) ? next.delete(topic) : next.add(topic);
    return next;
  });

  return (
    <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-lg font-semibold text-white">User: {userId}</h3>
          <p className="text-xs text-white/40 mt-1">Updated: {profile.updated_at}</p>
        </div>
        <span className="px-3 py-1 bg-teal-500/20 text-teal-400 rounded-full text-xs font-medium">
          {profile.profiles.length} entries
        </span>
      </div>
      <div className="space-y-2 font-mono text-sm">
        {Object.entries(grouped).map(([topic, entries]) => {
          const isOpen = expanded.has(topic);
          return (
            <div key={topic}>
              <button onClick={() => toggle(topic)} className="flex items-center gap-2 py-1.5 px-2 rounded hover:bg-white/5 w-full text-left">
                {isOpen ? <ChevronDown className="w-4 h-4 text-teal-400" /> : <ChevronRight className="w-4 h-4 text-white/50" />}
                <span className="text-teal-400">{topic}</span>
                <span className="text-white/30 text-xs ml-2">({entries.length})</span>
              </button>
              {isOpen && (
                <div className="ml-6 pl-4 border-l border-white/10 space-y-1 py-1">
                  {entries.map((entry, i) => (
                    <div key={i} className="flex items-start gap-2 py-1">
                      <span className="text-white/50">{entry.sub_topic}:</span>
                      <span className="text-white/80">&quot;{entry.content}&quot;</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
      {Object.keys(grouped).length === 0 && <EmptyState message="No profile entries extracted yet" />}
    </div>
  );
}

/* ─── Profile Config ─── */
function ProfileConfig() {
  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Profile Schema Configuration</h3>
        <p className="text-sm text-white/50 mb-6">Define which topics and sub-topics to extract from conversations.</p>
        <div className="flex items-center justify-between p-4 bg-white/5 rounded-lg border border-white/10 mb-6">
          <div>
            <p className="text-sm font-medium text-white">Strict Mode</p>
            <p className="text-xs text-white/40 mt-1">Only collect data matching the defined schema</p>
          </div>
          <button className="w-12 h-6 rounded-full bg-teal-500 relative">
            <span className="absolute right-1 top-1 w-4 h-4 rounded-full bg-white shadow-sm" />
          </button>
        </div>
        <div className="border border-white/10 rounded-lg overflow-hidden">
          <div className="grid grid-cols-3 gap-4 p-3 bg-white/5 text-xs font-medium text-white/60 uppercase">
            <span>Topic</span><span>Sub-topic</span><span>Description</span>
          </div>
          {[
            { topic: 'Preferences', sub: 'coding_style', desc: 'Programming style preferences' },
            { topic: 'Preferences', sub: 'language', desc: 'Preferred programming language' },
            { topic: 'Preferences', sub: 'theme', desc: 'UI theme preference' },
            { topic: 'Projects', sub: 'name', desc: 'Project being worked on' },
            { topic: 'Goals', sub: 'short_term', desc: 'Immediate goals' },
            { topic: 'Goals', sub: 'long_term', desc: 'Long-term objectives' },
          ].map((row, i) => (
            <div key={i} className="grid grid-cols-3 gap-4 p-3 border-t border-white/5 text-sm">
              <span className="text-teal-400">{row.topic}</span>
              <span className="text-white/70">{row.sub}</span>
              <span className="text-white/40">{row.desc}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Buffer Zone Settings</h3>
        <div className="grid grid-cols-2 gap-4">
          <div className="p-4 bg-white/5 rounded-lg border border-white/10">
            <p className="text-xs text-white/40 mb-1">Token Threshold</p>
            <p className="text-xl font-semibold text-white">1,024 <span className="text-xs text-white/40 font-normal">tokens</span></p>
          </div>
          <div className="p-4 bg-white/5 rounded-lg border border-white/10">
            <p className="text-xs text-white/40 mb-1">Idle Timeout</p>
            <p className="text-xl font-semibold text-white">1h</p>
          </div>
        </div>
        <p className="text-xs text-white/40 mt-4 flex items-center gap-1.5">
          <Sparkles className="w-3.5 h-3.5 text-amber-400" /> Fixed 3 LLM calls per flush: extract → merge → events
        </p>
      </div>
    </div>
  );
}

/* ─── Buffer Zone Monitor ─── */
function BufferZoneMonitor({ userId }: { userId: string | null }) {
  const { data: buffer, isLoading } = useBufferStatus(userId ?? '');
  if (!userId) return <EmptyState message="Select a user to view buffer status" />;
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
      <h3 className="text-lg font-semibold text-white mb-4">Buffer Zone — {userId}</h3>
      {buffer ? (
        <div className="space-y-6">
          <div>
            <div className="flex justify-between text-sm mb-2">
              <span className="text-white/60">Token Accumulation</span>
              <span className="text-white font-medium">{buffer.token_count} / {buffer.token_threshold}</span>
            </div>
            <div className="h-3 bg-white/5 rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-teal-500 to-cyan-500 rounded-full transition-all duration-500"
                style={{ width: `${Math.min((buffer.token_count / buffer.token_threshold) * 100, 100)}%` }} />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: 'Buffer Type', value: buffer.buffer_type },
              { label: 'Total Flushes', value: String(buffer.flush_count) },
              { label: 'Idle Timeout', value: buffer.idle_timeout },
            ].map((s) => (
              <div key={s.label} className="p-4 bg-white/5 rounded-lg border border-white/10 text-center">
                <p className="text-xs text-white/40">{s.label}</p>
                <p className="text-lg font-semibold text-white mt-1">{s.value}</p>
              </div>
            ))}
          </div>
          <div className="p-4 bg-white/5 rounded-lg border border-white/10">
            <p className="text-xs text-white/40">Last Flush</p>
            <p className="text-sm text-white mt-1">{buffer.last_flush}</p>
          </div>
        </div>
      ) : <EmptyState message="No active buffer for this user" />}
    </div>
  );
}

/* ─── Event Timeline ─── */
function EventTimeline({ userId }: { userId: string | null }) {
  const { data: events, isLoading } = useUserEvents(userId ?? '');
  if (!userId) return <EmptyState message="Select a user to view events" />;
  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
        <input type="text" placeholder="Search events by gist..."
          className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-teal-500/50" />
      </div>
      <div className="relative pl-6">
        <div className="absolute left-2.5 top-0 bottom-0 w-px bg-teal-500/30" />
        {(events ?? []).map((event) => (
          <div key={event.id} className="relative mb-4">
            <div className="absolute -left-3.5 w-3 h-3 rounded-full bg-teal-500 border-2 border-[#0f0f14]" />
            <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-4 ml-4">
              <p className="text-sm text-white">{event.gist}</p>
              <div className="flex items-center gap-2 mt-2">
                {event.tags.map((tag, i) => (
                  <span key={i} className="px-2 py-0.5 bg-teal-500/10 text-teal-400 rounded text-xs">{tag}</span>
                ))}
                <span className="text-xs text-white/30 ml-auto">{event.created_at}</span>
              </div>
            </div>
          </div>
        ))}
        {(!events || events.length === 0) && <EmptyState message="No events recorded yet" />}
      </div>
    </div>
  );
}

/* ─── Context Assembly Preview ─── */
function ContextAssemblyPreview({ userId }: { userId: string | null }) {
  const { data: ctx, isLoading } = useContextAssembly(userId ?? '');
  if (!userId) return <EmptyState message="Select a user to preview context assembly" />;
  if (isLoading) return <LoadingSkeleton />;
  if (!ctx) return <EmptyState message="No context available" />;

  const totalTokens = ctx.token_count || 1;
  const profilePct = Math.round((ctx.profile_section_tokens / totalTokens) * 100);
  const eventPct = 100 - profilePct;

  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Token Budget Allocation</h3>
        <div className="flex items-center gap-4 mb-3">
          <div className="flex-1 h-4 rounded-full overflow-hidden flex">
            <div className="bg-teal-500" style={{ width: `${profilePct}%` }} />
            <div className="bg-blue-500" style={{ width: `${eventPct}%` }} />
          </div>
          <span className="text-sm font-medium text-white">{ctx.token_count} tokens</span>
        </div>
        <div className="flex gap-4 text-xs">
          <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded bg-teal-500" /> Profile ({ctx.profile_section_tokens})</span>
          <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded bg-blue-500" /> Events ({ctx.event_section_tokens})</span>
        </div>
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6 flex-1">
        <p className="text-xs text-white/40 mb-1">Assembly Latency</p>
        <p className={`text-2xl font-semibold ${ctx.latency_ms < 100 ? 'text-green-400' : ctx.latency_ms < 200 ? 'text-yellow-400' : 'text-red-400'}`}>
          {ctx.latency_ms}ms
        </p>
        <p className="text-xs text-white/30 mt-1">Target: &lt; 100ms</p>
      </div>
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Prompt-Ready Context</h3>
        <pre className="p-4 bg-black/40 rounded-lg text-sm text-white/80 font-mono whitespace-pre-wrap border border-white/5 max-h-64 overflow-y-auto">
          {ctx.context_string}
        </pre>
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
      <button onClick={onRetry} className="flex items-center gap-2 px-4 py-2 bg-teal-500/20 text-teal-400 rounded-lg hover:bg-teal-500/30">
        <RefreshCw className="w-4 h-4" /> Retry
      </button>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-32 text-center">
      <User className="w-8 h-8 text-white/15 mb-2" />
      <p className="text-white/40 text-sm">{message}</p>
    </div>
  );
}
