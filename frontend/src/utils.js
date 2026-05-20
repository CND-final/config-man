export function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

export function statusClass(status) {
  const normalized = String(status).toLowerCase();
  if (['healthy', 'synced', 'approved', 'valid'].includes(normalized)) {
    return 'success';
  }
  if (['review', 'changed', 'pending', 'warning', 'modified'].includes(normalized)) {
    return 'warning';
  }
  if (['blocked', 'failed', 'danger', 'rejected'].includes(normalized)) {
    return 'danger';
  }
  return 'neutral';
}

export function initials(name) {
  return String(name)
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();
}

export function formatDateTime(value) {
  if (!value) return 'unknown time';
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(new Date(value));
}

export function displayVersionValue(value) {
  if (value === null || value === undefined) return 'No previous value';
  return String(value);
}

export function hasPreviousValue(version) {
  return version?.oldValue !== null && version?.oldValue !== undefined;
}

export function parseEnvironmentInput(value) {
  return value
    .split(',')
    .map((environment) => environment.trim().toLowerCase())
    .filter(Boolean);
}

export function isProdSensitiveEdit(config) {
  return (
    config.environment === 'prod' &&
    (config.isSensitive || /(database|db).*(password|url)|password|secret|token/i.test(config.key))
  );
}
