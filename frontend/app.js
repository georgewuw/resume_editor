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
import { createFiles } from './components/files.js';
import { element } from './dom.js';

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
    const workspace = await backend.workspace();
    let documents = workspace.documents;
    const loaded = workspace.current;
    const content = copyContent(loaded.content);
    let currentDocument = loaded.document;
    const groups = data.groups
      .map((group) => ({
        ...group,
        items: data.schema.filter((field) => field.group === group.id),
      }))
      .filter((group) => group.items.length);
    const preview = createPreview({ backend, notices, logger, errors });
    const titlebar = createTitlebar({ platform, onModeChange: () => preview.fit() });
    const actions = createDocumentActions({
      backend,
      content,
      notices,
      logger,
      errors,
      platform,
      getDocument: () => currentDocument,
      async onSaved(info, fresh) {
        if (info.filename === currentDocument.filename) {
          currentDocument = { ...info, jd: fresh ? info.jd : currentDocument.jd };
        }
        documents = await backend.listDocuments();
        files.setDocuments(documents, currentDocument);
      },
    });
    const editor = createEditor({
      schema: data.schema,
      groups,
      content,
      getProgress: (snapshot) => backend.progress(snapshot),
      onProgressError: (cause) =>
        logger.warn('progress_failed', cause, errors.codes.DOCUMENT_INVALID),
      onSelect: () => {},
      onChange(key, lines) {
        content.fields[key] = lines;
        notices.status('编辑中…', 'busy');
        actions.scheduleSave();
        preview.schedule(content);
      },
    });
    let changingDocument = false;

    async function documentTask(operation) {
      if (changingDocument) return false;
      changingDocument = true;
      element('workspace').inert = true;
      element('workspace').setAttribute('aria-busy', 'true');
      try {
        await actions.flush();
        await operation();
        return true;
      } catch (cause) {
        notices.status('操作失败', 'error');
        notices.failure(
          logger.error('document_operation_failed', cause, errors.codes.CONTENT_SAVE_FAILED),
        );
        return false;
      } finally {
        changingDocument = false;
        element('workspace').inert = false;
        element('workspace').removeAttribute('aria-busy');
      }
    }

    async function applyDocument(document) {
      currentDocument = document.document;
      content.fields = copyContent(document.content).fields;
      if (!documents.some((item) => item.filename === currentDocument.filename)) {
        documents.push(currentDocument);
      }
      titlebar.setDocument(currentDocument);
      files.setDocuments(documents, currentDocument);
      files.setDocument(currentDocument);
      editor.refresh();
      element('editor').scrollTop = 0;
      await preview.refresh(content);
      notices.status('已同步');
    }

    const files = createFiles({
      onOpen(filename) {
        if (filename === currentDocument.filename) return;
        return documentTask(async () => applyDocument(await backend.openDocument(filename)));
      },
      onCreate(input) {
        return documentTask(async () => {
          if (input.source !== 'template') await actions.save();
          const created = await backend.createDocument({
            company: input.company,
            role: input.role,
            jd: input.jd,
            source: input.source === 'template' ? '' : currentDocument.filename,
          });
          documents = await backend.listDocuments();
          await applyDocument(created);
          notices.toast('已创建 ' + created.document.filename);
        });
      },
      onJDChange(jd) {
        currentDocument.jd = jd;
        notices.status('编辑中…', 'busy');
        actions.scheduleSave();
      },
      onEdit: () => titlebar.setMode('content'),
    });
    createPanes({
      onResize: preview.fit,
      logger,
      errors,
      platform,
      storageKey: data.metadata.name + '-panes',
    });
    titlebar.setMode('content');
    editor.selectGroup(groups[0]?.id || '');
    await applyDocument(loaded);
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
