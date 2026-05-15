import type { ScanFilterRule } from './collector.service';

const SCAN_SETTINGS_PREFIX = 'zq.desktop.collector.scan';

export interface CollectorScanSettings {
  host: string;
  maxLinks: number;
  scanSitemap: boolean;
  filterRules: ScanFilterRule[];
  updatedAt: string;
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

function buildScanSettingsKey(host: string): string {
  return `${SCAN_SETTINGS_PREFIX}.${normaliseHost(host)}`;
}

export function extractScanSettingsHost(urlOrHost: string): string {
  return normaliseHost(urlOrHost);
}

export function loadCollectorScanSettings(urlOrHost: string): CollectorScanSettings | null {
  const host = normaliseHost(urlOrHost);
  if (!host) {
    return null;
  }

  const raw = localStorage.getItem(buildScanSettingsKey(host));
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<CollectorScanSettings>;
    if (!parsed.host || typeof parsed.maxLinks !== 'number' || !Array.isArray(parsed.filterRules)) {
      localStorage.removeItem(buildScanSettingsKey(host));
      return null;
    }
    return {
      host: parsed.host,
      maxLinks: parsed.maxLinks,
      scanSitemap: parsed.scanSitemap === true,
      filterRules: parsed.filterRules.filter(
        (rule): rule is ScanFilterRule =>
          Boolean(rule) && typeof rule.field === 'string' && typeof rule.operator === 'string' && typeof rule.value === 'string',
      ),
      updatedAt: parsed.updatedAt || '',
    };
  } catch {
    localStorage.removeItem(buildScanSettingsKey(host));
    return null;
  }
}

export function saveCollectorScanSettings(settings: CollectorScanSettings): void {
  const host = normaliseHost(settings.host);
  if (!host) {
    return;
  }

  const payload: CollectorScanSettings = {
    host,
    maxLinks: settings.maxLinks,
    scanSitemap: settings.scanSitemap === true,
    filterRules: Array.isArray(settings.filterRules) ? settings.filterRules : [],
    updatedAt: new Date().toISOString(),
  };

  localStorage.setItem(buildScanSettingsKey(host), JSON.stringify(payload));
}

export function deleteCollectorScanSettings(urlOrHost: string): void {
  const host = normaliseHost(urlOrHost);
  if (!host) {
    return;
  }
  localStorage.removeItem(buildScanSettingsKey(host));
}
