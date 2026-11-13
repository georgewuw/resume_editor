/* components/panes.js — 分栏宽度调节与左侧栏收起/展开
   - 两条中线分隔条：pointer capture + rAF，拖动时实时改写相邻栏宽度
   - 用户偏好（pref）与实际生效宽度（applied）分离：窗口缩放只重算生效值，
     不会永久改变用户拖出来的宽度，布局始终满足最小边界
   - 双击复位、方向键微调、Ctrl+B 收起、localStorage 记忆 */
(function () {
  'use strict';
  var UI = window.ResumeUI = window.ResumeUI || {};
  var $ = function (id) { return document.getElementById(id); };
  var KEY = 'resume-editor-panes';

  var MIN_SIDEBAR = 168, MAX_SIDEBAR = 360, RAIL_W = 52;
  var MIN_PREVIEW = 380, MIN_EDITOR = 320, SPLIT = 7;
  var DEF_SIDEBAR = 220, DEF_PREVIEW = 520;

  var root = document.documentElement;
  var workspace = $('workspace');
  var left = $('splitLeft');
  var right = $('splitRight');
  var preview = $('preview');
  if (!workspace || !left || !right) return;

  var pref = { sidebar: DEF_SIDEBAR, preview: DEF_PREVIEW, collapsed: false };
  var applied = { sidebar: DEF_SIDEBAR, preview: DEF_PREVIEW };
  try {
    var saved = JSON.parse(localStorage.getItem(KEY) || 'null');
    if (saved && typeof saved === 'object') {
      if (Number.isFinite(saved.sidebar)) pref.sidebar = saved.sidebar;
      if (Number.isFinite(saved.preview)) pref.preview = saved.preview;
      pref.collapsed = !!saved.collapsed;
    }
  } catch (e) { /* file:// 下 localStorage 可能不可用 */ }

  function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }
  function narrow() { return window.matchMedia('(max-width: 920px)').matches; }
  function total() { return workspace.getBoundingClientRect().width; }
  function effSidebar() { return pref.collapsed ? RAIL_W : pref.sidebar; }
  function maxPreview(sb) { return Math.max(MIN_PREVIEW, total() - sb - SPLIT * 2 - MIN_EDITOR); }
  function maxSidebar(pv) { return Math.max(MIN_SIDEBAR, Math.min(MAX_SIDEBAR, total() - pv - SPLIT * 2 - MIN_EDITOR)); }

  function save() {
    try { localStorage.setItem(KEY, JSON.stringify(pref)); } catch (e) { /* 忽略 */ }
  }

  // 由偏好计算实际可用宽度：优先保证编辑区不小于 MIN_EDITOR
  function compute() {
    if (narrow()) { applied.sidebar = pref.sidebar; applied.preview = pref.preview; return; }
    var budget = total() - SPLIT * 2 - MIN_EDITOR;
    var sb = clamp(pref.sidebar, MIN_SIDEBAR, MAX_SIDEBAR);
    sb = Math.min(sb, Math.max(MIN_SIDEBAR, budget - MIN_PREVIEW));
    var eff = pref.collapsed ? RAIL_W : sb;
    applied.sidebar = sb;
    applied.preview = clamp(pref.preview, MIN_PREVIEW, Math.max(MIN_PREVIEW, budget - eff));
  }

  function apply(emit) {
    root.style.setProperty('--sidebar-w', applied.sidebar + 'px');
    root.style.setProperty('--preview-w', applied.preview + 'px');
    root.classList.toggle('sidebar-collapsed', pref.collapsed);
    left.classList.toggle('disabled', pref.collapsed);
    left.setAttribute('aria-disabled', pref.collapsed ? 'true' : 'false');
    var b = $('btnSidebar');
    if (b) {
      b.setAttribute('aria-pressed', String(pref.collapsed));
      b.setAttribute('aria-label', pref.collapsed ? '展开侧栏' : '收起侧栏');
    }
    var c = $('btnCollapse');
    if (c) c.setAttribute('aria-label', pref.collapsed ? '展开侧栏' : '收起侧栏');
    if (emit) {
      document.dispatchEvent(new CustomEvent('resume:pane-resize', {
        detail: { previewWidth: preview ? preview.getBoundingClientRect().width : applied.preview }
      }));
    }
  }

  function update(emit) { compute(); apply(emit !== false); }

  // ---------------------------------------------------------------- 拖拽
  function beginDrag(e, which) {
    if (narrow() || e.button !== 0) return;
    if (which === 'left' && pref.collapsed) return;
    var sp = which === 'left' ? left : right;
    e.preventDefault();
    var startX = e.clientX;
    var startSidebar = pref.sidebar, startPreview = pref.preview;
    document.body.classList.add('is-resizing');
    sp.classList.add('dragging');
    sp.setAttribute('aria-grabbed', 'true');
    try { sp.setPointerCapture(e.pointerId); } catch (err) { /* 降级到 document 监听 */ }

    var pending = 0, raf = 0;
    function frame() {
      raf = 0;
      if (which === 'left') {
        pref.sidebar = clamp(startSidebar + pending, MIN_SIDEBAR, maxSidebar(startPreview));
      } else {
        pref.preview = clamp(startPreview - pending, MIN_PREVIEW, maxPreview(effSidebar()));
      }
      compute(); apply(true);
    }
    function move(ev) {
      pending = ev.clientX - startX;
      if (!raf) raf = requestAnimationFrame(frame);
    }
    function end() {
      if (raf) { cancelAnimationFrame(raf); frame(); }
      document.removeEventListener('pointermove', move);
      document.removeEventListener('pointerup', end);
      document.removeEventListener('pointercancel', end);
      document.body.classList.remove('is-resizing');
      sp.classList.remove('dragging');
      sp.setAttribute('aria-grabbed', 'false');
      try { sp.releasePointerCapture(e.pointerId); } catch (err) { /* 忽略 */ }
      save(); apply(true);
    }
    document.addEventListener('pointermove', move);
    document.addEventListener('pointerup', end);
    document.addEventListener('pointercancel', end);
  }

  function reset(which) {
    if (which === 'left') pref.sidebar = DEF_SIDEBAR;
    else pref.preview = DEF_PREVIEW;
    save(); update(true);
  }

  function keyAdjust(e, which) {
    if (e.key === 'Home') { e.preventDefault(); reset(which); return; }
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    var delta = (e.key === 'ArrowRight' ? 1 : -1) * (e.shiftKey ? 32 : 16);
    if (which === 'left') pref.sidebar = clamp(pref.sidebar + delta, MIN_SIDEBAR, maxSidebar(pref.preview));
    else pref.preview = clamp(pref.preview - delta, MIN_PREVIEW, maxPreview(effSidebar()));
    save(); update(true);
  }

  function toggleSidebar(force) {
    pref.collapsed = typeof force === 'boolean' ? force : !pref.collapsed;
    save(); update(true);
  }

  // ---------------------------------------------------------------- 绑定
  left.addEventListener('pointerdown', function (e) { beginDrag(e, 'left'); });
  right.addEventListener('pointerdown', function (e) { beginDrag(e, 'right'); });
  left.addEventListener('dblclick', function () { reset('left'); });
  right.addEventListener('dblclick', function () { reset('right'); });
  left.addEventListener('keydown', function (e) { keyAdjust(e, 'left'); });
  right.addEventListener('keydown', function (e) { keyAdjust(e, 'right'); });
  $('btnSidebar').addEventListener('click', function () { toggleSidebar(); });
  $('btnCollapse').addEventListener('click', function () { toggleSidebar(); });
  document.addEventListener('keydown', function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'b') {
      e.preventDefault(); toggleSidebar();
    }
  });

  // 窗口尺寸变化时重新计算生效宽度，保持布局不溢出
  window.addEventListener('resize', function () { update(true); });
  window.addEventListener('load', function () { update(true); });
  if (typeof ResizeObserver !== 'undefined') {
    new ResizeObserver(function () { update(true); }).observe(workspace);
  }

  update(true);
  UI.panes = {
    toggle: toggleSidebar,
    reset: function () { pref.sidebar = DEF_SIDEBAR; pref.preview = DEF_PREVIEW; save(); update(true); },
    get collapsed() { return pref.collapsed; },
    get sizes() { return { sidebar: applied.sidebar, preview: applied.preview }; }
  };
})();
