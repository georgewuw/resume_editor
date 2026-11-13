import { element, cssNumber } from '../dom.js';
import { copyContent } from '../content.js';

export function createPreview({ backend, notices, logger, errors }) {
  const paperWidth = cssNumber('--paper-w');
  const paperHeight = cssNumber('--paper-h');
  const previewCSS = `
html,body{margin:0!important;padding:0!important;background:#edf0f3!important}
.preview-page-guide{display:flex;align-items:center;gap:12px;width:210mm;height:36px;margin:0 auto;color:#5b6673;font:18px/1.4 "Microsoft YaHei",sans-serif;white-space:nowrap;user-select:none;pointer-events:none}
.preview-page-guide::before,.preview-page-guide::after{content:"";flex:1;height:1.5px;background:repeating-linear-gradient(to right,#98a2ae 0 7px,transparent 7px 12px)}
.page{margin:0 auto!important;box-shadow:none!important}
.page:last-of-type{margin-bottom:0!important}
@media print{
  html,body{background:#fff!important}
  .page{margin:0!important}
  .preview-page-guide{display:none}
}`;
  let zoom = 0.6;
  let autoFit = true;
  let previewTimer = 0;
  let fitFrame = 0;
  let request = 0;

  // Guides belong only to the sandboxed preview; export uses the original HTML.
  function withPageGuides(html) {
    const document = new DOMParser().parseFromString(html, 'text/html');
    const style = document.createElement('style');
    style.textContent = previewCSS;
    document.head.append(style);
    document.querySelectorAll('.page').forEach((page, index) => {
      const guide = document.createElement('div');
      guide.className = 'preview-page-guide';
      guide.textContent = '第 ' + (index + 1) + ' 页 ↓';
      page.before(guide);
    });
    return '<!doctype html>\n' + document.documentElement.outerHTML;
  }

  function applyZoom() {
    element('zoomLabel').textContent = Math.round(zoom * 100) + '%';
    element('paperHolder').style.width = Math.round(paperWidth * zoom) + 'px';
    element('paperHolder').style.height = Math.round(paperHeight * zoom) + 'px';
    element('paperStage').style.transform = 'scale(' + zoom + ')';
  }

  function fit() {
    if (!autoFit || fitFrame) return;
    fitFrame = requestAnimationFrame(() => {
      fitFrame = 0;
      const scroll = element('previewScroll');
      const style = getComputedStyle(scroll);
      const available =
        scroll.clientWidth -
        Number.parseFloat(style.paddingLeft) -
        Number.parseFloat(style.paddingRight);
      if (available > 200) {
        zoom = Math.min(0.85, Math.max(0.35, Math.floor((available / paperWidth) * 10) / 10));
        applyZoom();
      }
    });
  }

  /** @param {import('../contracts.js').ResumeContent} content */
  async function refresh(content) {
    const current = ++request;
    try {
      const html = await backend.fill(copyContent(content));
      if (current !== request) return;
      element('previewFrame').srcdoc = withPageGuides(html);
      element('previewHint').textContent = '分页线与页码仅供预览，不会导出';
    } catch (cause) {
      if (current !== request) return;
      const error = logger.error('preview_failed', cause, errors.codes.PREVIEW_FAILED);
      element('previewHint').textContent = errors.describe(error);
    }
  }

  function schedule(content) {
    clearTimeout(previewTimer);
    previewTimer = setTimeout(() => refresh(content), 280);
  }

  element('zoomIn').addEventListener('click', () => {
    autoFit = false;
    zoom = Math.min(0.95, Number((zoom + 0.1).toFixed(2)));
    applyZoom();
  });
  element('zoomOut').addEventListener('click', () => {
    autoFit = false;
    zoom = Math.max(0.35, Number((zoom - 0.1).toFixed(2)));
    applyZoom();
  });
  element('zoomLabel').addEventListener('click', () => {
    autoFit = true;
    fit();
    notices.toast('已恢复自适应缩放');
  });
  window.addEventListener('resize', fit);
  new ResizeObserver(fit).observe(element('previewScroll'));
  fit();
  return { refresh, schedule, fit };
}
