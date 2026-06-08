function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]';
}

function shouldEncodeJSONBody(body) {
  return isPlainObject(body) || Array.isArray(body);
}

const authStorageKey = 'app_access_token';

export function getAccessToken() {
  try {
    return globalThis.localStorage?.getItem(authStorageKey) || '';
  } catch {
    return '';
  }
}

export function setAccessToken(token) {
  try {
    if (token) {
      globalThis.localStorage?.setItem(authStorageKey, token);
    } else {
      globalThis.localStorage?.removeItem(authStorageKey);
    }
  } catch {
    // Storage may be unavailable in private or restricted environments.
  }
}

async function parseResponseBody(response) {
  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  if (text === '') {
    return null;
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return JSON.parse(text);
  }

  return text;
}

export async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  const init = { ...options, headers };
  const token = getAccessToken();

  if (token && !headers.has('authorization')) {
    headers.set('authorization', `Bearer ${token}`);
  }

  if (shouldEncodeJSONBody(init.body)) {
    init.body = JSON.stringify(init.body);
    if (!headers.has('content-type')) {
      headers.set('content-type', 'application/json');
    }
  }

  const response = await fetch(path, {
    ...init
  });

  const body = await parseResponseBody(response);

  if (!response.ok) {
    const message = typeof body === 'object' && body !== null
      ? body.error?.message || body.error || body.message
      : body;
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  if (typeof body === 'object' && body !== null && Object.hasOwn(body, 'success')) {
    if (body.success === true) {
      return Object.hasOwn(body, 'data') ? body.data : null;
    }

    const message = body.error?.message || 'Request failed';
    throw new Error(message);
  }

  return body;
}

export function getAuthStatus() {
  return request('/api/auth/status');
}

export function login(payload) {
  return request('/api/auth/login', {
    method: 'POST',
    body: payload
  }).then((result) => {
    setAccessToken(result?.access_token || '');
    return result;
  });
}

export function register(payload) {
  return request('/api/auth/register', {
    method: 'POST',
    body: payload
  }).then((result) => {
    setAccessToken(result?.access_token || '');
    return result;
  });
}

export function logout() {
  const result = request('/api/auth/logout', {
    method: 'POST'
  });
  setAccessToken('');
  return result;
}

export function forgotPassword(payload) {
  return request('/api/auth/forgot-password', {
    method: 'POST',
    body: payload
  });
}

export function resetPassword(payload) {
  return request('/api/auth/reset-password', {
    method: 'POST',
    body: payload
  });
}

export function getUser(id) {
  return request(`/api/users/${encodeURIComponent(id)}`);
}

export function listUsers(pagination = {}) {
  const params = new URLSearchParams();
  if (pagination.page) {
    params.set('page', String(pagination.page));
  }
  if (pagination.pageSize) {
    params.set('page_size', String(pagination.pageSize));
  }
  const query = params.toString();
  return request(`/api/users${query ? `?${query}` : ''}`);
}

export function setUserActive(id, active) {
  return request(`/api/users/${encodeURIComponent(id)}/active`, {
    method: 'PATCH',
    body: { active }
  });
}

export function getDictionaries(types) {
  const uniqueTypes = [...new Set((types || []).filter(Boolean))];
  const query = uniqueTypes.map((type) => encodeURIComponent(type)).join(',');
  return request(`/api/dictionaries?types=${query}`);
}

export function listDictionaryTypes() {
  return request('/api/dictionary/types');
}

export function createDictionaryType(payload) {
  return request('/api/dictionary/types', {
    method: 'POST',
    body: payload
  });
}

export function updateDictionaryType(id, payload) {
  return request(`/api/dictionary/types/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: payload
  });
}

export function setDictionaryTypeEnabled(id, enabled) {
  return request(`/api/dictionary/types/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled }
  });
}

export function listDictionaryValues(typeId) {
  return request(`/api/dictionary/types/${encodeURIComponent(typeId)}/values`);
}

export function createDictionaryValue(typeId, payload) {
  return request(`/api/dictionary/types/${encodeURIComponent(typeId)}/values`, {
    method: 'POST',
    body: payload
  });
}

export function updateDictionaryValue(typeId, id, payload) {
  return request(`/api/dictionary/types/${encodeURIComponent(typeId)}/values/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: payload
  });
}

export function setDictionaryValueEnabled(id, enabled) {
  return request(`/api/dictionary/values/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled }
  });
}

export function getUserOrders(userId, pagination = {}) {
  const params = new URLSearchParams();
  if (pagination.page) {
    params.set('page', String(pagination.page));
  }
  if (pagination.pageSize) {
    params.set('page_size', String(pagination.pageSize));
  }
  const query = params.toString();
  return request(`/api/orders/user/${encodeURIComponent(userId)}${query ? `?${query}` : ''}`);
}

export function createOrder(payload) {
  return request('/api/orders', {
    method: 'POST',
    body: payload
  });
}

export function payOrder(orderId) {
  return request(`/api/orders/${encodeURIComponent(orderId)}/pay`, {
    method: 'POST'
  });
}

export function createOrderPaymentCheckout(orderId) {
  return request(`/api/orders/${encodeURIComponent(orderId)}/payment-checkout`, {
    method: 'POST'
  });
}

export function getMyPoints() {
  return request('/api/points/me');
}

export function getProducts() {
  return request('/api/products');
}

export function triggerExportToast() {
  return request('/api/notifications/test-export-toast', {
    method: 'POST'
  });
}

export function listNotifications(filters = {}) {
  const params = new URLSearchParams();
  if (filters.page) {
    params.set('page', String(filters.page));
  }
  if (filters.pageSize) {
    params.set('page_size', String(filters.pageSize));
  }
  if (filters.type) {
    params.set('type', String(filters.type));
  }
  if (filters.email) {
    params.set('email', String(filters.email));
  }
  if (filters.phone) {
    params.set('phone', String(filters.phone));
  }
  const query = params.toString();
  return request(`/api/notifications${query ? `?${query}` : ''}`);
}

export function listScheduledTasks() {
  return request('/api/scheduler/tasks');
}

export function createScheduledTask(payload) {
  return request('/api/scheduler/tasks', {
    method: 'POST',
    body: payload
  });
}

export function updateScheduledTask(id, payload) {
  return request(`/api/scheduler/tasks/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: payload
  });
}

export function setScheduledTaskEnabled(id, enabled) {
  return request(`/api/scheduler/tasks/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled }
  });
}

export function listScheduledTaskHistory(id) {
  return request(`/api/scheduler/tasks/${encodeURIComponent(id)}/history`);
}

export function listEvents(pagination = {}) {
  const params = new URLSearchParams();
  if (pagination.page) {
    params.set('page', String(pagination.page));
  }
  if (pagination.pageSize) {
    params.set('page_size', String(pagination.pageSize));
  }
  const query = params.toString();
  return request(`/api/events${query ? `?${query}` : ''}`);
}

export function listEventDeliveries(eventId) {
  return request(`/api/events/${encodeURIComponent(eventId)}/deliveries`);
}

export function listMessages(queue = '') {
  const query = queue ? `?queue=${encodeURIComponent(queue)}` : '';
  return request(`/api/messages${query}`);
}

export function listParameterIntegrationChannels(scenario) {
  const query = scenario ? `?scenario=${encodeURIComponent(scenario)}` : '';
  return request(`/api/parameters/integration-channels${query}`);
}

export function listParameterIntegrationSchemas(scenario) {
  const query = scenario ? `?scenario=${encodeURIComponent(scenario)}` : '';
  return request(`/api/parameters/integration-schemas${query}`);
}

export function createParameterIntegrationChannel(payload) {
  return request('/api/parameters/integration-channels', {
    method: 'POST',
    body: payload
  });
}

export function updateParameterIntegrationChannel(id, payload) {
  return request(`/api/parameters/integration-channels/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: payload
  });
}

export function setParameterIntegrationChannelEnabled(id, enabled) {
  return request(`/api/parameters/integration-channels/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled }
  });
}

export function listVariables() {
  return request('/api/variables');
}

export function createVariable(payload) {
  return request('/api/variables', {
    method: 'POST',
    body: payload
  });
}

export function updateVariable(id, payload) {
  return request(`/api/variables/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: payload
  });
}

export function setVariableEnabled(id, enabled) {
  return request(`/api/variables/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled }
  });
}

export function pointsSSEURL(locationObject = globalThis.location) {
  const protocol = locationObject?.protocol === 'https:' ? 'https:' : 'http:';
  const host = locationObject?.host || '127.0.0.1:5173';
  const url = new URL(`${protocol}//${host}/api/points/sse`);
  const token = getAccessToken();
  if (token) {
    url.searchParams.set('access_token', token);
  }
  return url.toString();
}
