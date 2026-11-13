import { element } from '../dom.js';
import { copyContent } from '../content.js';

/** Owns navigation, field controls, search and completion display. */
export function createEditor({
  schema,
  groups,
  content,
  onChange,
  onSelect,
  getProgress,
  onProgressError,
}) {
  const state = { active: '', search: '' };
  const navProgress = new Map();
  let progressRequest = 0;

  function renderNav() {
    const ul = element('navList');
    ul.innerHTML = '';
    navProgress.clear();
    groups.forEach((g, i) => {
      const li = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'nav-item' + (g.id === state.active ? ' active' : '');
      if (g.id === state.active) button.setAttribute('aria-current', 'true');
      button.innerHTML =
        '<span class="nav-num">' +
        String(i + 1).padStart(2, '0') +
        '</span>' +
        '<span class="nav-copy"><span class="nav-name"></span><span class="nav-detail"></span></span>' +
        '<span class="nav-count"></span>';
      button.querySelector('.nav-name').textContent = g.title;
      button.querySelector('.nav-detail').textContent = g.items.length + ' 个字段';
      button.title = g.title;
      button.addEventListener('click', () => selectGroup(g.id));
      navProgress.set(g.id, button.querySelector('.nav-count'));
      li.appendChild(button);
      ul.appendChild(li);
    });
    element('totalCount').textContent = String(schema.length);
    updateProgress();
  }

  function selectGroup(id) {
    state.active = id;
    state.search = '';
    element('searchInput').value = '';
    const g = groups.find((x) => x.id === id);
    if (!g) return;
    const i = groups.indexOf(g);
    element('groupEyebrow').textContent = '模块 ' + String(i + 1).padStart(2, '0') + ' / ' + g.en;
    element('groupTitle').textContent = g.title;
    element('groupDesc').textContent = g.description;
    onSelect(g);
    renderNav();
    renderForm();
    element('editor').scrollTop = 0;
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
    hint.textContent = f.hint || (f.type === 'lines' ? '每行一条' : '');
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
    input.value = (content.fields[f.key] || []).join('\n');
    input.placeholder = f.placeholder || '请输入' + f.label;
    input.addEventListener('input', onInput);

    const key = document.createElement('span');
    key.className = 'field-key';
    key.textContent = f.key;
    wrap.append(head, input, key);
    return wrap;
  }

  function renderForm() {
    const form = element('form');
    form.innerHTML = '';
    const term = state.search.trim().toLowerCase();
    let items;
    if (term) {
      items = schema.filter(
        (f) =>
          f.label.toLowerCase().includes(term) ||
          f.key.toLowerCase().includes(term) ||
          (content.fields[f.key] || []).join('').toLowerCase().includes(term),
      );
    } else {
      const g = groups.find((x) => x.id === state.active);
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
    const lines = e.target.value.split('\n').filter((x) => x.trim() !== '');
    onChange(key, lines.length ? lines : ['【待填写】']);
    updateProgress();
  }

  async function updateProgress() {
    const request = ++progressRequest;
    let progress;
    try {
      progress = await getProgress(copyContent(content));
    } catch (cause) {
      if (request === progressRequest) onProgressError(cause);
      return;
    }
    if (request !== progressRequest) return;
    const n = progress.filled;
    groups.forEach((group) => {
      const count = progress.groups[group.id] || 0;
      const node = navProgress.get(group.id);
      if (!node) return;
      node.textContent = count + '/' + group.items.length;
      node.classList.toggle('complete', count === group.items.length);
      node.setAttribute('aria-label', '已填写 ' + count + ' / ' + group.items.length + ' 项');
    });
    element('filledCount').textContent = String(n);
    const ratio = schema.length ? n / schema.length : 0;
    element('progressBar').style.width = Math.round(ratio * 100) + '%';
    element('filledPct').textContent = Math.round(ratio * 100) + '%';
    element('collapsedProgressValue').style.strokeDasharray = ratio * 100 + ' 100';
    element('collapsedProgressCount').textContent = n + '/' + schema.length;
    element('collapsedProgress').setAttribute(
      'aria-label',
      '已填写 ' + n + ' / ' + schema.length + ' 项',
    );
  }

  element('searchInput').addEventListener('input', (event) => {
    state.search = event.target.value;
    renderForm();
  });
  updateProgress();
  function refresh() {
    renderNav();
    renderForm();
  }

  return { selectGroup, refresh };
}
