import { element } from '../dom.js';

export function createNotices(errors) {
  let timer = 0;

  function toast(message, isError = false) {
    const node = element('toast');
    node.textContent = message;
    node.className = 'toast show' + (isError ? ' error' : '');
    clearTimeout(timer);
    timer = setTimeout(() => {
      node.className = 'toast';
    }, 3200);
  }

  function status(message, kind = '') {
    element('saveText').textContent = message;
    element('saveState').className = 'save-state' + (kind ? ' ' + kind : '');
    element('saveState').title = message;
  }

  return { toast, status, failure: (error) => toast(errors.describe(error), true) };
}
