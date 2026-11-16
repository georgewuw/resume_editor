import assert from 'node:assert/strict';
import test from 'node:test';
import { toggleBold } from '../../frontend/components/bold.js';

test('bold toggles a selection and restores it', () => {
  const applied = toggleBold('增长30%', 2, 5);
  assert.deepEqual(applied, { value: '增长**30%**', start: 2, end: 9 });
  assert.deepEqual(toggleBold(applied.value, applied.start, applied.end), {
    value: '增长30%',
    start: 2,
    end: 5,
  });
  assert.deepEqual(toggleBold('增长**30%**', 4, 7), { value: '增长30%', start: 2, end: 5 });
});

test('bold handles insertion and multiline selection', () => {
  assert.deepEqual(toggleBold('增长', 2, 2), { value: '增长****', start: 4, end: 4 });
  const applied = toggleBold('第一行\n第二行', 0, 7);
  assert.equal(applied.value, '**第一行**\n**第二行**');
  assert.equal(toggleBold(applied.value, applied.start, applied.end).value, '第一行\n第二行');
  assert.equal(toggleBold('**第一行**\n第二行', 0, 11).value, '**第一行**\n**第二行**');
});
