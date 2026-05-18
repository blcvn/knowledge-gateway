import { useState } from 'react';
import { Settings, Users, Building2, CreditCard, Bell, Lock, Shield, Globe, Save, Plus, Trash2, Search, UserPlus, Crown, Eye, KeyRound, AlertCircle } from 'lucide-react';
import { useOrgSettings, useMembers, useRoles } from '../../hooks/useOrganizationSettings';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';
import { API_CONFIG } from '../../config/api.config';

type SettingsTab = 'general' | 'members' | 'security' | 'billing' | 'notifications';

const tabs: { id: SettingsTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'general', label: 'General', icon: Building2 },
  { id: 'members', label: 'Members & Roles', icon: Users },
  { id: 'security', label: 'Security', icon: Shield },
  { id: 'billing', label: 'Billing & Plans', icon: CreditCard },
  { id: 'notifications', label: 'Notifications', icon: Bell },
];

const ROLE_COLORS: Record<string, string> = {
  owner: 'bg-amber-500/20 text-amber-400',
  admin: 'bg-purple-500/20 text-purple-400',
  developer: 'bg-blue-500/20 text-blue-400',
  viewer: 'bg-white/10 text-white/50',
};

export function OrganizationSettings() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');

  return (
    <div className="flex-1 overflow-hidden flex">
      {/* Vertical sidebar */}
      <div className="w-56 flex-shrink-0 border-r border-white/10 p-4 space-y-1">
        <p className="text-xs font-semibold text-white/30 uppercase tracking-wider px-3 mb-3">Organization</p>
        {tabs.map(tab => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button key={tab.id} onClick={() => setActiveTab(tab.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors ${
                isActive ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white/80 hover:bg-white/5'
              }`}>
              <Icon className="w-4 h-4 flex-shrink-0" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-8">
        {activeTab === 'general' && <GeneralSettings />}
        {activeTab === 'members' && <MembersRoles />}
        {activeTab === 'security' && <SecuritySettings />}
        {activeTab === 'billing' && <BillingPlans />}
        {activeTab === 'notifications' && <NotificationSettings />}
      </div>
    </div>
  );
}

/* ─── General Settings ─── */
function GeneralSettings() {
  const { data: settings, isLoading } = useOrgSettings();
  const qc = useQueryClient();

  const saveMutation = useMutation({
    mutationFn: (data: Record<string, string>) =>
      API_CONFIG.useMockData
        ? new Promise(r => setTimeout(r, 500))
        : apiClient.put(`${API_CONFIG.engines.gateway.baseUrl}/v1/org/settings`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['org', 'settings'] }),
  });

  const mockSettings = {
    name: 'VNP Platform',
    slug: 'vnp-platform',
    domain: 'vnp-memory.io',
    timezone: 'Asia/Ho_Chi_Minh',
    maxAgents: 100,
    maxMemoriesPerUser: 10000,
  };

  const display = settings ?? mockSettings;

  return (
    <div className="space-y-8 max-w-2xl">
      <div>
        <h3 className="text-xl font-semibold text-white mb-1">General Settings</h3>
        <p className="text-sm text-white/40">Manage your organization's core configuration.</p>
      </div>

      <div className="space-y-5">
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Organization Name" value={display.name} placeholder="Acme Corp" />
          <FormField label="Slug" value={display.slug} placeholder="acme-corp" mono />
        </div>
        <FormField label="Primary Domain" value={display.domain} placeholder="example.com" />
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Max Agents" value={String(display.maxAgents)} type="number" />
          <FormField label="Max Memories / User" value={String(display.maxMemoriesPerUser)} type="number" />
        </div>
        <div>
          <label className="block text-sm text-white/50 mb-1.5">Timezone</label>
          <select defaultValue={display.timezone}
            className="w-full px-3 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white focus:outline-none focus:ring-1 focus:ring-white/30">
            <option>Asia/Ho_Chi_Minh</option>
            <option>UTC</option>
            <option>America/New_York</option>
            <option>Europe/London</option>
          </select>
        </div>
      </div>

      <div className="flex items-center justify-between pt-4 border-t border-white/10">
        <p className="text-sm text-white/30">Changes take effect immediately.</p>
        <button
          onClick={() => saveMutation.mutate({})}
          disabled={saveMutation.isPending}
          className="flex items-center gap-2 px-5 py-2.5 bg-white text-black font-medium rounded-lg hover:bg-white/90 disabled:opacity-50 transition-colors"
        >
          <Save className="w-4 h-4" />
          {saveMutation.isPending ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  );
}

/* ─── Members & Roles ─── */
function MembersRoles() {
  const { data: members, isLoading } = useMembers();
  const [search, setSearch] = useState('');

  const mockMembers = [
    { id: 'm1', name: 'Nguyen Binh', email: 'binh@vnp.io', role: 'owner', status: 'active', joinedAt: '2025-01-01' },
    { id: 'm2', name: 'Alice Chen', email: 'alice@vnp.io', role: 'admin', status: 'active', joinedAt: '2025-02-15' },
    { id: 'm3', name: 'Bob Kim', email: 'bob@vnp.io', role: 'developer', status: 'active', joinedAt: '2025-03-20' },
    { id: 'm4', name: 'Carol Liu', email: 'carol@vnp.io', role: 'developer', status: 'active', joinedAt: '2025-04-10' },
    { id: 'm5', name: 'Dave Park', email: 'dave@vnp.io', role: 'viewer', status: 'inactive', joinedAt: '2025-05-05' },
  ];

  const displayMembers = (members ?? mockMembers).filter(m =>
    !search || m.name.toLowerCase().includes(search.toLowerCase()) || m.email.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-xl font-semibold text-white">Members & Roles</h3>
          <p className="text-sm text-white/40 mt-1">Manage team access and RBAC permissions.</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 bg-white/10 text-white rounded-lg text-sm hover:bg-white/15 border border-white/10">
          <UserPlus className="w-4 h-4" /> Invite Member
        </button>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
        <input type="text" value={search} onChange={e => setSearch(e.target.value)}
          placeholder="Search members..."
          className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-white/20" />
      </div>

      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-4 gap-4 p-4 bg-white/5 text-xs font-medium text-white/40 uppercase tracking-wider">
          <span className="col-span-2">Member</span><span>Role</span><span>Actions</span>
        </div>
        {displayMembers.map(member => (
          <div key={member.id} className="grid grid-cols-4 gap-4 p-4 border-t border-white/5 items-center hover:bg-white/5">
            <div className="col-span-2 flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center text-sm font-medium text-white flex-shrink-0">
                {member.name.split(' ').map(n => n[0]).join('').slice(0, 2)}
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-white">{member.name}</p>
                  {member.role === 'owner' && <Crown className="w-3 h-3 text-amber-400" />}
                </div>
                <p className="text-xs text-white/40">{member.email}</p>
              </div>
            </div>
            <div>
              <select defaultValue={member.role} disabled={member.role === 'owner'}
                className="px-2.5 py-1.5 rounded-lg text-xs border bg-transparent focus:outline-none disabled:opacity-50 disabled:cursor-default"
                style={{ borderColor: 'transparent', backgroundColor: 'transparent' }}>
                {Object.keys(ROLE_COLORS).map(r => <option key={r} value={r}>{r}</option>)}
              </select>
              <span className={`text-xs px-2 py-0.5 rounded-full ${ROLE_COLORS[member.role]}`}>{member.role}</span>
            </div>
            <div className="flex items-center gap-2">
              {member.role !== 'owner' && (
                <>
                  <button className="p-1.5 bg-white/5 rounded text-white/40 hover:text-white hover:bg-white/10"><Eye className="w-3.5 h-3.5" /></button>
                  <button className="p-1.5 bg-white/5 rounded text-white/40 hover:text-red-400 hover:bg-red-500/10"><Trash2 className="w-3.5 h-3.5" /></button>
                </>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─── Security Settings ─── */
function SecuritySettings() {
  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h3 className="text-xl font-semibold text-white">Security</h3>
        <p className="text-sm text-white/40 mt-1">Enforce authentication and session policies.</p>
      </div>

      {[
        { title: 'Require MFA for all members', desc: 'Members must enable multi-factor authentication to access the console.', defaultOn: true, color: 'green' },
        { title: 'Enforce SSO (SAML/OIDC)', desc: 'Require members to authenticate via your configured identity provider.', defaultOn: false, color: 'blue' },
        { title: 'IP Allowlist', desc: 'Restrict access to specific IP ranges only.', defaultOn: false, color: 'orange' },
        { title: 'Audit Log Immutability', desc: 'Prevent deletion or modification of audit log entries.', defaultOn: true, color: 'purple' },
      ].map(setting => (
        <div key={setting.title} className="flex items-start justify-between p-5 bg-[#1a1a1f] border border-white/10 rounded-xl">
          <div className="flex items-start gap-3">
            <Shield className={`w-5 h-5 mt-0.5 text-${setting.color}-400`} />
            <div>
              <p className="text-sm font-medium text-white">{setting.title}</p>
              <p className="text-xs text-white/40 mt-0.5">{setting.desc}</p>
            </div>
          </div>
          <Toggle defaultOn={setting.defaultOn} />
        </div>
      ))}

      <div className="p-5 bg-[#1a1a1f] border border-white/10 rounded-xl">
        <div className="flex items-center gap-2 mb-4">
          <KeyRound className="w-5 h-5 text-white/50" />
          <h4 className="text-sm font-medium text-white">Session Management</h4>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Session Timeout (hours)" value="8" type="number" />
          <FormField label="Max Concurrent Sessions" value="5" type="number" />
        </div>
      </div>

      <div className="flex justify-end">
        <button className="px-5 py-2.5 bg-white text-black font-medium rounded-lg hover:bg-white/90">Save Security Settings</button>
      </div>
    </div>
  );
}

/* ─── Billing ─── */
function BillingPlans() {
  const plans = [
    { name: 'Starter', price: '$49', period: '/mo', features: ['5 agents', '100k memories', '2 engines', 'Community support'], current: false },
    { name: 'Pro', price: '$299', period: '/mo', features: ['25 agents', '1M memories', '4 engines', 'Priority support', 'RBAC'], current: true },
    { name: 'Enterprise', price: 'Custom', period: '', features: ['Unlimited agents', 'Unlimited memories', 'All 6 engines', 'SLA + dedicated support', 'SAML SSO', 'Audit logs'], current: false },
  ];

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <h3 className="text-xl font-semibold text-white">Billing & Plans</h3>
        <p className="text-sm text-white/40 mt-1">Manage your subscription and usage.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {plans.map(plan => (
          <div key={plan.name} className={`relative p-6 rounded-xl border ${
            plan.current ? 'bg-white/5 border-purple-500/40' : 'bg-[#1a1a1f] border-white/10'
          }`}>
            {plan.current && (
              <span className="absolute top-4 right-4 text-xs px-2 py-0.5 bg-purple-500/20 text-purple-400 rounded-full">Current</span>
            )}
            <h4 className="text-lg font-semibold text-white">{plan.name}</h4>
            <div className="mt-2 mb-4">
              <span className="text-2xl font-bold text-white">{plan.price}</span>
              <span className="text-white/40 text-sm">{plan.period}</span>
            </div>
            <ul className="space-y-2 mb-6">
              {plan.features.map(f => (
                <li key={f} className="flex items-center gap-2 text-xs text-white/60">
                  <div className="w-1.5 h-1.5 rounded-full bg-purple-400 flex-shrink-0" />
                  {f}
                </li>
              ))}
            </ul>
            <button className={`w-full py-2.5 rounded-lg text-sm font-medium transition-colors ${
              plan.current
                ? 'bg-white/10 text-white/50 cursor-default'
                : plan.name === 'Enterprise'
                ? 'bg-white/5 border border-white/20 text-white hover:bg-white/10'
                : 'bg-purple-600 text-white hover:bg-purple-500'
            }`}>
              {plan.current ? 'Current Plan' : plan.name === 'Enterprise' ? 'Contact Sales' : 'Upgrade'}
            </button>
          </div>
        ))}
      </div>

      <div className="p-5 bg-[#1a1a1f] border border-white/10 rounded-xl">
        <h4 className="text-sm font-medium text-white mb-4">Usage This Month</h4>
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'Memory Operations', used: 245000, limit: 1000000 },
            { label: 'API Calls', used: 18200, limit: 50000 },
            { label: 'Active Agents', used: 12, limit: 25 },
          ].map(u => (
            <div key={u.label}>
              <div className="flex items-center justify-between text-xs mb-1.5">
                <span className="text-white/50">{u.label}</span>
                <span className="text-white/70">{u.used.toLocaleString()} / {u.limit.toLocaleString()}</span>
              </div>
              <div className="h-2 bg-white/5 rounded-full overflow-hidden">
                <div className={`h-full rounded-full ${u.used / u.limit > 0.8 ? 'bg-red-500' : 'bg-purple-500'}`}
                  style={{ width: `${(u.used / u.limit) * 100}%` }} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ─── Notifications ─── */
function NotificationSettings() {
  const notificationGroups = [
    {
      group: 'System Alerts',
      items: [
        { id: 'n1', label: 'Engine health degradation', desc: 'Alert when any memory engine exceeds latency thresholds', email: true, slack: true, webhook: false },
        { id: 'n2', label: 'Pipeline failures', desc: 'Notify when ingestion or recall jobs fail', email: true, slack: true, webhook: true },
        { id: 'n3', label: 'Quota warnings', desc: 'Alert when approaching usage limits (80% / 95%)', email: true, slack: false, webhook: false },
      ],
    },
    {
      group: 'Security Events',
      items: [
        { id: 'n4', label: 'New member joined', desc: 'Notify when a new user joins the organization', email: true, slack: false, webhook: false },
        { id: 'n5', label: 'GDPR forget request', desc: 'Alert when a right-to-erasure request is submitted', email: true, slack: true, webhook: true },
        { id: 'n6', label: 'API key created / revoked', desc: 'Audit trail for key lifecycle events', email: false, slack: false, webhook: true },
      ],
    },
  ];

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <h3 className="text-xl font-semibold text-white">Notification Preferences</h3>
        <p className="text-sm text-white/40 mt-1">Configure how and when you receive alerts.</p>
      </div>

      {notificationGroups.map(group => (
        <div key={group.group} className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
          <div className="px-5 py-3 bg-white/5 border-b border-white/10">
            <p className="text-xs font-semibold text-white/50 uppercase tracking-wider">{group.group}</p>
          </div>
          <div className="divide-y divide-white/5">
            {group.items.map(item => (
              <div key={item.id} className="flex items-center justify-between px-5 py-4 hover:bg-white/5">
                <div className="flex-1 mr-6">
                  <p className="text-sm font-medium text-white">{item.label}</p>
                  <p className="text-xs text-white/40 mt-0.5">{item.desc}</p>
                </div>
                <div className="flex items-center gap-4">
                  {[
                    { key: 'email', label: 'Email', defaultOn: item.email },
                    { key: 'slack', label: 'Slack', defaultOn: item.slack },
                    { key: 'webhook', label: 'Webhook', defaultOn: item.webhook },
                  ].map(ch => (
                    <div key={ch.key} className="flex flex-col items-center gap-1">
                      <Toggle defaultOn={ch.defaultOn} small />
                      <span className="text-xs text-white/30">{ch.label}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}

      <div className="flex justify-end">
        <button className="px-5 py-2.5 bg-white text-black font-medium rounded-lg hover:bg-white/90">Save Preferences</button>
      </div>
    </div>
  );
}

/* ─── Shared ─── */
function FormField({ label, value, placeholder, type = 'text', mono = false }: {
  label: string; value: string; placeholder?: string; type?: string; mono?: boolean;
}) {
  return (
    <div>
      <label className="block text-sm text-white/50 mb-1.5">{label}</label>
      <input
        type={type}
        defaultValue={value}
        placeholder={placeholder}
        className={`w-full px-3 py-2.5 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/20 focus:outline-none focus:ring-1 focus:ring-white/30 ${mono ? 'font-mono' : ''}`}
      />
    </div>
  );
}

function Toggle({ defaultOn, small = false }: { defaultOn: boolean; small?: boolean }) {
  const [on, setOn] = useState(defaultOn);
  const size = small ? 'w-8 h-4' : 'w-10 h-5';
  const dot = small ? 'w-3 h-3' : 'w-4 h-4';
  return (
    <button
      onClick={() => setOn(!on)}
      className={`${size} rounded-full relative transition-colors ${on ? 'bg-purple-500' : 'bg-white/20'}`}
    >
      <span className={`absolute top-0.5 ${dot} rounded-full bg-white shadow transition-all ${on ? 'right-0.5' : 'left-0.5'}`} />
    </button>
  );
}
