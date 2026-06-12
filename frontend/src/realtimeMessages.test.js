import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  dispatchRealtimeMessage,
  normalizeRealtimeMessage,
  realtimePresentations,
  toastFromRealtimeMessage
} from './helpers/realtimeMessages.js';

test('normalizes realtime message default presentations', () => {
  assert.deepEqual(normalizeRealtimeMessage({
    type: 'points',
    payload: { balance: 10 }
  }), {
    type: 'points',
    presentation: realtimePresentations.refresh,
    payload: { balance: 10 }
  });

  assert.equal(
    normalizeRealtimeMessage({ type: 'async_export_task' }).presentation,
    realtimePresentations.toast
  );

  assert.equal(
    normalizeRealtimeMessage({ type: 'notification' }).presentation,
    realtimePresentations.toast
  );

  assert.equal(
    normalizeRealtimeMessage({ type: 'points', presentation: 'toast' }).presentation,
    realtimePresentations.toast
  );
});

test('dispatches points refresh messages', () => {
  let refreshedPayload;
  const handled = dispatchRealtimeMessage({
    type: 'points',
    payload: { user_id: 'u1', balance: 20 }
  }, {
    refreshPoints(payload) {
      refreshedPayload = payload;
    }
  });

  assert.equal(handled, true);
  assert.deepEqual(refreshedPayload, { user_id: 'u1', balance: 20 });
});

test('dispatches async export task messages as toast by default', () => {
  let toast;
  const handled = dispatchRealtimeMessage({
    type: 'async_export_task',
    payload: {
      task_id: 'export-1',
      status: 'completed',
      message: 'Export completed'
    }
  }, {
    toast(nextToast) {
      toast = nextToast;
    }
  });

  assert.equal(handled, true);
  assert.deepEqual(toast, {
    id: 'export-1',
    level: 'success',
    message: 'Export completed'
  });
});

test('dispatches notification messages as toast by default', () => {
  let toast;
  const handled = dispatchRealtimeMessage({
    type: 'notification',
    payload: {
      id: 'notification-1',
      title: 'Order paid',
      summary: 'Your points have been awarded',
      source_type: 'order',
      source_id: 'order-1'
    }
  }, {
    toast(nextToast) {
      toast = nextToast;
    }
  });

  assert.equal(handled, true);
  assert.deepEqual(toast, {
    id: 'notification-1',
    level: 'info',
    message: 'Your points have been awarded'
  });
});

test('maps notification status to toast level', () => {
  assert.deepEqual(toastFromRealtimeMessage({
    type: 'notification',
    payload: {
      id: 'notification-2',
      title: 'Order export failed',
      summary: 'Order export failed',
      status: 'failed'
    }
  }), {
    id: 'notification-2',
    level: 'error',
    message: 'Order export failed'
  });
});

test('ignores malformed or unsupported realtime messages', () => {
  assert.equal(dispatchRealtimeMessage(null, {}), false);
  assert.equal(dispatchRealtimeMessage({ payload: {} }, {}), false);
  assert.equal(dispatchRealtimeMessage({ type: 'unknown' }, {}), false);
});

test('builds fallback toast text for async export task messages', () => {
  assert.deepEqual(toastFromRealtimeMessage({
    type: 'async_export_task',
    payload: { task_id: 'export-2', status: 'failed' }
  }), {
    id: 'export-2',
    level: 'error',
    message: 'Export task updated'
  });
});
