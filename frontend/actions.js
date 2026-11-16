import { element } from './dom.js';
import { copyContent } from './content.js';

// Queued saves capture content and file identity before touching the bridge.
export function createDocumentActions({
  backend,
  content,
  getDocument,
  onSaved,
  notices,
  logger,
  errors,
  platform,
}) {
  let saveTimer = 0;
  let pending = null;
  let queue = Promise.resolve();
  let revision = 0;

  function snapshot() {
    const document = getDocument();
    return {
      filename: document.filename,
      jd: document.jd,
      content: copyContent(content),
      revision,
    };
  }

  function failure(cause) {
    notices.status('保存失败', 'error');
    notices.failure(logger.error('save_failed', cause, errors.codes.CONTENT_SAVE_FAILED));
  }

  async function flush() {
    clearTimeout(saveTimer);
    const next = pending;
    pending = null;
    if (!next) return queue;
    queue = queue
      .catch(() => {})
      .then(async () => {
        try {
          const info = await backend.saveDocument(next.filename, next.content, next.jd);
          await onSaved(info, next.revision === revision);
          if (next.revision === revision) notices.status('已保存');
          logger.info('save_completed');
        } catch (cause) {
          if (!pending && next.revision === revision) pending = next;
          throw cause;
        }
      });
    return queue;
  }

  function scheduleSave() {
    revision++;
    pending = snapshot();
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => flush().catch(failure), 900);
  }

  async function save(quiet = true, savedSnapshot = snapshot()) {
    if (element('workspace').inert && !quiet) return;
    pending = savedSnapshot;
    await flush();
    if (!quiet) notices.toast('当前简历已保存');
  }

  async function exportPDF() {
    const button = element('btnPdf');
    button.disabled = true;
    try {
      const exported = snapshot();
      await save(true, exported);
      notices.status('正在导出 PDF…', 'busy');
      const result = await backend.export(exported.filename, exported.content);
      notices.status('已保存');
      notices.toast('PDF 已导出：' + result);
      logger.info('export_completed');
    } catch (cause) {
      notices.status('导出失败', 'error');
      notices.failure(logger.error('export_failed', cause, errors.codes.EXPORT_FAILED));
    } finally {
      button.disabled = false;
    }
  }

  element('btnPdf').addEventListener('click', exportPDF);
  document.addEventListener('keydown', (event) => {
    if (platform.matchesShortcut(event, 's')) {
      event.preventDefault();
      save(false).catch(failure);
    }
  });

  return { scheduleSave, flush, save };
}
