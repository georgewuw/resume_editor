/** Desktop bridge adapter. All service names are owned by this module.
 * @returns {import('./contracts.js').ResumeBackend}
 */
export function createBackend(errors) {
  const bridge = window.mygo;
  if (!bridge || typeof bridge.call !== 'function') {
    throw errors.create(errors.codes.BRIDGE_CALL_FAILED);
  }
  const methods = {
    load: 'Resume.Load',
    fill: 'Resume.Fill',
    save: 'Resume.Save',
    export: 'Resume.Export',
    log: 'Resume.Log',
  };

  async function invoke(method, code, ...args) {
    try {
      return await bridge.call(methods[method], ...args);
    } catch (cause) {
      throw errors.normalize(cause, code);
    }
  }

  return {
    platform: bridge.platform,
    load: () => invoke('load', errors.codes.CONTENT_LOAD_FAILED),
    fill: (content) => invoke('fill', errors.codes.PREVIEW_FAILED, content),
    save: (content) => invoke('save', errors.codes.CONTENT_SAVE_FAILED, content),
    export: (content) => invoke('export', errors.codes.EXPORT_FAILED, content),
    log: (entry) => bridge.call(methods.log, entry),
  };
}
