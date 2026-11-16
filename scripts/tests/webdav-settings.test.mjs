import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { createWebDAVSettings } from '../../frontend/components/webdav-settings.js';

test('WebDAV settings mounts against the sidebar controls in the page', () => {
  const html = readFileSync(new URL('../../frontend/index.html', import.meta.url), 'utf8');
  const ids = new Set(Array.from(html.matchAll(/\bid="([^"]+)"/g), (match) => match[1]));
  const elements = new Map(
    Array.from(ids, (id) => [
      id,
      {
        addEventListener() {},
        querySelectorAll() {
          return [];
        },
      },
    ]),
  );
  const previousDocument = globalThis.document;
  globalThis.document = { getElementById: (id) => elements.get(id) ?? null };

  try {
    assert.doesNotThrow(() =>
      createWebDAVSettings({
        backend: {},
        errors: {},
        logger: {},
        beforeSync() {},
        afterSync() {},
      }),
    );
  } finally {
    if (previousDocument === undefined) delete globalThis.document;
    else globalThis.document = previousDocument;
  }
});
