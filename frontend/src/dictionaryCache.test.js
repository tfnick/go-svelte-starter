import assert from 'node:assert/strict';
import { test } from 'node:test';

import { createDictionaryCache } from './helpers/dictionaryCache.js';

test('dictionary cache batch loads missing types and reuses cached results', async () => {
  const calls = [];
  const cache = createDictionaryCache(async (types) => {
    calls.push(types);
    return {
      dictionaries: Object.fromEntries(types.map((type) => [type, [{ value: 'a', label: type }]]))
    };
  });

  await cache.load(['product_category', 'product_category', 'region']);
  await cache.load(['product_category']);

  assert.deepEqual(calls, [['product_category', 'region']]);
  assert.deepEqual(cache.options('product_category'), [{ value: 'a', label: 'product_category' }]);
});

test('dictionary cache shares a pending batch request for the same types', async () => {
  const calls = [];
  let resolveLoad;
  const cache = createDictionaryCache(
    (types) =>
      new Promise((resolve) => {
        calls.push(types);
        resolveLoad = resolve;
      })
  );

  const first = cache.load(['product_category']);
  const second = cache.load(['product_category']);

  resolveLoad({
    dictionaries: {
      product_category: [{ value: 'phone', label: 'Phone' }]
    }
  });

  await Promise.all([first, second]);

  assert.deepEqual(calls, [['product_category']]);
  assert.deepEqual(cache.options('product_category'), [{ value: 'phone', label: 'Phone' }]);
});
