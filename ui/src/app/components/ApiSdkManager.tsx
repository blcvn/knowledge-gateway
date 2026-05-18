import { useState } from 'react';
import { Key, Webhook, Gauge, Copy, Eye, EyeOff, Plus, Trash2, RefreshCw, AlertCircle, CheckCircle, Search, Code2, Terminal, Globe } from 'lucide-react';
import { useApiKeys, useRateLimits, useWebhooks } from '../../hooks/useApiSdk';

type ApiTab = 'keys' | 'rateLimits' | 'webhooks' | 'sdks';

const tabs: { id: ApiTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'keys', label: 'API Keys', icon: Key },
  { id: 'rateLimits', label: 'Rate Limits', icon: Gauge },
  { id: 'webhooks', label: 'Webhooks', icon: Webhook },
  { id: 'sdks', label: 'SDK Quickstart', icon: Code2 },
];

export function ApiSdkManager() {
  const [activeTab, setActiveTab] = useState<ApiTab>('keys');

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-lg bg-blue-500/20">
            <Key className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-white">API & SDK Manager</h2>
            <p className="text-sm text-white/50 mt-0.5">Manage access keys, rate limits, webhooks, and SDK quickstart guides</p>
          </div>
        </div>
      </div>

      <div className="px-6 pt-4 border-b border-white/10">
        <div className="flex items-center gap-1">
          {tabs.map(tab => {
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

      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'keys' && <ApiKeysPanel />}
        {activeTab === 'rateLimits' && <RateLimitsPanel />}
        {activeTab === 'webhooks' && <WebhooksPanel />}
        {activeTab === 'sdks' && <SdkQuickstart />}
      </div>
    </div>
  );
}

/* ─── API Keys Panel ─── */
function ApiKeysPanel() {
  const { data: keys, isLoading, isError, refetch } = useApiKeys();
  const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState('');

  const toggleReveal = (id: string) => {
    setRevealedKeys(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const copyKey = (value: string) => {
    navigator.clipboard.writeText(value).catch(console.error);
  };

  const mockKeys = [
    { id: 'key_1', name: 'Production Agent', key: 'vnp_prod_sk_3f9a2b8c...', scopes: ['memory:read', 'memory:write', 'graph:read'], createdAt: '2026-01-15', lastUsed: '2026-05-12', status: 'active' },
    { id: 'key_2', name: 'Staging Environment', key: 'vnp_stg_sk_7d1e4f2a...', scopes: ['memory:read'], createdAt: '2026-02-20', lastUsed: '2026-04-30', status: 'active' },
    { id: 'key_3', name: 'Dev CLI Tool', key: 'vnp_dev_sk_2c8b5e9f...', scopes: ['memory:read', 'debug:context'], createdAt: '2026-03-10', lastUsed: '2026-05-01', status: 'inactive' },
  ];

  const displayKeys = (keys ?? mockKeys).filter(k =>
    k.name.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" />
          <input
            type="text" value={search} onChange={e => setSearch(e.target.value)}
            placeholder="Search keys..."
            className="w-full pl-10 pr-4 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-white/30 focus:outline-none focus:ring-1 focus:ring-blue-500/50"
          />
        </div>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30 border border-blue-500/30">
          <Plus className="w-4 h-4" /> Generate Key
        </button>
      </div>

      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-5 gap-4 p-4 bg-white/5 text-xs font-medium text-white/40 uppercase tracking-wider">
          <span className="col-span-2">Name / Key</span><span>Scopes</span><span>Last Used</span><span>Status</span>
        </div>
        {displayKeys.map(key => (
          <div key={key.id} className="grid grid-cols-5 gap-4 p-4 border-t border-white/5 items-center hover:bg-white/5">
            <div className="col-span-2">
              <p className="text-sm font-medium text-white">{key.name}</p>
              <div className="flex items-center gap-2 mt-1">
                <code className="text-xs font-mono text-white/40">
                  {revealedKeys.has(key.id) ? key.key.replace('...', 'XXXX1234') : key.key}
                </code>
                <button onClick={() => toggleReveal(key.id)} className="text-white/30 hover:text-white/60">
                  {revealedKeys.has(key.id) ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                </button>
                <button onClick={() => copyKey(key.key)} className="text-white/30 hover:text-white/60">
                  <Copy className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
            <div className="flex flex-wrap gap-1">
              {key.scopes.map(s => (
                <span key={s} className="px-1.5 py-0.5 bg-white/10 text-white/50 rounded text-xs">{s}</span>
              ))}
            </div>
            <span className="text-xs text-white/40">{key.lastUsed}</span>
            <div className="flex items-center justify-between">
              <span className={`text-xs px-2 py-0.5 rounded ${key.status === 'active' ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/30'}`}>
                {key.status}
              </span>
              <button className="text-white/20 hover:text-red-400 transition-colors">
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─── Rate Limits Panel ─── */
function RateLimitsPanel() {
  const { data: limits, isLoading } = useRateLimits();

  const mockLimits = [
    { scope: 'Global (Default)', rps: 1000, rpm: 60000, burst: 2000, tier: 'enterprise' },
    { scope: 'memory:write', rps: 100, rpm: 6000, burst: 200, tier: 'standard' },
    { scope: 'graph:query', rps: 50, rpm: 3000, burst: 100, tier: 'standard' },
    { scope: 'debug:context', rps: 10, rpm: 600, burst: 20, tier: 'restricted' },
  ];

  const displayLimits = limits ?? mockLimits;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white">Rate Limit Configuration</h3>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30 border border-blue-500/30">
          <Plus className="w-4 h-4" /> Add Rule
        </button>
      </div>

      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl overflow-hidden">
        <div className="grid grid-cols-5 gap-4 p-4 bg-white/5 text-xs font-medium text-white/40 uppercase tracking-wider">
          <span className="col-span-2">Scope</span><span>Req/s</span><span>Req/min</span><span>Burst</span>
        </div>
        {displayLimits.map((limit: any) => (
          <div key={limit.scope} className="grid grid-cols-5 gap-4 p-4 border-t border-white/5 items-center hover:bg-white/5">
            <div className="col-span-2 flex items-center gap-2">
              <span className="text-sm text-white">{limit.scope}</span>
              <span className={`text-xs px-1.5 py-0.5 rounded ${
                limit.tier === 'enterprise' ? 'bg-amber-500/20 text-amber-400' :
                limit.tier === 'restricted' ? 'bg-red-500/20 text-red-400' :
                'bg-white/10 text-white/40'
              }`}>{limit.tier}</span>
            </div>
            <input type="number" defaultValue={limit.rps}
              className="w-20 px-2 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500/50" />
            <input type="number" defaultValue={limit.rpm}
              className="w-24 px-2 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500/50" />
            <input type="number" defaultValue={limit.burst}
              className="w-20 px-2 py-1.5 bg-white/5 border border-white/10 rounded text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500/50" />
          </div>
        ))}
      </div>

      <div className="flex justify-end">
        <button className="px-6 py-2.5 bg-blue-500 text-white font-medium rounded-lg hover:bg-blue-400 transition-colors">
          Save Rate Limits
        </button>
      </div>
    </div>
  );
}

/* ─── Webhooks Panel ─── */
function WebhooksPanel() {
  const { data: webhooks, isLoading, isError, refetch } = useWebhooks();

  const mockWebhooks = [
    { id: 'wh_1', url: 'https://api.example.com/webhooks/vnp', events: ['memory.created', 'memory.deleted', 'profile.updated'], status: 'active', successRate: 99.2 },
    { id: 'wh_2', url: 'https://monitoring.example.com/vnp-hook', events: ['engine.health_degraded', 'pipeline.failed'], status: 'active', successRate: 100 },
    { id: 'wh_3', url: 'https://legacy.example.com/old-hook', events: ['memory.created'], status: 'inactive', successRate: 45.0 },
  ];

  const displayWebhooks = webhooks ?? mockWebhooks;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white">Webhook Endpoints</h3>
        <button className="flex items-center gap-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30 border border-blue-500/30">
          <Plus className="w-4 h-4" /> Add Webhook
        </button>
      </div>

      <div className="space-y-3">
        {displayWebhooks.map((wh: any) => (
          <div key={wh.id} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5 hover:border-white/20">
            <div className="flex items-start justify-between mb-3">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Globe className="w-4 h-4 text-blue-400" />
                  <code className="text-sm text-blue-400 font-mono">{wh.url}</code>
                </div>
                <div className="flex flex-wrap gap-1 mt-2">
                  {wh.events.map((ev: string) => (
                    <span key={ev} className="text-xs px-2 py-0.5 bg-blue-500/10 text-blue-300/70 rounded border border-blue-500/20">{ev}</span>
                  ))}
                </div>
              </div>
              <div className="flex items-center gap-3 flex-shrink-0 ml-4">
                <div className="text-right">
                  <p className="text-xs text-white/40">Success rate</p>
                  <p className={`text-sm font-semibold ${wh.successRate >= 90 ? 'text-green-400' : wh.successRate >= 50 ? 'text-yellow-400' : 'text-red-400'}`}>
                    {wh.successRate}%
                  </p>
                </div>
                <span className={`text-xs px-2 py-1 rounded-full ${wh.status === 'active' ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/30'}`}>
                  {wh.status}
                </span>
                <button className="text-white/20 hover:text-red-400">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─── SDK Quickstart ─── */
const SDK_EXAMPLES: Record<string, string> = {
  curl: `curl -X POST https://gateway.vnp-memory.io/v1/memory/add \\
  -H "Authorization: Bearer vnp_prod_sk_..." \\
  -H "x-tenant-id: my-org" \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "user-123",
    "content": "User prefers TypeScript and dark themes",
    "engine": "supermemory",
    "metadata": {"source": "chat", "session_id": "sess-abc"}
  }'`,
  python: `from vnp_memory import VNPClient

client = VNPClient(
    api_key="vnp_prod_sk_...",
    tenant_id="my-org"
)

# Add memory to Supermemory (adaptive)
memory = await client.memory.add(
    user_id="user-123",
    content="User prefers TypeScript and dark themes",
    engine="supermemory",
    metadata={"source": "chat"}
)

# Retrieve context for LLM prompt
context = await client.context.assemble(
    agent_id="agent-001",
    user_id="user-123",
    query="What are the user's preferences?",
    engines=["supermemory", "memobase", "graphiti"],
    top_k=5,
)`,
  typescript: `import { VNPClient } from '@vnp-memory/sdk';

const client = new VNPClient({
  apiKey: process.env.VNP_API_KEY!,
  tenantId: 'my-org',
});

// Add episodic memory (Graphiti)
const memory = await client.memory.add({
  userId: 'user-123',
  content: 'User asked about memory system design patterns',
  engine: 'graphiti',
  metadata: { source: 'chat', sessionId: 'sess-xyz' },
});

// Assemble context for LLM
const { context, totalTokens } = await client.context.assemble({
  agentId: 'agent-001',
  userId: 'user-123',
  query: 'What does the user know about memory systems?',
  engines: ['graphiti', 'cognee', 'memobase'],
  topK: 5,
  minScore: 0.7,
});`,
  go: `import "github.com/vnp-memory/go-sdk/vnp"

client := vnp.NewClient(vnp.Config{
    APIKey:   os.Getenv("VNP_API_KEY"),
    TenantID: "my-org",
})

// Add procedural memory (OpenViking)
memory, err := client.Memory.Add(ctx, &vnp.AddMemoryRequest{
    UserID:  "user-123",
    Content: "User's preferred deployment workflow for Go services",
    Engine:  "openviking",
    Metadata: map[string]any{"source": "agent"},
})`,
};

function SdkQuickstart() {
  const [lang, setLang] = useState<keyof typeof SDK_EXAMPLES>('typescript');

  const copyCode = () => {
    navigator.clipboard.writeText(SDK_EXAMPLES[lang]).catch(console.error);
  };

  return (
    <div className="space-y-6">
      <div className="bg-[#1a1a1f] border border-white/10 rounded-xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Quickstart Examples</h3>
        <div className="flex gap-2 mb-4">
          {Object.keys(SDK_EXAMPLES).map(l => (
            <button key={l} onClick={() => setLang(l as any)}
              className={`px-4 py-2 rounded-lg text-sm transition-colors ${
                lang === l ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'bg-white/5 text-white/50 hover:bg-white/10'
              }`}>
              {l === 'typescript' ? 'TypeScript' : l === 'python' ? 'Python' : l === 'go' ? 'Go' : 'cURL'}
            </button>
          ))}
        </div>
        <div className="relative">
          <button onClick={copyCode} className="absolute top-3 right-3 p-2 bg-white/10 hover:bg-white/20 rounded text-white/60">
            <Copy className="w-4 h-4" />
          </button>
          <pre className="p-5 bg-black/50 rounded-xl border border-white/10 text-xs font-mono text-green-300/80 overflow-x-auto whitespace-pre leading-relaxed">
            {SDK_EXAMPLES[lang]}
          </pre>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {[
          { icon: Terminal, title: 'CLI Tool', desc: 'Install the VNP CLI', cmd: 'npm i -g @vnp-memory/cli' },
          { icon: Code2, title: 'OpenAPI Spec', desc: 'Download the full API schema', cmd: 'GET /openapi.json' },
          { icon: Globe, title: 'MCP Server', desc: 'Model Context Protocol integration', cmd: 'npx @vnp-memory/mcp-server' },
        ].map(item => (
          <div key={item.title} className="bg-[#1a1a1f] border border-white/10 rounded-xl p-5">
            <item.icon className="w-5 h-5 text-blue-400 mb-3" />
            <h4 className="text-sm font-medium text-white mb-1">{item.title}</h4>
            <p className="text-xs text-white/40 mb-3">{item.desc}</p>
            <code className="text-xs text-blue-300/70 bg-black/30 px-2 py-1 rounded">{item.cmd}</code>
          </div>
        ))}
      </div>
    </div>
  );
}
