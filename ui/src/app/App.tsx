import { useState, lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router';
import { Sidebar } from './components/Sidebar';
import { TopNav } from './components/TopNav';
import { ErrorBoundary } from './components/ErrorBoundary';
import { Loader2 } from 'lucide-react';

/* ─── Auth Modules ─── */
import { Login } from '../components/auth/Login';
import { Register } from '../components/auth/Register';
import { ProtectedRoute } from '../components/auth/ProtectedRoute';

/* ─── Lazy-loaded modules for code splitting ─── */
const Dashboard           = lazy(() => import('./components/Dashboard').then(m => ({ default: m.Dashboard })));
const MemoryExplorer      = lazy(() => import('./components/MemoryExplorer').then(m => ({ default: m.MemoryExplorer })));
const GraphStudio         = lazy(() => import('./components/GraphStudio').then(m => ({ default: m.GraphStudio })));
const UserProfiles        = lazy(() => import('./components/UserProfiles').then(m => ({ default: m.UserProfiles })));
const AdaptiveMemory      = lazy(() => import('./components/AdaptiveMemory').then(m => ({ default: m.AdaptiveMemory })));
const AgentContextDebugger = lazy(() => import('./components/AgentContextDebugger').then(m => ({ default: m.AgentContextDebugger })));
const SessionsExplorer    = lazy(() => import('./components/SessionsExplorer').then(m => ({ default: m.SessionsExplorer })));
const GovernanceCenter    = lazy(() => import('./components/GovernanceCenter').then(m => ({ default: m.GovernanceCenter })));
const PipelinesMonitor    = lazy(() => import('./components/PipelinesMonitor').then(m => ({ default: m.PipelinesMonitor })));
const InfrastructureHealth = lazy(() => import('./components/InfrastructureHealth').then(m => ({ default: m.InfrastructureHealth })));
const ObservabilityError  = lazy(() => import('./components/ObservabilityError').then(m => ({ default: m.ObservabilityError })));
const ApiSdkManager       = lazy(() => import('./components/ApiSdkManager').then(m => ({ default: m.ApiSdkManager })));
const OrganizationSettings = lazy(() => import('./components/OrganizationSettings').then(m => ({ default: m.OrganizationSettings })));

/* ─── Module loading fallback ─── */
function ModuleLoadingFallback() {
  return (
    <div className="flex-1 flex items-center justify-center flex-col gap-3">
      <Loader2 className="w-8 h-8 text-white/30 animate-spin" />
      <p className="text-white/30 text-sm">Loading module...</p>
    </div>
  );
}

function DashboardLayout() {
  const [activeSection, setActiveSection] = useState('overview');
  const [currentTenant] = useState('Acme Corporation');
  const [currentEnvironment] = useState('Production');

  const renderContent = () => {
    switch (activeSection) {
      case 'overview':      return <Dashboard />;
      case 'memory':        return <MemoryExplorer />;
      case 'graph':         return <GraphStudio />;
      case 'profiles':      return <UserProfiles />;
      case 'adaptive':      return <AdaptiveMemory />;
      case 'debugger':      return <AgentContextDebugger />;
      case 'sessions':      return <SessionsExplorer />;
      case 'governance':    return <GovernanceCenter />;
      case 'pipelines':     return <PipelinesMonitor />;
      case 'infrastructure': return <InfrastructureHealth />;
      case 'observability': return <ObservabilityError />;
      case 'api':           return <ApiSdkManager />;
      case 'settings':      return <OrganizationSettings />;
      default:              return <Dashboard />;
    }
  };

  return (
    <div className="size-full flex bg-[#0f0f14] dark">
      <Sidebar activeSection={activeSection} onSectionChange={setActiveSection} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <TopNav currentTenant={currentTenant} currentEnvironment={currentEnvironment} />
        <ErrorBoundary>
          <Suspense fallback={<ModuleLoadingFallback />}>
            {renderContent()}
          </Suspense>
        </ErrorBoundary>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route 
        path="/*" 
        element={
          <ProtectedRoute>
            <DashboardLayout />
          </ProtectedRoute>
        } 
      />
    </Routes>
  );
}