import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { Message } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import {
  collectWebPage,
  scanSiteLinks,
  type CollectWebPageResult,
  type ScanSiteLinksResult,
  type SiteLinkItem,
  type ScanFilterRule,
} from '../services/collector.service';
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
} from '../services/collector-site-rule.service';
import {
  loadCollectorScanSettings,
  saveCollectorScanSettings,
  deleteCollectorScanSettings,
  extractScanSettingsHost,
} from '../services/collector-scan-settings.service';
import {
  saveCollectedUrl,
  getCollectedUrlSet,
  loadCollectedUrls,
  deleteCollectedUrl,
  clearCollectedHistory,
  type CollectedHistoryItem,
} from '../services/collector-history.service';
import { buildLocalDraftKey, saveLocalDraft } from '../services/local-draft.service';
import { useAuthStore } from '../stores/auth.store';
import { useSiteStore } from '../stores/site.store';
import { useCollectorStore, type BatchCollectItem } from '../stores/collector.store';
import { initialiseDesktopContext } from '../utils/desktop-context';
import { isValidHttpUrl, normaliseExternalHttpUrl } from '../utils/url';

export function useCollectorPage() {
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
  let currentScanHost = '';

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
      syncingTenants.value = false;
    }
  }

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
    return {
      title: payload?.title || '',
      url: resolveCollectedOfficialUrl(payload),
      website: payload?.siteName || '',
      icon: payload?.iconUrl || '',
      thumbnail: resolveCollectedThumbnail(payload),
      description: payload?.description || payload?.excerpt || '',
      content: buildCollectedContentHtml(payload),
      featuresText: (Array.isArray(payload?.keywords) ? payload.keywords : []).join(', '),
      tagsText: (Array.isArray(payload?.suggestedTags) ? payload.suggestedTags : []).join(', '),
      metaTitle: payload?.title || '',
      metaKeywords: (Array.isArray(payload?.seoKeywords) ? payload.seoKeywords : []).join(', '),
      metaDescription: payload?.seoDescription || payload?.description || '',
    };
  }

  function buildDefaultArticleMapping(payload: CollectWebPageResult | null): ArticleDraftMapping {
    return {
      title: payload?.title || '',
      thumbnail: resolveCollectedThumbnail(payload),
      excerpt: payload?.description || payload?.excerpt || '',
      content: buildCollectedContentHtml(payload),
      publishedAt: payload?.publishedAt || '',
      tagsText: (Array.isArray(payload?.suggestedTags) ? payload.suggestedTags : []).join(', '),
      metaTitle: payload?.title || '',
      metaKeywords: (Array.isArray(payload?.seoKeywords) ? payload.seoKeywords : []).join(', '),
      metaDescription: payload?.seoDescription || payload?.description || '',
    };
  }

  function buildToolDraftPayload(mapping: ToolDraftMapping) {
    return {
      title: mapping.title,
      url: mapping.url,
      website: mapping.website,
      icon: mapping.icon,
      thumbnail: mapping.thumbnail,
      description: mapping.description,
      content: mapping.content,
      features: mapping.featuresText.split(',').map((item: string) => item.trim()).filter(Boolean),
      tags: mapping.tagsText.split(',').map((item: string) => item.trim()).filter(Boolean),
      metaTitle: mapping.metaTitle,
      metaKeywords: mapping.metaKeywords,
      metaDescription: mapping.metaDescription,
    };
  }

  function buildArticleDraftPayload(mapping: ArticleDraftMapping) {
    return {
      title: mapping.title,
      thumbnail: mapping.thumbnail,
      excerpt: mapping.excerpt,
      content: mapping.content,
      publishedAt: mapping.publishedAt,
      tags: mapping.tagsText.split(',').map((item: string) => item.trim()).filter(Boolean),
      metaTitle: mapping.metaTitle,
      metaKeywords: mapping.metaKeywords,
      metaDescription: mapping.metaDescription,
    };
  }

  function cleanupPickerListeners() {
    if (removePickerListeners) {
      removePickerListeners();
      removePickerListeners = null;
    }
  }

  function isHtmlElementLike(node: unknown): node is HTMLElement {
    return Boolean(node && typeof node === 'object' && 'tagName' in node);
  }

  function markPickerElement(element: HTMLElement) {
    cleanupPickerListeners();
  }

  function resolvePickerElement(element: HTMLElement): HTMLElement {
    let current: HTMLElement | null = element;
    while (current && current.tagName !== 'BODY') {
      if (current.tagName === 'A' || current.tagName === 'BUTTON' || current.tagName === 'IMG') {
        return current;
      }
      current = current.parentElement;
    }
    return element;
  }

  function escapeSelectorValue(value: string): string {
    return CSS.escape(value);
  }

  function isSelectorClassName(value: string): boolean {
    return /^[\w-]+$/.test(value);
  }

  function buildSelectorSegment(element: HTMLElement): string {
    if (element.id) {
      return `#${CSS.escape(element.id)}`;
    }

    const tagName = element.tagName.toLowerCase();
    const classes = Array.from(element.classList).filter(isSelectorClassName);
    if (classes.length) {
      return `${tagName}.${classes.map((c) => CSS.escape(c)).join('.')}`;
    }

    const parentElement = element.parentElement;
    if (!parentElement) {
      return tagName;
    }

    const sameTagSiblings = Array.from(parentElement.children).filter(
      (child) => child.tagName === element.tagName,
    );
    if (sameTagSiblings.length > 1) {
      const index = sameTagSiblings.indexOf(element) + 1;
      return `${tagName}:nth-of-type(${index})`;
    }

    return tagName;
  }

  function buildUniqueSelector(element: HTMLElement, documentNode: Document): string {
    if (element.id) {
      const idSelector = `#${CSS.escape(element.id)}`;
      try {
        if (documentNode.querySelectorAll(idSelector).length === 1) {
          return idSelector;
        }
      } catch { /* ignore */ }
    }

    const segments: string[] = [];
    let current: HTMLElement | null = element;
    while (current && current.tagName !== 'BODY') {
      const segment = buildSelectorSegment(current);
      segments.unshift(segment);
      try {
        const selector = segments.join(' > ');
        if (documentNode.querySelectorAll(selector).length === 1) {
          return selector;
        }
      } catch { /* ignore */ }
      current = current.parentElement;
    }

    return segments.join(' > ');
  }

  function buildGroupSelector(element: HTMLElement, documentNode: Document): string {
    const tagName = element.tagName.toLowerCase();
    const classes = Array.from(element.classList).filter(isSelectorClassName);

    if (classes.length) {
      const classSelector = `${tagName}.${classes.map((c) => CSS.escape(c)).join('.')}`;
      try {
        const count = documentNode.querySelectorAll(classSelector).length;
        if (count >= 2 && count <= 24) {
          return classSelector;
        }
      } catch { /* ignore */ }
    }

    if (element.parentElement) {
      const parentTag = element.parentElement.tagName.toLowerCase();
      const parentClasses = Array.from(element.parentElement.classList).filter(isSelectorClassName);
      if (parentClasses.length) {
        const parentSelector = `${parentTag}.${parentClasses.map((c) => CSS.escape(c)).join('.')}`;
        const childSelector = `${parentSelector} > ${tagName}`;
        try {
          const count = documentNode.querySelectorAll(childSelector).length;
          if (count >= 2 && count <= 24) {
            return childSelector;
          }
        } catch { /* ignore */ }
      }
    }

    return buildUniqueSelector(element, documentNode);
  }

  function buildContentSelector(element: HTMLElement, documentNode: Document): string {
    return buildGroupSelector(element, documentNode);
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
    const candidates: string[] = [];

    const addCandidate = (url: string | null | undefined) => {
      const nextUrl = String(url || '').trim();
      if (nextUrl) {
        candidates.push(resolvePickerAssetUrl(nextUrl, baseUrl));
      }
    };

    const lazyImageAttributes = [
      'data-src', 'data-original', 'data-lazy-src',
      'data-thumb', 'data-thumbnail', 'data-cover',
    ];
    lazyImageAttributes.forEach((attr) => addCandidate(image.getAttribute(attr)));

    const srcset = image.getAttribute('srcset') || image.getAttribute('data-srcset');
    if (srcset) {
      const srcsetValues = srcset.split(',').map((s) => s.trim().split(/\s+/)[0]).filter(Boolean);
      srcsetValues.forEach((url) => addCandidate(url));
    }

    addCandidate(image.getAttribute('src'));

    return candidates;
  }

  function markPickerBrokenImages(documentNode: Document) {
    documentNode.querySelectorAll('img').forEach((image) => {
      if (!image.complete || image.naturalWidth === 0) {
        image.setAttribute('data-zq-image-broken', '1');
      }
    });
  }

  function repairPickerPreviewImages(documentNode: Document, baseUrl: string) {
    Array.from(documentNode.images).forEach((image) => {
      let candidateIndex = 0;
      const candidates = collectPickerImageFallbackCandidates(image, baseUrl);

      const tryNextCandidate = () => {
        while (candidateIndex < candidates.length) {
          const nextCandidate = candidates[candidateIndex++];
          if (!nextCandidate || nextCandidate === image.currentSrc) {
            continue;
          }
          image.src = nextCandidate;
          return true;
        }
        return false;
      };

      if (image.complete && image.naturalWidth > 0) {
        return;
      }

      image.addEventListener('load', () => {
        image.removeAttribute('data-zq-image-broken');
      }, { once: true });

      image.addEventListener('error', () => {
        if (!tryNextCandidate()) {
          image.setAttribute('data-zq-image-broken', '1');
        }
      }, { once: false });

      if ((!image.getAttribute('src') || image.naturalWidth === 0) && candidates.length) {
        tryNextCandidate();
      }
    });
  }

  function refreshDisplayedResult() {
    const payload = baseResult.value;
    if (!payload) {
      return;
    }

    result.value = { ...payload };

    if (previewMode.value === 'picker') {
      void nextTick(() => bindPickerFrameEvents());
    }
  }

  async function ensureCurrentRuleDraft(host: string): Promise<CollectorSiteRuleItem> {
    if (currentRule.value?.host === host) {
      return currentRule.value;
    }
    const saved = await getCollectorSiteRule(host);
    currentRule.value = saved || createEmptyCollectorSiteRule(host);
    return currentRule.value;
  }

  async function loadCurrentSiteRule(payload: CollectWebPageResult | null) {
    if (!payload) {
      return;
    }
    const host = extractHost(payload.finalUrl);
    if (host) {
      await ensureCurrentRuleDraft(host);
    }
  }

  async function handlePickerSelection(documentNode: Document, element: HTMLElement) {
    if (!baseResult.value) {
      return;
    }
    if (!pickerField.value) {
      Message.warning('请先在工具栏中选择一个字段');
      return;
    }

    const host = extractHost(baseResult.value.finalUrl);
    if (!host) {
      return;
    }

    const rule = await ensureCurrentRuleDraft(host);
    const field = pickerField.value;
    const target = resolvePickerElement(element);
    const selector = buildSelectorForField(field, target, documentNode);
    const pathKey = getCollectorRuleFieldPathKey(field);
    const previewValue = buildRulePreviewValue(baseResult.value, field);

    rule[pathKey as keyof CollectorSiteRuleItem] = selector;
    currentRule.value = { ...rule };
    pickerField.value = field;
    pickerLastValue.value = selector;

    if (!previewValue) {
      pickerSelectionSummary.value = {
        field,
        fieldLabel: getPickerFieldLabel(field),
        selector,
        matchCount: 1,
        targetTag: target.tagName.toLowerCase(),
        previewValue: selector,
      };
    } else {
      pickerSelectionSummary.value = {
        field,
        fieldLabel: getPickerFieldLabel(field),
        selector,
        matchCount: getPickerSelectionMatchCount(documentNode, selector),
        targetTag: target.tagName.toLowerCase(),
        previewValue,
      };
    }

    pickerStatus.value = `已选中 ${getPickerFieldLabel(field)}，选择器：${selector}`;
    scrollToDraftMappingField(field);
  }

  function bindPickerFrameEvents() {
    if (previewMode.value !== 'picker' || !pickerPreviewHtml.value) {
      return;
    }

    const iframe = pickerFrameRef.value;
    if (!iframe || !iframe.contentDocument) {
      return;
    }

    const documentNode = iframe.contentDocument;
    if (!documentNode) {
      return;
    }

    repairPickerPreviewImages(documentNode, String(baseResult.value?.finalUrl || ''));
    markPickerBrokenImages(documentNode);

    let currentHoverElement: HTMLElement | null = null;

    const handleMouseOver = (event: Event) => {
      const target = event.target;
      if (!isHtmlElementLike(target)) {
        return;
      }
      if (target === currentHoverElement) {
        return;
      }
      if (currentHoverElement) {
        currentHoverElement.removeAttribute('data-zq-picker-hover');
      }
      currentHoverElement = target;
      target.setAttribute('data-zq-picker-hover', '1');
    };

    const handleClick = (event: Event) => {
      event.preventDefault();
      event.stopPropagation();
      const target = event.target;
      if (!isHtmlElementLike(target)) {
        return;
      }
      if (!pickerField.value) {
        Message.warning('请先在右侧工具栏中选择一个字段');
        return;
      }
      if (!target) {
        return;
      }
      handlePickerSelection(documentNode, target);
    };

    removePickerListeners = () => {
      documentNode.removeEventListener('mouseover', handleMouseOver, true);
      documentNode.removeEventListener('click', handleClick, true);
    };

    documentNode.addEventListener('mouseover', handleMouseOver, true);
    documentNode.addEventListener('click', handleClick, true);
  }

  function handlePickerFrameLoad() {
    if (previewMode.value === 'picker') {
      void nextTick(() => bindPickerFrameEvents());
    }
  }

  async function saveCurrentSiteRuleDraft() {
    if (!activeRuleHost.value || !currentRule.value) {
      Message.warning('没有可保存的规则');
      return;
    }
    if (isCollectorSiteRuleEmpty(currentRule.value)) {
      Message.warning('当前规则为空，无内容可保存');
      return;
    }

    try {
      await saveCollectorSiteRule(currentRule.value);
      Message.success('站点规则已保存');
    } catch (error) {
      Message.error(error instanceof Error ? error.message : '保存规则失败');
    }
  }

  async function deleteCurrentSiteRuleDraft() {
    if (!activeRuleHost.value) {
      Message.warning('没有可删除的规则');
      return;
    }

    try {
      await deleteCollectorSiteRule(activeRuleHost.value);
      currentRule.value = createEmptyCollectorSiteRule(activeRuleHost.value);
      Message.success('站点规则已删除');
    } catch (error) {
      Message.error(error instanceof Error ? error.message : '删除规则失败');
    }
  }

  async function handleCollect() {
    if (!currentSiteId.value) {
      Message.warning('请先选择站点');
      return;
    }
    if (!currentTenantId.value) {
      Message.warning('请先选择租户');
      return;
    }

    const url = normaliseExternalHttpUrl(sourceUrl.value.trim());
    if (!isValidHttpUrl(url)) {
      Message.warning('请输入有效的网页地址');
      return;
    }

    collecting.value = true;
    try {
      const payload = await collectWebPage(url, {
        useBrowserRender: useBrowserRender.value,
        siteId: currentSiteId.value,
        tenantId: currentTenantId.value,
      });
      baseResult.value = payload;
      result.value = { ...payload };
      previewHtml;
      Object.assign(toolMapping, buildDefaultToolMapping(payload));
      Object.assign(articleMapping, buildDefaultArticleMapping(payload));
      void loadCurrentSiteRule(payload);
      Message.success('采集完成');
    } catch (error) {
      Message.error(error instanceof Error ? error.message : '采集失败');
    } finally {
      collecting.value = false;
    }
  }

  function handleReset() {
    sourceUrl.value = '';
    baseResult.value = null;
    result.value = null;
    previewMode.value = 'rendered';
    Object.assign(toolMapping, buildDefaultToolMapping(null));
    Object.assign(articleMapping, buildDefaultArticleMapping(null));
    pickerField.value = '';
    pickerStatus.value = '切到"点选规则"后，先选字段，再直接点击页面里的目标区域。';
    pickerLastValue.value = '';
    pickerSelectionSummary.value = null;
    currentRule.value = null;
    scanResult.value = null;
    scanning.value = false;
    scanUrl.value = '';
    scannedSelectedLinks.value = new Set();
    scanFilterRules.value = [];
    collectMode.value = 'single';
  }

  function getItemUrl(item: BatchCollectItem): string {
    return item.url;
  }

  function getItemTitle(item: BatchCollectItem): string {
    return item.title || '';
  }

  async function handleBatchCollect() {
    if (!currentSiteId.value) {
      Message.warning('请先选择站点');
      return;
    }
    if (!currentTenantId.value) {
      Message.warning('请先选择租户');
      return;
    }

    const urls = batchUrlsText.value
      .split('\n')
      .map((u: string) => normaliseExternalHttpUrl(u.trim()))
      .filter((u: string) => isValidHttpUrl(u));
    if (!urls.length) {
      Message.warning('请输入有效的网页地址');
      return;
    }

    if (collectorStore.batchStatus === 'idle') {
      collectorStore.resetBatch();
    }

    collectorStore.initBatch(urls, batchDraftType.value);
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

      const url = collectorStore.batchResults[index].url;
      const itemHost = extractHost(url);
      if (skipCollectedUrls.value && itemHost && collectedUrlSet.value.has(url)) {
        collectorStore.updateBatchItem(index, {
          url,
          status: 'success',
          error: '已采集过（跳过）',
        });
        collectorStore.batchProgress.current += 1;
        continue;
      }

      try {
        if (useBrowserRender.value && index > 0) {
          await new Promise((resolve) => setTimeout(resolve, 1200));
        }
        const payload = await collectWebPage(url, {
          useBrowserRender: useBrowserRender.value,
          siteId: currentSiteId.value,
          tenantId: currentTenantId.value,
        });

        const rule = await getCollectorSiteRule(itemHost);
        const refined = rule ? applyCollectorSiteRule(payload, rule) : payload;

        collectorStore.updateBatchItem(index, {
          url,
          title: refined.title,
          status: 'success',
          ruleHost: itemHost || undefined,
        });

        const toolTargetId = buildCollectorDraftId('tool');
        const articleTargetId = buildCollectorDraftId('article');

        if (toolTargetId && currentTenantId.value) {
          await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, 'tool', toolTargetId), {
            tenantId: currentTenantId.value,
            contentType: 'tool',
            targetId: toolTargetId,
            title: toolMapping.title || refined.title || '采集草稿',
            payload: { content: refined.contentHtml, sourceUrl: refined.finalUrl, collectedFrom: url },
          });
        }

        if (articleTargetId && currentTenantId.value) {
          await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, 'article', articleTargetId), {
            tenantId: currentTenantId.value,
            contentType: 'article',
            targetId: articleTargetId,
            title: articleMapping.title || refined.title || '采集草稿',
            payload: { content: refined.contentHtml, sourceUrl: refined.finalUrl, collectedFrom: url },
          });
        }

        collectorStore.updateBatchItem(index, {
          toolDraftId: toolTargetId,
          articleDraftId: articleTargetId,
        });
      } catch {
        collectorStore.updateBatchItem(index, {
          url,
          status: 'failed',
          error: '采集失败',
        });
      }

      collectorStore.batchProgress.current += 1;
    }

    if (collectorStore.batchStatus === 'running') {
      collectorStore.completeBatch();
    }
  }

  function handleBatchPause() {
    collectorStore.pauseBatch();
  }

  function handleBatchResume() {
    collectorStore.startBatch();
  }

  function handleBatchStop() {
    collectorStore.stopBatch();
  }

  function handleBatchReset() {
    collectorStore.resetBatch();
  }

  function handleDeleteHistoryItem(host: string, url: string) {
    deleteCollectedUrl(host, url);
    historyVersion.value += 1;
    Message.success('已删除');
  }

  function handleClearHistory() {
    if (!historyPanelHost.value) {
      return;
    }
    clearCollectedHistory(historyPanelHost.value);
    historyVersion.value += 1;
    Message.success('已清空');
  }

  function toggleHistoryPanel() {
    showHistoryPanel.value = !showHistoryPanel.value;
  }

  async function handleScanSite() {
    if (!currentSiteId.value) {
      Message.warning('请先选择站点');
      return;
    }
    if (!currentTenantId.value) {
      Message.warning('请先选择租户');
      return;
    }

    const url = normaliseExternalHttpUrl(scanUrl.value.trim());
    if (!isValidHttpUrl(url)) {
      Message.warning('请输入有效的网页地址');
      return;
    }

    scanning.value = true;
    try {
      const result = await scanSiteLinks({
        url,
        maxLinks: scanMaxLinks.value,
        scanSitemap: scanSitemap.value,
        siteId: currentSiteId.value,
        tenantId: currentTenantId.value,
        filterRules: scanFilterRules.value,
      });
      scanResult.value = result;
      scannedSelectedLinks.value = new Set(result.links.map((link: SiteLinkItem) => link.url));
      Message.success(`扫描完成，发现 ${result.links.length} 个链接`);
    } catch (error) {
      Message.error(error instanceof Error ? error.message : '扫描失败');
    } finally {
      scanning.value = false;
    }
  }

  function addScanFilterRule() {
    scanFilterRules.value = [...scanFilterRules.value, { field: 'url', operator: 'contains', value: '' }];
  }

  function removeScanFilterRule(index: number) {
    scanFilterRules.value = scanFilterRules.value.filter((_: ScanFilterRule, i: number) => i !== index);
  }

  function updateScanFilterRule(index: number, key: 'field' | 'operator' | 'value', value: string) {
    if (index >= 0 && index < scanFilterRules.value.length) {
      const rules = [...scanFilterRules.value];
      rules[index] = { ...rules[index], [key]: value };
      scanFilterRules.value = rules;
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
    if (!scanResult.value) {
      return;
    }
    const allUrls = new Set(scanResult.value.links.map((link: SiteLinkItem) => link.url));
    if (scannedSelectedLinks.value.size === allUrls.size) {
      scannedSelectedLinks.value = new Set();
    } else {
      scannedSelectedLinks.value = allUrls;
    }
  }

  function sendSelectedToBatch() {
    if (!scannedSelectedLinks.value.size) {
      return;
    }
    const selected = Array.from(scannedSelectedLinks.value);
    const currentText = batchUrlsText.value.trim();
    const newText = currentText ? `${currentText}\n${selected.join('\n')}` : selected.join('\n');
    batchUrlsText.value = newText;
    collectMode.value = 'batch';
    Message.success(`已添加 ${selected.length} 个链接到批量采集`);
  }

  function handleScanReset() {
    scanResult.value = null;
    scannedSelectedLinks.value = new Set();
    scanFilterRules.value = [];
  }

  function applyScanSettingsFromHost(host: string) {
    const settings = loadCollectorScanSettings(host);
    if (!settings) {
      return;
    }
    scanMaxLinks.value = settings.maxLinks ?? 50;
    scanSitemap.value = settings.scanSitemap ?? true;
    scanFilterRules.value = settings.filterRules ?? [];
  }

  function persistScanSettings() {
    const host = extractScanSettingsHost(scanUrl.value);
    if (!host) {
      return;
    }
    saveCollectorScanSettings({
      host,
      maxLinks: scanMaxLinks.value,
      scanSitemap: scanSitemap.value,
      filterRules: scanFilterRules.value,
      updatedAt: '',
    });
  }

  function forgetScanSettings() {
    const host = extractScanSettingsHost(scanUrl.value);
    if (host) {
      deleteCollectorScanSettings(host);
      Message.success('已清除保存的设置');
    }
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
      await saveLocalDraft(buildLocalDraftKey(currentTenantId.value, contentType, targetId), {
        tenantId: currentTenantId.value,
        contentType,
        targetId,
        title: toolMapping.title || result.value.title || '采集草稿',
        payload: {
          content: contentType === 'tool' ? toolMapping.content || result.value.contentHtml : articleMapping.content || result.value.contentHtml,
          sourceUrl: result.value.finalUrl,
        },
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

  return {
    router, siteStore, authStore, collectorStore,
    batchUrlsText, batchResults, batchProgress, batchDraftType,
    sourceUrl, collecting, draftGenerating, previewMode, baseResult, result,
    draftMappingTab, syncingTenants, pickerField, pickerStatus, pickerLastValue,
    currentRule, pickerFrameRef, pickerSelectionSummary, pickerBrokenImageCount,
    collectMode, useBrowserRender, skipCollectedUrls, skippedUrlCount,
    showHistoryPanel, historyVersion, scanning, scanUrl, scanResult,
    scanMaxLinks, scanSitemap, scanFilterRules, scannedSelectedLinks,
    removePickerListeners,
    PICKER_FIELD_OPTIONS, SCAN_OPERATOR_OPTIONS, SCAN_FIELD_OPTIONS,
    toolMapping, articleMapping,
    currentSite, currentTenant, filteredTenants, currentSiteId, currentTenantId,
    resultImages, resultKeywords, resultSeoKeywords, resultSuggestedTags,
    hasResult, imageCount, canCollect, canGenerateDraft, contentLength,
    pickerEnabled, activeRuleHost, hasCurrentRule, hasScanSettings,
    pickerPreviewHtml, quickImageChoices, currentRuleFieldEntries,
    historyPanelHost, collectedHistoryItems, collectedUrlSet,
    pickerFieldDescription, previewHtml, currentScanHost,
    handleSiteChange, handleTenantChange, syncCurrentSiteTenants,
    buildCollectorDraftId, escapeHtmlAttribute, extractHost,
    resolvePickerAssetUrl, normalisePickerSrcset, isPlaceholderImageSrc,
    normalisePickerPreviewHtml, resolvePickerSourceHtml, buildPickerFrameHtml,
    buildCollectedContentHtml, firstContentImageUrl, resolveCollectedThumbnail,
    resolveCollectedOfficialUrl, buildRulePreviewValue, getPickerFieldLabel,
    getPickerSelectionMatchCount, getDraftMappingAnchorId,
    scrollToDraftMappingField, applyQuickImageSelection, resetMappings,
    buildDefaultToolMapping, buildDefaultArticleMapping,
    buildToolDraftPayload, buildArticleDraftPayload, cleanupPickerListeners,
    isHtmlElementLike, markPickerElement, resolvePickerElement,
    escapeSelectorValue, isSelectorClassName, buildSelectorSegment,
    buildUniqueSelector, buildGroupSelector, buildContentSelector,
    buildSelectorForField, collectPickerImageFallbackCandidates,
    markPickerBrokenImages, repairPickerPreviewImages, refreshDisplayedResult,
    ensureCurrentRuleDraft, loadCurrentSiteRule, handlePickerSelection,
    bindPickerFrameEvents, handlePickerFrameLoad, saveCurrentSiteRuleDraft,
    deleteCurrentSiteRuleDraft, handleCollect, handleReset, handleBatchCollect,
    handleBatchPause, handleBatchResume, handleBatchStop, handleBatchReset,
    handleDeleteHistoryItem, handleClearHistory, toggleHistoryPanel,
    handleScanSite, addScanFilterRule, removeScanFilterRule,
    updateScanFilterRule, toggleScannedLink, selectAllScannedLinks,
    sendSelectedToBatch, handleScanReset, applyScanSettingsFromHost,
    persistScanSettings, forgetScanSettings, createDraft,
  };
}
