/** @param {import('./contracts.js').ResumeBackend} backend */
export function createLogger(backend, errors) {
  function write(level, event, cause, fallback) {
    const error = level === 'info' ? null : errors.normalize(cause, fallback);
    const entry = { level, event, ...(error ? { code: error.code } : {}) };
    console[level](JSON.stringify({ time: new Date().toISOString(), ...entry }));
    backend.log(entry).catch(() => {
      console.warn(
        JSON.stringify({
          time: new Date().toISOString(),
          level: 'warn',
          event: 'log_forward_failed',
          code: errors.codes.LOG_WRITE_FAILED,
        }),
      );
    });
    return error;
  }

  return {
    info: (event) => write('info', event),
    warn: (event, cause, fallback) => write('warn', event, cause, fallback),
    error: (event, cause, fallback) => write('error', event, cause, fallback),
  };
}
