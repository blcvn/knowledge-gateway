import { ReactNode } from 'react';
import { BrainCircuit } from 'lucide-react';

interface AuthLayoutProps {
  children: ReactNode;
  title: string;
  subtitle: string;
}

export function AuthLayout({ children, title, subtitle }: AuthLayoutProps) {
  return (
    <div className="min-h-screen w-full flex bg-[#0f0f14] text-white">
      {/* Left side - Graphic/Branding */}
      <div className="hidden lg:flex flex-1 relative overflow-hidden bg-gradient-to-br from-indigo-900/40 via-purple-900/20 to-[#0f0f14] items-center justify-center">
        {/* Abstract Background Shapes */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-indigo-500/20 rounded-full mix-blend-screen filter blur-3xl animate-blob" />
        <div className="absolute top-1/3 right-1/4 w-96 h-96 bg-purple-500/20 rounded-full mix-blend-screen filter blur-3xl animate-blob animation-delay-2000" />
        <div className="absolute bottom-1/4 left-1/3 w-96 h-96 bg-pink-500/20 rounded-full mix-blend-screen filter blur-3xl animate-blob animation-delay-4000" />

        <div className="relative z-10 max-w-lg p-12 backdrop-blur-sm bg-white/5 border border-white/10 rounded-2xl shadow-2xl">
          <div className="flex items-center gap-3 mb-8">
            <div className="p-3 bg-indigo-500/20 rounded-xl border border-indigo-500/30">
              <BrainCircuit className="w-8 h-8 text-indigo-400" />
            </div>
            <span className="text-3xl font-bold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent">
              VNP Memory
            </span>
          </div>
          <h2 className="text-4xl font-semibold leading-tight mb-6">
            The intelligent memory platform for your agents.
          </h2>
          <p className="text-white/60 text-lg">
            Empower your AI agents with long-term memory, observability, and adaptive context management.
          </p>
        </div>
      </div>

      {/* Right side - Form */}
      <div className="flex-1 flex flex-col justify-center px-4 sm:px-6 lg:px-20 xl:px-32 relative">
        {/* Mobile Header */}
        <div className="lg:hidden flex items-center gap-2 absolute top-8 left-6">
          <div className="p-2 bg-indigo-500/20 rounded-lg border border-indigo-500/30">
            <BrainCircuit className="w-6 h-6 text-indigo-400" />
          </div>
          <span className="text-xl font-bold">VNP Memory</span>
        </div>

        <div className="mx-auto w-full max-w-sm">
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-white mb-2">{title}</h1>
            <p className="text-white/50">{subtitle}</p>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
