import assert from 'node:assert/strict';
import { test } from 'node:test';

import { orderStatusLabel, orderStatusOptions } from './enums/orderStatus.ts';

test('order status label and options are generated from the same definitions', () => {
  assert.equal(orderStatusLabel('pending'), '待支付');
  assert.equal(orderStatusLabel('paid'), '已支付');
  assert.equal(orderStatusLabel('unknown'), 'unknown');

  assert.deepEqual(
    orderStatusOptions.map((option) => option.value),
    ['pending', 'paid', 'shipped', 'completed', 'cancelled']
  );
  assert.equal(
    orderStatusOptions.find((option) => option.value === 'completed')?.label,
    orderStatusLabel('completed')
  );
});
