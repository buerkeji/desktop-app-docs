const HISTORY_PREFIX = 'zq.desktop.collector.history';

export interface CollectedHistoryItem {
  url: string;
  title: string;
  host: string;
  fetchedAt: string;
}

function normaliseHost(value: string): string {
  const nextValue = value.trim();
  if (!nextValue) {
    return '';
  }
  try {
    return new URL(nextValue).hostname.toLowerCase();
  } catch {
    return nextValue.replace(/^https?:\/\//i, '').replace(/\/.*$/, '').toLowerCase();
  }
}

function buildHistoryKey(host: string): string {
  return `${HISTORY_PREFIX}.${normaliseHost(host)}`;
}

export function loadCollectedUrls(host: string): CollectedHistoryItem[] {
  const key = buildHistoryKey(host);
  if (!key.endsWith('.') && normaliseHost(host)) {
    const raw = localStorage.getItem(key);
    if (raw) {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
          return parsed.filter(
            (item): item is CollectedHistoryItem =>
              Boolean(item) && typeof item.url === 'string' && typeof item.host === 'string',
          );
        }
      } catch {
        localStorage.removeItem(key);
      }
    }
  }
  return [];
}

export function saveCollectedUrl(url: string, title: string, host: string): void {
  const normalised = normaliseHost(host);
  if (!normalised || !url) {
    return;
  }

  const key = buildHistoryKey(normalised);
  const existing = loadCollectedUrls(normalised);

  const alreadyExists = existing.some((item) => item.url === url);
  if (alreadyExists) {
    return;
  }

  existing.push({
    url,
    title: title || url,
    host: normalised,
    fetchedAt: new Date().toISOString(),
  });

  const trimmed = existing.slice(-500);
  localStorage.setItem(key, JSON.stringify(trimmed));
}

export function deleteCollectedUrl(host: string, url: string): void {
  const normalised = normaliseHost(host);
  if (!normalised) return;

  const key = buildHistoryKey(normalised);
  const existing = loadCollectedUrls(normalised);
  const filtered = existing.filter((item) => item.url !== url);

  if (filtered.length) {
    localStorage.setItem(key, JSON.stringify(filtered));
  } else {
    localStorage.removeItem(key);
  }
}

export function clearCollectedHistory(host: string): void {
  const normalised = normaliseHost(host);
  if (!normalised) return;

  localStorage.removeItem(buildHistoryKey(normalised));
}

export function getCollectedUrlSet(host: string): Set<string> {
  const items = loadCollectedUrls(host);
  return new Set(items.map((item) => item.url));
}
