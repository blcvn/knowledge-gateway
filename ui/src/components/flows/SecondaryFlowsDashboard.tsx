import React from 'react';

export const P5IDEPluginUserFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50">
    <h3 className="text-lg font-bold text-white">P5 - IDE Plugin User Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: Connect IDE to VNP Memory</li>
      <li>Flow 2: Sync Workspace Context</li>
      <li>Flow 3: Query Local/Remote Graph</li>
      <li>Flow 21: Revoke Compromised Key</li>
    </ul>
  </div>
);

export const P6FrameworkIntegratorFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P6 - Framework Integrator Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: View Webhook Logs</li>
      <li>Flow 2: Configure Sync Endpoints</li>
      <li>Flow 22: Monitor Delivery Failures</li>
    </ul>
  </div>
);

export const P7PowerUserFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P7 - AI Power User Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: Personal Graph Overview</li>
      <li>Flow 19: Manual Entity Merging</li>
      <li>Flow 20: Export Personal Vault</li>
    </ul>
  </div>
);

export const P8ProductManagerFlows = () => (
  <div className="p-4 border border-zinc-800 rounded bg-zinc-900/50 mt-4">
    <h3 className="text-lg font-bold text-white">P8 - Product Manager Flows</h3>
    <ul className="list-disc pl-5 mt-2 text-zinc-400">
      <li>Flow 1: View Usage Analytics</li>
      <li>Flow 2: Monitor AI Feature Adoption</li>
      <li>Flow 3: Export Product Reports</li>
    </ul>
  </div>
);

export const SecondaryFlowsDashboard = () => {
  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-6 text-white">Secondary Ecosystem & User Flows</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <P5IDEPluginUserFlows />
        <P6FrameworkIntegratorFlows />
        <P7PowerUserFlows />
        <P8ProductManagerFlows />
      </div>
    </div>
  );
};
