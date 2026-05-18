import React from 'react';

/**
 * Lazy-loaded route components for code splitting.
 * Heavy modules (Recharts, CodeMirror, ReactFlow) are only loaded
 * when the user navigates to the corresponding screen.
 */

export const LazyDashboard = React.lazy(
  () => import('../app/components/Dashboard').then(m => ({ default: m.Dashboard }))
);

export const LazyMemoryExplorer = React.lazy(
  () => import('../app/components/MemoryExplorer').then(m => ({ default: m.MemoryExplorer }))
);

export const LazyGraphStudio = React.lazy(
  () => import('../app/components/GraphStudio').then(m => ({ default: m.GraphStudio }))
);

export const LazyAgentContextDebugger = React.lazy(
  () => import('../app/components/AgentContextDebugger').then(m => ({ default: m.AgentContextDebugger }))
);

export const LazySessionsExplorer = React.lazy(
  () => import('../app/components/SessionsExplorer').then(m => ({ default: m.SessionsExplorer }))
);

export const LazyGovernanceCenter = React.lazy(
  () => import('../app/components/GovernanceCenter').then(m => ({ default: m.GovernanceCenter }))
);

export const LazyPipelinesMonitor = React.lazy(
  () => import('../app/components/PipelinesMonitor').then(m => ({ default: m.PipelinesMonitor }))
);

export const LazyInfrastructureHealth = React.lazy(
  () => import('../app/components/InfrastructureHealth').then(m => ({ default: m.InfrastructureHealth }))
);

export const LazyObservabilityError = React.lazy(
  () => import('../app/components/ObservabilityError').then(m => ({ default: m.ObservabilityError }))
);

export const LazyApiSdkManager = React.lazy(
  () => import('../app/components/ApiSdkManager').then(m => ({ default: m.ApiSdkManager }))
);

export const LazyOrganizationSettings = React.lazy(
  () => import('../app/components/OrganizationSettings').then(m => ({ default: m.OrganizationSettings }))
);
