// Error codes and safe messages come from internal/apperr, via generated data.
export function createErrorTools(messages) {
  const codes = Object.freeze(
    Object.fromEntries(Object.keys(messages).map((code) => [code, code])),
  );

  function create(code, cause) {
    return Object.assign(new Error(messages[code]), { code, cause });
  }

  function normalize(cause, fallback = codes.APP_UNEXPECTED) {
    if (cause?.code && messages[cause.code]) return create(cause.code, cause);
    try {
      const parsed = JSON.parse(cause?.message || String(cause));
      if (messages[parsed.code]) return create(parsed.code, cause);
    } catch {
      // Native and browser exceptions need not carry our bridge envelope.
    }
    return create(fallback, cause);
  }

  function describe(cause) {
    const error = normalize(cause);
    return error.message + ' [' + error.code + ']';
  }

  return { codes, create, normalize, describe };
}
