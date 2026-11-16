import { element } from '../dom.js';
import { copyContent } from '../content.js';
import { createIcon } from './icons.js';
import { createRepeatable } from './repeatable.js';
import { toggleBold } from './bold.js';

/** Owns navigation, field controls, search and completion display. */
export function createEditor({
  schema,
  groups,
  content,
  onChange,
  onEntriesChange,
  onSelect,
  getProgress,
  onProgressError,
}) {
  const state = { active: '', search: '' };
  const navProgress = new Map();
  let progressRequest = 0;
  const repeatableOptions = {
    schema,
    content,
    fieldNode,
    fieldValue,
    onChange: onEntriesChange,
    onRefresh() {
      renderForm();
      updateProgress();
    },
  };
  const work = createRepeatable({
    ...repeatableOptions,
    group: 'work',
    slotPattern: /^work([1-5])(Period|Company|Dept|Role|Desc)$/,
    fields: [
      { key: 'Period', label: '起止时间' },
      { key: 'Company', label: '公司名称' },
      { key: 'Dept', label: '部门' },
      { key: 'Role', label: '职位名称' },
      { key: 'Desc', label: '工作描述' },
    ],
    titleKey: 'Company',
    itemLabel: '经历',
    emptyTitle: '添加你的第一段工作经历',
    emptyHint: '记录任职时间、公司、职位和主要成果。',
  });
  const projects = createRepeatable({
    ...repeatableOptions,
    group: 'project',
    slotPattern: /^proj([1-6])(Name|Desc)$/,
    fields: [
      { key: 'Name', label: '项目名称' },
      { key: 'Desc', label: '项目描述' },
    ],
    titleKey: 'Name',
    itemLabel: '项目',
    itemUnit: '个',
    emptyTitle: '添加你的第一个项目',
    emptyHint: '记录项目背景、你的职责和取得的成果。',
  });
  const advantages = createRepeatable({
    ...repeatableOptions,
    group: 'advantage',
    slotPattern: /^adv([1-6])(Lead|Text)$/,
    fields: [
      { key: 'Lead', label: '优势要点' },
      { key: 'Text', label: '具体描述' },
    ],
    titleKey: 'Lead',
    itemLabel: '优势',
    itemUnit: '项',
    emptyTitle: '添加你的第一项优势',
    emptyHint: '用具体能力与成果补充上方简述，最多添加 6 项。',
  });
  const education = createRepeatable({
    ...repeatableOptions,
    group: 'education',
    slotPattern: /^edu([1-3])(Period|School|Major|Honor|Achievement)$/,
    fields: [
      { key: 'Period', label: '起止时间' },
      { key: 'School', label: '学校名称' },
      { key: 'Major', label: '专业 / 学位' },
      { key: 'Honor', label: '主要荣誉' },
      { key: 'Achievement', label: '学术成果', optional: true },
    ],
    titleKey: 'School',
    itemLabel: '教育经历',
    emptyTitle: '添加你的第一段教育经历',
    emptyHint: '填写就读时间、学校、专业或学位。',
  });
  const repeatables = { advantage: advantages, work, project: projects, education };

  function fieldValue(field) {
    const lines = content.fields[field.key] || [];
    if (lines.length > 1 && lines.every((line) => !line.trim() || /^【[^】]*】$/.test(line.trim())))
      return '';
    const value = (content.fields[field.key] || []).join('\n');
    // Template markers stay in the document, while untouched controls appear empty.
    return /^【[^】\n]*】$/.test(value.trim()) ? '' : value;
  }

  function renderNav() {
    const ul = element('navList');
    ul.innerHTML = '';
    navProgress.clear();
    groups.forEach((g) => {
      const li = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'nav-item' + (g.id === state.active ? ' active' : '');
      if (g.id === state.active) button.setAttribute('aria-current', 'true');
      const name = document.createElement('span');
      name.className = 'nav-name';
      name.textContent = g.title;
      const count = document.createElement('span');
      count.className = 'nav-count';
      button.append(createIcon(g.id), name, count);
      button.title = g.title;
      button.setAttribute('aria-label', g.title);
      button.addEventListener('click', () => selectGroup(g.id));
      navProgress.set(g.id, count);
      li.appendChild(button);
      ul.appendChild(li);
    });
    updateProgress();
  }

  function selectGroup(id) {
    state.active = id;
    state.search = '';
    element('searchInput').value = '';
    const g = groups.find((x) => x.id === id);
    if (!g) return;
    element('groupTitle').textContent = g.title;
    onSelect(g);
    renderNav();
    renderForm();
    element('form').scrollTop = 0;
  }

  function fieldNode(f, compactLabel = f.label) {
    const wrap = document.createElement('div');
    wrap.className = 'field';
    if (f.type !== 'text' || f.key === 'email' || f.key === 'photo') {
      wrap.classList.add('field-wide');
    }
    const label = document.createElement('label');
    label.setAttribute('for', 'f_' + f.key);
    label.textContent = compactLabel;

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
    input.setAttribute('aria-label', f.label);
    input.value = fieldValue(f);
    input.placeholder = f.placeholder || '';
    if (f.hint || f.type === 'lines') input.title = f.hint || '每行一条';
    input.addEventListener('input', onInput);

    if (f.type !== 'text') {
      const applyBold = () => {
        const result = toggleBold(input.value, input.selectionStart, input.selectionEnd);
        input.value = result.value;
        input.focus();
        input.setSelectionRange(result.start, result.end);
        input.dispatchEvent(new Event('input', { bubbles: true }));
      };
      input.addEventListener('keydown', (event) => {
        if ((event.ctrlKey || event.metaKey) && !event.altKey && event.key.toLowerCase() === 'b') {
          event.preventDefault();
          applyBold();
        }
      });
    }
    wrap.append(label, input);
    return wrap;
  }

  function renderForm() {
    const form = element('form');
    form.replaceChildren();
    form.dataset.group = state.active;
    const term = state.search.trim().toLowerCase();
    form.classList.toggle('search-results', !!term);
    element('groupTitle').textContent = term
      ? '搜索结果'
      : groups.find((group) => group.id === state.active)?.title || '内容';
    let items;
    if (!term && state.active === 'advantage') {
      const intro = document.createElement('div');
      intro.className = 'repeatable-intro';
      intro.append(fieldNode(schema.find((field) => field.key === 'brief')));
      form.append(intro);
    }
    if (!term && repeatables[state.active]) {
      repeatables[state.active].render(form);
      return;
    }
    if (term) {
      items = schema.filter(
        (f) =>
          f.key in content.fields &&
          (f.label.toLowerCase().includes(term) ||
            f.key.toLowerCase().includes(term) ||
            fieldValue(f).toLowerCase().includes(term)),
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
    let lastSection = '';
    let grid;
    for (const f of items) {
      if (term && f.group !== lastGroup) {
        lastGroup = f.group;
        const d = document.createElement('div');
        d.className = 'group-divider';
        d.textContent = f.groupTitle;
        form.appendChild(d);
        grid = null;
      }
      // A label prefix such as “经历 1 ·” groups related fields into one section.
      const separator = term ? -1 : f.label.indexOf(' · ');
      const section = separator < 0 ? '' : f.label.slice(0, separator);
      const label = separator < 0 ? f.label : f.label.slice(separator + 3);
      if (!grid || section !== lastSection) {
        grid = document.createElement('div');
        grid.className = 'field-grid';
        if (section) {
          const fieldset = document.createElement('fieldset');
          fieldset.className = 'form-section';
          const legend = document.createElement('legend');
          legend.textContent = section;
          fieldset.append(legend, grid);
          form.append(fieldset);
        } else {
          form.append(grid);
        }
        lastSection = section;
      }
      grid.appendChild(fieldNode(f, label));
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
    groups.forEach((group) => {
      const count = progress.groups[group.id] || 0;
      const node = navProgress.get(group.id);
      if (!node) return;
      const total = group.items.filter((field) => field.key in content.fields).length;
      node.textContent = count + '/' + total;
      node.setAttribute('aria-label', '已填写 ' + count + ' / ' + total + ' 项');
    });
  }

  element('searchInput').addEventListener('input', (event) => {
    state.search = event.target.value;
    renderForm();
    element('form').scrollTop = 0;
  });
  updateProgress();
  function refresh() {
    renderNav();
    renderForm();
  }

  return { selectGroup, refresh };
}
