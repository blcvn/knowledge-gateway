import { createRoot }          from 'react-dom/client';
import { QueryClientProvider }  from '@tanstack/react-query';
import { BrowserRouter }        from 'react-router';
import { queryClient }          from './lib/query-client';
import { AuthProvider }         from './lib/auth';
import App                      from './app/App.tsx';
import './styles/index.css';

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <AuthProvider>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </AuthProvider>
  </QueryClientProvider>
);