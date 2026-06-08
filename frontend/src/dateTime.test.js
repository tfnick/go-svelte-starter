import assert from 'node:assert/strict';
import { test } from 'node:test';

import { formatLocalDateTime, parseUTCDateTime } from './helpers/dateTime.js';

test('parseUTCDateTime treats SQLite timestamps as UTC', () => {
  const date = parseUTCDateTime('2026-06-07 12:30:00');

  assert.equal(date.toISOString(), '2026-06-07T12:30:00.000Z');
});

test('parseUTCDateTime preserves explicit timezone offsets', () => {
  const date = parseUTCDateTime('2026-06-07T20:30:00+08:00');

  assert.equal(date.toISOString(), '2026-06-07T12:30:00.000Z');
});

test('formatLocalDateTime falls back for empty or invalid values', () => {
  assert.equal(formatLocalDateTime(''), '--');
  assert.equal(formatLocalDateTime('not-a-date'), 'not-a-date');
});
