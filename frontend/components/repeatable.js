import { createIcon } from './icons.js';

/** Renders ordered, optional field groups as editable cards. */
export function createRepeatable({
  schema,
  content,
  fieldNode,
  fieldValue,
  onChange,
  onRefresh,
  group,
  slotPattern,
  fields,
  titleKey,
  itemLabel,
  itemUnit = '段',
  emptyTitle,
  emptyHint,
}) {
  const slotsByIndex = new Map();
  for (const field of schema) {
    if (field.group !== group) continue;
    const match = field.key.match(slotPattern);
    if (!match) continue;
    const index = Number(match[1]);
    const key = match[2];
    const slot = slotsByIndex.get(index) || {};
    slot[key] = field;
    slotsByIndex.set(index, slot);
  }
  const slots = [...slotsByIndex.entries()]
    .sort(([a], [b]) => a - b)
    .map(([index, slot]) => ({ index, fields: slot }));

  function active() {
    return slots.filter((slot) =>
      fields.every(({ key, optional }) => optional || slot.fields[key].key in content.fields),
    );
  }

  function snapshot() {
    return active().map((slot) =>
      Object.fromEntries(
        fields.map(({ key }) => [
          key,
          (content.fields[slot.fields[key].key] || ['【待填写】']).slice(),
        ]),
      ),
    );
  }

  function replace(items) {
    const next = { ...content.fields };
    for (const slot of slots) {
      for (const { key } of fields) delete next[slot.fields[key].key];
    }
    items.forEach((item, index) => {
      for (const { key } of fields) {
        next[slots[index].fields[key].key] = item[key];
      }
    });
    onChange(next);
    onRefresh();
  }

  function button(label, icon, className) {
    const node = document.createElement('button');
    node.type = 'button';
    node.className = className;
    if (icon) node.append(createIcon(icon));
    node.append(document.createTextNode(label));
    return node;
  }

  function render(form) {
    const items = active();
    const list = document.createElement('div');
    list.className = 'repeatable-list';
    const summary = document.createElement('p');
    summary.className = 'repeatable-summary';
    summary.textContent = `${items.length} ${itemUnit}${itemLabel} · 最多 ${slots.length} ${itemUnit}`;
    list.append(summary);

    items.forEach((slot, index) => {
      const card = document.createElement('article');
      card.className = 'repeatable-card';
      const titleId = `${group}-card-title-${index}`;
      card.setAttribute('aria-labelledby', titleId);
      const header = document.createElement('header');
      header.className = 'repeatable-card-header';
      const number = document.createElement('span');
      number.className = 'repeatable-number';
      number.textContent = String(index + 1).padStart(2, '0');
      const title = document.createElement('h3');
      title.id = titleId;
      title.textContent = fieldValue(slot.fields[titleKey]) || `未命名${itemLabel}`;
      const actions = document.createElement('div');
      actions.className = 'repeatable-card-actions';
      const remove = button('删除', 'trash', 'repeatable-remove');
      remove.setAttribute('aria-label', `删除${itemLabel} ${index + 1}`);

      function deleteItem() {
        const remaining = snapshot();
        remaining.splice(index, 1);
        replace(remaining);
        const next = Math.min(index, remaining.length - 1);
        if (next >= 0)
          document.getElementById('f_' + slots[next].fields[fields[0].key].key)?.focus();
        else form.querySelector('.repeatable-add')?.focus();
      }

      remove.addEventListener('click', () => {
        const hasValue = fields.some(({ key }) => fieldValue(slot.fields[key]));
        if (!hasValue) {
          deleteItem();
          return;
        }
        const prompt = document.createElement('span');
        prompt.textContent = `删除此${itemLabel}？`;
        const cancel = button('取消', null, 'repeatable-cancel');
        const confirm = button('删除', null, 'repeatable-remove');
        confirm.setAttribute('aria-label', `确认删除${itemLabel} ${index + 1}`);
        cancel.addEventListener('click', () => {
          actions.replaceChildren(remove);
          remove.focus();
        });
        confirm.addEventListener('click', deleteItem);
        actions.replaceChildren(prompt, cancel, confirm);
        cancel.focus();
      });
      actions.append(remove);
      header.append(number, title, actions);

      const fieldGrid = document.createElement('div');
      fieldGrid.className = 'field-grid';
      for (const { key, label } of fields) {
        const node = fieldNode(slot.fields[key], label);
        if (key === titleKey) {
          node.querySelector('input').addEventListener('input', (event) => {
            title.textContent = event.target.value.trim() || `未命名${itemLabel}`;
          });
        }
        fieldGrid.append(node);
      }
      card.append(header, fieldGrid);
      list.append(card);
    });

    if (!items.length) {
      const empty = document.createElement('div');
      empty.className = 'repeatable-empty';
      const heading = document.createElement('h3');
      heading.textContent = emptyTitle;
      const hint = document.createElement('p');
      hint.textContent = emptyHint;
      empty.append(createIcon(group), heading, hint);
      list.append(empty);
    }

    const add = button(`新增${itemLabel}`, 'plus', 'repeatable-add');
    add.disabled = items.length >= slots.length;
    if (add.disabled) add.title = `最多添加 ${slots.length} ${itemUnit}${itemLabel}`;
    add.addEventListener('click', () => {
      const next = snapshot();
      next.push(Object.fromEntries(fields.map(({ key }) => [key, ['【待填写】']])));
      replace(next);
      const input = document.getElementById(
        'f_' + slots[next.length - 1].fields[fields[0].key].key,
      );
      input?.focus({ preventScroll: true });
      input?.closest('.repeatable-card').scrollIntoView({ block: 'nearest' });
    });
    list.append(add);
    form.append(list);
  }

  return { render };
}
