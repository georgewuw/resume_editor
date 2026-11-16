import { element } from '../dom.js';

const jianguoyunServerURL = 'https://dav.jianguoyun.com/dav/';

function providerForURL(value) {
  try {
    return new URL(value).href === jianguoyunServerURL ? 'jianguoyun' : 'custom';
  } catch {
    return 'custom';
  }
}

export function createWebDAVSettings({ backend, errors, logger, beforeSync, afterSync }) {
  const dialog = element('webDAVDialog');
  const form = element('webDAVForm');
  const fields = element('webDAVFields');
  const status = element('webDAVStatus');
  const conflictList = element('webDAVConflicts');
  const provider = element('webDAVProvider');
  const providerOptions = element('webDAVProviderOptions');
  const options = Array.from(providerOptions.querySelectorAll('[role="option"]'));
  const serverURL = element('webDAVServerURL');
  const buttons = [
    element('btnCloseWebDAV'),
    element('btnTestWebDAV'),
    element('btnSaveWebDAV'),
    element('btnSyncWebDAV'),
    element('btnForgetWebDAV'),
  ];
  let busy = false;
  let openRevision = 0;
  let savedSettings;
  let customServerURL = '';
  let activeProviderIndex = 0;

  function setProvider(value) {
    provider.value = value;
    element('webDAVProviderValue').textContent =
      value === 'jianguoyun' ? '坚果云' : '自定义 WebDAV';
    options.forEach((option) =>
      option.setAttribute('aria-selected', String(option.value === value)),
    );
  }

  function setActiveProvider(index) {
    activeProviderIndex = index;
    options.forEach((option, optionIndex) => {
      option.dataset.active = String(optionIndex === index);
    });
    provider.setAttribute('aria-activedescendant', options[index].id);
  }

  function setProviderExpanded(expanded) {
    provider.setAttribute('aria-expanded', String(expanded));
    providerOptions.hidden = !expanded;
    if (expanded) {
      setActiveProvider(options.findIndex((option) => option.value === provider.value));
    } else {
      provider.removeAttribute('aria-activedescendant');
    }
  }

  function chooseProvider(value) {
    const changed = value !== provider.value;
    if (changed) {
      if (value === 'jianguoyun') {
        customServerURL = serverURL.value;
        serverURL.value = jianguoyunServerURL;
      } else {
        serverURL.value = customServerURL;
      }
    }
    setProvider(value);
    setProviderExpanded(false);
    if (changed && value === 'custom') serverURL.focus();
    else provider.focus();
    setPasswordHint();
  }

  function input() {
    return {
      serverURL: element('webDAVServerURL').value.trim(),
      username: element('webDAVUsername').value.trim(),
      password: element('webDAVPassword').value,
      remotePath: element('webDAVPath').value.trim(),
    };
  }

  function setBusy(value) {
    busy = value;
    if (value) setProviderExpanded(false);
    fields.disabled = value;
    buttons.forEach((button) => (button.disabled = value));
    dialog.setAttribute('aria-busy', String(value));
    conflictList.querySelectorAll('button').forEach((button) => (button.disabled = value));
  }

  function formatDate(value) {
    if (!value) return '尚未同步';
    const date = new Date(value);
    return Number.isNaN(date.valueOf())
      ? '上次同步时间不可用'
      : '上次同步：' +
          new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(
            date,
          );
  }

  function setPasswordHint() {
    const canReuse =
      savedSettings?.hasPassword &&
      savedSettings.serverURL === serverURL.value.trim() &&
      savedSettings.username === element('webDAVUsername').value.trim();
    element('webDAVPassword').placeholder = canReuse ? '留空保留已保存的密码' : '输入应用密码';
    element('webDAVPasswordHint').textContent = canReuse
      ? '密码已保存，留空即可继续使用'
      : provider.value === 'jianguoyun'
        ? '请使用坚果云的第三方应用密码'
        : '使用服务商提供的应用密码或授权密码';
    element('btnForgetWebDAV').hidden = !savedSettings?.hasPassword;
  }

  function setStatus(message, kind = '') {
    status.textContent = message;
    status.dataset.kind = kind;
  }

  function setSettings(settings) {
    savedSettings = settings;
    serverURL.value = settings.serverURL;
    setProvider(providerForURL(settings.serverURL));
    customServerURL = provider.value === 'custom' ? settings.serverURL : '';
    element('webDAVUsername').value = settings.username;
    element('webDAVPath').value = settings.remotePath;
    element('webDAVPassword').value = '';
    setPasswordHint();
    setStatus(formatDate(settings.lastSyncedAt));
  }

  function renderResult(result, prefix = '同步完成') {
    const message = `${prefix}：上传 ${result.uploaded}，下载 ${result.downloaded}，删除 ${result.deleted}`;
    setStatus(
      result.conflicts.length
        ? `${message}；${result.conflicts.length} 个文件需要选择版本`
        : message,
      result.conflicts.length ? 'warning' : 'success',
    );
    conflictList.replaceChildren();
    conflictList.hidden = result.conflicts.length === 0;
    for (const filename of result.conflicts) {
      const item = document.createElement('li');
      const name = document.createElement('span');
      name.className = 'webdav-conflict-name';
      name.textContent = filename;
      const local = document.createElement('button');
      local.type = 'button';
      local.className = 'btn';
      local.textContent = '保留本机';
      local.addEventListener('click', () => resolve(filename, 'local'));
      const remote = document.createElement('button');
      remote.type = 'button';
      remote.className = 'btn';
      remote.textContent = '使用云端';
      remote.addEventListener('click', () => resolve(filename, 'remote'));
      item.append(name, local, remote);
      conflictList.append(item);
    }
  }

  async function run(task, eventName) {
    if (busy) return;
    const pendingActions = {
      webdav_connection_failed: ['btnTestWebDAV', '连接中…'],
      webdav_settings_save_failed: ['btnSaveWebDAV', '保存中…'],
      webdav_sync_failed: ['btnSyncWebDAV', '同步中…'],
    };
    const action = pendingActions[eventName];
    if (action && !form.reportValidity()) return;
    const button = action && element(action[0]);
    const previousLabel = button?.textContent;
    if (button) button.textContent = action[1];
    setBusy(true);
    try {
      await task();
    } catch (cause) {
      const normalized = logger.error(eventName, cause);
      setStatus(errors.describe(normalized), 'error');
    } finally {
      if (button) button.textContent = previousLabel;
      setBusy(false);
    }
  }

  async function save() {
    setStatus('正在保存设置…', 'busy');
    const settings = await backend.saveWebDAVSettings(input());
    savedSettings = settings;
    element('webDAVPassword').value = '';
    setPasswordHint();
    setStatus(
      settings.hasPassword
        ? '设置已保存 · ' + formatDate(settings.lastSyncedAt)
        : '设置已保存，请填写应用密码',
    );
  }

  async function testConnection() {
    setStatus('正在测试连接…', 'busy');
    const result = await backend.testWebDAVConnection(input());
    setStatus(result, 'success');
  }

  async function sync() {
    setStatus('正在准备同步…', 'busy');
    await beforeSync();
    const settings = await backend.saveWebDAVSettings(input());
    savedSettings = settings;
    element('webDAVPassword').value = '';
    setPasswordHint();
    setStatus('正在同步…', 'busy');
    const result = await backend.syncWebDAV();
    renderResult(result);
    await afterSync();
  }

  async function resolve(filename, choice) {
    await run(async () => {
      setStatus('正在处理 ' + filename + '…', 'busy');
      await backend.resolveWebDAVConflict(filename, choice);
      const result = await backend.syncWebDAV();
      renderResult(result, '冲突已处理');
      await afterSync();
    }, 'webdav_conflict_resolve_failed');
  }

  async function open() {
    const revision = ++openRevision;
    conflictList.replaceChildren();
    conflictList.hidden = true;
    setStatus('正在读取设置…', 'busy');
    setBusy(true);
    dialog.showModal();
    try {
      const settings = await backend.getWebDAVSettings();
      if (revision !== openRevision || !dialog.open) return;
      setSettings(settings);
    } catch (cause) {
      if (revision === openRevision && dialog.open) {
        setStatus(errors.describe(logger.error('webdav_settings_load_failed', cause)), 'error');
      }
    } finally {
      if (revision === openRevision) setBusy(false);
    }
  }

  provider.addEventListener('click', () => {
    setProviderExpanded(providerOptions.hidden);
  });
  provider.addEventListener('keydown', (event) => {
    if (event.key === 'Tab') {
      setProviderExpanded(false);
      return;
    }
    if (event.key === 'Escape') {
      if (!providerOptions.hidden) {
        event.preventDefault();
        event.stopPropagation();
        setProviderExpanded(false);
      }
      return;
    }
    if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      if (providerOptions.hidden) setProviderExpanded(true);
      else if (event.key === 'ArrowDown') {
        setActiveProvider((activeProviderIndex + 1) % options.length);
      } else if (event.key === 'ArrowUp') {
        setActiveProvider((activeProviderIndex - 1 + options.length) % options.length);
      }
      if (event.key === 'Home') setActiveProvider(0);
      if (event.key === 'End') setActiveProvider(options.length - 1);
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      if (providerOptions.hidden) setProviderExpanded(true);
      else chooseProvider(options[activeProviderIndex].value);
    }
  });
  options.forEach((option, index) => {
    option.addEventListener('click', () => chooseProvider(option.value));
    option.addEventListener('pointermove', () => setActiveProvider(index));
  });
  dialog.addEventListener('pointerdown', (event) => {
    if (!element('webDAVProviderControl').contains(event.target)) setProviderExpanded(false);
  });
  serverURL.addEventListener('input', () => {
    setProvider(providerForURL(serverURL.value));
    if (provider.value === 'custom') customServerURL = serverURL.value;
    setPasswordHint();
  });
  element('webDAVUsername').addEventListener('input', setPasswordHint);
  element('side-settings').addEventListener('click', open);
  element('btnCloseWebDAV').addEventListener('click', () => dialog.close());
  element('btnTestWebDAV').addEventListener('click', () =>
    run(testConnection, 'webdav_connection_failed'),
  );
  element('btnSaveWebDAV').addEventListener('click', () =>
    run(save, 'webdav_settings_save_failed'),
  );
  element('btnSyncWebDAV').addEventListener('click', () => run(sync, 'webdav_sync_failed'));
  element('btnForgetWebDAV').addEventListener('click', () =>
    run(async () => {
      const settings = await backend.saveWebDAVSettings({
        ...input(),
        password: '',
        clearPassword: true,
      });
      savedSettings = settings;
      element('webDAVPassword').value = '';
      setPasswordHint();
      setStatus('已清除已保存的密码');
    }, 'webdav_password_clear_failed'),
  );
  form.addEventListener('submit', (event) => event.preventDefault());
  dialog.addEventListener('cancel', (event) => {
    if (busy || !providerOptions.hidden) {
      event.preventDefault();
      setProviderExpanded(false);
    }
  });
  dialog.addEventListener('close', () => setProviderExpanded(false));

  return { open };
}
