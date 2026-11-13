import data from './data.js';
import { createBackend } from './backend.js';
import { createDesktopPlatform } from './platform.js';
import { copyContent } from './content.js';
import { createErrorTools } from './errors.js';
import { createLogger } from './logger.js';
import { createDocumentActions } from './actions.js';
import { createNotices } from './components/notices.js';
import { createTitlebar } from './components/titlebar.js';
import { createPanes } from './components/panes.js';
import { createEditor } from './components/editor.js';
import { createPreview } from './components/preview.js';

// Composition root: components receive their data and collaborators explicitly.
async function startApp(data) {
  const errors = createErrorTools(data.errors);
  let logger;
  const notices = createNotices(errors);
  window.addEventListener('error', (event) => {
    logger?.error('frontend_unhandled_error', event.error, errors.codes.APP_UNEXPECTED);
  });
  window.addEventListener('unhandledrejection', (event) => {
    logger?.error('frontend_unhandled_rejection', event.reason, errors.codes.APP_UNEXPECTED);
  });

  try {
    const backend = createBackend(errors);
    const platform = createDesktopPlatform(backend.platform);
    logger = createLogger(backend, errors);
    let content;
    try {
      const loaded = await backend.load();
      content = loaded?.fields ? loaded : { fields: {} };
    } catch (cause) {
      content = copyContent(data.initial);
      notices.failure(logger.error('load_failed', cause, errors.codes.CONTENT_LOAD_FAILED));
    }
    for (const field of data.schema) {
      if (!content.fields[field.key]?.length) {
        content.fields[field.key] = (data.initial.fields[field.key] || ['【待填写】']).slice();
      }
    }
    const groups = data.groups
      .map((group) => ({
        ...group,
        items: data.schema.filter((field) => field.group === group.id),
      }))
      .filter((group) => group.items.length);
    const titlebar = createTitlebar({
      groups,
      platform,
      onSelect: (id) => editor.selectGroup(id),
    });
    const preview = createPreview({ backend, notices, logger, errors });
    const actions = createDocumentActions({ backend, content, notices, logger, errors, platform });
    const editor = createEditor({
      schema: data.schema,
      groups,
      initial: data.initial,
      content,
      onSelect: (group) => titlebar.setActive(group.id),
      onChange(key, lines) {
        content.fields[key] = lines;
        notices.status('编辑中…', 'busy');
        actions.scheduleSave();
        preview.schedule(content);
      },
    });
    createPanes({
      onResize: preview.fit,
      logger,
      errors,
      platform,
      storageKey: data.metadata.name + '-panes',
    });
    editor.selectGroup(groups[0]?.id || '');
    await preview.refresh(content);
    notices.status('已同步');
    logger.info('editor_ready');
  } catch (cause) {
    notices.status('启动失败', 'error');
    const error = errors.normalize(cause);
    if (logger) logger.error('editor_start_failed', error);
    else
      console.error(
        JSON.stringify({ level: 'error', event: 'editor_start_failed', code: error.code }),
      );
    notices.failure(error);
  }
}

startApp(data);
