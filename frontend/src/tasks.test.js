import assert from 'node:assert/strict';
import { test } from 'node:test';

import { canDownloadTask, taskFilename, taskTitle, taskTypes } from './helpers/tasks.js';

test('completed order export tasks are downloadable from task center', () => {
  assert.equal(canDownloadTask({
    task_type: taskTypes.ordersExcelExport,
    status: 'completed',
    result_json: '{}'
  }), true);
});

test('task download is limited to completed order export tasks', () => {
  assert.equal(canDownloadTask({
    task_type: taskTypes.ordersExcelExport,
    status: 'processing'
  }), false);
  assert.equal(canDownloadTask({
    task_type: 'test_export',
    status: 'completed'
  }), false);
});

test('task helpers provide stable labels and filename fallback', () => {
  assert.equal(taskTitle({ task_type: taskTypes.ordersExcelExport }), 'Orders Excel export');
  assert.equal(taskTitle({ task_type: 'test_export' }), 'test_export');
  assert.equal(taskFilename({
    result_json: '{"filename":"orders-20260612.xlsx"}'
  }), 'orders-20260612.xlsx');
  assert.equal(taskFilename({
    result_json: '{invalid'
  }), 'orders.xlsx');
});
