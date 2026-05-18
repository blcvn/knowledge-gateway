import {
  LayoutDashboard,
  Database,
  Network,
  MessageSquare,
  Shield,
  GitBranch,
  Server,
  Activity,
  Key,
  Settings,
  Terminal,
  UserCircle,
  Sparkles
} from 'lucide-react';

interface NavItem {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  color?: string;
}

const navItems: NavItem[] = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'memory', label: 'Memory Explorer', icon: Database },
  { id: 'graph', label: 'Graph Studio', icon: Network },
  { id: 'profiles', label: 'User Profiles', icon: UserCircle, color: 'text-teal-500' },
  { id: 'adaptive', label: 'Adaptive Memory', icon: Sparkles, color: 'text-amber-500' },
  { id: 'debugger', label: 'Context Debugger', icon: Terminal },
  { id: 'sessions', label: 'Sessions', icon: MessageSquare },
  { id: 'governance', label: 'Governance', icon: Shield },
  { id: 'pipelines', label: 'Pipelines', icon: GitBranch },
  { id: 'infrastructure', label: 'Infrastructure', icon: Server },
  { id: 'observability', label: 'Observability', icon: Activity },
  { id: 'api', label: 'API & SDK', icon: Key },
  { id: 'settings', label: 'Settings', icon: Settings },
];

interface SidebarProps {
  activeSection: string;
  onSectionChange: (section: string) => void;
}

export function Sidebar({ activeSection, onSectionChange }: SidebarProps) {
  return (
    <div className="w-64 h-full bg-[#1a1a1f] border-r border-white/10 flex flex-col">
      {/* Logo */}
      <div className="p-6 border-b border-white/10">
        <h1 className="text-xl font-semibold text-white">VNP Memory</h1>
        <p className="text-xs text-white/50 mt-1">Control Plane</p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeSection === item.id;

          return (
            <button
              key={item.id}
              onClick={() => onSectionChange(item.id)}
              className={`
                w-full flex items-center gap-3 px-3 py-2.5 rounded-lg
                transition-all duration-200
                ${isActive
                  ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                  : 'text-white/70 hover:bg-white/5 hover:text-white'
                }
              `}
            >
              <Icon className={`w-5 h-5 flex-shrink-0 ${item.color || ''}`} />
              <span className="text-sm">{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-white/10">
        <div className="text-xs text-white/40">
          v1.0.0 • Enterprise Edition
        </div>
      </div>
    </div>
  );
}
