/* 简历编辑器 · 同一份前端，两种运行方式
   - mygo 桌面模式：通过 window.mygo.call 调用 Go 服务读写文件、导出 PDF
   - 浏览器降级模式：直接用浏览器打开 dist/index.html，本地填充 + 本地存储

   界面按 MyGit 的方式拆分为「按区域划分的模块」：
     components/titlebar.js  顶栏交互与窗口控制
     components/panes.js     分栏宽度调节与侧栏折叠
     app.js                  数据、表单、预览的编排（本文件）
   模块之间只通过 document 上的自定义事件通信，互不直接依赖。 */
(function () {
  'use strict';
  const $ = id => document.getElementById(id);
  const PREVIEW_W = 794;
  const PREVIEW_H = 2380;
  const PREVIEW_CSS = `<style>
html,body{margin:0!important;padding:0!important;background:#fff!important}
.toolbar{display:none!important}
.page{margin:8mm auto!important;box-shadow:none!important}
.page:first-of-type{margin-top:0!important}
</style>`;

  const isMyGo = !!(window.mygo && typeof window.mygo.call === 'function');
  const schema = window.__RESUME_SCHEMA__ || [];
  const initial = window.__RESUME_INITIAL__ || { fields: {} };
  const tpl = window.__RESUME_TEMPLATE__ || '';

  const state = {
    content: { fields: {} },
    groups: [],
    active: '',
    search: '',
    zoom: 0.6,
    autoFit: true,
    saveTimer: 0,
    previewTimer: 0,
    toastTimer: 0,
  };

  // ---------------------------------------------------------------- 工具
  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    })[c]);
  }

  // 与 Go 端 FillTemplate 完全相同的扫描器，保证两种模式渲染一致
  function fillTemplate(src, fields) {
    let out = '', rest = src;
    for (;;) {
      const idx = rest.indexOf('data-f="');
      if (idx < 0) { out += rest; break; }
      const ks = idx + 8;
      const ke = rest.indexOf('"', ks);
      if (ke < 0) throw new Error('模板中存在未闭合的 data-f 属性');
      const key = rest.slice(ks, ke);
      const openStart = rest.lastIndexOf('<', idx);
      const openEnd = rest.indexOf('>', idx);
      if (openStart < 0 || openEnd < 0) throw new Error('占位标签不完整：' + key);
      const attrs = rest.slice(openStart + 1, openEnd);
      const tag = (/^\w+/.exec(attrs) || ['span'])[0];
      const closeTag = '</' + tag + '>';
      const closeRel = rest.indexOf(closeTag, openEnd + 1);
      if (closeRel < 0) throw new Error('缺少闭合标签：' + key);
      const lines = fields[key] || [];
      let inner = '';
      if (attrs.indexOf('data-multiline') >= 0) {
        inner = lines.map(l => '<span class="ln">' + escapeHtml(l) + '</span>').join('');
      } else if (key === 'photo') {
        const val = lines.join('\n').trim();
        const m = /^!\[(.*?)\]\((.+?)\)$/.exec(val);
        inner = m
          ? '<img src="' + escapeHtml(m[2]) + '" alt="' + escapeHtml(m[1] || '证件照') + '">'
          : '<span class="photo-tip">' + escapeHtml(val) + '</span>';
      } else {
        inner = lines.map(escapeHtml).join('<br>');
      }
      out += rest.slice(0, openStart) + '<' + attrs + '>' + inner + closeTag;
      rest = rest.slice(closeRel + closeTag.length);
    }
    return out.replace('<body data-layout="double">',
      '<body data-layout="double" class="filled">');
  }

  function toast(msg, isError) {
    const el = $('toast');
    el.textContent = msg;
    el.className = 'toast show' + (isError ? ' error' : '');
    clearTimeout(state.toastTimer);
    state.toastTimer = setTimeout(() => { el.className = 'toast'; }, 3200);
  }

  function setStatus(text, kind) {
    $('saveText').textContent = text;
    $('saveState').className = 'save-state' + (kind ? ' ' + kind : '');
  }

  // ---------------------------------------------------------------- 后端
  const mygoBackend = {
    load: () => window.mygo.call('Resume.Load'),
    fill: c => window.mygo.call('Resume.Fill', c),
    save: c => window.mygo.call('Resume.Save', c),
    export: c => window.mygo.call('Resume.Export', c),
    info: () => window.mygo.call('Resume.Info'),
  };

  const browserBackend = {
    async load() {
      try {
        const saved = localStorage.getItem('resume-editor-content');
        if (saved) return JSON.parse(saved);
      } catch (e) { /* file:// 下可能不可用 */ }
      return JSON.parse(JSON.stringify(initial));
    },
    async fill(c) {
      if (!tpl) throw new Error('缺少内嵌模板，请先运行 node scripts/build-web.mjs');
      return fillTemplate(tpl, c.fields);
    },
    async save(c) {
      try { localStorage.setItem('resume-editor-content', JSON.stringify(c)); }
      catch (e) { /* 忽略 */ }
      return '浏览器本地存储';
    },
    async export(c) {
      if (!tpl) throw new Error('缺少内嵌模板，请先运行 node scripts/build-web.mjs');
      const html = fillTemplate(tpl, c.fields);
      const url = URL.createObjectURL(new Blob([html], { type: 'text/html;charset=utf-8' }));
      const w = window.open(url, '_blank');
      if (!w) {
        URL.revokeObjectURL(url);
        throw new Error('打印窗口被拦截，请允许弹出窗口后重试');
      }
      w.addEventListener('load', () => setTimeout(() => { try { w.print(); } catch (e) {} }, 500));
      setTimeout(() => URL.revokeObjectURL(url), 30000);
      return '已打开打印窗口（另存为 PDF）';
    },
    async info() { return { mode: 'browser', dataDir: '浏览器本地存储' }; },
  };

  const backend = isMyGo ? mygoBackend : browserBackend;

  // ---------------------------------------------------------------- 渲染
  function buildGroups() {
    const list = [];
    for (const f of schema) {
      let g = list.find(x => x.id === f.group);
      if (!g) {
        g = { id: f.group, title: f.groupTitle, en: f.groupEn, items: [] };
        list.push(g);
      }
      g.items.push(f);
    }
    state.groups = list;
    // 顶栏模块切换器与左侧导航共用这份数据
    document.dispatchEvent(new CustomEvent('resume:groups', { detail: { groups: list } }));
  }

  const DESCS = {
    personal: '姓名与三行信息按整页中心居中，证件照固定在内容区右上角。',
    advantage: '首行为不加粗的一句话简述，其余条目为「加粗要点 + 描述」。',
    work: '四列网格对齐：起止时间 / 公司 / 部门 / 职位。',
    project: '第 1~4 条落在第 1 页，第 5~6 条续在第 2 页，标题不重复。',
    education: '三列网格对齐：起止时间 / 学校 / 专业学位。',
    certificate: '每行一条，生成时自动渲染为独立行。',
  };

  function renderNav() {
    const ul = $('navList');
    ul.innerHTML = '';
    state.groups.forEach((g, i) => {
      const li = document.createElement('li');
      li.className = 'nav-item' + (g.id === state.active ? ' active' : '');
      li.innerHTML = '<span class="nav-num">' + String(i + 1).padStart(2, '0') + '</span>' +
        '<span class="nav-name"></span><span class="nav-count">' + g.items.length + '</span>';
      li.querySelector('.nav-name').textContent = g.title;
      li.title = g.title + '（' + g.en + '）';
      li.addEventListener('click', () => selectGroup(g.id));
      ul.appendChild(li);
    });
    $('totalCount').textContent = String(schema.length);
  }

  function selectGroup(id) {
    state.active = id;
    state.search = '';
    $('searchInput').value = '';
    const g = state.groups.find(x => x.id === id);
    if (!g) return;
    const i = state.groups.indexOf(g);
    $('groupEyebrow').textContent = '模块 ' + String(i + 1).padStart(2, '0') + ' / ' + g.en;
    $('groupTitle').textContent = g.title;
    $('groupDesc').textContent = DESCS[g.id] || '';
    document.dispatchEvent(new CustomEvent('resume:group', {
      detail: { id: g.id, title: g.title, en: g.en }
    }));
    renderNav();
    renderForm();
    $('editor').scrollTop = 0;
  }

  function fieldNode(f) {
    const wrap = document.createElement('div');
    wrap.className = 'field' + (f.type === 'lines' ? ' lines' : '');
    const head = document.createElement('div');
    head.className = 'field-head';
    const label = document.createElement('label');
    label.setAttribute('for', 'f_' + f.key);
    label.textContent = f.label;
    const hint = document.createElement('span');
    hint.className = 'field-hint';
    hint.textContent = f.type === 'lines' ? '每行一条' : (f.key === 'photo' ? '可填图片路径' : '');
    head.append(label, hint);

    let input;
    if (f.type === 'text') {
      input = document.createElement('input');
      input.type = 'text';
    } else {
      input = document.createElement('textarea');
      input.rows = f.type === 'lines' ? 4 : 3;
    }
    input.id = 'f_' + f.key;
    input.dataset.key = f.key;
    input.value = (state.content.fields[f.key] || []).join('\n');
    input.placeholder = f.key === 'photo'
      ? '【证件照】 或 ![证件照](./photo.jpg)'
      : '请输入' + f.label;
    input.addEventListener('input', onInput);

    const key = document.createElement('span');
    key.className = 'field-key';
    key.textContent = f.key;
    wrap.append(head, input, key);
    return wrap;
  }

  function renderForm() {
    const form = $('form');
    form.innerHTML = '';
    const term = state.search.trim().toLowerCase();
    let items;
    if (term) {
      items = schema.filter(f =>
        f.label.toLowerCase().includes(term) ||
        f.key.toLowerCase().includes(term) ||
        (state.content.fields[f.key] || []).join('').toLowerCase().includes(term));
    } else {
      const g = state.groups.find(x => x.id === state.active);
      items = g ? g.items : [];
    }
    if (!items.length) {
      const e = document.createElement('p');
      e.className = 'empty';
      e.textContent = '没有匹配的字段';
      form.appendChild(e);
      return;
    }
    let lastGroup = '';
    for (const f of items) {
      if (term && f.group !== lastGroup) {
        lastGroup = f.group;
        const d = document.createElement('div');
        d.className = 'group-divider';
        d.textContent = f.groupTitle;
        form.appendChild(d);
      }
      form.appendChild(fieldNode(f));
    }
  }

  function onInput(e) {
    const key = e.target.dataset.key;
    const lines = e.target.value.split('\n').filter(x => x.trim() !== '');
    state.content.fields[key] = lines.length ? lines : ['【待填写】'];
    setStatus('编辑中…', 'busy');
    scheduleSave();
    schedulePreview();
    updateProgress();
  }

  function updateProgress() {
    let n = 0;
    for (const f of schema) {
      const cur = (state.content.fields[f.key] || []).join('\n').trim();
      const ini = (initial.fields[f.key] || []).join('\n').trim();
      if (cur && cur !== ini) n++;
    }
    $('filledCount').textContent = String(n);
    $('progressBar').style.width = Math.round(n / schema.length * 100) + '%';
  }

  function scheduleSave() {
    clearTimeout(state.saveTimer);
    state.saveTimer = setTimeout(saveNow, 900);
  }

  async function saveNow(quiet) {
    clearTimeout(state.saveTimer);
    try {
      await backend.save(state.content);
      setStatus('已保存', '');
      if (!quiet) toast('内容已保存');
    } catch (err) {
      setStatus('保存失败', 'error');
      toast('保存失败：' + (err && err.message ? err.message : err), true);
    }
  }

  function schedulePreview() {
    clearTimeout(state.previewTimer);
    state.previewTimer = setTimeout(refreshPreview, 280);
  }

  async function refreshPreview() {
    try {
      const html = await backend.fill(state.content);
      $('previewFrame').srcdoc = html.replace('</head>', PREVIEW_CSS + '</head>');
      $('previewHint').textContent = isMyGo
        ? '与导出成品使用同一份模板'
        : '浏览器模式 · 打印时请选择 A4 / 100% / 背景图形';
    } catch (err) {
      $('previewHint').textContent = '预览失败：' + (err && err.message ? err.message : err);
    }
  }

  function applyZoom() {
    const z = state.zoom;
    $('zoomLabel').textContent = Math.round(z * 100) + '%';
    $('paperHolder').style.width = Math.round(PREVIEW_W * z) + 'px';
    $('paperHolder').style.height = Math.round(PREVIEW_H * z) + 'px';
    $('paperStage').style.transform = 'scale(' + z + ')';
  }

  // 预览栏宽度变化（拖动分隔条、折叠侧栏、窗口缩放）时自动适配缩放，
  // 用户手动 +/- 后切换为手动模式，点缩放数值可恢复自适应。
  function fitZoom() {
    if (!state.autoFit) return;
    const avail = $('previewScroll').clientWidth - 28;
    if (avail > 200) {
      state.zoom = Math.min(0.85, Math.max(0.35, Math.floor(avail / PREVIEW_W * 10) / 10));
      applyZoom();
    }
  }

  // ---------------------------------------------------------------- 事件
  function bind() {
    $('searchInput').addEventListener('input', e => {
      state.search = e.target.value;
      renderForm();
    });
    $('zoomIn').addEventListener('click', () => {
      state.autoFit = false;
      state.zoom = Math.min(0.95, +(state.zoom + 0.1).toFixed(2));
      applyZoom();
    });
    $('zoomOut').addEventListener('click', () => {
      state.autoFit = false;
      state.zoom = Math.max(0.35, +(state.zoom - 0.1).toFixed(2));
      applyZoom();
    });
    $('zoomLabel').addEventListener('click', () => {
      state.autoFit = true;
      fitZoom();
      toast('已恢复自适应缩放');
    });
    $('btnPdf').addEventListener('click', () => doExport());
    document.addEventListener('keydown', e => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault();
        saveNow();
      }
    });
    document.addEventListener('resume:select-group', e => {
      const id = e.detail && e.detail.id;
      if (id) selectGroup(id);
    });
    let fitRaf = 0;
    document.addEventListener('resume:pane-resize', () => {
      if (fitRaf) return;
      fitRaf = requestAnimationFrame(() => { fitRaf = 0; fitZoom(); });
    });
    window.addEventListener('resize', fitZoom);
  }

  async function doExport() {
    setStatus('正在导出 PDF…', 'busy');
    try {
      await backend.save(state.content);
      const result = await backend.export(state.content);
      setStatus('已保存', '');
      toast('PDF 已导出' + (isMyGo ? '：' + result : ''));
    } catch (err) {
      setStatus('导出失败', 'error');
      toast('PDF 导出失败：' + (err && err.message ? err.message : err), true);
    }
  }

  // ---------------------------------------------------------------- 启动
  async function boot() {
    buildGroups();
    try {
      const loaded = await backend.load();
      state.content = loaded && loaded.fields ? loaded : { fields: {} };
    } catch (err) {
      state.content = JSON.parse(JSON.stringify(initial));
      toast('读取内容失败，已使用默认占位：' + err.message, true);
    }
    // 补齐模板中新增但旧数据里没有的字段
    for (const f of schema) {
      if (!state.content.fields[f.key] || !state.content.fields[f.key].length) {
        state.content.fields[f.key] = (initial.fields[f.key] || ['【待填写】']).slice();
      }
    }
    const info = await backend.info().catch(() => ({ mode: isMyGo ? 'mygo' : 'browser', dataDir: '' }));
    $('modeNote').textContent = isMyGo
      ? '桌面模式 · 内容保存在 ' + (info.dataDir || '应用数据目录')
      : '浏览器模式 · 内容保存在本地浏览器，Ctrl+S 保存';

    bind();
    fitZoom();
    selectGroup(state.groups.length ? state.groups[0].id : '');
    updateProgress();
    refreshPreview();
    setStatus('已同步', '');
  }

  boot();
})();
