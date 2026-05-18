import { Search, Bell, Bot, ChevronDown } from 'lucide-react';

interface TopNavProps {
  currentTenant: string;
  currentEnvironment: string;
}

export function TopNav({ currentTenant, currentEnvironment }: TopNavProps) {
  return (
    <div className="h-16 bg-[#1a1a1f] border-b border-white/10 flex items-center justify-between px-6">
      {/* Left: Tenant & Environment */}
      <div className="flex items-center gap-4">
        <button className="flex items-center gap-2 px-3 py-1.5 bg-white/5 rounded-lg border border-white/10 hover:bg-white/10 transition-colors">
          <div className="w-2 h-2 rounded-full bg-green-500"></div>
          <span className="text-sm text-white">{currentTenant}</span>
          <ChevronDown className="w-4 h-4 text-white/50" />
        </button>

        <div className="flex items-center gap-2 px-3 py-1.5 bg-white/5 rounded-lg border border-white/10">
          <div className={`w-2 h-2 rounded-full ${
            currentEnvironment === 'Production' ? 'bg-red-500' :
            currentEnvironment === 'Staging' ? 'bg-yellow-500' :
            'bg-blue-500'
          }`}></div>
          <span className="text-sm text-white/70">{currentEnvironment}</span>
        </div>
      </div>

      {/* Center: Global Search */}
      <div className="flex-1 max-w-2xl mx-8">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
          <input
            type="text"
            placeholder="Search memories, sessions, entities..."
            className="w-full pl-10 pr-4 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
          />
          <kbd className="absolute right-3 top-1/2 -translate-y-1/2 px-2 py-0.5 bg-white/10 rounded text-xs text-white/50">
            ⌘K
          </kbd>
        </div>
      </div>

      {/* Right: Actions & Profile */}
      <div className="flex items-center gap-3">
        <button className="p-2 hover:bg-white/5 rounded-lg transition-colors relative">
          <Bell className="w-5 h-5 text-white/70" />
          <div className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full"></div>
        </button>

        <button className="p-2 hover:bg-white/5 rounded-lg transition-colors">
          <Bot className="w-5 h-5 text-white/70" />
        </button>

        <button className="flex items-center gap-2 pl-3 pr-2 py-1.5 hover:bg-white/5 rounded-lg transition-colors">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center text-sm font-medium text-white">
            AD
          </div>
          <ChevronDown className="w-4 h-4 text-white/50" />
        </button>
      </div>
    </div>
  );
}
