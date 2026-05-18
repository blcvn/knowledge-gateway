/**
 * Enterprise Logger Module
 * Provides controlled logging that disables console output in production.
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const isProduction = import.meta.env.PROD;
const minLevel: LogLevel = isProduction ? 'error' : 'debug';

function shouldLog(level: LogLevel): boolean {
  return LOG_LEVELS[level] >= LOG_LEVELS[minLevel];
}

export const logger = {
  debug: (...args: unknown[]) => {
    if (shouldLog('debug')) console.debug('[DEBUG]', ...args);
  },
  info: (...args: unknown[]) => {
    if (shouldLog('info')) console.info('[INFO]', ...args);
  },
  warn: (...args: unknown[]) => {
    if (shouldLog('warn')) console.warn('[WARN]', ...args);
  },
  error: (...args: unknown[]) => {
    if (shouldLog('error')) {
      console.error('[ERROR]', ...args);
      // In production, send to error tracking service (e.g., Sentry)
      // if (typeof window !== 'undefined' && window.Sentry) {
      //   window.Sentry.captureException(args[0]);
      // }
    }
  },
};
