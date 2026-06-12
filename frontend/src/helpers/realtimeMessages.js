export const realtimeMessageTypes = Object.freeze({
  points: 'points',
  asyncExportTask: 'async_export_task',
  notification: 'notification',
  heavyTask: 'heavy_task'
});

export const realtimePresentations = Object.freeze({
  refresh: 'refresh',
  toast: 'toast'
});

export function defaultPresentation(type) {
  if (type === realtimeMessageTypes.asyncExportTask || type === realtimeMessageTypes.notification || type === realtimeMessageTypes.heavyTask) {
    return realtimePresentations.toast;
  }
  return realtimePresentations.refresh;
}

export function normalizeRealtimeMessage(value) {
  if (!isObject(value) || typeof value.type !== 'string' || value.type.trim() === '') {
    return null;
  }

  return {
    type: value.type,
    presentation: typeof value.presentation === 'string' && value.presentation.trim() !== ''
      ? value.presentation
      : defaultPresentation(value.type),
    payload: isObject(value.payload) ? value.payload : {}
  };
}

export function dispatchRealtimeMessage(value, handlers = {}) {
  const message = normalizeRealtimeMessage(value);
  if (!message) {
    return false;
  }

  if (message.presentation === realtimePresentations.toast) {
    handlers.toast?.(toastFromRealtimeMessage(message), message);
    return Boolean(handlers.toast);
  }

  if (message.presentation === realtimePresentations.refresh && message.type === realtimeMessageTypes.points) {
    handlers.refreshPoints?.(message.payload, message);
    return Boolean(handlers.refreshPoints);
  }

  return false;
}

export function toastFromRealtimeMessage(message) {
  const payload = isObject(message.payload) ? message.payload : {};
  const status = normalizedToastStatus(message.type, payload.status);
  return {
    id: payload.task_id || payload.id || `${message.type}-${status}`,
    level: toastLevel(status),
    message: payload.message || notificationToastMessage(message.type, payload) || fallbackToastMessage(message.type, status)
  };
}

function notificationToastMessage(type, payload) {
  if (type !== realtimeMessageTypes.notification) {
    return '';
  }
  if (typeof payload.summary === 'string' && payload.summary.trim() !== '') {
    return payload.summary;
  }
  if (typeof payload.title === 'string' && payload.title.trim() !== '') {
    return payload.title;
  }
  return '';
}

function fallbackToastMessage(type, status) {
  if (type === realtimeMessageTypes.asyncExportTask) {
    return status === 'completed' ? 'Export completed' : 'Export task updated';
  }
  return 'Realtime notification';
}

function normalizedToastStatus(type, status) {
  if (typeof status === 'string' && status.trim() !== '') {
    return status;
  }
  if (type === realtimeMessageTypes.notification) {
    return 'info';
  }
  return 'info';
}

function toastLevel(status) {
  switch (status) {
    case 'completed':
    case 'success':
      return 'success';
    case 'failed':
    case 'error':
      return 'error';
    case 'running':
    case 'pending':
      return 'info';
    default:
      return 'info';
  }
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}
