const sqliteUTCDateTimePattern = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?$/;
const isoWithoutTimezonePattern = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?$/;

export function parseUTCDateTime(value) {
  const raw = String(value || '').trim();
  if (!raw) return null;

  if (sqliteUTCDateTimePattern.test(raw)) {
    return new Date(`${raw.replace(' ', 'T')}Z`);
  }
  if (isoWithoutTimezonePattern.test(raw)) {
    return new Date(`${raw}Z`);
  }
  return new Date(raw);
}

export function formatLocalDateTime(value, fallback = '--') {
  if (!value) return fallback;

  const date = parseUTCDateTime(value);
  if (!date || Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}
