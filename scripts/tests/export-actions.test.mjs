import assert from 'node:assert/strict';
import test from 'node:test';
import { createDocumentActions } from '../../frontend/actions.js';
import { createBackend } from '../../frontend/backend.js';

test('export keeps the document and content captured before the save completes', async () => {
  const elements = { btnPdf: {}, workspace: {} };
  elements.btnPdf.addEventListener = (_name, handler) => {
    elements.btnPdf.click = handler;
  };
  globalThis.document = { getElementById: (id) => elements[id], addEventListener() {} };
  let releaseSave;
  const saved = new Promise((resolve) => {
    releaseSave = resolve;
  });
  let current = { filename: '简历_甲_工程师.md', jd: 'JD 甲' };
  const content = { fields: { name: ['甲内容'] } };
  const calls = [];
  const backend = {
    async saveDocument(filename, data, jd) {
      calls.push(['save', filename, data.fields.name[0], jd]);
      await saved;
      return { filename, jd };
    },
    async export(filename, data) {
      calls.push(['export', filename, data.fields.name[0]]);
      return 'result.pdf';
    },
  };
  createDocumentActions({
    backend,
    content,
    getDocument: () => current,
    onSaved: async () => {},
    notices: {
      status() {},
      toast() {},
      failure(error) {
        throw error;
      },
    },
    logger: {
      info() {},
      error(_event, error) {
        return error;
      },
    },
    errors: { codes: { EXPORT_FAILED: 'EXPORT_FAILED' } },
    platform: { matchesShortcut: () => false },
  });
  const exporting = elements.btnPdf.click();
  await Promise.resolve();
  current = { filename: '简历_乙_设计师.md', jd: 'JD 乙' };
  content.fields.name = ['乙内容'];
  releaseSave();
  await exporting;
  assert.deepEqual(calls, [
    ['save', '简历_甲_工程师.md', '甲内容', 'JD 甲'],
    ['export', '简历_甲_工程师.md', '甲内容'],
  ]);
  assert.equal(elements.btnPdf.disabled, false);
});

test('desktop bridge forwards the export document identity', async () => {
  let call;
  globalThis.window = {
    mygo: {
      call: async (...args) => {
        call = args;
      },
    },
  };
  const backend = createBackend({ codes: { EXPORT_FAILED: 'EXPORT_FAILED' } });
  const content = { fields: { name: ['甲内容'] } };
  await backend.export('简历_甲_工程师.md', content);
  assert.deepEqual(call, ['Resume.Export', '简历_甲_工程师.md', content]);
});
