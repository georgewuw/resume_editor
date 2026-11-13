import { element } from '../dom.js';

// Native window controls are managed by mygo; this component owns navigation.
export function createTitlebar({ groups, onSelect, platform }) {
  const button = element('btnModules');
  const menu = element('moduleMenu');
  element('btnSidebar').title = '收起 / 展开侧栏（' + platform.shortcutLabel('b') + '）';
  let active = '';

  function closeMenu() {
    menu.hidden = true;
    button.setAttribute('aria-expanded', 'false');
  }

  function renderMenu() {
    menu.replaceChildren();
    groups.forEach((group, index) => {
      const item = document.createElement('button');
      item.type = 'button';
      item.setAttribute('role', 'menuitem');
      item.className = group.id === active ? 'active' : '';
      const number = document.createElement('span');
      number.className = 'menu-num';
      number.textContent = String(index + 1).padStart(2, '0');
      const name = document.createElement('span');
      name.textContent = group.title;
      item.append(number, name);
      item.addEventListener('click', () => {
        closeMenu();
        onSelect(group.id);
      });
      menu.appendChild(item);
    });
  }

  function setActive(id) {
    active = id;
    element('crumb').textContent = groups.find((group) => group.id === id)?.title || '内容模块';
    if (!menu.hidden) renderMenu();
  }

  button.addEventListener('click', (event) => {
    event.stopPropagation();
    if (menu.hidden) {
      renderMenu();
      menu.hidden = false;
      button.setAttribute('aria-expanded', 'true');
    } else {
      closeMenu();
    }
  });
  document.addEventListener('click', (event) => {
    if (!menu.hidden && !menu.contains(event.target)) closeMenu();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeMenu();
  });

  return { setActive };
}
