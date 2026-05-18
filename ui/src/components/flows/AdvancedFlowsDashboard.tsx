import React from 'react';

export const P1AgentDeveloperFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50">
    <h3 className="text-lg font-bold text-white">P1 - AI Agent Developer Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: Debugging Agent Context</li>
      <li>Flow 2: Auto-ontology Evaluation</li>
      <li>Flow 3: Temporal Timeline Query</li>
      <li>Flow 4: Auto-forget Configuration</li>
      <li>Flow 5: Fallback RAG Testing</li>
      <li>Flow 6: Multi-Agent Memory Sharing</li>
      <li>Flow 7: Bulk Import</li>
      <li>Flow 8: Export Graph Snapshot</li>
    </ul>
  </div>
);

export const P2DevOpsEngineerFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P2 - Platform / DevOps Engineer Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: System Health Monitoring</li>
      <li>Flow 2: System Troubleshooting</li>
      <li>Flow 3: Tenant Provisioning</li>
      <li>Flow 9: Rollback Engine Config</li>
      <li>Flow 10: SSL/TLS & Webhook Auth Management</li>
      <li>Flow 11: Data Archiving</li>
      <li>Flow 12: Kill Switch</li>
    </ul>
  </div>
);

export const P3MLEngineerFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P3 - ML/AI Engineer Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: Knowledge Graph Exploration</li>
      <li>Flow 2: Vector Pipeline Optimization</li>
      <li>Flow 3: Evaluation Pipeline Run</li>
      <li>Flow 13: Custom Embedding Model Integration</li>
      <li>Flow 14: NER Precision Testing</li>
      <li>Flow 15: Cost Analysis (Token vs Context)</li>
    </ul>
  </div>
);

export const P4EnterpriseArchitectFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P4 - Enterprise Architect Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: Right to be Forgotten (GDPR)</li>
      <li>Flow 2: ABAC Policy Update</li>
      <li>Flow 3: Audit Trail</li>
      <li>Flow 4: Data Retention & TTL</li>
      <li>Flow 16: SOC2 / HIPAA Audit Report</li>
      <li>Flow 17: RBAC Configuration</li>
      <li>Flow 18: Data Masking/PII Redaction</li>
    </ul>
  </div>
);

export const AdvancedFlowsDashboard = () => {
  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-6 text-white">Advanced Navigation Flows Dashboard</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <P1AgentDeveloperFlows />
        <P2DevOpsEngineerFlows />
        <P3MLEngineerFlows />
        <P4EnterpriseArchitectFlows />
      </div>
    </div>
  );
};
