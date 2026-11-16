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
    listDocuments: 'Resume.ListDocuments',
    workspace: 'Resume.Workspace',
    progress: 'Resume.Progress',
    openDocument: 'Resume.OpenDocument',
    createDocument: 'Resume.CreateDocument',
    deleteDocument: 'Resume.DeleteDocument',
    saveDocument: 'Resume.SaveDocument',
    getWebDAVSettings: 'Resume.GetWebDAVSettings',
    saveWebDAVSettings: 'Resume.SaveWebDAVSettings',
    testWebDAVConnection: 'Resume.TestWebDAVConnection',
    syncWebDAV: 'Resume.SyncWebDAV',
    resolveWebDAVConflict: 'Resume.ResolveWebDAVConflict',
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
    export: (filename, content) => invoke('export', errors.codes.EXPORT_FAILED, filename, content),
    log: (entry) => bridge.call(methods.log, entry),
    listDocuments: () => invoke('listDocuments', errors.codes.CONTENT_LOAD_FAILED),
    workspace: () => invoke('workspace', errors.codes.CONTENT_LOAD_FAILED),
    progress: (content) => invoke('progress', errors.codes.DOCUMENT_INVALID, content),
    openDocument: (filename) => invoke('openDocument', errors.codes.CONTENT_LOAD_FAILED, filename),
    createDocument: (input) => invoke('createDocument', errors.codes.CONTENT_SAVE_FAILED, input),
    deleteDocument: (filename) =>
      invoke('deleteDocument', errors.codes.DOCUMENT_DELETE_FAILED, filename),
    saveDocument: (filename, content, jd) =>
      invoke('saveDocument', errors.codes.CONTENT_SAVE_FAILED, filename, content, jd),
    getWebDAVSettings: () => invoke('getWebDAVSettings', errors.codes.STORAGE_READ_FAILED),
    saveWebDAVSettings: (settings) =>
      invoke('saveWebDAVSettings', errors.codes.STORAGE_WRITE_FAILED, settings),
    testWebDAVConnection: (settings) =>
      invoke('testWebDAVConnection', errors.codes.WEBDAV_CONNECT_FAILED, settings),
    syncWebDAV: () => invoke('syncWebDAV', errors.codes.WEBDAV_SYNC_FAILED),
    resolveWebDAVConflict: (filename, choice) =>
      invoke('resolveWebDAVConflict', errors.codes.WEBDAV_CONFLICT_CHANGED, filename, choice),
  };
}
