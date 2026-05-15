export function isValidHttpUrl(value: string): boolean {
  const nextValue = value.trim();
  if (!nextValue) {
    return false;
  }

  try {
    const parsed = new URL(nextValue);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}

export function normaliseExternalHttpUrl(value: string): string {
  const nextValue = value.trim();
  if (!nextValue) {
    return '';
  }
  if (isValidHttpUrl(nextValue)) {
    return nextValue;
  }
  if (nextValue.startsWith('//')) {
    return `https:${nextValue}`;
  }
  if (/^[a-z0-9.-]+\.[a-z]{2,}([/:?#]|$)/i.test(nextValue)) {
    return `https://${nextValue}`;
  }

  return nextValue;
}

function getAssetOrigin(apiBaseUrl?: string | null): string | null {
  if (!apiBaseUrl?.trim()) {
    return null;
  }

  try {
    return new URL(apiBaseUrl).origin;
  } catch {
    return null;
  }
}

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, '')}/${path.replace(/^\/+/, '')}`;
}

export function resolveTenantAssetUrl(value: string, apiBaseUrl?: string | null): string {
  const nextValue = value.trim();
  if (!nextValue) {
    return '';
  }
  if (
    isValidHttpUrl(nextValue)
    || nextValue.startsWith('data:')
    || nextValue.startsWith('blob:')
  ) {
    return nextValue;
  }
  if (nextValue.startsWith('//')) {
    return `https:${nextValue}`;
  }

  const origin = getAssetOrigin(apiBaseUrl);
  if (!origin) {
    return nextValue;
  }

  if (nextValue.startsWith('/storage/')) {
    return `${origin}${nextValue}`;
  }

  if (nextValue === '/storage') {
    return `${origin}${nextValue}`;
  }

  if (nextValue.startsWith('storage/')) {
    return joinUrl(origin, nextValue);
  }

  return joinUrl(`${origin}/storage`, nextValue);
}

function resolveSrcsetValue(value: string, apiBaseUrl?: string | null): string {
  return value
    .split(',')
    .map((candidate) => candidate.trim())
    .filter(Boolean)
    .map((candidate) => {
      const [url, descriptor] = candidate.split(/\s+/, 2);
      const resolvedUrl = resolveTenantAssetUrl(url || '', apiBaseUrl);
      return descriptor ? `${resolvedUrl} ${descriptor}` : resolvedUrl;
    })
    .join(', ');
}

export function normaliseTenantHtmlAssets(html: string, apiBaseUrl?: string | null): string {
  if (!html.trim() || !apiBaseUrl?.trim()) {
    return html;
  }

  if (typeof DOMParser === 'undefined') {
    return html
      .replace(/(\s(?:src|poster)=["'])([^"']+)(["'])/gi, (_match, prefix, value, suffix) => (
        `${prefix}${resolveTenantAssetUrl(value, apiBaseUrl)}${suffix}`
      ))
      .replace(/(\ssrcset=["'])([^"']+)(["'])/gi, (_match, prefix, value, suffix) => (
        `${prefix}${resolveSrcsetValue(value, apiBaseUrl)}${suffix}`
      ));
  }

  const parser = new DOMParser();
  const documentNode = parser.parseFromString(html, 'text/html');

  documentNode.querySelectorAll<HTMLElement>('[src], [poster], [srcset]').forEach((element) => {
    const src = element.getAttribute('src');
    if (src) {
      element.setAttribute('src', resolveTenantAssetUrl(src, apiBaseUrl));
    }

    const poster = element.getAttribute('poster');
    if (poster) {
      element.setAttribute('poster', resolveTenantAssetUrl(poster, apiBaseUrl));
    }

    const srcset = element.getAttribute('srcset');
    if (srcset) {
      element.setAttribute('srcset', resolveSrcsetValue(srcset, apiBaseUrl));
    }
  });

  return documentNode.body.innerHTML;
}
