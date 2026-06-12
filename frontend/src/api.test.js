import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';

import {
  clearMyNotifications,
  clearMyTasks,
  createDictionaryType,
  createDictionaryValue,
  createProduct,
  createVariable,
  createOrder,
  createOrderPaymentCheckout,
  createParameterIntegrationChannel,
  createScheduledTask,
  exchangeOAuthLoginResult,
  exportAdminOrders,
  exportMyOrders,
  getAccessToken,
  getCurrentUser,
  getDictionaries,
  getMyOrders,
  getMyPoints,
  getProducts,
  getSiteSettings,
  getTaskDownload,
  getUser,
  getUserOrders,
  listAdminOrders,
  listUsers,
  listEventDeliveries,
  listEvents,
  listMessages,
  listNotifications,
  listDictionaryTypes,
  listDictionaryValues,
  listParameterIntegrationChannels,
  listParameterIntegrationSchemas,
  listScheduledTaskHistory,
  listScheduledTasks,
  listVariables,
  login,
  logout,
  oauthLoginURL,
  payOrder,
  realtimeWebSocketURL,
  request,
  setAccessToken,
  setDictionaryTypeEnabled,
  setDictionaryValueEnabled,
  setParameterIntegrationChannelEnabled,
  setScheduledTaskEnabled,
  setUserActive,
  setVariableEnabled,
  startOAuthLogin,
  summarizeTextWithLLM,
  triggerExportToast,
  updateDictionaryType,
  updateDictionaryValue,
  updateParameterIntegrationChannel,
  updateProduct,
  updateScheduledTask,
  updateVariable,
  uploadSiteLogo
} from './api.js';

const originalFetch = globalThis.fetch;
const originalLocalStorage = globalThis.localStorage;

afterEach(() => {
  globalThis.fetch = originalFetch;
  setAccessToken('');
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: originalLocalStorage
  });
});

function installMemoryStorage() {
  const values = new Map();
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem(key) {
        return values.has(key) ? values.get(key) : null;
      },
      setItem(key, value) {
        values.set(key, String(value));
      },
      removeItem(key) {
        values.delete(key);
      }
    }
  });
}

function jsonResponse(body, init = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status || 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers || {})
    }
  });
}

test('request encodes plain object bodies as JSON', async () => {
  let fetchPath;
  let fetchOptions;

  globalThis.fetch = async (path, options) => {
    fetchPath = path;
    fetchOptions = options;
    return jsonResponse({ success: true, data: { ok: true } });
  };

  const body = await request('/api/example', {
    method: 'POST',
    body: { name: 'Ada' }
  });

  assert.deepEqual(body, { ok: true });
  assert.equal(fetchPath, '/api/example');
  assert.equal(fetchOptions.method, 'POST');
  assert.equal(fetchOptions.headers.get('content-type'), 'application/json');
  assert.equal(fetchOptions.body, '{"name":"Ada"}');
});

test('request adds bearer token when one is stored', async () => {
  installMemoryStorage();
  setAccessToken('jwt-123');
  let fetchOptions;

  globalThis.fetch = async (_path, options) => {
    fetchOptions = options;
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await request('/api/example');

  assert.equal(fetchOptions.headers.get('authorization'), 'Bearer jwt-123');
});

test('request preserves caller authorization header', async () => {
  installMemoryStorage();
  setAccessToken('stored-token');
  let fetchOptions;

  globalThis.fetch = async (_path, options) => {
    fetchOptions = options;
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await request('/api/example', {
    headers: {
      authorization: 'Bearer caller-token'
    }
  });

  assert.equal(fetchOptions.headers.get('authorization'), 'Bearer caller-token');
});

test('request preserves caller content type headers', async () => {
  let fetchOptions;

  globalThis.fetch = async (_path, options) => {
    fetchOptions = options;
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await request('/api/example', {
    method: 'POST',
    headers: {
      'content-type': 'application/custom+json'
    },
    body: { name: 'Ada' }
  });

  assert.equal(fetchOptions.headers.get('content-type'), 'application/custom+json');
  assert.equal(fetchOptions.body, '{"name":"Ada"}');
});

test('request does not force JSON headers for FormData bodies', async () => {
  let fetchOptions;
  const form = new FormData();
  form.set('name', 'Ada');

  globalThis.fetch = async (_path, options) => {
    fetchOptions = options;
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await request('/api/example', {
    method: 'POST',
    body: form
  });

  assert.equal(fetchOptions.headers.has('content-type'), false);
  assert.equal(fetchOptions.body, form);
});

test('request returns null for empty success responses', async () => {
  globalThis.fetch = async () => new Response(null, { status: 204 });

  const body = await request('/api/example', { method: 'DELETE' });

  assert.equal(body, null);
});

test('request unwraps internal api success envelope data', async () => {
  globalThis.fetch = async () => jsonResponse({
    success: true,
    data: { id: 'u1', name: 'Ada' }
  });

  const body = await request('/api/example');

  assert.deepEqual(body, { id: 'u1', name: 'Ada' });
});

test('request throws safe server-provided envelope error messages', async () => {
  globalThis.fetch = async () => jsonResponse({
    success: false,
    error: {
      code: 'unauthorized',
      message: 'not logged in'
    }
  }, { status: 401 });

  await assert.rejects(
    () => request('/api/example'),
    (error) => {
      assert.equal(error.message, 'not logged in');
      return true;
    }
  );
});

test('request preserves legacy flat error messages while callers migrate', async () => {
  globalThis.fetch = async () => jsonResponse({ error: 'not logged in' }, { status: 401 });

  await assert.rejects(
    () => request('/api/example'),
    (error) => {
      assert.equal(error.message, 'not logged in');
      return true;
    }
  );
});

test('api helpers use relative api paths and shared request behavior', async () => {
  installMemoryStorage();
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    if (path === '/api/auth/login') {
      return jsonResponse({ success: true, data: { access_token: 'jwt-login', user: { id: 'u1', name: 'Ada' } } });
    }
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await login({ email: 'ada@example.com', password: 'secret' });
  await logout();

  assert.equal(calls[0].path, '/api/auth/login');
  assert.equal(calls[0].options.body, '{"email":"ada@example.com","password":"secret"}');
  assert.equal(getAccessToken(), '');
  assert.equal(calls[1].path, '/api/auth/logout');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.headers.get('authorization'), 'Bearer jwt-login');
});

test('oauth api helpers build start URL and store exchanged token', async () => {
  installMemoryStorage();
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({
      success: true,
      data: {
        access_token: 'jwt-oauth',
        user: { id: 'u1', name: 'Ada' }
      }
    });
  };

  const originalLocation = globalThis.location;
  let assignedURL = '';
  Object.defineProperty(globalThis, 'location', {
    configurable: true,
    value: {
      assign(url) {
        assignedURL = url;
      }
    }
  });

  assert.equal(
    oauthLoginURL('google', '/app/orders?tab=mine'),
    '/api/auth/oauth/google/start?redirect_path=%2Fapp%2Forders%3Ftab%3Dmine'
  );
  startOAuthLogin('github', '/app/products');
  assert.equal(assignedURL, '/api/auth/oauth/github/start?redirect_path=%2Fapp%2Fproducts');

  const result = await exchangeOAuthLoginResult('one-time-token');
  assert.equal(result.access_token, 'jwt-oauth');
  assert.equal(getAccessToken(), 'jwt-oauth');
  assert.equal(calls[0].path, '/api/auth/oauth/exchange');
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(calls[0].options.body, '{"token":"one-time-token"}');

  Object.defineProperty(globalThis, 'location', {
    configurable: true,
    value: originalLocation
  });
});

test('dictionary and order api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await getDictionaries(['product_category', 'product_category', 'region']);
  await getMyOrders({ page: 2, pageSize: 10, status: 'pending' });
  await exportMyOrders({ status: 'paid' });
  await listAdminOrders({ userId: '019ea0c1-0001-7000-8000-000000000001', status: 'paid', page: 1, pageSize: 20 });
  await exportAdminOrders({ userId: '019ea0c1-0001-7000-8000-000000000001', status: 'pending' });
  await getUserOrders('019ea0c1-0001-7000-8000-000000000001', { page: 2, pageSize: 10 });
  await createOrder({
    product_id: 'product-1'
  });
  await createOrderPaymentCheckout('o001');
  await payOrder('o001');
  await getMyPoints();
  await getProducts();
  await triggerExportToast();
  await clearMyNotifications();
  await summarizeTextWithLLM({
    text: 'Source text',
    prompt: 'Summarize briefly',
    dimensions: ['summary']
  });
  await clearMyTasks();
  await getTaskDownload('task 1');
  await listNotifications({
    page: 2,
    pageSize: 10,
    type: 'sms',
    email: 'ada@example.com',
    phone: '13800000000'
  });

  assert.equal(calls[0].path, '/api/public/dictionaries?types=product_category,region');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/user/orders?page=2&page_size=10&status=pending');
  assert.equal(calls[2].path, '/api/user/orders/export?status=paid');
  assert.equal(calls[2].options.method, 'POST');
  assert.equal(calls[3].path, '/api/admin/orders?user_id=019ea0c1-0001-7000-8000-000000000001&status=paid&page=1&page_size=20');
  assert.equal(calls[4].path, '/api/admin/orders/export?user_id=019ea0c1-0001-7000-8000-000000000001&status=pending');
  assert.equal(calls[4].options.method, 'POST');
  assert.equal(calls[5].path, '/api/orders/user/019ea0c1-0001-7000-8000-000000000001?page=2&page_size=10');
  assert.equal(calls[6].path, '/api/user/orders');
  assert.equal(calls[6].options.method, 'POST');
  assert.equal(calls[6].options.body, '{"product_id":"product-1"}');
  assert.equal(calls[7].path, '/api/orders/o001/payment-checkout');
  assert.equal(calls[7].options.method, 'POST');
  assert.equal(calls[8].path, '/api/orders/o001/pay');
  assert.equal(calls[8].options.method, 'POST');
  assert.equal(calls[9].path, '/api/user/points');
  assert.equal(calls[10].path, '/api/products');
  assert.equal(calls[11].path, '/api/user/notifications/test-export-toast');
  assert.equal(calls[11].options.method, 'POST');
  assert.equal(calls[12].path, '/api/user/notifications/clear');
  assert.equal(calls[12].options.method, 'POST');
  assert.equal(calls[13].path, '/api/llm/summaries');
  assert.equal(calls[13].options.method, 'POST');
  assert.equal(calls[13].options.body, '{"text":"Source text","prompt":"Summarize briefly","dimensions":["summary"]}');
  assert.equal(calls[14].path, '/api/user/tasks/clear');
  assert.equal(calls[14].options.method, 'POST');
  assert.equal(calls[15].path, '/api/user/tasks/task%201/download');
  assert.equal(calls[16].path, '/api/admin/notifications?page=2&page_size=10&type=sms&email=ada%40example.com&phone=13800000000');
});

test('current user helper uses user persona path', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { user: { id: 'user 1' } } });
  };

  await getCurrentUser();

  assert.equal(calls[0].path, '/api/user/me');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
});

test('dictionary management api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  const typePayload = {
    type_key: 'order_status',
    name: 'Order status',
    enabled: true,
    description: 'Order lifecycle'
  };
  const valuePayload = {
    dictionary_type_id: 'type 1',
    value_code: 'pending',
    label: 'Pending',
    sort_order: 10,
    enabled: true,
    description: 'Waiting'
  };

  await listDictionaryTypes();
  await createDictionaryType(typePayload);
  await updateDictionaryType('type 1', typePayload);
  await setDictionaryTypeEnabled('type 1', false);
  await listDictionaryValues('type 1');
  await createDictionaryValue('type 1', valuePayload);
  await updateDictionaryValue('type 1', 'value 1', valuePayload);
  await setDictionaryValueEnabled('value 1', false);

  assert.equal(calls[0].path, '/api/admin/dictionary/types');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/dictionary/types');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.body, JSON.stringify(typePayload));
  assert.equal(calls[2].path, '/api/admin/dictionary/types/type%201');
  assert.equal(calls[2].options.method, 'PUT');
  assert.equal(calls[3].path, '/api/admin/dictionary/types/type%201/enabled');
  assert.equal(calls[3].options.method, 'PATCH');
  assert.equal(calls[3].options.body, '{"enabled":false}');
  assert.equal(calls[4].path, '/api/admin/dictionary/types/type%201/values');
  assert.equal(calls[5].path, '/api/admin/dictionary/types/type%201/values');
  assert.equal(calls[5].options.method, 'POST');
  assert.equal(calls[5].options.body, JSON.stringify(valuePayload));
  assert.equal(calls[6].path, '/api/admin/dictionary/types/type%201/values/value%201');
  assert.equal(calls[6].options.method, 'PUT');
  assert.equal(calls[7].path, '/api/admin/dictionary/values/value%201/enabled');
  assert.equal(calls[7].options.method, 'PATCH');
  assert.equal(calls[7].options.body, '{"enabled":false}');
});

test('user management api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  await getUser('user 0');
  await listUsers({ page: 2, pageSize: 10 });
  await setUserActive('user 1', false);
  await setUserActive('user 2', true);

  assert.equal(calls[0].path, '/api/admin/users/user%200');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/users?page=2&page_size=10');
  assert.equal(calls[2].path, '/api/admin/users/user%201/active');
  assert.equal(calls[2].options.method, 'PATCH');
  assert.equal(calls[2].options.body, '{"active":false}');
  assert.equal(calls[3].path, '/api/admin/users/user%202/active');
  assert.equal(calls[3].options.method, 'PATCH');
  assert.equal(calls[3].options.body, '{"active":true}');
});

test('scheduler and message api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  const payload = {
    name: 'Nightly export',
    job_name: 'scheduler.noop',
    schedule_type: 'cron',
    schedule_value: '*/5 * * * *',
    payload_json: '{}',
    enabled: true
  };

  await listScheduledTasks();
  await createScheduledTask(payload);
  await updateScheduledTask('task 1', payload);
  await setScheduledTaskEnabled('task 1', false);
  await listScheduledTaskHistory('task 1');
  await listEvents({ page: 3, pageSize: 10 });
  await listEventDeliveries('event 1');
  await listMessages();
  await listMessages('domain-events');

  assert.equal(calls[0].path, '/api/admin/scheduler/tasks');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/scheduler/tasks');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.body, JSON.stringify(payload));
  assert.equal(calls[2].path, '/api/admin/scheduler/tasks/task%201');
  assert.equal(calls[2].options.method, 'PUT');
  assert.equal(calls[3].path, '/api/admin/scheduler/tasks/task%201/enabled');
  assert.equal(calls[3].options.method, 'PATCH');
  assert.equal(calls[3].options.body, '{"enabled":false}');
  assert.equal(calls[4].path, '/api/admin/scheduler/tasks/task%201/history');
  assert.equal(calls[5].path, '/api/admin/events?page=3&page_size=10');
  assert.equal(calls[6].path, '/api/admin/events/event%201/deliveries');
  assert.equal(calls[7].path, '/api/admin/messages');
  assert.equal(calls[8].path, '/api/admin/messages?queue=domain-events');
});

test('parameter integration api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  const payload = {
    scenario: 'payment',
    channel_code: 'creem',
    provider_code: 'creem',
    adapter_key: 'payment.creem.hosted_checkout',
    environment: 'test',
    enabled: true,
    priority: 10,
    webhook_enabled: true,
    is_primary: true,
    config_json: '{}',
    metadata_json: '{}',
    credential_type: 'payment_bundle',
    credential_value: 'secret'
  };

  await listParameterIntegrationChannels('payment');
  await listParameterIntegrationSchemas('payment');
  await listParameterIntegrationChannels('email');
  await listParameterIntegrationSchemas('email');
  await listParameterIntegrationChannels('oss');
  await listParameterIntegrationSchemas('oss');
  await createParameterIntegrationChannel(payload);
  await updateParameterIntegrationChannel('channel 1', payload);
  await setParameterIntegrationChannelEnabled('channel 1', false);

  assert.equal(calls[0].path, '/api/admin/parameters/integration-channels?scenario=payment');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/parameters/integration-schemas?scenario=payment');
  assert.equal(calls[2].path, '/api/admin/parameters/integration-channels?scenario=email');
  assert.equal(calls[3].path, '/api/admin/parameters/integration-schemas?scenario=email');
  assert.equal(calls[4].path, '/api/admin/parameters/integration-channels?scenario=oss');
  assert.equal(calls[5].path, '/api/admin/parameters/integration-schemas?scenario=oss');
  assert.equal(calls[6].path, '/api/admin/parameters/integration-channels');
  assert.equal(calls[6].options.method, 'POST');
  assert.equal(calls[6].options.body, JSON.stringify(payload));
  assert.equal(calls[7].path, '/api/admin/parameters/integration-channels/channel%201');
  assert.equal(calls[7].options.method, 'PUT');
  assert.equal(calls[8].path, '/api/admin/parameters/integration-channels/channel%201/enabled');
  assert.equal(calls[8].options.method, 'PATCH');
  assert.equal(calls[8].options.body, '{"enabled":false}');
});

test('setting api helpers use relative api paths and multipart upload', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({
      success: true,
      data: {
        logo_url: '/logo.png',
        logo_configured: false,
        logo_updated_at: '',
        logo_upload_available: false,
        logo_upload_unavailable_reason: 'Primary OSS provider is not configured'
      }
    });
  };

  const logoBlob = new Blob(['logo'], { type: 'image/png' });
  await getSiteSettings();
  await uploadSiteLogo(logoBlob);

  assert.equal(calls[0].path, '/api/public/settings/site');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/settings/site/logo');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.body instanceof FormData, true);
  assert.equal(calls[1].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].options.headers.has('content-type'), false);
});

test('variable api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { ok: true } });
  };

  const payload = {
    key: 'feature.new_checkout',
    name: 'New checkout',
    value_type: 'boolean',
    value_json: 'true',
    enabled: true,
    description: 'Feature switch'
  };

  await listVariables();
  await createVariable(payload);
  await updateVariable('variable 1', payload);
  await setVariableEnabled('variable 1', false);

  assert.equal(calls[0].path, '/api/admin/variables');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[1].path, '/api/admin/variables');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.body, JSON.stringify(payload));
  assert.equal(calls[2].path, '/api/admin/variables/variable%201');
  assert.equal(calls[2].options.method, 'PUT');
  assert.equal(calls[3].path, '/api/admin/variables/variable%201/enabled');
  assert.equal(calls[3].options.method, 'PATCH');
  assert.equal(calls[3].options.body, '{"enabled":false}');
});

test('product api helpers use relative api paths', async () => {
  installMemoryStorage();
  setAccessToken('jwt-api');
  const calls = [];

  globalThis.fetch = async (path, options) => {
    calls.push({ path, options });
    return jsonResponse({ success: true, data: { product: { id: 'product 1' } } });
  };

  const payload = {
    name: 'Premium',
    creem_product_id: 'prod_1',
    billing_type: 'subscription',
    membership_level: 'premium',
    subscription_interval: 'month',
    enabled: true
  };

  await createProduct(payload);
  await updateProduct('product 1', payload);

  assert.equal(calls[0].path, '/api/admin/products');
  assert.equal(calls[0].options.headers.get('authorization'), 'Bearer jwt-api');
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(calls[0].options.body, JSON.stringify(payload));
  assert.equal(calls[1].path, '/api/admin/products/product%201');
  assert.equal(calls[1].options.method, 'PUT');
  assert.equal(calls[1].options.body, JSON.stringify(payload));
});

test('realtime websocket helper uses the current host and websocket scheme', () => {
  assert.equal(
    realtimeWebSocketURL({ protocol: 'http:', host: '127.0.0.1:5173' }),
    'ws://127.0.0.1:5173/api/user/realtime/ws'
  );
  assert.equal(
    realtimeWebSocketURL({ protocol: 'https:', host: 'example.com' }),
    'wss://example.com/api/user/realtime/ws'
  );
});

test('realtime websocket helper includes stored access token and client id', () => {
  installMemoryStorage();
  setAccessToken('jwt socket');

  assert.equal(
    realtimeWebSocketURL({ protocol: 'http:', host: '127.0.0.1:5173' }, { clientId: 'client 1' }),
    'ws://127.0.0.1:5173/api/user/realtime/ws?access_token=jwt+socket&client_id=client+1'
  );
});
