import assert from 'node:assert/strict';
import { test } from 'node:test';

import { createRealtimeWebSocketClient } from './helpers/realtimeWebSocket.js';

test('realtime websocket client opens and dispatches parsed JSON messages', () => {
  const sockets = [];
  const messages = [];
  const statuses = [];

  const client = createRealtimeWebSocketClient({
    url: () => 'ws://example.test/api/user/realtime/ws',
    WebSocketCtor: fakeWebSocketCtor(sockets),
    onStatusChange: (status) => statuses.push(status),
    onMessage: (message) => messages.push(message)
  });

  client.connect();
  assert.equal(sockets[0].url, 'ws://example.test/api/user/realtime/ws');
  sockets[0].open();
  sockets[0].message({ type: 'points', payload: { balance: 10 } });

  assert.deepEqual(statuses, ['Connecting', 'Connected']);
  assert.deepEqual(messages, [{ type: 'points', payload: { balance: 10 } }]);
});

test('realtime websocket client reports malformed JSON without closing the socket', () => {
  const sockets = [];
  const malformed = [];

  const client = createRealtimeWebSocketClient({
    url: 'ws://example.test/api/user/realtime/ws',
    WebSocketCtor: fakeWebSocketCtor(sockets),
    onMalformedMessage: (event) => malformed.push(event.data)
  });

  client.connect();
  sockets[0].open();
  sockets[0].message('{bad json');

  assert.deepEqual(malformed, ['{bad json']);
  assert.equal(sockets[0].closed, false);
});

test('realtime websocket client closes errored sockets and schedules reconnects', () => {
  const sockets = [];
  const timers = [];
  const statuses = [];
  let reconnect = true;

  const client = createRealtimeWebSocketClient({
    url: 'ws://example.test/api/user/realtime/ws',
    WebSocketCtor: fakeWebSocketCtor(sockets),
    shouldReconnect: () => reconnect,
    onStatusChange: (status) => statuses.push(status),
    setTimeoutFn(fn, delay) {
      timers.push({ fn, delay });
      return timers.length;
    }
  });

  client.connect();
  sockets[0].open();
  sockets[0].error();

  assert.equal(sockets[0].closed, true);
  assert.deepEqual(statuses, ['Connecting', 'Connected', 'Error']);
  assert.equal(timers.length, 1);
  assert.equal(timers[0].delay, 5000);

  timers[0].fn();
  assert.equal(sockets.length, 2);
  assert.equal(client.getStatus(), 'Connecting');

  reconnect = false;
  sockets[1].closeFromServer();
  assert.equal(timers.length, 1);
});

test('realtime websocket client disconnects without firing reconnect', () => {
  const sockets = [];
  const clearedTimers = [];
  const timers = [];
  const statuses = [];

  const client = createRealtimeWebSocketClient({
    url: 'ws://example.test/api/user/realtime/ws',
    WebSocketCtor: fakeWebSocketCtor(sockets),
    shouldReconnect: () => true,
    onStatusChange: (status) => statuses.push(status),
    setTimeoutFn(fn) {
      timers.push(fn);
      return 'timer-1';
    },
    clearTimeoutFn(timer) {
      clearedTimers.push(timer);
    }
  });

  client.connect();
  sockets[0].closeFromServer();
  assert.equal(timers.length, 1);

  client.disconnect();
  assert.deepEqual(clearedTimers, ['timer-1']);
  assert.equal(client.getSocket(), undefined);
  assert.equal(client.getStatus(), 'Disconnected');
  assert.equal(statuses.at(-1), 'Disconnected');
});

function fakeWebSocketCtor(instances) {
  return class FakeWebSocket {
    constructor(url) {
      this.url = url;
      this.closed = false;
      instances.push(this);
    }

    open() {
      this.onopen?.({});
    }

    message(data) {
      this.onmessage?.({ data: typeof data === 'string' ? data : JSON.stringify(data) });
    }

    error() {
      this.onerror?.({ type: 'error' });
    }

    close() {
      this.closed = true;
      this.onclose?.({ code: 1000 });
    }

    closeFromServer() {
      this.closed = true;
      this.onclose?.({ code: 1000 });
    }
  };
}
