import { CollectWebPage } from '../../wailsjs/go/main/App';
import { hasAppMethod, invokeApp } from './bridge';

export interface CollectorImageItem {
  url: string;
  alt?: string;
}

export interface CollectWebPageResult {
  requestedUrl: string;
  finalUrl: string;
  canonicalUrl?: string;
  officialUrl?: string;
  host: string;
  sourceHtml?: string;
  browserPreviewHtml?: string;
  siteName?: string;
  title: string;
  description?: string;
  excerpt?: string;
  iconUrl?: string;
  thumbnailUrl?: string;
  contentHtml: string;
  contentText: string;
  keywords: string[];
  seoTitle?: string;
  seoDescription?: string;
  seoKeywords: string[];
  suggestedTags: string[];
  images: CollectorImageItem[];
  publishedAt?: string;
  fetchedAt: string;
}

interface CollectWebPageInput {
  url: string;
  siteId?: number;
  tenantId?: number;
  useBrowserRender?: boolean;
}

interface CollectWebPageOptions {
  siteId?: number;
  tenantId?: number;
  useBrowserRender?: boolean;
}

export interface BrowserRenderedPreviewResult {
  requestedUrl: string;
  finalUrl: string;
  host: string;
  browser: string;
  html: string;
  renderedAt: string;
}

export async function collectWebPage(url: string, options: CollectWebPageOptions = {}): Promise<CollectWebPageResult> {
  if (!hasAppMethod('CollectWebPage')) {
    throw new Error('当前桌面端尚未绑定网页采集能力');
  }

  const payload = await CollectWebPage({
    url,
    siteId: typeof options.siteId === 'number' ? options.siteId : undefined,
    tenantId: typeof options.tenantId === 'number' ? options.tenantId : undefined,
    useBrowserRender: options.useBrowserRender === true,
  } satisfies CollectWebPageInput) as unknown as Partial<CollectWebPageResult>;

  return {
    requestedUrl: String(payload?.requestedUrl || url),
    finalUrl: String(payload?.finalUrl || url),
    canonicalUrl: payload?.canonicalUrl,
    officialUrl: payload?.officialUrl,
    host: String(payload?.host || ''),
    sourceHtml: typeof payload?.sourceHtml === 'string' ? payload.sourceHtml : '',
    browserPreviewHtml: typeof payload?.browserPreviewHtml === 'string' ? payload.browserPreviewHtml : '',
    siteName: payload?.siteName,
    title: String(payload?.title || ''),
    description: payload?.description,
    excerpt: payload?.excerpt,
    iconUrl: payload?.iconUrl,
    thumbnailUrl: payload?.thumbnailUrl,
    contentHtml: String(payload?.contentHtml || ''),
    contentText: String(payload?.contentText || ''),
    keywords: Array.isArray(payload?.keywords) ? payload.keywords.filter((item): item is string => typeof item === 'string') : [],
    seoTitle: payload?.seoTitle,
    seoDescription: payload?.seoDescription,
    seoKeywords: Array.isArray(payload?.seoKeywords) ? payload.seoKeywords.filter((item): item is string => typeof item === 'string') : [],
    suggestedTags: Array.isArray(payload?.suggestedTags) ? payload.suggestedTags.filter((item): item is string => typeof item === 'string') : [],
    images: Array.isArray(payload?.images)
      ? payload.images.filter((item): item is CollectorImageItem => Boolean(item && typeof item.url === 'string'))
      : [],
    publishedAt: payload?.publishedAt,
    fetchedAt: String(payload?.fetchedAt || new Date().toISOString()),
  };
}

export async function renderWebPagePreview(url: string, options: CollectWebPageOptions = {}): Promise<BrowserRenderedPreviewResult> {
  if (!hasAppMethod('RenderWebPagePreview')) {
    throw new Error('当前桌面端尚未绑定浏览器渲染预览能力');
  }

  const payload = await invokeApp<Partial<BrowserRenderedPreviewResult>>(
    'RenderWebPagePreview',
    {
      url,
      siteId: typeof options.siteId === 'number' ? options.siteId : undefined,
      tenantId: typeof options.tenantId === 'number' ? options.tenantId : undefined,
    } satisfies CollectWebPageInput,
  );

  return {
    requestedUrl: String(payload?.requestedUrl || url),
    finalUrl: String(payload?.finalUrl || url),
    host: String(payload?.host || ''),
    browser: String(payload?.browser || ''),
    html: String(payload?.html || ''),
    renderedAt: String(payload?.renderedAt || new Date().toISOString()),
  };
}

export interface ScanFilterRule {
  field: string;
  operator: string;
  value: string;
}

export interface ScanSiteLinksInput {
  url: string;
  maxLinks: number;
  scanSitemap: boolean;
  filterRules: ScanFilterRule[];
  siteId?: number;
  tenantId?: number;
}

export interface SiteLinkItem {
  url: string;
  title: string;
  source?: string;
}

export interface ScanSiteLinksResult {
  requestedUrl: string;
  finalUrl: string;
  host: string;
  siteName?: string;
  title: string;
  links: SiteLinkItem[];
  pageHtmlCount: number;
  sitemapUrlCount: number;
  sitemapSources?: string[];
  scannedAt: string;
}

export async function scanSiteLinks(input: ScanSiteLinksInput): Promise<ScanSiteLinksResult> {
  if (!hasAppMethod('ScanSiteLinks')) {
    throw new Error('当前桌面端尚未绑定全站扫描能力');
  }

  const payload = await invokeApp<Partial<ScanSiteLinksResult>>('ScanSiteLinks', input);

  return {
    requestedUrl: String(payload?.requestedUrl || input.url),
    finalUrl: String(payload?.finalUrl || input.url),
    host: String(payload?.host || ''),
    siteName: payload?.siteName,
    title: String(payload?.title || ''),
    links: Array.isArray(payload?.links)
      ? payload.links.filter((item): item is SiteLinkItem => Boolean(item && typeof item.url === 'string'))
      : [],
    pageHtmlCount: typeof payload?.pageHtmlCount === 'number' ? payload.pageHtmlCount : 0,
    sitemapUrlCount: typeof payload?.sitemapUrlCount === 'number' ? payload.sitemapUrlCount : 0,
    sitemapSources: Array.isArray(payload?.sitemapSources) ? payload.sitemapSources : [],
    scannedAt: String(payload?.scannedAt || new Date().toISOString()),
  };
}
