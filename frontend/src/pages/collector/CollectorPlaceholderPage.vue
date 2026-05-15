<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { Message } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import SiteTenantContextBar from '../../components/SiteTenantContextBar.vue';
import {
  collectWebPage,
  scanSiteLinks,
  type CollectWebPageResult,
  type ScanSiteLinksResult,
  type SiteLinkItem,
  type ScanFilterRule,
} from '../../services/collector.service';
import {
  applyCollectorSiteRule,
  createEmptyCollectorSiteRule,
  deleteCollectorSiteRule,
  getCollectorRuleFieldPathKey,
  getCollectorSiteRule,
  isCollectorSiteRuleEmpty,
  saveCollectorSiteRule,
  resolveHtmlImageUrls,
  type CollectorRuleField,
  type CollectorSiteRuleItem,
} from '../../services/collector-site-rule.service';
import {
  loadCollectorScanSettings,
  saveCollectorScanSettings,
  deleteCollectorScanSettings,
  extractScanSettingsHost,
} from '../../services/collector-scan-settings.service';
import {
  saveCollectedUrl,
  getCollectedUrlSet,
  loadCollectedUrls,
  deleteCollectedUrl,
  clearCollectedHistory,
  type CollectedHistoryItem,
} from '../../services/collector-history.service';
import { buildLocalDraftKey, saveLocalDraft } from '../../services/local-draft.service';
import { useAuthStore } from '../../stores/auth.store';
import { useSiteStore } from '../../stores/site.store';
import { useCollectorStore, type BatchCollectItem } from '../../stores/collector.store';
import { initialiseDesktopContext } from '../../utils/desktop-context';
import { isValidHttpUrl, normaliseExternalHttpUrl } from '../../utils/url';

const router = useRouter();
const siteStore = useSiteStore();
const authStore = useAuthStore();
const collectorStore = useCollectorStore();
const {
  batchUrlsText,
  batchResults,
  batchProgress,
  batchDraftType,
} = storeToRefs(collectorStore);

interface PickerFieldOption {
  value: CollectorRuleField;
  label: string;
  description: string;
}

interface QuickImageChoice {
  url: string;
  label: string;
  type: 'icon' | 'thumbnail' | 'image';
}

interface PickerSelectionSummary {
  field: CollectorRuleField;
  fieldLabel: string;
  selector: string;
  matchCount: number;
  targetTag: string;
  previewValue: string;
}

const PICKER_FIELD_OPTIONS: PickerFieldOption[] = [
  { value: 'title', label: '标题', description: '点页面标题区域' },
  { value: 'description', label: '摘要', description: '点页面简介或摘要段落' },
  { value: 'excerpt', label: '导语', description: '点文章摘要或首段导语' },
  { value: 'content', label: '正文', description: '点文章正文容器' },
  { value: 'thumbnail', label: '缩略图', description: '点封面图或头图' },
  { value: 'icon', label: '图标', description: '点站点 logo 或小图标' },
  { value: 'tags', label: '标签', description: '点任意一个标签' },
  { value: 'publishedAt', label: '时间', description: '点发布时间文本' },
  { value: 'siteName', label: '站点名', description: '点站点名称区域' },
];

const sourceUrl = ref('');
const collecting = ref(false);
const draftGenerating = ref<'tool' | 'article' | null>(null);
const previewMode = ref<'rendered' | 'source' | 'picker'>('rendered');
const baseResult = ref<CollectWebPageResult | null>(null);
const result = ref<CollectWebPageResult | null>(null);
const draftMappingTab = ref<'tool' | 'article'>('tool');
const syncingTenants = ref(false);
const pickerField = ref<CollectorRuleField | ''>('');
const pickerStatus = ref('切到"点选规则"后，先选字段，再直接点击页面里的目标区域。');
const pickerLastValue = ref('');
const currentRule = ref<CollectorSiteRuleItem | null>(null);
const pickerFrameRef = ref<HTMLIFrameElement | null>(null);
const pickerSelectionSummary = ref<PickerSelectionSummary | null>(null);
const pickerBrokenImageCount = ref(0);
const collectMode = ref<'single' | 'batch' | 'scan'>('single');
const useBrowserRender = ref(false);
const skipCollectedUrls = ref(true);
const skippedUrlCount = ref(0);
const showHistoryPanel = ref(false);
const historyVersion = ref(0);
const scanning = ref(false);
const scanUrl = ref('');
const scanResult = ref<ScanSiteLinksResult | null>(null);
const scanMaxLinks = ref(50);
const scanSitemap = ref(true);
const scanFilterRules = ref<ScanFilterRule[]>([]);
const scannedSelectedLinks = ref<Set<string>>(new Set());

const SCAN_OPERATOR_OPTIONS = [
  { value: 'contains', label: '包含' },
  { value: 'not_contains', label: '不包含' },
  { value: 'prefix', label: '前缀' },
  { value: 'suffix', label: '后缀' },
  { value: 'path_prefix', label: '路径前缀' },
  { value: 'path_contains', label: '路径包含' },
];

const SCAN_FIELD_OPTIONS = [
  { value: 'url', label: 'URL' },
  { value: 'title', label: '标题' },
  { value: 'all', label: '全部' },
];

let removePickerListeners: (() => void) | null = null;

interface ToolDraftMapping {
  title: string;
  url: string;
  website: string;
  icon: string;
  thumbnail: string;
  description: string;
  content: string;
  featuresText: string;
  tagsText: string;
  metaTitle: string;
  metaKeywords: string;
  metaDescription: string;
}

interface ArticleDraftMapping {
  title: string;
  thumbnail: string;
  excerpt: string;
  content: string;
  publishedAt: string;
  tagsText: string;
  metaTitle: string;
  metaKeywords: string;
  metaDescription: string;
}

const toolMapping = reactive<ToolDraftMapping>({
  title: '',
  url: '',
  website: '',
  icon: '',
  thumbnail: '',
  description: '',
  content: '',
  featuresText: '',
  tagsText: '',
  metaTitle: '',
  metaKeywords: '',
  metaDescription: '',
});

const articleMapping = reactive<ArticleDraftMapping>({
  title: '',
  thumbnail: '',
  excerpt: '',
  content: '',
  publishedAt: '',
  tagsText: '',
  metaTitle: '',
  metaKeywords: '',
  metaDescription: '',
});

const currentSite = computed(() => siteStore.currentSite);
const currentTenant = computed(() => siteStore.currentTenant);
const filteredTenants = computed(() => siteStore.filteredTenants);
const currentSiteId = computed(() => {
  const id = siteStore.currentSite?.id;
  return typeof id === 'number' ? id : null;
});
const currentTenantId = computed(() => {
  const id = siteStore.currentTenant?.id;
  return typeof id === 'number' ? id : null;
});
const resultImages = computed(() => (Array.isArray(result.value?.images) ? result.value.images : []));
const resultKeywords = computed(() => (Array.isArray(result.value?.keywords) ? result.value.keywords : []));
const resultSeoKeywords = computed(() => (Array.isArray(result.value?.seoKeywords) ? result.value.seoKeywords : []));
const resultSuggestedTags = computed(() => (Array.isArray(result.value?.suggestedTags) ? result.value.suggestedTags : []));
const hasResult = computed(() => Boolean(result.value));
const imageCount = computed(() => resultImages.value.length);
const canCollect = computed(() => Boolean(currentSiteId.value && currentTenantId.value));
const canGenerateDraft = computed(() => Boolean(currentTenantId.value && result.value));
const contentLength = computed(() => result.value?.contentText.length || 0);
const pickerEnabled = computed(() => Boolean(baseResult.value?.finalUrl?.trim()));
const activeRuleHost = computed(() => currentRule.value?.host || extractHost(result.value?.finalUrl || baseResult.value?.finalUrl || sourceUrl.value));
const hasCurrentRule = computed(() => Boolean(currentRule.value && !isCollectorSiteRuleEmpty(currentRule.value)));
const hasScanSettings = computed(() => Boolean(currentScanHost && loadCollectorScanSettings(currentScanHost)));
const pickerPreviewHtml = computed(() => buildPickerFrameHtml(baseResult.value));
const quickImageChoices = computed<QuickImageChoice[]>(() => {
  const choices: QuickImageChoice[] = [];
  const seen = new Set<string>();

  const pushChoice = (url: string | undefined, label: string, type: QuickImageChoice['type']) => {
    const nextUrl = String(url || '').trim();
    if (!nextUrl || seen.has(nextUrl)) {
      return;
    }
    seen.add(nextUrl);
    choices.push({ url: nextUrl, label, type });
  };

  pushChoice(result.value?.thumbnailUrl, '当前缩略图', 'thumbnail');
  pushChoice(result.value?.iconUrl, '当前图标', 'icon');
  resultImages.value.forEach((image, index) => {
  pushChoice(image.url, image.alt || `正文图 ${index + 1}`, 'image');
  });

  return choices;
});
const currentRuleFieldEntries = computed(() => {
  if (!currentRule.value) {
    return [];
  }

  return PICKER_FIELD_OPTIONS
    .map((option) => ({
      label: option.label,
      path: String(currentRule.value?.[getCollectorRuleFieldPathKey(option.value)] || '').trim(),
    }))
    .filter((item) => item.path);
});
const historyPanelHost = computed(() => {
  const scanHost = extractScanSettingsHost(scanUrl.value);
  if (scanHost) return scanHost;
  if (batchResults.value.length) {
    try {
      return new URL(batchResults.value[0].url).hostname.toLowerCase();
    } catch { /* ignore */ }
  }
  return '';
});
const collectedHistoryItems = computed<CollectedHistoryItem[]>(() => {
  void historyVersion.value;
  if (!historyPanelHost.value) return [];
  return loadCollectedUrls(historyPanelHost.value);
});
const collectedUrlSet = computed<Set<string>>(() => {
  void historyVersion.value;
  if (!historyPanelHost.value) return new Set();
  return getCollectedUrlSet(historyPanelHost.value);
});
const pickerFieldDescription = computed(() => {
  const matched = PICKER_FIELD_OPTIONS.find((option) => option.value === pickerField.value);
  return matched?.description || '请选择一个字段后，再点击原页面对应内容。';
});

type SelectValue = string | number | boolean | Record<string, unknown> | Array<string | number | boolean | Record<string, unknown>>;

function handleSiteChange(value: SelectValue) {
  if (Array.isArray(value) || (typeof value !== 'number' && typeof value !== 'string')) {
    return;
  }
  siteStore.selectSite(Number(value));
  handleReset();
}

function handleTenantChange(value: SelectValue) {
  if (Array.isArray(value) || (typeof value !== 'number' && typeof value !== 'string')) {
    return;
  }
  siteStore.selectTenant(Number(value));
}

async function syncCurrentSiteTenants() {
  if (!currentSiteId.value) {
    Message.warning('请先选择站点');
    return;
  }

  syncingTenants.value = true;
  try {
    const tenants = await siteStore.syncSiteTenants(currentSiteId.value);
    if (!tenants.length) {
      Message.warning('当前站点未发现可用租户');
      return;
    }
    Message.success(`已同步 ${tenants.length} 个租户`);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '同步租户失败');
  } finally {

function buildCollectorDraftId(contentType: 'tool' | 'article'): string {
  return `local:collector:${contentType}:${Date.now()}`;
}

function escapeHtmlAttribute(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function extractHost(value: string | undefined | null): string {
  const nextValue = String(value || '').trim();
  if (!nextValue) {
    return '';
  }

  try {
    return new URL(nextValue).hostname.toLowerCase();
  } catch {
    return '';
  }
}

function resolvePickerAssetUrl(value: string | null | undefined, baseUrl: string): string {
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

function normalisePickerSrcset(value: string | null | undefined, baseUrl: string): string {
  return String(value || '')
    .split(',')
    .map((candidate) => candidate.trim())
    .filter(Boolean)
    .map((candidate) => {
      const [url, descriptor] = candidate.split(/\s+/, 2);
      const resolvedUrl = resolvePickerAssetUrl(url || '', baseUrl);
      return descriptor ? `${resolvedUrl} ${descriptor}` : resolvedUrl;
    })
    .join(', ');
}

function isPlaceholderImageSrc(src: string | null | undefined): boolean {
  if (!src) {
    return true;
  }
  const trimmed = src.trim();
  if (!trimmed) {
    return true;
  }
  if (trimmed.startsWith('data:')) {
    return true;
  }
  if (trimmed.startsWith('blob:')) {
    return true;
  }
  return false;
}

function normalisePickerPreviewHtml(html: string, baseUrl: string): string {
  if (!html.trim() || typeof DOMParser === 'undefined') {
    return html;
  }

  const documentNode = new DOMParser().parseFromString(html, 'text/html');
  const lazyImageAttributes = [
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

  documentNode.querySelectorAll<HTMLElement>('[src], [href], [poster], [srcset], [style], img, source, video, link').forEach((element) => {
    const rawSrc = element.getAttribute('src');

    if (element.tagName === 'IMG') {
      const isPlaceholder = isPlaceholderImageSrc(rawSrc);
      const lazySrc = isPlaceholder ? lazyImageAttributes.map((attribute) => element.getAttribute(attribute)).find(Boolean) : null;
      const effectiveSrc = lazySrc || rawSrc;
      if (effectiveSrc) {
        element.setAttribute('src', resolvePickerAssetUrl(effectiveSrc, baseUrl));
      }
    } else {
      const promotedSrc = rawSrc || lazyImageAttributes.map((attribute) => element.getAttribute(attribute)).find(Boolean) || '';
      if (promotedSrc) {
        element.setAttribute('src', resolvePickerAssetUrl(promotedSrc, baseUrl));
      }
    }

    const href = element.getAttribute('href');
    if (href) {
      element.setAttribute('href', resolvePickerAssetUrl(href, baseUrl));
    }

    const poster = element.getAttribute('poster');
    if (poster) {
      element.setAttribute('poster', resolvePickerAssetUrl(poster, baseUrl));
    }

    const srcset = element.getAttribute('srcset') || element.getAttribute('data-srcset');
    if (srcset) {
      element.setAttribute('srcset', normalisePickerSrcset(srcset, baseUrl));
    }

    if (element.tagName === 'IMG') {
      element.removeAttribute('loading');
      element.removeAttribute('decoding');
      element.removeAttribute('lazy');
    }

    const style = element.getAttribute('style');
    if (style && /url\(/i.test(style)) {
      const nextStyle = style.replace(/url\((['"]?)(.*?)\1\)/gi, (_match, quote, assetUrl) => (
        `url(${quote}${resolvePickerAssetUrl(assetUrl, baseUrl)}${quote})`
      ));
      element.setAttribute('style', nextStyle);
    }
  });

  return documentNode.documentElement.outerHTML || html;
}

function resolvePickerSourceHtml(payload: CollectWebPageResult | null): string {
  if (!payload) {
    return '';
  }

  return String(payload.browserPreviewHtml || payload.sourceHtml || '').trim();
}

function buildPickerFrameHtml(payload: CollectWebPageResult | null): string {
  const html = normalisePickerPreviewHtml(
    resolvePickerSourceHtml(payload),
    String(payload?.finalUrl || ''),
  ).trim();
  if (!html) {
    return '';
  }

  const baseTag = `<base href="${escapeHtmlAttribute(String(payload?.finalUrl || ''))}">`;
  const helperStyle = `
    <style id="zq-picker-style">
      html { scroll-behavior: smooth; }
      body { margin: 0 auto; max-width: 960px; padding: 20px; box-sizing: border-box; }
      img, video, iframe { max-width: 100%; height: auto; }
      [data-zq-image-broken="1"] { outline: 2px dashed #f53f3f !important; background: repeating-linear-gradient(-45deg, #fef2f2, #fef2f2 8px, #fde2e2 8px, #fde2e2 16px) !important; min-height: 48px; }
      [data-zq-picker-hover="1"] { outline: 2px solid #ff7d00 !important; cursor: crosshair !important; }
      [data-zq-picker-selected="1"] { outline: 2px solid #165dff !important; }
    </style>
  `.trim();

  if (/<head[\s>]/i.test(html)) {
    return html.replace(/<\/head>/i, `${baseTag}${helperStyle}</head>`);
  }

  return `<!doctype html><html><head>${baseTag}${helperStyle}</head><body>${html}</body></html>`;
}

function buildCollectedContentHtml(payload: CollectWebPageResult | null): string {
  if (!payload) {
    return '';
  }

  const html = String(payload.contentHtml || '').trim();
  return resolveHtmlImageUrls(html, payload.finalUrl);
}

function firstContentImageUrl(payload: CollectWebPageResult | null): string {
  if (!payload || !Array.isArray(payload.images)) {
    return '';
  }

  const ignoredAltPatterns = [
    '备案', 'beian', 'icp', 'icp备案', '公安备案',
    'logo', 'icon', 'favicon', '头像', 'avatar',
    '二维码', 'qrcode', '小程序码',
    '分享', 'share', '广告', 'ad',
    'banner', '导航', 'nav', 'menu',
    'loading', 'spinner', 'placeholder',
    'screenshot', '截图',
    'email', '邮件', '电话', 'phone',
  ];

  const ignoredUrlPatterns = [
    'favicon', 'beian', 'icp', 'badge',
    'logo.', 'icon.', 'avatar.',
    'sprite', 'spacer', 'blank',
    'pixel', 'tracking', 'analytics',
    '1x1', 'transparent',
  ];

  for (const image of payload.images) {
    const url = (image.url || '').toLowerCase();
    const alt = (image.alt || '').toLowerCase();

    const hasIgnoredAlt = ignoredAltPatterns.some((p) => alt.includes(p));
    if (hasIgnoredAlt) {
      continue;
    }

    const hasIgnoredUrl = ignoredUrlPatterns.some((p) => url.includes(p));
    if (hasIgnoredUrl) {
      continue;
    }

    if (url) {
      return image.url;
    }
  }

  return '';
}

function resolveCollectedThumbnail(payload: CollectWebPageResult | null): string {
  if (!payload) {
    return '';
  }

  const ignoredThumbnailPatterns = [
    'beian', 'icp', 'badge', 'favicon',
    'logo.', 'icon.', 'avatar.',
    'sprite', '1x1', 'transparent',
    '备案', '图标', '徽标',
  ];

  const thumbnailUrl = String(payload.thumbnailUrl || '').trim();
  const lowerThumb = thumbnailUrl.toLowerCase();
  const hasIgnoredPattern = ignoredThumbnailPatterns.some((p) => lowerThumb.includes(p));

  if (thumbnailUrl && !hasIgnoredPattern) {
    return thumbnailUrl;
  }

  return firstContentImageUrl(payload);
}

function resolveCollectedOfficialUrl(payload: CollectWebPageResult | null): string {
  if (!payload) {
    return '';
  }
  return String(payload.officialUrl || '').trim();
}

const previewHtml = computed(() => buildCollectedContentHtml(result.value));

function buildRulePreviewValue(payload: CollectWebPageResult | null, field: CollectorRuleField): string {
  if (!payload) {
    return '';
  }

  switch (field) {
    case 'title':
      return payload.title || '';
    case 'description':
      return payload.description || '';
    case 'excerpt':
      return payload.excerpt || '';
    case 'content':
      return payload.contentText.slice(0, 120);
    case 'thumbnail':
      return payload.thumbnailUrl || '';
    case 'icon':
      return payload.iconUrl || '';
    case 'tags':
      return (Array.isArray(payload.suggestedTags) ? payload.suggestedTags : []).join(', ');
    case 'publishedAt':
      return payload.publishedAt || '';
    case 'siteName':
      return payload.siteName || '';
    default:
      return '';
  }
}

function getPickerFieldLabel(field: CollectorRuleField): string {
  return PICKER_FIELD_OPTIONS.find((option) => option.value === field)?.label || field;
}

function getPickerSelectionMatchCount(documentNode: Document, selector: string): number {
  try {
    return documentNode.querySelectorAll(selector).length;
  } catch {
    return 0;
  }
}

function getDraftMappingAnchorId(field: CollectorRuleField): string {
  if (field === 'icon') {
    draftMappingTab.value = 'tool';
    return 'collector-tool-icon';
  }
  if (field === 'publishedAt') {
    draftMappingTab.value = 'article';
    return 'collector-article-publishedAt';
  }
  if (field === 'siteName') {
    draftMappingTab.value = 'tool';
    return 'collector-tool-website';
  }

  if (draftMappingTab.value === 'article') {
    if (field === 'description' || field === 'excerpt') {
      return 'collector-article-excerpt';
    }
    if (field === 'thumbnail') {
      return 'collector-article-thumbnail';
    }
    if (field === 'tags') {
      return 'collector-article-tags';
    }
    if (field === 'content') {
      return 'collector-article-content';
    }
    return 'collector-article-title';
  }

  if (field === 'description' || field === 'excerpt') {
    return 'collector-tool-description';
  }
  if (field === 'thumbnail') {
    return 'collector-tool-thumbnail';
  }
  if (field === 'tags') {
    return 'collector-tool-tags';
  }
  if (field === 'content') {
    return 'collector-tool-content';
  }
  return 'collector-tool-title';
}

function scrollToDraftMappingField(field: CollectorRuleField) {
  const targetId = getDraftMappingAnchorId(field);
  void nextTick(() => {
    document.getElementById(targetId)?.scrollIntoView({
      behavior: 'smooth',
      block: 'center',
    });
  });
}

function applyQuickImageSelection(choice: QuickImageChoice) {
  if (!result.value) {
    Message.warning('请先完成采集');
    return;
  }

  if (pickerField.value === 'icon') {
    result.value = {
      ...result.value,
      iconUrl: choice.url,
    };
    toolMapping.icon = choice.url;
    pickerLastValue.value = choice.url;
    pickerStatus.value = `已快捷选中图标：${choice.label}。本次生成工具草稿会直接使用这张图。`;
    pickerSelectionSummary.value = {
      field: 'icon',
      fieldLabel: getPickerFieldLabel('icon'),
      selector: '快捷图片选择',
      matchCount: 1,
      targetTag: 'img',
      previewValue: choice.url,
    };
    scrollToDraftMappingField('icon');
    Message.success('已设为本次采集图标');
    return;
  }

  result.value = {
    ...result.value,
    thumbnailUrl: choice.url,
  };
  toolMapping.thumbnail = choice.url;
  articleMapping.thumbnail = choice.url;
  pickerLastValue.value = choice.url;
  pickerStatus.value = `已快捷选中缩略图：${choice.label}。本次生成工具和文章草稿都会直接使用这张图。`;
  pickerSelectionSummary.value = {
    field: 'thumbnail',
    fieldLabel: getPickerFieldLabel('thumbnail'),
    selector: '快捷图片选择',
    matchCount: 1,
    targetTag: 'img',
    previewValue: choice.url,
  };
  scrollToDraftMappingField('thumbnail');
  Message.success('已设为本次采集缩略图');
}

function resetMappings(payload: CollectWebPageResult | null) {
  Object.assign(toolMapping, buildDefaultToolMapping(payload));
  Object.assign(articleMapping, buildDefaultArticleMapping(payload));
}

function buildDefaultToolMapping(payload: CollectWebPageResult | null): ToolDraftMapping {
  if (!payload) {
    return {
      title: '',
      url: '',
      website: '',
      icon: '',
      thumbnail: '',
      description: '',
      content: '',
      featuresText: '',
      tagsText: '',
      metaTitle: '',
      metaKeywords: '',
      metaDescription: '',
    };
  }

  const keywords = Array.isArray(payload.seoKeywords)
    ? payload.seoKeywords
    : Array.isArray(payload.keywords) ? payload.keywords : [];
  const tags = Array.isArray(payload.suggestedTags) ? payload.suggestedTags : [];
  const thumbnail = payload.thumbnailUrl || firstContentImageUrl(payload) || '';
  const officialUrl = resolveCollectedOfficialUrl(payload);
  const website = officialUrl ? new URL(officialUrl).origin : '';
  const contentHtml = buildCollectedContentHtml(payload);

  return {
    title: payload.title || '',
    url: officialUrl || payload.finalUrl,
    website,
    icon: String(payload.iconUrl || '').trim(),
    thumbnail,
    description: payload.description || payload.excerpt || '',
    content: contentHtml,
    featuresText: keywords.join('\n'),
    tagsText: tags.join(', '),
    metaTitle: payload.seoTitle || '',
    metaKeywords: keywords.join(', '),
    metaDescription: payload.seoDescription || '',
  };
}

function buildDefaultArticleMapping(payload: CollectWebPageResult | null): ArticleDraftMapping {
  if (!payload) {
    return {
      title: '',
      thumbnail: '',
      excerpt: '',
      content: '',
      publishedAt: '',
      tagsText: '',
      metaTitle: '',
      metaKeywords: '',
      metaDescription: '',
    };
  }

  const keywords = Array.isArray(payload.seoKeywords)
    ? payload.seoKeywords
    : Array.isArray(payload.keywords) ? payload.keywords : [];
  const tags = Array.isArray(payload.suggestedTags) ? payload.suggestedTags : [];
  const contentHtml = buildCollectedContentHtml(payload);
  return {
    title: payload.title || '',
    thumbnail: payload.thumbnailUrl || firstContentImageUrl(payload) || '',
    excerpt: payload.excerpt || payload.description || '',
    content: contentHtml,
    publishedAt: payload.publishedAt || '',
    tagsText: tags.join(', '),
    metaTitle: payload.seoTitle || '',
    metaKeywords: keywords.join(', '),
    metaDescription: payload.seoDescription || '',
  };
}

function buildToolDraftPayload(mapping: ToolDraftMapping) {
  const icon = mapping.icon.trim();
  const thumbnail = mapping.thumbnail.trim();

  return {
    form: {
      title: mapping.title.trim(),
      slug: '',
      url: mapping.url.trim(),
      mediaCategoryId: undefined,
      icon,
      thumbnail,
      categoryId: undefined,
      description: mapping.description.trim(),
      content: mapping.content,
      featuresText: mapping.featuresText,
      website: mapping.website.trim(),
      isFeatured: false,
      isActive: true,
      sortOrder: 0,
      metaTitle: mapping.metaTitle.trim(),
      metaKeywords: mapping.metaKeywords.trim(),
      metaDescription: mapping.metaDescription.trim(),
      tagsText: mapping.tagsText.trim(),
    },
  };
}

function buildArticleDraftPayload(mapping: ArticleDraftMapping) {
  return {
    form: {
      title: mapping.title.trim(),
      slug: '',
      categoryId: undefined,
      mediaCategoryId: undefined,
      thumbnail: mapping.thumbnail.trim(),
      excerpt: mapping.excerpt.trim(),
      content: mapping.content,
      authorId: undefined,
      metaTitle: mapping.metaTitle.trim(),
      metaKeywords: mapping.metaKeywords.trim(),
      metaDescription: mapping.metaDescription.trim(),
      isFeatured: false,
      isPinned: false,
      isPublished: true,
      publishedAt: mapping.publishedAt.trim(),
      tagsText: mapping.tagsText.trim(),
    },
  };
}

function cleanupPickerListeners() {
  removePickerListeners?.();
  removePickerListeners = null;
}

function isHtmlElementLike(value: unknown): value is HTMLElement {
  return Boolean(value && typeof value === 'object' && 'nodeType' in value && (value as Node).nodeType === Node.ELEMENT_NODE);
}

function markPickerElement(attribute: 'data-zq-picker-hover' | 'data-zq-picker-selected', element: HTMLElement | null) {
  const documentNode = element?.ownerDocument;
  if (!documentNode) {
    return;
  }

  documentNode.querySelectorAll(`[${attribute}="1"]`).forEach((item) => {
    item.removeAttribute(attribute);
  });
  if (element) {
    element.setAttribute(attribute, '1');
  }
}

function resolvePickerElement(target: EventTarget | null): HTMLElement | null {
  if (!isHtmlElementLike(target)) {
    return null;
  }

  const ignoredTags = new Set(['HTML', 'BODY', 'SCRIPT', 'STYLE', 'LINK', 'META', 'HEAD']);
  const candidate = target.closest<HTMLElement>('*');
  if (!candidate || ignoredTags.has(candidate.tagName)) {
    return null;
  }

  return candidate;
}

function escapeSelectorValue(value: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(value);
  }

  return value.replace(/([ !"#$%&'()*+,./:;<=>?@[\\\]^`{|}~])/g, '\\$1');
}

function isSelectorClassName(value: string): boolean {
  return /^[a-z_][a-z0-9_-]{1,40}$/i.test(value) && !/^\d/.test(value);
}

function buildSelectorSegment(element: HTMLElement): string {
  const tagName = element.tagName.toLowerCase();
  const classes = Array.from(element.classList).filter(isSelectorClassName).slice(0, 2);
  let segment = tagName;
  if (classes.length) {
    segment += classes.map((item) => `.${escapeSelectorValue(item)}`).join('');
  }

  const parentElement = element.parentElement;
  if (!parentElement) {
    return segment;
  }

  const sameTagSiblings = Array.from(parentElement.children)
    .filter((item) => isHtmlElementLike(item) && item.tagName === element.tagName);
  if (sameTagSiblings.length > 1) {
    const index = sameTagSiblings.indexOf(element) + 1;
    segment += `:nth-of-type(${index})`;
  }

  return segment;
}

function buildUniqueSelector(element: HTMLElement, documentNode: Document): string {
  if (element.id) {
    const idSelector = `#${escapeSelectorValue(element.id)}`;
    if (documentNode.querySelectorAll(idSelector).length === 1) {
      return idSelector;
    }
  }

  const segments: string[] = [];
  let current: HTMLElement | null = element;
  while (current && current.tagName !== 'BODY') {
    segments.unshift(buildSelectorSegment(current));
    const selector = segments.join(' > ');
    try {
      if (documentNode.querySelectorAll(selector).length === 1) {
        return selector;
      }
    } catch {
      // Continue walking up the DOM until we build a valid selector.
    }
    current = current.parentElement;
  }

  return `body > ${segments.join(' > ')}`;
}

function buildGroupSelector(element: HTMLElement, documentNode: Document): string {
  const tagElement = element.closest<HTMLElement>('a, span, li, div') || element;
  const classes = Array.from(tagElement.classList).filter(isSelectorClassName).slice(0, 2);
  if (classes.length) {
    const selector = `${tagElement.tagName.toLowerCase()}${classes.map((item) => `.${escapeSelectorValue(item)}`).join('')}`;
    try {
      const count = documentNode.querySelectorAll(selector).length;
      if (count >= 2 && count <= 24) {
        return selector;
      }
    } catch {
      // Fall back to a unique selector below.
    }
  }

  if (tagElement.parentElement) {
    const parentSelector = buildUniqueSelector(tagElement.parentElement, documentNode);
    const selector = `${parentSelector} > ${tagElement.tagName.toLowerCase()}`;
    try {
      const count = documentNode.querySelectorAll(selector).length;
      if (count >= 2 && count <= 24) {
        return selector;
      }
    } catch {
      // Fall back to a unique selector below.
    }
  }

  return buildUniqueSelector(tagElement, documentNode);
}

function buildContentSelector(element: HTMLElement, documentNode: Document): string {
  const container = element.closest<HTMLElement>('article, main, section, [class*="content"], [class*="article"], [class*="post"], [class*="entry"]') || element;
  return buildUniqueSelector(container, documentNode);
}

function buildSelectorForField(field: CollectorRuleField, element: HTMLElement, documentNode: Document): string {
  if (field === 'tags') {
    return buildGroupSelector(element, documentNode);
  }
  if (field === 'content') {
    return buildContentSelector(element, documentNode);
  }
  return buildUniqueSelector(element, documentNode);
}

function collectPickerImageFallbackCandidates(image: HTMLImageElement, baseUrl: string): string[] {
  const candidates = [
    image.currentSrc,
    image.getAttribute('src'),
    image.getAttribute('data-src'),
    image.getAttribute('data-original'),
    image.getAttribute('data-thumb'),
    image.getAttribute('data-thumbnail'),
    image.getAttribute('data-cover'),
    image.getAttribute('data-image'),
  ]
    .map((item) => resolvePickerAssetUrl(item, baseUrl))
    .filter(Boolean);

  const srcsetValues = [
    image.getAttribute('srcset'),
    image.getAttribute('data-srcset'),
  ]
    .map((item) => normalisePickerSrcset(item, baseUrl))
    .filter(Boolean)
    .flatMap((item) => item.split(',').map((candidate) => candidate.trim().split(/\s+/, 2)[0]).filter(Boolean));

  return Array.from(new Set([...candidates, ...srcsetValues]));
}

function markPickerBrokenImages(documentNode: Document) {
  pickerBrokenImageCount.value = Array.from(documentNode.images).filter((image) => image.getAttribute('data-zq-image-broken') === '1').length;
}

function repairPickerPreviewImages(documentNode: Document, baseUrl: string) {
  Array.from(documentNode.images).forEach((image) => {
    const candidates = collectPickerImageFallbackCandidates(image, baseUrl);
    let candidateIndex = 0;

    const tryNextCandidate = () => {
      while (candidateIndex < candidates.length) {
        const nextCandidate = candidates[candidateIndex];
        candidateIndex += 1;
        if (!nextCandidate || nextCandidate === image.currentSrc) {
          continue;
        }
        image.removeAttribute('data-zq-image-broken');
        image.src = nextCandidate;
        return true;
      }
      image.setAttribute('data-zq-image-broken', '1');
      markPickerBrokenImages(documentNode);
      return false;
    };

    if (image.complete && image.naturalWidth > 0) {
      image.removeAttribute('data-zq-image-broken');
      return;
    }

    image.addEventListener('load', () => {
      image.removeAttribute('data-zq-image-broken');
      markPickerBrokenImages(documentNode);
    }, { once: true });

    image.addEventListener('error', () => {
      if (!tryNextCandidate()) {
        markPickerBrokenImages(documentNode);
      }
    }, { once: false });

    if ((!image.getAttribute('src') || image.naturalWidth === 0) && candidates.length) {
      tryNextCandidate();
    }
  });

  markPickerBrokenImages(documentNode);
}

function refreshDisplayedResult() {
  const payload = baseResult.value;
  if (!payload) {
    result.value = null;
    resetMappings(null);
    return;
  }

  const nextResult = currentRule.value ? applyCollectorSiteRule(payload, currentRule.value) : payload;
  result.value = nextResult;
  resetMappings(nextResult);

  if (previewMode.value === 'picker') {
    void nextTick(() => bindPickerFrameEvents());
  }
}

function ensureCurrentRuleDraft(host: string): CollectorSiteRuleItem {
  if (currentRule.value?.host === host) {
    return { ...currentRule.value };
  }

  return createEmptyCollectorSiteRule(host);
}

async function loadCurrentSiteRule(payload: CollectWebPageResult | null) {
  if (!payload) {
    currentRule.value = null;
    return;
  }

  currentRule.value = await getCollectorSiteRule(payload.finalUrl);
}

function handlePickerSelection(documentNode: Document, element: HTMLElement) {
  if (!baseResult.value) {
    Message.warning('请先完成页面采集');
    return;
  }
  if (!pickerField.value) {
      Message.warning('请先选择一个字段');
    return;
  }

  const host = extractHost(baseResult.value.finalUrl);
  if (!host) {
    Message.warning('当前页面未识别到有效域名');
    return;
  }

  const selector = buildSelectorForField(pickerField.value, element, documentNode);
  const nextRule = ensureCurrentRuleDraft(host);
  nextRule[getCollectorRuleFieldPathKey(pickerField.value)] = selector;
  nextRule.updatedAt = new Date().toISOString();

  const nextResult = applyCollectorSiteRule(baseResult.value, nextRule);
  const previewValue = buildRulePreviewValue(nextResult, pickerField.value);
  if (!previewValue) {
      Message.warning('已生成规则，但当前没有提取到值，请换一个更准确的元素再试');
    return;
  }

  currentRule.value = nextRule;
  result.value = nextResult;
  resetMappings(nextResult);
  pickerLastValue.value = previewValue;
  pickerSelectionSummary.value = {
    field: pickerField.value,
    fieldLabel: getPickerFieldLabel(pickerField.value),
    selector,
    matchCount: getPickerSelectionMatchCount(documentNode, selector),
    targetTag: element.tagName.toLowerCase(),
    previewValue,
  };
    pickerStatus.value = `已选中「${getPickerFieldLabel(pickerField.value)}」，规则已回填到当前结果。确认无误后点「保存当前站点规则」。`;
  markPickerElement('data-zq-picker-selected', element);
  scrollToDraftMappingField(pickerField.value);
  Message.success('已从页面生成字段规则');
}

function bindPickerFrameEvents() {
  cleanupPickerListeners();

  if (previewMode.value !== 'picker' || !pickerPreviewHtml.value) {
    return;
  }

  const documentNode = pickerFrameRef.value?.contentDocument;
  if (!documentNode) {
    return;
  }

  repairPickerPreviewImages(documentNode, String(baseResult.value?.finalUrl || ''));

  let currentHoverElement: HTMLElement | null = null;

  const handleMouseOver = (event: Event) => {
    const element = resolvePickerElement(event.target);
    if (element === currentHoverElement) {
      return;
    }
    currentHoverElement = element;
    markPickerElement('data-zq-picker-hover', element);
  };

  const handleClick = (event: Event) => {
    event.preventDefault();
    event.stopPropagation();

    if (!pickerField.value) {
      Message.warning('请先选择要采集的字段');
      return;
    }

    const element = resolvePickerElement(event.target);
    if (!element) {
      return;
    }

    handlePickerSelection(documentNode, element);
  };

  documentNode.addEventListener('mouseover', handleMouseOver, true);
  documentNode.addEventListener('click', handleClick, true);

  removePickerListeners = () => {
    documentNode.removeEventListener('mouseover', handleMouseOver, true);
    documentNode.removeEventListener('click', handleClick, true);
    markPickerElement('data-zq-picker-hover', null);
  };
}

function handlePickerFrameLoad() {
  if (previewMode.value === 'picker') {
    bindPickerFrameEvents();
  }
}

async function saveCurrentSiteRuleDraft() {
  if (!activeRuleHost.value || !currentRule.value) {
    Message.warning('请先点选至少一个字段后再保存规则');
    return;
  }
  if (isCollectorSiteRuleEmpty(currentRule.value)) {
    Message.warning('当前还没有可保存的站点规则');
    return;
  }

  currentRule.value = await saveCollectorSiteRule(currentRule.value);
  refreshDisplayedResult();
  pickerStatus.value = `已保存 ${currentRule.value.host} 的定向规则。下次采集同站点页面会自动套用。`;
  refreshDisplayedResult();
  Message.success('站点定向规则已保存');
}
async function deleteCurrentSiteRuleDraft() {
  if (!activeRuleHost.value) {
    Message.warning('当前没有可删除的站点规则');
    return;
  }

  await deleteCollectorSiteRule(activeRuleHost.value);
  currentRule.value = null;
  pickerLastValue.value = '';
  pickerSelectionSummary.value = null;
  pickerBrokenImageCount.value = 0;
  pickerStatus.value = '已删除当前站点规则，采集结果恢复为通用规则。';
  refreshDisplayedResult();
  Message.success('当前站点规则已删除');
}

async function handleCollect() {
  if (!currentSiteId.value) {
    Message.warning('请先选择当前站点');
    return;
  }
  if (!currentTenantId.value) {
    Message.warning('请先选择当前租户');
    return;
  }

  const url = normaliseExternalHttpUrl(sourceUrl.value);
  if (!isValidHttpUrl(url)) {
    Message.warning('请输入有效的网页地址');
    return;
  }

  collecting.value = true;
  try {
    const payload = await collectWebPage(url, {
      siteId: currentSiteId.value || undefined,
      tenantId: currentTenantId.value || undefined,
      useBrowserRender: useBrowserRender.value,
    });
    baseResult.value = payload;
    await loadCurrentSiteRule(payload);
    refreshDisplayedResult();
    sourceUrl.value = payload.finalUrl || url;
    previewMode.value = 'rendered';
    pickerStatus.value = currentRule.value
      ? `已自动套用 ${currentRule.value.host} 的定向规则。你也可以切到"点选规则"继续补齐。`
      : '页面采集完成。切到"点选规则"后，先选字段，再点击原页面里的对应内容。';
    pickerBrokenImageCount.value = 0;
    Message.success(currentRule.value ? '页面采集完成，已自动应用站点规则' : '页面采集完成，已生成预览结果');
  } catch (error) {
    baseResult.value = null;
    result.value = null;
    currentRule.value = null;
    pickerBrokenImageCount.value = 0;
    Message.error(error instanceof Error ? error.message : '网页采集失败');
  } finally {
    collecting.value = false;
  }
}

function handleReset() {
  cleanupPickerListeners();
  baseResult.value = null;
  result.value = null;
  currentRule.value = null;
  previewMode.value = 'rendered';
  pickerField.value = '';
    pickerStatus.value = '切到"点选规则"后，先选字段，再直接点击页面里的目标区域。';
  pickerLastValue.value = '';
  pickerSelectionSummary.value = null;
  pickerBrokenImageCount.value = 0;
  resetMappings(null);
}

async function handleBatchCollect() {
  if (!currentSiteId.value) {
    Message.warning('请先选择当前站点');
    return;
  }
  if (!currentTenantId.value) {
    Message.warning('请先选择当前租户');
    return;
  }

  const urls = batchUrlsText.value
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .filter((line) => isValidHttpUrl(normaliseExternalHttpUrl(line)));

  if (!urls.length) {
    Message.warning('请输入至少一个有效的网页地址');
    return;
  }

  if (collectorStore.batchStatus === 'idle') {
    collectorStore.initBatch(urls, batchDraftType.value);
  }
  collectorStore.startBatch();

  for (let index = 0; index < collectorStore.batchResults.length; index += 1) {
    while (collectorStore.batchStatus === 'paused') {
      await new Promise<void>((resolve) => {
        const check = setInterval(() => {
          if (collectorStore.batchStatus !== 'paused') {
            clearInterval(check);
            resolve();
          }
        }, 300);
      });
    }

    if (collectorStore.batchStatus === 'idle') {
      break;
    }

    const item = collectorStore.batchResults[index];
    if (!item || item.status === 'success' || item.status === 'failed') continue;

    const url = normaliseExternalHttpUrl(item.url);

    let itemHost = '';
    try {
      itemHost = new URL(url).hostname.toLowerCase();
    } catch { /* ignore */ }

    if (skipCollectedUrls.value && itemHost && collectedUrlSet.value.has(url)) {
      collectorStore.updateBatchItem(index, { status: 'failed', error: '已采集过（跳过）', title: url });
      skippedUrlCount.value += 1;
      collectorStore.batchProgress.current = index + 1;
      continue;
    }

    collectorStore.updateBatchItem(index, { status: 'collecting' });
    collectorStore.batchProgress.current = index + 1;

    try {
      if (useBrowserRender.value && index > 0) {
        await new Promise((resolve) => setTimeout(resolve, 800));
      }

      const payload = await collectWebPage(url, {
        siteId: currentSiteId.value || undefined,
        tenantId: currentTenantId.value || undefined,
        useBrowserRender: useBrowserRender.value,
      });
      const rule = await getCollectorSiteRule(payload.finalUrl);
      if (rule) {
        currentRule.value = rule;
      }
      const refined = rule ? applyCollectorSiteRule(payload, rule) : payload;

      collectorStore.updateBatchItem(index, {
        title: refined.title || url,
        status: 'success',
        ruleHost: rule?.host,
      });

      saveCollectedUrl(url, refined.title || '', refined.host || itemHost);

      const draftType = batchDraftType.value;
      const toolTargetId = draftType !== 'article' ? buildCollectorDraftId('tool') : undefined;
      const articleTargetId = draftType !== 'tool' ? buildCollectorDraftId('article') : undefined;

      if (toolTargetId && currentTenantId.value) {
        const toolMappingData = buildDefaultToolMapping(refined);
        await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, 'tool', toolTargetId), {
          tenantId: currentTenantId.value,
          contentType: 'tool',
          targetId: toolTargetId,
          title: toolMappingData.title || refined.title || '采集草稿',
          payload: buildToolDraftPayload(toolMappingData),
        });
        collectorStore.updateBatchItem(index, { toolDraftId: toolTargetId });
      }

      if (articleTargetId && currentTenantId.value) {
        const articleMappingData = buildDefaultArticleMapping(refined);
        await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, 'article', articleTargetId), {
          tenantId: currentTenantId.value,
          contentType: 'article',
          targetId: articleTargetId,
          title: articleMappingData.title || refined.title || '采集草稿',
          payload: buildArticleDraftPayload(articleMappingData),
        });
        collectorStore.updateBatchItem(index, { articleDraftId: articleTargetId });
      }
    } catch (error) {
      collectorStore.updateBatchItem(index, {
        status: 'failed',
        error: error instanceof Error ? error.message : '采集失败',
        title: url,
      });
    }
  }

  if (collectorStore.batchStatus === 'running') {
    collectorStore.completeBatch();
    const successCount = collectorStore.batchSuccessCount;
    const failCount = collectorStore.batchFailCount;
    const draftLabel = batchDraftType.value === 'both' ? '工具和文章草稿' : `${batchDraftType.value === 'tool' ? '工具' : '文章'}草稿`;
    const skipHint = skippedUrlCount.value > 0 ? `，已跳过 ${skippedUrlCount.value} 个已采集` : '';
    Message.success(`批量采集完成：成功 ${successCount} 个，失败 ${failCount} 个${skipHint}，已自动保存${draftLabel}`);
  }
  skippedUrlCount.value = 0;
}

function handleBatchPause() {
  collectorStore.pauseBatch();
    Message.info('已暂停批量采集，可点击「继续采集」恢复');
}

function handleBatchResume() {
  collectorStore.startBatch();
  handleBatchCollect();
}

function handleBatchStop() {
  collectorStore.stopBatch();
    Message.info('已停止批量采集');
}

function handleBatchReset() {
  collectorStore.resetBatch();
  skippedUrlCount.value = 0;
}

function handleDeleteHistoryItem(host: string, url: string) {
  deleteCollectedUrl(host, url);
  historyVersion.value += 1;
  Message.success('已删除该采集记录');
}

function handleClearHistory() {
  if (!historyPanelHost.value) return;
  clearCollectedHistory(historyPanelHost.value);
  historyVersion.value += 1;
    Message.success(`已清空 ${historyPanelHost.value} 的采集记录`);
}

function toggleHistoryPanel() {
  if (!showHistoryPanel.value) {
    historyVersion.value += 1;
  }
  showHistoryPanel.value = !showHistoryPanel.value;
}

async function handleScanSite() {
  if (!currentSiteId.value) {
    Message.warning('请先选择当前站点');
    return;
  }
  if (!currentTenantId.value) {
    Message.warning('请先选择当前租户');
    return;
  }

  const url = normaliseExternalHttpUrl(scanUrl.value.trim());
  if (!isValidHttpUrl(url)) {
    Message.warning('请输入有效的网页地址');
    return;
  }

  scanning.value = true;
  scanResult.value = null;
  scannedSelectedLinks.value = new Set();
  try {
    const cleanRules = scanFilterRules.value.filter((rule) => rule.value.trim().length > 0);
    const result = await scanSiteLinks({
      url,
      maxLinks: scanMaxLinks.value,
      scanSitemap: scanSitemap.value,
      filterRules: cleanRules,
      siteId: currentSiteId.value || undefined,
      tenantId: currentTenantId.value || undefined,
    });
    scanResult.value = result;
    const parts: string[] = [];
    if (result.pageHtmlCount > 0) parts.push(`页面 ${result.pageHtmlCount} 个`);
    if (result.sitemapUrlCount > 0) parts.push(`站点地图 ${result.sitemapUrlCount} 个`);
    Message.success(`扫描完成，共 ${result.links.length} 个链接（${parts.join('，')}）`);
    persistScanSettings();
  } catch (error) {
    scanResult.value = null;
    Message.error(error instanceof Error ? error.message : '全站扫描失败');
  } finally {
    scanning.value = false;
  }
}

function addScanFilterRule() {
  scanFilterRules.value.push({ field: 'url', operator: 'path_contains', value: '' });
}

function removeScanFilterRule(index: number) {
  scanFilterRules.value.splice(index, 1);
}

function updateScanFilterRule(index: number, key: 'field' | 'operator' | 'value', val: string) {
  if (index >= 0 && index < scanFilterRules.value.length) {
    const next = [...scanFilterRules.value];
    next[index] = { ...next[index], [key]: val };
    scanFilterRules.value = next;
  }
}

function toggleScannedLink(url: string) {
  const next = new Set(scannedSelectedLinks.value);
  if (next.has(url)) {
    next.delete(url);
  } else {
    next.add(url);
  }
  scannedSelectedLinks.value = next;
}

function selectAllScannedLinks() {
  if (!scanResult.value) return;
  const next = new Set(scannedSelectedLinks.value);
  const allSelected = scanResult.value.links.every((link) => next.has(link.url));
  if (allSelected) {
    scannedSelectedLinks.value = new Set();
  } else {
    scannedSelectedLinks.value = new Set(scanResult.value.links.map((link) => link.url));
  }
}

function sendSelectedToBatch() {
  if (!scanResult.value) return;
  const selected = scanResult.value.links.filter((link) => scannedSelectedLinks.value.has(link.url));
  if (!selected.length) {
    Message.warning('请至少选择一个链接');
    return;
  }

  collectorStore.resetBatch();
  batchUrlsText.value = selected.map((link) => link.url).join('\n');
  collectMode.value = 'batch';
    Message.success(`已发送 ${selected.length} 个链接到批量采集`);
}

function handleScanReset() {
  scanResult.value = null;
  scannedSelectedLinks.value = new Set();
  scanUrl.value = '';
  currentScanHost = '';
}

function applyScanSettingsFromHost(host: string) {
  const settings = loadCollectorScanSettings(host);
  if (!settings) {
    return;
  }
  scanMaxLinks.value = settings.maxLinks;
  scanSitemap.value = settings.scanSitemap;
  scanFilterRules.value = settings.filterRules.length ? settings.filterRules : [];
}

let lastScanSettingsHost = '';
function persistScanSettings() {
  const host = extractScanSettingsHost(scanUrl.value || lastScanSettingsHost);
  if (!host) {
    return;
  }
  lastScanSettingsHost = host;
  saveCollectorScanSettings({
    host,
    maxLinks: scanMaxLinks.value,
    scanSitemap: scanSitemap.value,
    filterRules: scanFilterRules.value,
    updatedAt: new Date().toISOString(),
  });
}

function forgetScanSettings() {
  const host = extractScanSettingsHost(scanUrl.value || lastScanSettingsHost);
  if (host) {
    deleteCollectorScanSettings(host);
  }
  lastScanSettingsHost = '';
}

async function createDraft(contentType: 'tool' | 'article') {
  if (!currentTenantId.value) {
    Message.warning('请先选择租户后再生成草稿');
    return;
  }
  if (!result.value) {
    Message.warning('请先完成采集');
    return;
  }

  draftGenerating.value = contentType;
  try {
    const targetId = buildCollectorDraftId(contentType);
    const payload = contentType === 'tool'
      ? buildToolDraftPayload(toolMapping)
      : buildArticleDraftPayload(articleMapping);
    const draftTitle = contentType === 'tool' ? toolMapping.title.trim() : articleMapping.title.trim();

    await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, contentType, targetId), {
      tenantId: currentTenantId.value,
      contentType,
      targetId,
      title: draftTitle || result.value.title || '采集草稿',
      payload,
    });

    Message.success(contentType === 'tool' ? '已生成工具草稿' : '已生成文章草稿');
    void router.push({
      path: contentType === 'tool' ? '/tools/new' : '/articles/new',
      query: {
        from: 'collector',
        draft: targetId,
      },
    });
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '生成草稿失败');
  } finally {
    draftGenerating.value = null;
  }
}

let currentScanHost = '';

watch(scanUrl, (value) => {
  const host = extractScanSettingsHost(value);
  if (host && host !== currentScanHost) {
    currentScanHost = host;
    applyScanSettingsFromHost(host);
  }
});

onMounted(async () => {
  await initialiseDesktopContext(siteStore, authStore);
  const host = extractScanSettingsHost(scanUrl.value);
  if (host) {
    currentScanHost = host;
    applyScanSettingsFromHost(host);
  }

  if (collectorStore.batchStatus === 'running' || collectorStore.batchStatus === 'paused') {
    collectMode.value = 'batch';
  }
});

watch(previewMode, async (mode) => {
  if (mode === 'picker') {
    refreshDisplayedResult();
  }
});

watch([previewMode, pickerField, pickerPreviewHtml], () => {
  cleanupPickerListeners();
  if (previewMode.value === 'picker' && pickerPreviewHtml.value) {
    void nextTick(() => bindPickerFrameEvents());
  }
});

onBeforeUnmount(() => {
  cleanupPickerListeners();
});
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>本地采集</h2>
        <p>当前先接入静态网页采集，支持抓取标题、摘要、正文和图片，并按预览结果直接生成工具或文章草稿?/p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-space wrap>
            <a-tag color="purple">{{ currentSite?.name || '未选站点? }}</a-tag>
            <a-tag color="arcoblue">{{ currentTenant?.name || '未选租户? }}</a-tag>
          </a-space>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :disabled="!result" @click="handleReset">清空结果</a-button>
          <a-button
            type="primary"
            :disabled="!canGenerateDraft"
            :loading="draftGenerating === 'article'"
            @click="createDraft('article')"
          >
            生成文章草稿
          </a-button>
          <a-button
            type="primary"
            :disabled="!canGenerateDraft"
            :loading="draftGenerating === 'tool'"
            @click="createDraft('tool')"
          >
            生成工具草稿
          </a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-card title="采集入口" class="page-section-card">
        <a-space direction="vertical" fill size="large">
          <SiteTenantContextBar
            :sites="siteStore.sites"
            :tenants="filteredTenants"
            :current-site-id="currentSiteId"
            :current-tenant-id="currentTenantId"
            :syncing="syncingTenants"
            @site-change="handleSiteChange"
            @tenant-change="handleTenantChange"
            @sync-tenants="syncCurrentSiteTenants"
            @manage-sites="router.push('/sites')"
            @login-tenant="router.push('/login')"
          />

          <a-radio-group v-model="collectMode" type="button" size="small">
            <a-radio value="single">单条采集</a-radio>
            <a-radio value="batch">批量采集</a-radio>
            <a-radio value="scan">全站扫描</a-radio>
          </a-radio-group>

          <template v-if="collectMode === 'single'">
          <a-input-search
            v-model="sourceUrl"
            placeholder="请输入要采集的网页地址，例?https://example.com/article"
            search-button
            allow-clear
            :loading="collecting"
            :disabled="!canCollect"
            @search="handleCollect"
          />
          <a-space wrap>
            <a-tag color="green">静态抓取?/a-tag>
            <a-tag color="blue">支持字段映射</a-tag>
            <a-tag color="purple">可转本地草稿</a-tag>
          </a-space>
          <a-space wrap style="margin-top: 4px;">
            <a-checkbox v-model="useBrowserRender" :disabled="collecting">
              浏览器渲染（对需要JS渲染的动态页面效果更好，但会更慢）?            </a-checkbox>
          </a-space>
          </template>

          <template v-else-if="collectMode === 'scan'">
            <a-input
              v-model="scanUrl"
              placeholder="请输入要扫描的网站首页或列表页地址，例?https://example.com"
              allow-clear
              :disabled="scanning"
              @keyup.enter="handleScanSite"
            >
              <template #suffix>
                <a-button type="primary" size="small" :loading="scanning" :disabled="scanning" @click="handleScanSite">开始扫描?/a-button>
              </template>
            </a-input>
            <a-space wrap>
              <a-space>
                <span style="font-size: 13px; color: #4e5969;">最多采集：</span>
                <a-input-number v-model="scanMaxLinks" :min="1" :max="500" :step="1" size="small" style="width: 100px" :disabled="scanning" />
                <span style="font-size: 13px; color: #86909c;">?/span>
              </a-space>
              <a-checkbox v-model="scanSitemap" :disabled="scanning">扫描 robots.txt / sitemap.xml</a-checkbox>
              <a-tag color="green">自动发现同站链接</a-tag>
              <a-tag color="blue">支持选择后批量采集?/a-tag>
              <a-tag color="purple">自动保存草稿</a-tag>
              <a-tag v-if="hasScanSettings" color="orange">{{ currentScanHost }} 已保存设置?/a-tag>
            </a-space>

            <a-space v-if="hasScanSettings" wrap style="margin-top: 4px;">
              <a-button size="mini" status="warning" @click="forgetScanSettings">清除 {{ currentScanHost }} 的保存设置?/a-button>
            </a-space>

            <a-space v-if="currentScanHost" wrap style="margin-top: 4px;">
              <a-button size="mini" @click="toggleHistoryPanel">
                {{ showHistoryPanel ? '收起记录' : `采集记录 (${collectedHistoryItems.length})` }}
              </a-button>
            </a-space>

            <a-collapse :default-active-key="scanFilterRules.length ? ['filter'] : []" :bordered="false" size="small">
              <a-collapse-item key="filter" header="自定义过滤规则?>
                <template #extra>
                  <a-button size="mini" type="text" @click.stop="addScanFilterRule">+ 添加规则</a-button>
                </template>
                <a-space direction="vertical" fill size="small">
                  <div v-for="(rule, ruleIndex) in scanFilterRules" :key="ruleIndex" class="scan-filter-rule-row">
                    <a-select
                      :model-value="rule.field" :options="SCAN_FIELD_OPTIONS" size="small" style="width: 90px" :disabled="scanning"
                      @change="(val: string | number | boolean | Record<string, unknown> | Array<string | number | boolean | Record<string, unknown>>) => updateScanFilterRule(ruleIndex, 'field', String(val))"
                    />
                    <a-select
                      :model-value="rule.operator" :options="SCAN_OPERATOR_OPTIONS" size="small" style="width: 120px" :disabled="scanning"
                      @change="(val: string | number | boolean | Record<string, unknown> | Array<string | number | boolean | Record<string, unknown>>) => updateScanFilterRule(ruleIndex, 'operator', String(val))"
                    />
                    <a-input
                      :model-value="rule.value" placeholder="关键词或路径" size="small" style="flex: 1; min-width: 120px" :disabled="scanning"
                      @input="(val: string) => updateScanFilterRule(ruleIndex, 'value', val)"
                    />
                    <a-button size="mini" status="danger" :disabled="scanning" @click="removeScanFilterRule(ruleIndex)">删除</a-button>
                  </div>
                  <a-empty v-if="!scanFilterRules.length" description="暂无自定义规则，将扫描所有发现的同站链接" style="padding: 8px 0" />
                </a-space>
              </a-collapse-item>
            </a-collapse>

            <template v-if="scanResult">
              <a-divider style="margin: 8px 0" />
              <div class="scan-summary">
                <a-space>
                  <strong>{{ scanResult.title || scanResult.finalUrl }}</strong>
                  <a-tag color="arcoblue">{{ scanResult.host }}</a-tag>
                  <a-tag color="green">{{ scanResult.links.length }} 个链接?/a-tag>
                  <a-tag
                    v-if="scanResult.links.some((link) => collectedUrlSet.has(link.url))"
                    color="orangered"
                  >
                    已采集过（跳过）s.filter((link) => collectedUrlSet.has(link.url)).length }} ?                  </a-tag>
                  <a-tag v-if="scanResult.siteName" color="purple">{{ scanResult.siteName }}</a-tag>
                  <a-tag v-if="scanResult.sitemapUrlCount > 0" color="blue">站点地图 {{ scanResult.sitemapUrlCount }}</a-tag>
                  <a-tag v-if="scanResult.pageHtmlCount > 0" color="cyan">页面 {{ scanResult.pageHtmlCount }}</a-tag>
                </a-space>
              </div>
              <div class="scan-actions">
                <a-space wrap>
                  <a-button size="small" @click="selectAllScannedLinks">{{ scanResult.links.every((link) => scannedSelectedLinks.has(link.url)) ? '取消全? : '全? }}</a-button>
                  <a-button type="primary" size="small" :disabled="!scannedSelectedLinks.size" @click="sendSelectedToBatch">批量采集选定（{{ scannedSelectedLinks.size }}?/a-button>
                  <a-button size="small" @click="handleScanReset">清空结果</a-button>
                </a-space>
              </div>
              <div class="scan-links-table">
                <div class="scan-links-table__header">
                  <span class="scan-links-table__col-check">选择</span>
                  <span class="scan-links-table__col-title">标题</span>
                  <span class="scan-links-table__col-url">URL</span>
                  <span class="scan-links-table__col-source">来源</span>
                </div>
                <div
                  v-for="(link, linkIndex) in scanResult.links" :key="linkIndex"
                  class="scan-links-table__row"
                  :class="{ 'scan-links-table__row--selected': scannedSelectedLinks.has(link.url) }"
                  @click="toggleScannedLink(link.url)"
                >
                  <span class="scan-links-table__col-check">
                    <a-checkbox :model-value="scannedSelectedLinks.has(link.url)" @click.stop @update:model-value="toggleScannedLink(link.url)" />
                  </span>
                  <span class="scan-links-table__col-title" :title="link.title">{{ link.title || '-' }}</span>
                  <span class="scan-links-table__col-url" :title="link.url">{{ link.url }}</span>
                  <span class="scan-links-table__col-source">
                    <a-tag v-if="link.source === 'sitemap'" size="small" color="blue">站点地图</a-tag>
                    <a-tag v-else size="small" color="cyan">页面</a-tag>
                    <a-tag v-if="collectedUrlSet.has(link.url)" size="small" color="red">已采集过（跳过）
                  </span>
                </div>
              </div>
            </template>
          </template>

          <template v-else>
            <a-textarea
              v-model="batchUrlsText"
              placeholder="请输入要批量采集的网页地址，每行一个，例如 2026-04-28T09:30:00+08:00e.com/article/1&#10;https://example.com/article/2&#10;https://example.com/article/3"
              :auto-size="{ minRows: 4, maxRows: 8 }"
              :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused"
            />
            <a-space direction="vertical" fill size="small">
              <a-space>
                <span style="font-size: 13px; color: #4e5969;">草稿类型?/span>
                <a-radio-group v-model="batchDraftType" type="button" size="small" :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused">
                  <a-radio value="tool">仅工具?/a-radio>
                  <a-radio value="article">仅文章?/a-radio>
                  <a-radio value="both">全部</a-radio>
                </a-radio-group>
              </a-space>
              <a-space wrap>
                <a-button
                  v-if="collectorStore.batchStatus === 'idle' || collectorStore.batchStatus === 'completed'"
                  type="primary"
                  :disabled="!canCollect"
                  @click="handleBatchCollect"
                >
                  开始批量采集?                </a-button>
                <a-button
                  v-if="collectorStore.isBatchRunning"
                  status="warning"
                  @click="handleBatchPause"
                >
                  暂停采集
                </a-button>
                <a-button
                  v-if="collectorStore.isBatchPaused"
                  type="primary"
                  @click="handleBatchResume"
                >
                  继续采集
                </a-button>
                <a-button
                  v-if="collectorStore.isBatchRunning || collectorStore.isBatchPaused"
                  status="danger"
                  @click="handleBatchStop"
                >
                  停止采集
                </a-button>
                <a-button
                  v-if="collectorStore.batchStatus === 'idle' || collectorStore.batchStatus === 'completed'"
                  :disabled="!batchResults.length"
                  @click="handleBatchReset"
                >
                  清空结果
                </a-button>
                <a-tag color="green">逐条采集</a-tag>
                <a-tag color="purple">自动保存草稿</a-tag>
                <a-tag color="blue">复用站点规则</a-tag>
                <a-tag v-if="useBrowserRender" color="orange">浏览器渲染?/a-tag>
              </a-space>
              <a-space wrap>
                <a-checkbox v-model="skipCollectedUrls" :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused">
                  跳过已采集链接?                </a-checkbox>
                <a-button
                  size="mini"
                  :disabled="!historyPanelHost"
                  @click="toggleHistoryPanel"
                >
                  {{ showHistoryPanel ? '收起记录' : `采集记录 (${collectedHistoryItems.length})` }}
                </a-button>
              </a-space>
              <a-progress
                v-if="collectorStore.isBatchRunning || collectorStore.isBatchPaused || collectorStore.batchStatus === 'completed'"
                :percent="batchProgress.total > 0 ? Math.round((batchProgress.current / batchProgress.total) * 100) : 0"
                :status="collectorStore.batchStatus === 'completed' ? 'success' : collectorStore.isBatchPaused ? 'warning' : undefined"
              >
                <template #text>
                <template #text>
                  <span v-if="collectorStore.isBatchRunning">{{ `采集中 ${batchProgress.current}/${batchProgress.total}` }}</span>
                  <span v-else-if="collectorStore.isBatchPaused">{{ `已暂停 ${batchProgress.current}/${batchProgress.total}` }}</span>
                </template>
              </a-progress>
            </a-space>
          </template>
        </a-space>
      </a-card>

      <a-alert
        v-if="!currentSiteId"
        type="warning"
        title="尚未选择站点"
        content="请先在上方选择当前站点，再同步该站点的租户列表；采集与草稿上下文都应在当前站点下完成?
      />

      <a-alert
        v-else-if="!filteredTenants.length"
        type="warning"
        title="当前站点下暂无租户?
        content="请先同步该站点的租户列表，或前往站点/租户管理页补充租户配置?
      />

      <a-alert
        v-else-if="!currentTenantId"
        type="warning"
        title="尚未选择租户"
        content="采集本身不依赖登录态，但生成草稿前需要先切换到目标租户，这样草稿箱和后续编辑页才能正确隔离?
      />

      <template v-if="collectMode === 'batch' && batchResults.length">
        <a-card title="批量采集结果" class="page-section-card">
          <div class="batch-progress">
            <div class="batch-progress__summary">
              <span>?{{ batchResults.length }} 个链接，</span>
              <span>成功 {{ collectorStore.batchSuccessCount }} 个，</span>
              <span>失败 {{ collectorStore.batchFailCount }} ?/span>
              <span v-if="collectorStore.isBatchPaused" style="color: #ff7d00;">（已暂停?/span>
              <span v-if="collectorStore.batchStatus === 'idle'" style="color: #f53f3f;">已停止批量采集
            </div>
          </div>
          <div class="batch-results-table">
            <div class="batch-results-table__header">
              <span class="batch-results-table__col-url">URL</span>
              <span class="batch-results-table__col-title">标题</span>
              <span class="batch-results-table__col-status">状态?/span>
              <span class="batch-results-table__col-draft">草稿</span>
            </div>
            <div
              v-for="(item, index) in batchResults"
              :key="index"
              class="batch-results-table__row"
              :class="`batch-results-table__row--${item.status}`"
            >
              <span class="batch-results-table__col-url" :title="item.url">{{ item.url }}</span>
              <span class="batch-results-table__col-title">{{ item.title || '-' }}</span>
              <span class="batch-results-table__col-status">
                <a-tag v-if="item.status === 'pending'" color="gray">等待</a-tag>
                <a-tag v-else-if="item.status === 'paused'" color="orangered">暂停</a-tag>
                <a-tag v-else-if="item.status === 'collecting'" color="arcoblue">采集草稿
                <a-tag v-else-if="item.status === 'success'" color="green">{{ item.ruleHost ? '已套用规则? : '成功' }}</a-tag>
                <a-tag v-else-if="item.status === 'failed'" color="red" :title="item.error">{{ item.error === '已采集过（跳过）' ? '已跳过? : '失败' }}</a-tag>
              </span>
              <span class="batch-results-table__col-draft">
                <template v-if="item.status === 'success'">
                  <a-tag v-if="item.toolDraftId" color="purple" size="small">工具</a-tag>
                  <a-tag v-if="item.articleDraftId" color="blue" size="small">文章</a-tag>
                  <span class="batch-results-table__draft-hint">已保存?/span>
                </template>
                <span v-else-if="item.status === 'failed'" class="batch-results-table__error" :title="item.error">{{ item.error || '采集失败' }}</span>
                <span v-else>-</span>
              </span>
            </div>
          </div>
        </a-card>
      </template>

      <a-card
        v-if="showHistoryPanel && historyPanelHost && collectedHistoryItems.length"
        title="采集记录"
        class="page-section-card"
      >
        <template #extra>
          <a-space>
            <span style="font-size: 13px; color: #86909c;">{{ historyPanelHost }}</span>
            <span style="font-size: 13px; color: #86909c;">{{ collectedHistoryItems.length }} ?/span>
            <a-button size="mini" status="danger" @click="handleClearHistory">清空记录</a-button>
          </a-space>
        </template>
        <div class="batch-results-table" style="max-height: 320px; overflow: auto;">
          <div class="batch-results-table__header">
            <span class="batch-results-table__col-url">URL</span>
            <span class="batch-results-table__col-title">标题</span>
            <span class="batch-results-table__col-status">采集时间</span>
            <span class="batch-results-table__col-draft">操作</span>
          </div>
          <div
            v-for="item in collectedHistoryItems.slice().reverse()"
            :key="item.url"
            class="batch-results-table__row"
          >
            <span class="batch-results-table__col-url" :title="item.url" style="font-size: 12px;">{{ item.url }}</span>
            <span class="batch-results-table__col-title">{{ item.title || '-' }}</span>
            <span class="batch-results-table__col-status" style="font-size: 12px; color: #86909c;">{{ item.fetchedAt.slice(0, 10) }}</span>
            <span class="batch-results-table__col-draft">
              <a-button size="mini" status="danger" @click="handleDeleteHistoryItem(item.host, item.url)">删除</a-button>
            </span>
          </div>
        </div>
      </a-card>

      <a-row v-if="collectMode === 'single'" :gutter="16" class="collector-grid">
        <a-col :span="8" class="collector-grid__side">
          <a-card title="采集结果" class="page-section-card collector-card">
            <a-space direction="vertical" fill size="large">
              <div class="collector-summary">
                <div class="collector-summary__hero">
                  <h3>{{ result?.title || '等待采集结果' }}</h3>
                  <p>{{ result?.seoDescription || result?.description || result?.excerpt || '采集成功后，这里会显示页面摘要? }}</p>
                </div>

                <div class="collector-stats">
                  <div class="collector-stat">
                    <span class="collector-stat__label">站点</span>
                    <strong class="collector-stat__value">{{ result?.siteName || '待识别? }}</strong>
                  </div>
                  <div class="collector-stat">
                    <span class="collector-stat__label">正文</span>
                    <strong class="collector-stat__value">{{ contentLength }} ?/strong>
                  </div>
                  <div class="collector-stat">
                    <span class="collector-stat__label">图片</span>
                    <strong class="collector-stat__value">{{ imageCount }} ?/strong>
                  </div>
                  <div class="collector-stat">
                    <span class="collector-stat__label">链接</span>
                    <strong class="collector-stat__value">{{ result?.finalUrl ? '已解析? : '待采集? }}</strong>
                  </div>
                </div>

                <div class="collector-summary-block">
                  <span class="collector-summary-block__label">最终地址</span>
                  <a-typography-paragraph
                    v-if="result?.finalUrl"
                    class="collector-summary-block__url"
                    :copyable="{ text: result.finalUrl }"
                    :ellipsis="{ rows: 2, expandable: true }"
                  >
                    {{ result.finalUrl }}
                  </a-typography-paragraph>
                  <span v-else class="collector-summary-block__empty">待采集?/span>
                </div>

                <div class="collector-summary-block">
                  <span class="collector-summary-block__label">图标 / 缩略图候选
                  <div v-if="result?.iconUrl || result?.thumbnailUrl" class="collector-media-preview-list">
                    <a
                      v-if="result?.iconUrl"
                      :href="result.iconUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="collector-media-preview collector-media-preview--icon"
                    >
                      <img :src="result.iconUrl" alt="采集图标" />
                      <span>图标</span>
                    </a>
                    <a
                      v-if="result?.thumbnailUrl"
                      :href="result.thumbnailUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="collector-media-preview collector-media-preview--thumbnail"
                    >
                      <img :src="result.thumbnailUrl" alt="采集缩略图? />
                      <span>缩略图候选
                    </a>
                  </div>
                  <span v-else class="collector-summary-block__empty">未识别到摘要信息
                </div>

                <div class="collector-summary-block">
                  <span class="collector-summary-block__label">建议标签</span>
                  <div v-if="resultSuggestedTags.length" class="collector-chip-list">
                    <a-tag v-for="tag in resultSuggestedTags" :key="tag" size="small" color="arcoblue">{{ tag }}</a-tag>
                  </div>
                  <span v-else class="collector-summary-block__empty">未识别到摘要信息
                </div>

                <div class="collector-summary-block">
                  <span class="collector-summary-block__label">站点定向规则</span>
                  <div class="collector-chip-list">
                    <a-tag :color="hasCurrentRule ? 'green' : 'gray'">
                      {{ hasCurrentRule ? `已配置?${activeRuleHost || ''}` : '未配置? }}
                    </a-tag>
                  </div>
                  <span class="collector-summary-block__empty">
                    {{ hasCurrentRule ? '当前采集结果已支持站点定向补齐? : '可切到右侧“点选规则”，用点击方式保存当前站点专属规则? }}
                  </span>
                </div>

                <div class="collector-summary-block">
                  <span class="collector-summary-block__label">SEO 关键词或路径
                  <div v-if="resultSeoKeywords.length" class="collector-chip-list">
                    <a-tag v-for="keyword in resultSeoKeywords" :key="keyword" size="small" color="purple">{{ keyword }}</a-tag>
                  </div>
                  <span v-else class="collector-summary-block__empty">未识别到摘要信息
                </div>
              </div>

              <a-card title="采集到的图片资源" :bordered="false" class="collector-inner-card">
                <div v-if="resultImages.length" class="collector-image-list">
                  <div v-for="image in resultImages.slice(0, 6)" :key="image.url" class="collector-image-item">
                    <img :src="image.url" :alt="image.alt || result?.title || ''" />
                    <span>{{ image.alt || '未命名图? }}</span>
                  </div>
                </div>
                <a-empty v-else description="当前页面没有识别到额外图片资源? />
              </a-card>
            </a-space>
          </a-card>
        </a-col>

        <a-col :span="16" class="collector-grid__main">
          <a-space direction="vertical" fill size="large" class="collector-main-stack">
            <a-card title="采集结果预览" class="page-section-card collector-card collector-card--preview">
              <template #extra>
                <a-radio-group v-model="previewMode" type="button" size="small">
                  <a-radio value="rendered">渲染预览</a-radio>
                  <a-radio value="source">HTML 源码</a-radio>
                  <a-radio value="picker" :disabled="!pickerEnabled">点选规则?/a-radio>
                </a-radio-group>
              </template>

              <template v-if="result">
                <a-space direction="vertical" fill size="large">
                  <div class="collector-preview-meta">
                    <h3>{{ result.title }}</h3>
                    <p>{{ result.description || result.excerpt || '未识别到摘要信息' }}</p>
                  </div>

                  <div v-if="previewMode === 'rendered'" class="collector-preview-body collector-scroll-panel" v-html="previewHtml" />

                  <div v-else-if="previewMode === 'picker'" class="collector-picker-panel">
                    <div class="collector-picker-toolbar">
                      <div class="collector-picker-toolbar__header">
                        <strong>点选字段?/strong>
                        <span>{{ pickerFieldDescription }}</span>
                      </div>
                      <div class="collector-picker-toolbar__meta">
                        <a-space wrap>
                          <a-tag :color="useBrowserRender ? 'orange' : 'arcoblue'">{{ useBrowserRender ? '浏览器渲染? : '静态源码? }}</a-tag>
                          <a-tag v-if="pickerBrokenImageCount > 0" color="red">
                            {{ `灰图/失败?${pickerBrokenImageCount}` }}
                          </a-tag>
                        </a-space>
                      </div>
                      <a-radio-group v-model="pickerField" type="button" size="small" class="collector-picker-toolbar__fields">
                        <a-radio v-for="option in PICKER_FIELD_OPTIONS" :key="option.value" :value="option.value">
                          {{ option.label }}
                        </a-radio>
                      </a-radio-group>
                    </div>

                    <a-alert
                      type="info"
                      class="collector-picker-alert"
                      :content="pickerStatus"
                    />

                    <div v-if="pickerSelectionSummary" class="collector-picker-selection">
                      <div class="collector-picker-selection__header">
                        <strong>本次点选结果?/strong>
                        <a-button size="mini" @click="scrollToDraftMappingField(pickerSelectionSummary.field)">
                          定位到字段?                        </a-button>
                      </div>
                      <div class="collector-picker-selection__grid">
                        <div class="collector-picker-selection__item">
                          <span>字段</span>
                          <strong>{{ pickerSelectionSummary.fieldLabel }}</strong>
                        </div>
                        <div class="collector-picker-selection__item">
                          <span>命中数?/span>
                          <strong>{{ pickerSelectionSummary.matchCount }}</strong>
                        </div>
                        <div class="collector-picker-selection__item">
                          <span>元素</span>
                          <strong>{{ pickerSelectionSummary.targetTag }}</strong>
                        </div>
                      </div>
                      <code class="collector-picker-selection__selector">{{ pickerSelectionSummary.selector }}</code>
                      <div class="collector-picker-selection__value">
                        {{ pickerSelectionSummary.previewValue }}
                      </div>
                    </div>

                    <div v-if="quickImageChoices.length" class="collector-quick-image-panel">
                      <div class="collector-quick-image-panel__header">
                        <strong>图片快捷选择</strong>
                        <span>
                          {{ pickerField === 'icon' ? '当前会设为图标? : '当前会设为缩略图' }}
                        </span>
                      </div>
                      <div class="collector-quick-image-grid">
                        <button
                          v-for="choice in quickImageChoices"
                          :key="choice.url"
                          type="button"
                          class="collector-quick-image-item"
                          :title="`${choice.label} - ${choice.type === 'icon' ? '图标候选? : choice.type === 'thumbnail' ? '缩略图候选? : '正文图片'}`"
                          @click="applyQuickImageSelection(choice)"
                        >
                          <div class="collector-quick-image-thumb">
                            <img :src="choice.url" :alt="choice.label" />
                            <span class="collector-quick-image-item__badge">
                              {{ choice.type === 'icon' ? '图标' : choice.type === 'thumbnail' ? 缩略图候选文图片
                            </span>
                            <span class="collector-quick-image-item__hover">
                              {{ choice.label }}
                            </span>
                          </div>
                        </button>
                      </div>
                    </div>

                    <div class="collector-picker-actions">
                      <a-space wrap>
                        <a-button type="primary" :disabled="!hasCurrentRule" @click="saveCurrentSiteRuleDraft">
                          保存当前站点规则
                        </a-button>
                        <a-button status="danger" :disabled="!hasCurrentRule" @click="deleteCurrentSiteRuleDraft">
                          删除当前站点规则
                        </a-button>
                      </a-space>
                      <span v-if="pickerLastValue" class="collector-picker-actions__value">
                        当前提取值：{{ pickerLastValue }}
                      </span>
                    </div>

                    <div v-if="currentRuleFieldEntries.length" class="collector-rule-summary">
                      <div v-for="item in currentRuleFieldEntries" :key="item.label" class="collector-rule-summary__item">
                        <span class="collector-rule-summary__label">{{ item.label }}</span>
                        <code class="collector-rule-summary__path">{{ item.path }}</code>
                      </div>
                    </div>

                    <iframe
                      ref="pickerFrameRef"
                      class="collector-picker-frame collector-scroll-panel"
                      :srcdoc="pickerPreviewHtml"
                      sandbox="allow-same-origin"
                      @load="handlePickerFrameLoad"
                    />
                  </div>

                  <a-typography-paragraph
                    v-else
                    class="collector-preview-source collector-scroll-panel"
                    :copyable="{ text: previewHtml }"
                  >
                    <pre>{{ previewHtml }}</pre>
                  </a-typography-paragraph>
                </a-space>
              </template>

              <a-empty v-else description="输入网址后开始采集，成功后会在这里显示固定尺寸预览，可滚动查看全文? />
            </a-card>

            <a-card title="草稿字段映射" class="page-section-card">
              <template #extra>
                <a-space>
                  <a-button :disabled="!hasResult" @click="resetMappings(result)">恢复默认映射</a-button>
                  <a-button
                    type="primary"
                    :disabled="!canGenerateDraft"
                    :loading="draftGenerating === draftMappingTab"
                    @click="createDraft(draftMappingTab)"
                  >
                    生成当前草稿
                  </a-button>
                </a-space>
              </template>

              <a-tabs v-model:active-key="draftMappingTab" lazy-load>
                <a-tab-pane key="tool" title="工具草稿">
                  <a-form :model="toolMapping" layout="vertical">
                    <div class="collector-form-section">
                      <div class="collector-form-section__title">基础信息</div>
                      <a-row :gutter="16">
                        <a-col id="collector-tool-title" :span="12">
                          <a-form-item label="标题">
                            <a-input v-model="toolMapping.title" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="12">
                          <a-form-item label="来源链接">
                            <a-input v-model="toolMapping.url" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-tool-website" :span="12">
                          <a-form-item label="官网">
                            <a-input v-model="toolMapping.website" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-tool-tags" :span="12">
                          <a-form-item label="标签">
                            <a-input v-model="toolMapping.tagsText" placeholder="多个标签用英文逗号分隔" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-tool-icon" :span="12">
                          <a-form-item label="图标">
                            <a-input v-model="toolMapping.icon" placeholder="优先使用站点图标或页面分享图" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-tool-thumbnail" :span="12">
                          <a-form-item label=缩略图候选
                            <a-input v-model="toolMapping.thumbnail" placeholder="优先页面缩略图，其次正文首图" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-tool-description" :span="24">
                          <a-form-item label="摘要">
                            <a-textarea v-model="toolMapping.description" :auto-size="{ minRows: 3, maxRows: 5 }" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="24">
                          <a-form-item label="特性?>
                            <a-textarea v-model="toolMapping.featuresText" :auto-size="{ minRows: 3, maxRows: 5 }" />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>

                    <div class="collector-form-section">
                      <div class="collector-form-section__title">SEO</div>
                      <a-row :gutter="16">
                        <a-col :span="8">
                          <a-form-item label="SEO 标题">
                            <a-input v-model="toolMapping.metaTitle" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="8">
                          <a-form-item label="SEO 关键词或路径
                            <a-input v-model="toolMapping.metaKeywords" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="8">
                          <a-form-item label="SEO 描述">
                            <a-input v-model="toolMapping.metaDescription" />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>

                    <div class="collector-form-section">
                      <div class="collector-form-section__title">正文</div>
                      <a-row :gutter="16">
                        <a-col id="collector-tool-content" :span="24">
                          <a-form-item label="正文 HTML">
                            <a-textarea
                              v-model="toolMapping.content"
                              class="collector-content-editor"
                              :auto-size="{ minRows: 12, maxRows: 12 }"
                              placeholder=这里的 HTML 会原样进入编辑器正文区域
                            />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>
                  </a-form>
                </a-tab-pane>

                <a-tab-pane key="article" title="文章草稿">
                  <a-form :model="articleMapping" layout="vertical">
                    <div class="collector-form-section">
                      <div class="collector-form-section__title">基础信息</div>
                      <a-row :gutter="16">
                        <a-col id="collector-article-title" :span="12">
                          <a-form-item label="标题">
                            <a-input v-model="articleMapping.title" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-article-thumbnail" :span="12">
                          <a-form-item label=缩略图候选
                            <a-input v-model="articleMapping.thumbnail" placeholder="可手动替换为其他图片地址" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-article-publishedAt" :span="12">
                          <a-form-item label="发布时间">
                            <a-input v-model="articleMapping.publishedAt" placeholder="例如 2026-04-28T09:30:00+08:00" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-article-tags" :span="12">
                          <a-form-item label="标签">
                            <a-input v-model="articleMapping.tagsText" placeholder="多个标签用英文逗号分隔" />
                          </a-form-item>
                        </a-col>
                        <a-col id="collector-article-excerpt" :span="24">
                          <a-form-item label="摘要">
                            <a-textarea v-model="articleMapping.excerpt" :auto-size="{ minRows: 3, maxRows: 5 }" />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>

                    <div class="collector-form-section">
                      <div class="collector-form-section__title">SEO</div>
                      <a-row :gutter="16">
                        <a-col :span="8">
                          <a-form-item label="SEO 标题">
                            <a-input v-model="articleMapping.metaTitle" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="8">
                          <a-form-item label="SEO 关键词或路径
                            <a-input v-model="articleMapping.metaKeywords" />
                          </a-form-item>
                        </a-col>
                        <a-col :span="8">
                          <a-form-item label="SEO 描述">
                            <a-input v-model="articleMapping.metaDescription" />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>

                    <div class="collector-form-section">
                      <div class="collector-form-section__title">正文</div>
                      <a-row :gutter="16">
                        <a-col id="collector-article-content" :span="24">
                          <a-form-item label="正文 HTML">
                            <a-textarea
                              v-model="articleMapping.content"
                              class="collector-content-editor"
                              :auto-size="{ minRows: 12, maxRows: 12 }"
                              placeholder=这里的 HTML 会原样进入编辑器正文区域
                            />
                          </a-form-item>
                        </a-col>
                      </a-row>
                    </div>
                  </a-form>
                </a-tab-pane>
              </a-tabs>
            </a-card>
          </a-space>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<style scoped>
.collector-grid__side,
.collector-grid__main {
  display: flex;
  flex-direction: column;
}

.collector-main-stack {
  width: 100%;
}

.collector-card {
  height: 100%;
}

.collector-card--preview :deep(.arco-card-body) {
  min-height: 680px;
}

.collector-inner-card {
  background: #f7f8fa;
  border-radius: 12px;
}

.collector-summary {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.collector-summary__hero {
  padding: 16px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: linear-gradient(180deg, #f7f8fa 0%, #ffffff 100%);
}

.collector-summary__hero h3 {
  margin: 0 0 8px;
  font-size: 20px;
  line-height: 1.4;
}

.collector-summary__hero p {
  margin: 0;
  color: #4e5969;
  line-height: 1.7;
}

.collector-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.collector-stat {
  padding: 14px 16px;
  border-radius: 12px;
  background: #f7f8fa;
  border: 1px solid #eef0f3;
}

.collector-stat__label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  color: #86909c;
}

.collector-stat__value {
  display: block;
  color: #1d2129;
  font-size: 14px;
  line-height: 1.5;
}

.collector-summary-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.collector-summary-block__label {
  font-size: 12px;
  font-weight: 600;
  color: #4e5969;
}

.collector-summary-block__url {
  margin: 0;
}

.collector-summary-block__empty {
  color: #86909c;
  line-height: 1.6;
}

.collector-chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.collector-media-preview-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.collector-media-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
  color: #1d2129;
  text-decoration: none;
}

.collector-media-preview img {
  display: block;
  object-fit: cover;
  border-radius: 8px;
  background: #e5e6eb;
  flex-shrink: 0;
}

.collector-media-preview span {
  font-size: 12px;
  color: #4e5969;
  white-space: nowrap;
}

.collector-media-preview--icon img {
  width: 32px;
  height: 32px;
}

.collector-media-preview--thumbnail img {
  width: 72px;
  height: 40px;
}

.collector-image-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.collector-image-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.collector-image-item img {
  width: 100%;
  height: 120px;
  object-fit: cover;
  border-radius: 10px;
  background: #e5e6eb;
}

.collector-image-item span {
  font-size: 12px;
  color: #86909c;
  line-height: 1.5;
}

.collector-preview-meta h3 {
  margin: 0 0 8px;
  font-size: 24px;
}

.collector-preview-meta p {
  margin: 0;
  color: #4e5969;
  line-height: 1.7;
}

.collector-scroll-panel {
  height: 560px;
  overflow: auto;
}

.collector-preview-body {
  padding: 20px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid #e5e6eb;
  line-height: 1.8;
  box-sizing: border-box;
}

.collector-preview-body :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 10px;
}

.collector-preview-body :deep(pre) {
  overflow: auto;
  padding: 12px;
  border-radius: 8px;
  background: #f2f3f5;
}

.collector-preview-source {
  margin: 0;
}

.collector-preview-source pre {
  margin: 0;
  min-height: 100%;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 16px;
  border-radius: 12px;
  background: #1d2129;
  color: #f2f3f5;
  box-sizing: border-box;
}

.collector-picker-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.collector-picker-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
}

.collector-picker-toolbar__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.collector-picker-toolbar__header strong {
  font-size: 13px;
  color: #1d2129;
}

.collector-picker-toolbar__header span {
  color: #4e5969;
  font-size: 12px;
  line-height: 1.5;
}

.collector-picker-toolbar__fields {
  display: flex;
  flex-wrap: wrap;
}

.collector-picker-toolbar__meta {
  display: flex;
  justify-content: flex-start;
}

.collector-picker-alert {
  margin: 0;
}

.collector-picker-selection {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid #d9e8ff;
  border-radius: 12px;
  background: #f2f7ff;
}

.collector-picker-selection__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.collector-picker-selection__header strong {
  font-size: 13px;
  color: #1d2129;
}

.collector-picker-selection__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.collector-picker-selection__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.75);
}

.collector-picker-selection__item span {
  color: #86909c;
  font-size: 12px;
}

.collector-picker-selection__item strong {
  color: #1d2129;
  font-size: 13px;
  line-height: 1.5;
}

.collector-picker-selection__selector {
  display: block;
  padding: 10px 12px;
  border-radius: 10px;
  background: #1d2129;
  color: #f2f3f5;
  font-size: 12px;
  line-height: 1.6;
  word-break: break-all;
  white-space: pre-wrap;
}

.collector-picker-selection__value {
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.75);
  color: #4e5969;
  font-size: 12px;
  line-height: 1.6;
  word-break: break-word;
}

.collector-quick-image-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
}

.collector-quick-image-panel__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.collector-quick-image-panel__header strong {
  font-size: 13px;
  color: #1d2129;
}

.collector-quick-image-panel__header span {
  color: #4e5969;
  font-size: 12px;
}

.collector-quick-image-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.collector-quick-image-item {
  display: block;
  padding: 6px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #fff;
  cursor: pointer;
  text-align: left;
  overflow: hidden;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.collector-quick-image-item:hover {
  border-color: #94bfff;
  box-shadow: 0 8px 18px rgba(22, 93, 255, 0.12);
  transform: translateY(-1px);
}

.collector-quick-image-thumb {
  position: relative;
  overflow: hidden;
  border-radius: 10px;
  background: #f7f8fa;
}

.collector-quick-image-item img {
  width: 100%;
  aspect-ratio: 4 / 3;
  height: auto;
  object-fit: contain;
  display: block;
  background: #e5e6eb;
  transition: transform 0.2s ease;
}

.collector-quick-image-item:hover img {
  transform: scale(1.06);
}

.collector-quick-image-item__badge {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 2px 6px;
  border-radius: 999px;
  background: rgba(29, 33, 41, 0.75);
  color: #fff;
  font-size: 11px;
  line-height: 1.4;
}

.collector-quick-image-item__hover {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  padding: 8px;
  background: linear-gradient(180deg, rgba(29, 33, 41, 0) 0%, rgba(29, 33, 41, 0.78) 100%);
  color: #fff;
  font-size: 12px;
  line-height: 1.5;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity 0.2s ease, transform 0.2s ease;
  pointer-events: none;
}

.collector-quick-image-item:hover .collector-quick-image-item__hover {
  opacity: 1;
  transform: translateY(0);
}

.collector-picker-actions {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.collector-picker-actions__value {
  flex: 1;
  min-width: 240px;
  color: #4e5969;
  font-size: 12px;
  line-height: 1.6;
}

.collector-rule-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.collector-rule-summary__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid #eef0f3;
  border-radius: 12px;
  background: #f7f8fa;
}

.collector-rule-summary__label {
  color: #86909c;
  font-size: 12px;
}

.collector-rule-summary__path {
  font-size: 12px;
  line-height: 1.5;
  word-break: break-all;
  white-space: pre-wrap;
}

.collector-picker-frame {
  width: 100%;
  height: 560px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #fff;
}

.collector-form-section + .collector-form-section {
  margin-top: 8px;
}

.collector-form-section__title {
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
}

.collector-content-editor :deep(textarea) {
  max-height: 280px;
  overflow: auto;
}

.batch-progress {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.batch-progress__summary {
  font-size: 14px;
  color: #4e5969;
}

.batch-results-table {
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  overflow: hidden;
}

.batch-results-table__header {
  display: grid;
  grid-template-columns: 2fr 2fr 100px 140px;
  gap: 0;
  padding: 12px 16px;
  background: #f7f8fa;
  border-bottom: 1px solid #e5e6eb;
  font-size: 12px;
  font-weight: 600;
  color: #4e5969;
}

.batch-results-table__row {
  display: grid;
  grid-template-columns: 2fr 2fr 100px 140px;
  gap: 0;
  padding: 10px 16px;
  border-bottom: 1px solid #f2f3f5;
  font-size: 13px;
  align-items: center;
  transition: background 0.2s ease;
}

.batch-results-table__row:last-child {
  border-bottom: none;
}

.batch-results-table__row:hover {
  background: #f7f8fa;
}

.batch-results-table__row--success {
  background: #f6ffed;
}

.batch-results-table__row--failed {
  background: #fff7f7;
}

.batch-results-table__col-url {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #165dff;
}

.batch-results-table__col-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #1d2129;
}

.batch-results-table__col-status {
  display: flex;
  align-items: center;
}

.batch-results-table__col-draft {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.batch-results-table__draft-hint {
  color: #86909c;
  font-size: 11px;
}

.batch-results-table__error {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #f53f3f;
  font-size: 12px;
}

.scan-summary {
  padding: 12px 16px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
}

.scan-actions {
  display: flex;
  align-items: center;
}

.scan-links-table {
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  overflow: hidden;
  max-height: 480px;
  overflow-y: auto;
}

.scan-links-table__header {
  display: grid;
  grid-template-columns: 60px 2fr 2fr 90px;
  gap: 0;
  padding: 10px 16px;
  background: #f7f8fa;
  border-bottom: 1px solid #e5e6eb;
  font-size: 12px;
  font-weight: 600;
  color: #4e5969;
  position: sticky;
  top: 0;
  z-index: 1;
}

.scan-links-table__row {
  display: grid;
  grid-template-columns: 60px 2fr 2fr 90px;
  gap: 0;
  padding: 8px 16px;
  border-bottom: 1px solid #f2f3f5;
  font-size: 13px;
  align-items: center;
  cursor: pointer;
  transition: background 0.15s ease;
}

.scan-links-table__row:last-child {
  border-bottom: none;
}

.scan-links-table__row:hover {
  background: #f0f5ff;
}

.scan-links-table__row--selected {
  background: #e8f3ff;
}

.scan-links-table__col-check {
  display: flex;
  align-items: center;
}

.scan-links-table__col-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #1d2129;
}

.scan-links-table__col-url {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #165dff;
  font-size: 12px;
}

.scan-links-table__col-source {
  display: flex;
  align-items: center;
}

.scan-filter-rule-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
</style>
