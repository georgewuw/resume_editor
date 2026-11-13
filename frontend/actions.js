import { element } from './dom.js';
import { copyContent } from './content.js';

// Save scheduling and export orchestration are independent of form rendering.
export function createDocumentActions({ backend, content, notices, logger, errors, platform }) {
  let saveTimer = 0;

  async function save(quiet = true) {
    clearTimeout(saveTimer);
    try {
      await backend.save(copyContent(content));
      notices.status('已保存');
      if (!quiet) notices.toast('内容已保存');
      logger.info('save_completed');
    } catch (cause) {
      notices.status('保存失败', 'error');
      notices.failure(logger.error('save_failed', cause, errors.codes.CONTENT_SAVE_FAILED));
    }
  }

  function scheduleSave() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => save(), 900);
  }

  async function exportPDF() {
    clearTimeout(saveTimer);
    notices.status('正在导出 PDF…', 'busy');
    logger.info('export_started');
    try {
      const snapshot = copyContent(content);
      await backend.save(snapshot);
      const result = await backend.export(snapshot);
      notices.status('已保存');
      notices.toast('PDF 已导出：' + result);
      logger.info('export_completed');
    } catch (cause) {
      notices.status('导出失败', 'error');
      notices.failure(logger.error('export_failed', cause, errors.codes.EXPORT_FAILED));
    }
  }

  element('btnPdf').addEventListener('click', exportPDF);
  document.addEventListener('keydown', (event) => {
    if (platform.matchesShortcut(event, 's')) {
      event.preventDefault();
      save(false);
    }
  });

  return { scheduleSave };
}
