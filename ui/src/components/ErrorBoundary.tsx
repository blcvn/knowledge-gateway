import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error:', error, errorInfo);
    // Here you would log to an error reporting service like Sentry
  }

  public render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="p-6 bg-red-900/20 border border-red-500/50 rounded-lg text-white">
          <h2 className="text-lg font-semibold mb-2">Đã có lỗi xảy ra ở thành phần này</h2>
          <p className="text-sm text-gray-300 mb-4">{this.state.error?.message}</p>
          <button 
            className="px-4 py-2 bg-red-600 hover:bg-red-700 rounded transition-colors"
            onClick={() => this.setState({ hasError: false, error: null })}
          >
            Thử lại
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
