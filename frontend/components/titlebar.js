import { element } from '../dom.js';

// The segmented control chooses a workspace; modules live only in the sidebar.
export function createTitlebar({ platform, onModeChange }) {
  const segments = [element('tabContent'), element('tabFiles')];
  element('btnSidebar').title = '收起 / 展开侧栏（' + platform.shortcutLabel('b') + '）';

  function setMode(mode) {
    segments.forEach((segment) => {
      segment.setAttribute('aria-pressed', String(segment.dataset.mode === mode));
    });
    element('workspace').dataset.mode = mode;
    element('contentNavigation').hidden = mode !== 'content';
    element('fileNavigation').hidden = mode !== 'files';
    element('editor').hidden = mode !== 'content';
    element('fileManager').hidden = mode !== 'files';
    element('sidebar').setAttribute('aria-label', mode === 'content' ? '内容模块' : '简历文件');
    onModeChange(mode);
  }

  segments.forEach((segment) => {
    segment.addEventListener('click', () => setMode(segment.dataset.mode));
  });

  function setDocument(info) {
    element('activeFilename').textContent = info.filename;
    element('activeFilename').title = info.filename;
  }

  return { setMode, setDocument };
}
