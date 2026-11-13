import { element, cssNumber } from '../dom.js';

/* components/panes.js — 分栏宽度调节与左侧栏收起/展开
   - 两条中线分隔条：pointer capture + rAF，拖动时实时改写相邻栏宽度
   - 用户偏好（pref）与实际生效宽度（applied）分离：窗口缩放只重算生效值，
     不会永久改变用户拖出来的宽度，布局始终满足最小边界
   - 双击复位、方向键微调、Ctrl+B 收起、localStorage 记忆 */
export function createPanes({ onResize, logger, errors, storageKey, platform }) {
  const MIN_SIDEBAR = cssNumber('--sidebar-min-w');
  const MAX_SIDEBAR = cssNumber('--sidebar-max-w');
  const RAIL_W = cssNumber('--sidebar-rail-w');
  const MIN_PREVIEW = cssNumber('--preview-min-w');
  const MIN_EDITOR = cssNumber('--editor-min-w');
  const SPLIT = cssNumber('--splitter-w');
  const DEF_SIDEBAR = cssNumber('--sidebar-default-w');
  const DEF_PREVIEW = cssNumber('--preview-default-w');

  const root = document.documentElement;
  const workspace = element('workspace');
  const left = element('splitLeft');
  const right = element('splitRight');
  if (!workspace || !left || !right) return;

  const pref = { sidebar: DEF_SIDEBAR, preview: DEF_PREVIEW, collapsed: false };
  const applied = { sidebar: DEF_SIDEBAR, preview: DEF_PREVIEW };
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey) || 'null');
    if (saved && typeof saved === 'object') {
      if (Number.isFinite(saved.sidebar)) pref.sidebar = saved.sidebar;
      if (Number.isFinite(saved.preview)) pref.preview = saved.preview;
      pref.collapsed = !!saved.collapsed;
    }
  } catch (cause) {
    logger.warn('pane_preferences_load_failed', cause, errors.codes.STORAGE_READ_FAILED);
  }

  function clamp(v, lo, hi) {
    return Math.max(lo, Math.min(hi, v));
  }
  function narrow() {
    return getComputedStyle(workspace).display === 'block';
  }
  function total() {
    return workspace.getBoundingClientRect().width;
  }
  function effSidebar() {
    return pref.collapsed ? RAIL_W : pref.sidebar;
  }
  function maxPreview(sb) {
    return Math.max(MIN_PREVIEW, total() - sb - SPLIT * 2 - MIN_EDITOR);
  }
  function maxSidebar(pv) {
    return Math.max(MIN_SIDEBAR, Math.min(MAX_SIDEBAR, total() - pv - SPLIT * 2 - MIN_EDITOR));
  }

  function save() {
    try {
      localStorage.setItem(storageKey, JSON.stringify(pref));
    } catch (cause) {
      logger.warn('pane_preferences_save_failed', cause, errors.codes.STORAGE_WRITE_FAILED);
    }
  }

  // 由偏好计算实际可用宽度：优先保证编辑区不小于 MIN_EDITOR
  function compute() {
    if (narrow()) {
      applied.sidebar = pref.sidebar;
      applied.preview = pref.preview;
      return;
    }
    const budget = total() - SPLIT * 2 - MIN_EDITOR;
    let sb = clamp(pref.sidebar, MIN_SIDEBAR, MAX_SIDEBAR);
    sb = Math.min(sb, Math.max(MIN_SIDEBAR, budget - MIN_PREVIEW));
    const eff = pref.collapsed ? RAIL_W : sb;
    applied.sidebar = sb;
    applied.preview = clamp(pref.preview, MIN_PREVIEW, Math.max(MIN_PREVIEW, budget - eff));
  }

  function apply(emit) {
    root.style.setProperty('--sidebar-w', applied.sidebar + 'px');
    root.style.setProperty('--preview-w', applied.preview + 'px');
    root.classList.toggle('sidebar-collapsed', pref.collapsed);
    left.classList.toggle('disabled', pref.collapsed);
    left.setAttribute('aria-disabled', pref.collapsed ? 'true' : 'false');
    const b = element('btnSidebar');
    if (b) {
      b.setAttribute('aria-pressed', String(pref.collapsed));
      b.setAttribute('aria-label', pref.collapsed ? '展开侧栏' : '收起侧栏');
    }
    const c = element('btnCollapse');
    if (c) c.setAttribute('aria-label', pref.collapsed ? '展开侧栏' : '收起侧栏');
    if (emit) {
      onResize();
    }
  }

  function update(emit) {
    compute();
    apply(emit !== false);
  }

  // ---------------------------------------------------------------- 拖拽
  function beginDrag(e, which) {
    if (narrow() || e.button !== 0) return;
    if (which === 'left' && pref.collapsed) return;
    const sp = which === 'left' ? left : right;
    e.preventDefault();
    const startX = e.clientX;
    const startSidebar = pref.sidebar,
      startPreview = pref.preview;
    document.body.classList.add('is-resizing');
    sp.classList.add('dragging');
    sp.setAttribute('aria-grabbed', 'true');
    try {
      sp.setPointerCapture(e.pointerId);
    } catch {
      /* document listeners remain available without pointer capture. */
    }

    let pending = 0,
      raf = 0;
    function frame() {
      raf = 0;
      if (which === 'left') {
        pref.sidebar = clamp(startSidebar + pending, MIN_SIDEBAR, maxSidebar(startPreview));
      } else {
        pref.preview = clamp(startPreview - pending, MIN_PREVIEW, maxPreview(effSidebar()));
      }
      compute();
      apply(true);
    }
    function move(ev) {
      pending = ev.clientX - startX;
      if (!raf) raf = requestAnimationFrame(frame);
    }
    function end() {
      if (raf) {
        cancelAnimationFrame(raf);
        frame();
      }
      document.removeEventListener('pointermove', move);
      document.removeEventListener('pointerup', end);
      document.removeEventListener('pointercancel', end);
      document.body.classList.remove('is-resizing');
      sp.classList.remove('dragging');
      sp.setAttribute('aria-grabbed', 'false');
      try {
        sp.releasePointerCapture(e.pointerId);
      } catch {
        /* Pointer capture may already have been released. */
      }
      save();
      apply(true);
    }
    document.addEventListener('pointermove', move);
    document.addEventListener('pointerup', end);
    document.addEventListener('pointercancel', end);
  }

  function reset(which) {
    if (which === 'left') pref.sidebar = DEF_SIDEBAR;
    else pref.preview = DEF_PREVIEW;
    save();
    update(true);
  }

  function keyAdjust(e, which) {
    if (e.key === 'Home') {
      e.preventDefault();
      reset(which);
      return;
    }
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    const delta = (e.key === 'ArrowRight' ? 1 : -1) * (e.shiftKey ? 32 : 16);
    if (which === 'left')
      pref.sidebar = clamp(pref.sidebar + delta, MIN_SIDEBAR, maxSidebar(pref.preview));
    else pref.preview = clamp(pref.preview - delta, MIN_PREVIEW, maxPreview(effSidebar()));
    save();
    update(true);
  }

  function toggleSidebar(force) {
    pref.collapsed = typeof force === 'boolean' ? force : !pref.collapsed;
    save();
    update(true);
  }

  // ---------------------------------------------------------------- 绑定
  left.addEventListener('pointerdown', (e) => {
    beginDrag(e, 'left');
  });
  right.addEventListener('pointerdown', (e) => {
    beginDrag(e, 'right');
  });
  left.addEventListener('dblclick', () => {
    reset('left');
  });
  right.addEventListener('dblclick', () => {
    reset('right');
  });
  left.addEventListener('keydown', (e) => {
    keyAdjust(e, 'left');
  });
  right.addEventListener('keydown', (e) => {
    keyAdjust(e, 'right');
  });
  element('btnSidebar').addEventListener('click', () => {
    toggleSidebar();
  });
  element('btnCollapse').addEventListener('click', () => {
    toggleSidebar();
  });
  document.addEventListener('keydown', (e) => {
    if (platform.matchesShortcut(e, 'b')) {
      e.preventDefault();
      toggleSidebar();
    }
  });

  // 窗口尺寸变化时重新计算生效宽度，保持布局不溢出
  window.addEventListener('resize', () => {
    update(true);
  });
  window.addEventListener('load', () => {
    update(true);
  });
  if (typeof ResizeObserver !== 'undefined') {
    new ResizeObserver(() => {
      update(true);
    }).observe(workspace);
  }

  update(true);
}
