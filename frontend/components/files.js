import { element } from '../dom.js';

function documentTitle(info) {
  return info.role || '通用简历';
}

function modifiedLabel(info) {
  if (!info.modifiedAt) return '尚未保存';
  return (
    '更新于 ' +
    new Intl.DateTimeFormat('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).format(new Date(info.modifiedAt))
  );
}

/** File management owns its list and creation form, but never mutates editor content. */
export function createFiles({ onOpen, onCreate, onJDChange, onEdit }) {
  const dialog = element('newDocumentDialog');
  const form = element('newDocumentForm');
  let active;

  function setDocuments(documents, info) {
    active = info;
    element('fileCount').textContent = String(documents.length);
    const list = element('fileList');
    list.replaceChildren();
    documents.forEach((entry) => {
      const item = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'file-item' + (entry.filename === info.filename ? ' active' : '');
      if (entry.filename === info.filename) button.setAttribute('aria-current', 'true');
      const icon = document.createElement('span');
      icon.className = 'file-icon';
      icon.textContent = 'MD';
      icon.setAttribute('aria-hidden', 'true');
      const copy = document.createElement('span');
      copy.className = 'file-item-copy';
      const title = document.createElement('span');
      title.className = 'file-item-title';
      title.textContent = documentTitle(entry);
      const company = document.createElement('span');
      company.className = 'file-item-company';
      company.textContent = entry.company || (entry.role ? '未指定公司' : '适用于所有岗位');
      copy.append(title, company);
      button.append(icon, copy);
      button.title = entry.filename;
      button.setAttribute(
        'aria-label',
        entry.filename + (entry.filename === info.filename ? '，当前文件' : ''),
      );
      button.addEventListener('click', () => onOpen(entry.filename));
      item.append(button);
      list.append(item);
    });
    element('sideDocumentName').textContent = documentTitle(info);
    element('sideDocumentName').title = info.filename;
    element('documentTitle').textContent = documentTitle(info);
    element('documentCompany').textContent =
      info.company || (info.role ? '未指定公司' : '所有岗位');
    element('documentFilename').textContent = info.filename;
    element('documentModified').textContent = modifiedLabel(info);
  }

  function setDocument(info) {
    active = info;
    element('documentJD').value = info.jd || '';
  }

  function openCreate() {
    form.reset();
    element('newDocumentError').textContent = '';
    element('sourceCurrentLabel').textContent = documentTitle(active);
    dialog.showModal();
  }

  element('btnNewDocument').addEventListener('click', openCreate);
  element('btnNewDocumentMain').addEventListener('click', openCreate);
  element('btnCancelDocument').addEventListener('click', () => dialog.close());
  element('btnCloseDocument').addEventListener('click', () => dialog.close());
  element('btnEditDocument').addEventListener('click', onEdit);
  element('documentJD').addEventListener('input', (event) => onJDChange(event.target.value));

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;
    const role = element('newRole').value.trim();
    const source = new FormData(form).get('source') || 'current';
    const submit = element('btnCreateDocument');
    submit.disabled = true;
    element('newDocumentFields').disabled = true;
    element('btnCancelDocument').disabled = true;
    element('btnCloseDocument').disabled = true;
    dialog.oncancel = (event) => event.preventDefault();
    try {
      const created = await onCreate({
        company: element('newCompany').value.trim(),
        role,
        jd: element('newJD').value.trim(),
        source,
      });
      if (created) dialog.close();
      else element('newDocumentError').textContent = '创建失败，当前内容已保留，请重试。';
    } finally {
      submit.disabled = false;
      element('newDocumentFields').disabled = false;
      element('btnCancelDocument').disabled = false;
      element('btnCloseDocument').disabled = false;
      dialog.oncancel = null;
    }
  });

  return { setDocuments, setDocument };
}
