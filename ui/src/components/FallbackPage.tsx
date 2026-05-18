import React from 'react';

interface FallbackProps {
  type?: '404' | '500' | 'chunk-error' | 'generic';
  error?: Error | null;
  onRetry?: () => void;
}

export const FallbackPage: React.FC<FallbackProps> = ({ type = 'generic', error, onRetry }) => {
  const configs = {
    '404': {
      emoji: '🔍',
      title: 'Page Not Found',
      description: 'The page you are looking for does not exist or has been moved.',
    },
    '500': {
      emoji: '⚠️',
      title: 'Internal Server Error',
      description: 'Something went wrong on our end. Please try again later.',
    },
    'chunk-error': {
      emoji: '📦',
      title: 'Failed to Load Module',
      description: 'A new version may have been deployed. Please refresh the page.',
    },
    generic: {
      emoji: '😵',
      title: 'Đã có lỗi xảy ra',
      description: 'An unexpected error occurred in this component.',
    },
  };

  const config = configs[type];

  return (
    <div className="flex-1 flex items-center justify-center bg-[#0a0a0f] p-8">
      <div className="text-center max-w-md">
        <div className="text-6xl mb-6">{config.emoji}</div>
        <h1 className="text-2xl font-bold text-white mb-3">{config.title}</h1>
        <p className="text-zinc-400 mb-6">{config.description}</p>
        {error && (
          <pre className="text-xs text-red-400/70 bg-red-900/10 border border-red-500/20 rounded p-3 mb-6 text-left overflow-auto max-h-32">
            {error.message}
          </pre>
        )}
        <div className="flex gap-3 justify-center">
          {onRetry && (
            <button
              onClick={onRetry}
              className="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
            >
              Thử lại
            </button>
          )}
          <button
            onClick={() => window.location.reload()}
            className="px-5 py-2.5 bg-zinc-800 hover:bg-zinc-700 text-white rounded-lg font-medium transition-colors border border-zinc-700"
          >
            Tải lại trang
          </button>
        </div>
      </div>
    </div>
  );
};
