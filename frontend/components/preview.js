import { element, cssNumber } from '../dom.js';
import { copyContent } from '../content.js';

export function createPreview({ backend, notices, logger, errors }) {
  const paperWidth = cssNumber('--paper-w');
  const paperHeight = cssNumber('--paper-h');
  const previewCSS = `<style>
html,body{margin:0!important;padding:0!important;background:#fff!important}
.page{margin:8mm auto!important;box-shadow:none!important}
.page:first-of-type{margin-top:0!important}
</style>`;
  let zoom = 0.6;
  let autoFit = true;
  let previewTimer = 0;
  let fitFrame = 0;
  let request = 0;

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
      element('previewFrame').srcdoc = html.replace('</head>', previewCSS + '</head>');
      element('previewHint').textContent = '与导出成品使用同一份模板';
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
