import { useState } from 'react';
import { MessageSquare, Search, User, RefreshCw, Bot, Clock, Layers, Circle, Tag, AlertCircle, Loader2 } from 'lucide-react';
import { useSessionList, useSessionDetail } from '../../hooks/useSessions';

const STATUS_CONFIG: Record<string, { label: string; color: string; dot: string }> = {
  active:    { label: 'Active',     color: 'text-green-400',   dot: 'bg-green-500' },
  completed: { label: 'Completed',  color: 'text-white/40',    dot: 'bg-white/20' },
  failed:    { label: 'Failed',     color: 'text-red-400',     dot: 'bg-red-500' },
};

export function SessionsExplorer() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const { data: sessions, isLoading, error, refetch } = useSessionList();
  const { data: sessionDetail, isLoading: isLoadingDetail } = useSessionDetail(selectedId ?? '');

  const filteredSessions = (sessions as any[] ?? []).filter((s: any) =>
    !search || s.title?.toLowerCase().includes(search.toLowerCase()) || s.user_id?.toLowerCase().includes(search.toLowerCase())
  );

  const selectedSession = (sessions as any[] ?? []).find((s: any) => s.id === selectedId);

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-white/10 flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-rose-500/20">
            <MessageSquare className="w-5 h-5 text-rose-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">Sessions Explorer</h2>
            <p className="text-sm text-white/50 mt-0.5">Replay agent-user sessions and inspect memory retrieval per message</p>
          </div>
        </div>
      </div>

      {/* Main layout */}
      <div className="flex-1 overflow-hidden flex">
        {/* Session list */}
        <div className="w-80 flex-shrink-0 border-r border-white/10 flex flex-col">
          <div className="p-4 border-b border-white/10">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
              <input
                type="text" value={search} onChange={e => setSearch(e.target.value)}
                placeholder="Search sessions..."
                className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-rose-500/50"
              />
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-3 space-y-2">
            {isLoading && (
              <div className="flex items-center justify-center h-32">
                <Loader2 className="w-6 h-6 text-white/30 animate-spin" />
              </div>
            )}
            {error && (
              <div className="flex flex-col items-center justify-center h-32">
                <AlertCircle className="w-6 h-6 text-red-400 mb-2" />
                <button onClick={() => refetch()} className="text-sm text-rose-400">Retry</button>
              </div>
            )}
            {!isLoading && !error && filteredSessions.map((session: any) => {
              const status = STATUS_CONFIG[session.status] ?? STATUS_CONFIG.completed;
              const isSelected = selectedId === session.id;
              return (
                <button
                  key={session.id}
                  onClick={() => setSelectedId(session.id)}
                  className={`w-full text-left p-3 rounded-xl border transition-all ${
                    isSelected ? 'bg-rose-500/10 border-rose-500/30' : 'bg-[#1a1a1f] border-white/5 hover:border-white/15 hover:bg-white/5'
                  }`}
                >
                  <div className="flex items-start justify-between mb-2">
                    <p className="text-sm font-medium text-white line-clamp-1 flex-1 mr-2">{session.title ?? `Session ${session.id}`}</p>
                    <div className={`flex items-center gap-1 flex-shrink-0`}>
                      <div className={`w-2 h-2 rounded-full ${status.dot}`} />
                    </div>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-white/40">
                    <span className="flex items-center gap-1"><User className="w-3 h-3" />{session.user_id}</span>
                    <span className="flex items-center gap-1"><MessageSquare className="w-3 h-3" />{session.message_count ?? '?'}</span>
                  </div>
                  <p className="text-xs text-white/25 mt-1">{session.agent_id}</p>
                </button>
              );
            })}
            {!isLoading && !error && filteredSessions.length === 0 && (
              <div className="text-center py-10 text-white/30 text-sm">No sessions found</div>
            )}
          </div>
        </div>

        {/* Session Detail */}
        <div className="flex-1 overflow-hidden flex flex-col">
          {!selectedId ? (
            <div className="flex-1 flex flex-col items-center justify-center text-center px-8">
              <div className="w-16 h-16 rounded-2xl bg-rose-500/10 flex items-center justify-center mb-4">
                <MessageSquare className="w-8 h-8 text-rose-400/50" />
              </div>
              <h3 className="text-lg font-medium text-white/50 mb-1">Select a session</h3>
              <p className="text-sm text-white/30">View conversation, memory retrieval sources, and working memory state.</p>
            </div>
          ) : isLoadingDetail ? (
            <div className="flex-1 flex items-center justify-center">
              <Loader2 className="w-8 h-8 text-white/30 animate-spin" />
            </div>
          ) : (
            <div className="flex-1 overflow-hidden flex flex-col">
              {/* Session header */}
              <div className="px-6 py-4 border-b border-white/10 flex-shrink-0">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{selectedSession?.title}</h3>
                    <div className="flex items-center gap-4 text-xs text-white/40 mt-1">
                      <span className="flex items-center gap-1"><User className="w-3 h-3" />{selectedSession?.user_id}</span>
                      <span className="flex items-center gap-1"><Bot className="w-3 h-3" />{selectedSession?.agent_id}</span>
                      <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{selectedSession?.updated_at?.slice(0, 10)}</span>
                    </div>
                  </div>
                  <div className={`px-2.5 py-1 rounded-full text-xs ${
                    selectedSession?.status === 'active' ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/40'
                  }`}>
                    {selectedSession?.status}
                  </div>
                </div>
              </div>

              {/* Messages */}
              <div className="flex-1 overflow-y-auto p-6 space-y-4">
                {(sessionDetail as any)?.messages?.map((msg: any) => {
                  const isUser = msg.role === 'user';
                  return (
                    <div key={msg.id} className={`flex flex-col ${isUser ? 'items-end' : 'items-start'} gap-2`}>
                      <div className={`max-w-[75%] p-4 rounded-2xl text-sm leading-relaxed ${
                        isUser
                          ? 'bg-rose-500/15 border border-rose-500/20 text-white ml-auto'
                          : 'bg-white/5 border border-white/10 text-white/90'
                      }`}>
                        <div className="flex items-center gap-2 mb-2">
                          {isUser ? <User className="w-3.5 h-3.5 text-rose-400" /> : <Bot className="w-3.5 h-3.5 text-blue-400" />}
                          <span className="text-xs font-medium capitalize text-white/50">{msg.role}</span>
                          <span className="text-xs text-white/20 ml-auto">{msg.timestamp?.slice(11, 16)}</span>
                        </div>
                        {msg.content}
                      </div>
                      {/* Memory sources */}
                      {msg.memory_sources && msg.memory_sources.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 max-w-[75%]">
                          {msg.memory_sources.map((src: string) => {
                            const [engine] = src.split(':');
                            const colors: Record<string, string> = {
                              graphiti: 'text-purple-400 bg-purple-500/10 border-purple-500/20',
                              memobase: 'text-teal-400 bg-teal-500/10 border-teal-500/20',
                              cognee: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
                              zep: 'text-green-400 bg-green-500/10 border-green-500/20',
                              supermemory: 'text-amber-400 bg-amber-500/10 border-amber-500/20',
                            };
                            const c = colors[engine] ?? 'text-white/40 bg-white/5 border-white/10';
                            return (
                              <span key={src} className={`text-xs px-2 py-0.5 rounded border font-mono ${c}`}>
                                {src}
                              </span>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
                {!(sessionDetail as any)?.messages?.length && (
                  <div className="text-center text-white/30 text-sm py-8">No messages in this session</div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
