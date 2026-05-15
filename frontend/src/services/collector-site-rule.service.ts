import { hasAppMethod, invokeApp } from './bridge';
import type { CollectWebPageResult, CollectorImageItem } from './collector.service';

export interface CollectorSiteRuleItem {
  host: string;
  siteNamePath?: string;
  titlePath?: string;
  descriptionPath?: string;
  excerptPath?: string;
  contentPath?: string;
  iconPath?: string;
  thumbnailPath?: string;
  tagsPath?: string;
  publishedAtPath?: string;
  updatedAt: string;
}

export type CollectorRuleField =
  | 'siteName'
  | 'title'
  | 'description'
  | 'excerpt'
  | 'content'
  | 'icon'
  | 'thumbnail'
  | 'tags'
  | 'publishedAt';

const COLLECTOR_RULE_PREFIX = 'zq.desktop.collector.rule';

const RULE_FIELD_PATH_MAP: Record<CollectorRuleField, keyof CollectorSiteRuleItem> = {
  siteName: 'siteNamePath',
  title: 'titlePath',
  description: 'descriptionPath',
  excerpt: 'excerptPath',
  content: 'contentPath',
  icon: 'iconPath',
  thumbnail: 'thumbnailPath',
  tags: 'tagsPath',
  publishedAt: 'publishedAtPath',
};

interface CollectorSiteRuleBridgeInput {
  url: string;
}

interface SaveCollectorSiteRuleBridgeInput {
  host: string;
  siteNamePath?: string;
  titlePath?: string;
  descriptionPath?: string;
  excerptPath?: string;
  contentPath?: string;
  iconPath?: string;
  thumbnailPath?: string;
  tagsPath?: string;
  publishedAtPath?: string;
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

function buildCollectorRuleKey(host: string): string {
  return `${COLLECTOR_RULE_PREFIX}.${normaliseHost(host)}`;
}

function mapCollectorSiteRule(source: Partial<CollectorSiteRuleItem> | null | undefined): CollectorSiteRuleItem | null {
  if (!source) {
    return null;
  }

  const host = normaliseHost(String(source.host || ''));
  if (!host) {
    return null;
  }

  return {
    host,
    siteNamePath: String(source.siteNamePath || '').trim() || undefined,
    titlePath: String(source.titlePath || '').trim() || undefined,
    descriptionPath: String(source.descriptionPath || '').trim() || undefined,
    excerptPath: String(source.excerptPath || '').trim() || undefined,
    contentPath: String(source.contentPath || '').trim() || undefined,
    iconPath: String(source.iconPath || '').trim() || undefined,
    thumbnailPath: String(source.thumbnailPath || '').trim() || undefined,
    tagsPath: String(source.tagsPath || '').trim() || undefined,
    publishedAtPath: String(source.publishedAtPath || '').trim() || undefined,
    updatedAt: String(source.updatedAt || new Date().toISOString()),
  };
}

function readLocalCollectorSiteRule(host: string): CollectorSiteRuleItem | null {
  const raw = localStorage.getItem(buildCollectorRuleKey(host));
  if (!raw) {
    return null;
  }

  try {
    return mapCollectorSiteRule(JSON.parse(raw) as Partial<CollectorSiteRuleItem>);
  } catch {
    localStorage.removeItem(buildCollectorRuleKey(host));
    return null;
  }
}

function writeLocalCollectorSiteRule(rule: CollectorSiteRuleItem): CollectorSiteRuleItem {
  const payload = mapCollectorSiteRule(rule);
  if (!payload) {
    throw new Error('站点规则缺少有效域名');
  }
  localStorage.setItem(buildCollectorRuleKey(payload.host), JSON.stringify(payload));
  return payload;
}

export function createEmptyCollectorSiteRule(host: string): CollectorSiteRuleItem {
  return {
    host: normaliseHost(host),
    updatedAt: new Date().toISOString(),
  };
}

export function isCollectorSiteRuleEmpty(rule: Partial<CollectorSiteRuleItem> | null | undefined): boolean {
  if (!rule) {
    return true;
  }

  return !Object.values(RULE_FIELD_PATH_MAP).some((key) => String(rule[key] || '').trim());
}

export function getCollectorRuleFieldPathKey(field: CollectorRuleField): keyof CollectorSiteRuleItem {
  return RULE_FIELD_PATH_MAP[field];
}

export async function getCollectorSiteRule(urlOrHost: string): Promise<CollectorSiteRuleItem | null> {
  const host = normaliseHost(urlOrHost);
  if (!host) {
    return null;
  }

  if (hasAppMethod('GetCollectorSiteRule')) {
    try {
      const payload = await invokeApp<Partial<CollectorSiteRuleItem> | null>(
        'GetCollectorSiteRule',
        { url: urlOrHost } satisfies CollectorSiteRuleBridgeInput,
      );
      const mapped = mapCollectorSiteRule(payload);
      if (mapped) {
        writeLocalCollectorSiteRule(mapped);
      }
      return mapped;
    } catch {
      // Fall back to local storage when the bridge method is unavailable at runtime.
    }
  }

  return readLocalCollectorSiteRule(host);
}

export async function saveCollectorSiteRule(rule: CollectorSiteRuleItem): Promise<CollectorSiteRuleItem> {
  const payload = mapCollectorSiteRule(rule);
  if (!payload) {
    throw new Error('站点规则缺少有效域名');
  }

  if (hasAppMethod('SaveCollectorSiteRule')) {
    try {
      const saved = await invokeApp<Partial<CollectorSiteRuleItem>>(
        'SaveCollectorSiteRule',
        {
          host: payload.host,
          siteNamePath: payload.siteNamePath,
          titlePath: payload.titlePath,
          descriptionPath: payload.descriptionPath,
          excerptPath: payload.excerptPath,
          contentPath: payload.contentPath,
          iconPath: payload.iconPath,
          thumbnailPath: payload.thumbnailPath,
          tagsPath: payload.tagsPath,
          publishedAtPath: payload.publishedAtPath,
        } satisfies SaveCollectorSiteRuleBridgeInput,
      );
      const mapped = mapCollectorSiteRule(saved);
      if (mapped) {
        return writeLocalCollectorSiteRule(mapped);
      }
    } catch {
      // Fall back to local storage when the bridge method is unavailable at runtime.
    }
  }

  return writeLocalCollectorSiteRule({
    ...payload,
    updatedAt: new Date().toISOString(),
  });
}

export async function deleteCollectorSiteRule(urlOrHost: string): Promise<void> {
  const host = normaliseHost(urlOrHost);
  if (!host) {
    return;
  }

  localStorage.removeItem(buildCollectorRuleKey(host));

  if (hasAppMethod('DeleteCollectorSiteRule')) {
    try {
      await invokeApp<void>(
        'DeleteCollectorSiteRule',
        { url: urlOrHost } satisfies CollectorSiteRuleBridgeInput,
      );
    } catch {
      // Local storage has already been cleared.
    }
  }
}

function parseCollectorDocument(html: string): Document | null {
  if (!html.trim() || typeof DOMParser === 'undefined') {
    return null;
  }

  return new DOMParser().parseFromString(html, 'text/html');
}

function querySelectorSafe(documentNode: Document, selector: string): Element | null {
  try {
    return documentNode.querySelector(selector);
  } catch {
    return null;
  }
}

function querySelectorAllSafe(documentNode: Document, selector: string): Element[] {
  try {
    return Array.from(documentNode.querySelectorAll(selector));
  } catch {
    return [];
  }
}

function collapseWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

function resolveRuleUrl(value: string | null | undefined, baseUrl: string): string {
  const nextValue = String(value || '').trim();
  if (!nextValue) {
    return '';
  }
  if (nextValue.startsWith('data:') || nextValue.startsWith('blob:')) {
    return nextValue;
  }

  try {
    return new URL(nextValue, baseUrl).toString();
  } catch {
    return nextValue;
  }
}

function extractBackgroundImage(value: string): string {
  const match = value.match(/background(?:-image)?\s*:\s*[^;]*url\((['"]?)(.*?)\1\)/i);
  return match ? match[2].trim() : '';
}

function extractMediaFromElement(element: Element | null, baseUrl: string): string {
  if (!element) {
    return '';
  }

  const htmlElement = element as HTMLElement;
  const tagName = element.tagName.toLowerCase();
  const directAttributes = [
    'src',
    'href',
    'content',
    'poster',
    'data-src',
    'data-original',
    'data-thumb',
    'data-thumbnail',
    'data-cover',
    'data-image',
    'data-bg',
    'data-bg-src',
    'data-background',
  ];

  if (tagName === 'img' || tagName === 'source') {
    const srcset = element.getAttribute('srcset');
    if (srcset) {
      const firstCandidate = srcset.split(',')[0]?.trim().split(/\s+/, 2)[0] || '';
      const resolvedSrcset = resolveRuleUrl(firstCandidate, baseUrl);
      if (resolvedSrcset) {
        return resolvedSrcset;
      }
    }
  }

  for (const attribute of directAttributes) {
    const resolved = resolveRuleUrl(element.getAttribute(attribute), baseUrl);
    if (resolved) {
      return resolved;
    }
  }

  const inlineBackground = extractBackgroundImage(htmlElement.getAttribute('style') || '');
  if (inlineBackground) {
    return resolveRuleUrl(inlineBackground, baseUrl);
  }

  if (typeof window !== 'undefined' && htmlElement.ownerDocument?.defaultView) {
    const computedStyle = htmlElement.ownerDocument.defaultView.getComputedStyle(htmlElement);
    const computedBackground = extractBackgroundImage(computedStyle.cssText || computedStyle.backgroundImage || '');
    if (computedBackground) {
      return resolveRuleUrl(computedBackground, baseUrl);
    }
  }

  const nestedMedia = element.querySelector('img, source, video, [style*="background"]');
  if (nestedMedia) {
    return extractMediaFromElement(nestedMedia, baseUrl);
  }

  return '';
}

function extractTextBySelector(documentNode: Document, selector?: string): string {
  if (!selector?.trim()) {
    return '';
  }
  const element = querySelectorSafe(documentNode, selector);
  return element ? collapseWhitespace(element.textContent || '') : '';
}

export function resolveHtmlImageUrls(html: string, baseUrl: string): string {
  if (!html.trim() || typeof DOMParser === 'undefined') {
    return html;
  }

  const documentNode = new DOMParser().parseFromString(html, 'text/html');
  const images = Array.from(documentNode.body.querySelectorAll('img'));
  for (const image of images) {
    const src = image.getAttribute('src');
    if (src) {
      image.setAttribute('src', resolveRuleUrl(src, baseUrl));
    }
    const srcset = image.getAttribute('srcset');
    if (srcset) {
      image.setAttribute('srcset', normalisePickerSrcsetValue(srcset, baseUrl));
    }
  }
  return documentNode.body.innerHTML;
}

function normalisePickerSrcsetValue(value: string, baseUrl: string): string {
  return value
    .split(',')
    .map((candidate) => candidate.trim())
    .filter(Boolean)
    .map((candidate) => {
      const [url, ...descriptors] = candidate.split(/\s+/);
      const resolvedUrl = resolveRuleUrl(url || '', baseUrl);
      return descriptors.length > 0 ? `${resolvedUrl} ${descriptors.join(' ')}` : resolvedUrl;
    })
    .join(', ');
}

function extractHtmlBySelector(documentNode: Document, baseUrl: string, selector?: string): { html: string; text: string; images: CollectorImageItem[] } {
  if (!selector?.trim()) {
    return { html: '', text: '', images: [] };
  }

  const element = querySelectorSafe(documentNode, selector) as HTMLElement | null;
  if (!element) {
    return { html: '', text: '', images: [] };
  }

  const html = resolveHtmlImageUrls(element.innerHTML.trim(), baseUrl);
  const text = collapseWhitespace(element.textContent || '');
  const images = Array.from(element.querySelectorAll('img'))
    .map((item) => ({
      url: resolveRuleUrl(item.getAttribute('src') || item.getAttribute('data-src'), baseUrl),
      alt: collapseWhitespace(item.getAttribute('alt') || ''),
    }))
    .filter((item) => Boolean(item.url));

  return { html, text, images };
}

function extractMediaBySelector(documentNode: Document, baseUrl: string, selector?: string): string {
  if (!selector?.trim()) {
    return '';
  }
  return extractMediaFromElement(querySelectorSafe(documentNode, selector), baseUrl);
}

function extractTagsBySelector(documentNode: Document, selector?: string): string[] {
  if (!selector?.trim()) {
    return [];
  }

  const tags = querySelectorAllSafe(documentNode, selector)
    .map((element) => collapseWhitespace(element.textContent || ''))
    .filter((item) => item.length > 0 && item.length <= 40);

  return Array.from(new Set(tags));
}

function normaliseStaticHTMLImages(documentNode: Document, baseUrl: string) {
  const lazyAttrs = ['data-src', 'data-original', 'data-thumb', 'data-thumbnail', 'data-cover', 'data-image', 'data-bg', 'data-bg-src', 'data-background'];

  const images = Array.from(documentNode.querySelectorAll<HTMLImageElement>('img'));
  for (const img of images) {
    const existingSrc = img.getAttribute('src');
    if (existingSrc && !existingSrc.startsWith('data:') && !existingSrc.startsWith('blob:')) {
      continue;
    }
    for (const attr of lazyAttrs) {
      const val = img.getAttribute(attr);
      if (val) {
        img.setAttribute('src', resolveRuleUrl(val, baseUrl));
        break;
      }
    }
  }

  const sources = Array.from(documentNode.querySelectorAll<HTMLSourceElement>('source'));
  for (const source of sources) {
    const srcset = source.getAttribute('srcset') || source.getAttribute('data-srcset');
    if (srcset && !source.getAttribute('src')) {
      const first = srcset.split(',')[0].trim().split(/\s+/)[0];
      if (first) {
        source.setAttribute('src', resolveRuleUrl(first, baseUrl));
      }
    }
  }
}

export function applyCollectorSiteRule(
  payload: CollectWebPageResult,
  rule: CollectorSiteRuleItem | null,
): CollectWebPageResult {
  const hasBrowserHtml = Boolean(payload.browserPreviewHtml?.trim());
  const sourceHtml = String(hasBrowserHtml ? payload.browserPreviewHtml : payload.sourceHtml || '').trim();
  if (!rule || !sourceHtml) {
    return payload;
  }

  const documentNode = parseCollectorDocument(sourceHtml);
  if (!documentNode) {
    return payload;
  }

  if (!hasBrowserHtml) {
    normaliseStaticHTMLImages(documentNode, payload.finalUrl);
  }

  if (!documentNode.URL || documentNode.URL === 'about:blank') {
    const baseElement = documentNode.createElement('base');
    baseElement.href = payload.finalUrl;
    const head = documentNode.head || documentNode.querySelector('head');
    if (head) {
      head.prepend(baseElement);
    }
  }

  const nextPayload: CollectWebPageResult = {
    ...payload,
    keywords: Array.isArray(payload.keywords) ? [...payload.keywords] : [],
    seoKeywords: Array.isArray(payload.seoKeywords) ? [...payload.seoKeywords] : [],
    suggestedTags: Array.isArray(payload.suggestedTags) ? [...payload.suggestedTags] : [],
    images: Array.isArray(payload.images) ? payload.images.map((item) => ({ ...item })) : [],
  };

  const siteName = extractTextBySelector(documentNode, rule.siteNamePath);
  if (siteName) {
    nextPayload.siteName = siteName;
  }

  const title = extractTextBySelector(documentNode, rule.titlePath);
  if (title) {
    nextPayload.title = title;
  }

  const description = extractTextBySelector(documentNode, rule.descriptionPath);
  if (description) {
    nextPayload.description = description;
  }

  const excerpt = extractTextBySelector(documentNode, rule.excerptPath);
  if (excerpt) {
    nextPayload.excerpt = excerpt;
  }

  const iconUrl = extractMediaBySelector(documentNode, payload.finalUrl, rule.iconPath);
  if (iconUrl) {
    nextPayload.iconUrl = iconUrl;
  }

  const thumbnailUrl = extractMediaBySelector(documentNode, payload.finalUrl, rule.thumbnailPath);
  if (thumbnailUrl) {
    if (hasBrowserHtml || !payload.thumbnailUrl) {
      nextPayload.thumbnailUrl = thumbnailUrl;
    }
  }

  const content = extractHtmlBySelector(documentNode, payload.finalUrl, rule.contentPath);
  if (content.html) {
    nextPayload.contentHtml = content.html;
    nextPayload.contentText = content.text;
    if (hasBrowserHtml) {
      nextPayload.images = content.images;
    } else {
      const existingUrls = new Set(nextPayload.images.map((img) => img.url));
      for (const img of content.images) {
        if (!existingUrls.has(img.url)) {
          nextPayload.images.push(img);
          existingUrls.add(img.url);
        }
      }
    }
  }

  const tags = extractTagsBySelector(documentNode, rule.tagsPath);
  if (tags.length) {
    nextPayload.suggestedTags = tags;
  }

  const publishedAt = extractTextBySelector(documentNode, rule.publishedAtPath);
  if (publishedAt) {
    nextPayload.publishedAt = publishedAt;
  }

  return nextPayload;
}
