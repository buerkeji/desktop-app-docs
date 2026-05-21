<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import EditorPageShell from '../../components/EditorPageShell.vue';
import MediaUploadField from '../../components/MediaUploadField.vue';
import RichContentEditor from '../../components/RichContentEditor.vue';
import type { ArticleUpsertPayload } from '../../types/content';
import { createArticle, getArticleDetail, updateArticle } from '../../services/content.service';
import {
  isDataUrl,
  uploadDeferredDataUrl,
  uploadDeferredHtmlImages,
  uploadRemoteHtmlImages,
  uploadRemoteMedia,
} from '../../services/deferred-media.service';
import {
  buildLocalDraftKey,
  clearLocalDraft,
  isVirtualLocalDraftTarget,
  loadLocalDraft,
  saveLocalDraft,
  type LocalDraftContentType,
} from '../../services/local-draft.service';
import { useAuthStore } from '../../stores/auth.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import { useSiteStore } from '../../stores/site.store';
import { initialiseDesktopContext } from '../../utils/desktop-context';
import { resolveTenantAssetUrl } from '../../utils/url';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const siteStore = useSiteStore();
const dictionaryStore = useDictionaryStore();

const loading = ref(false);
const saving = ref(false);
const draftUpdatedAt = ref('');
const hasDraft = ref(false);
const autoSaveReady = ref(false);
const autoSaving = ref(false);
const autoSaveError = ref('');
const initialSnapshot = ref('');
const suppressLeaveGuard = ref(false);
const contextWatchReady = ref(false);
let autoSaveTimer: number | undefined;
const contentEditorRef = ref<InstanceType<typeof RichContentEditor> | null>(null);

const isEditMode = computed(() => Boolean(route.params.id));
const articleId = computed(() => String(route.params.id || ''));
const openedFromDraftBox = computed(() => route.query.from === 'drafts');
const articleCategories = computed(() =>
  dictionaryStore.categories.filter((item) => item.type === 'article_category' && item.enabled),
);
const mediaCategories = computed(() =>
  dictionaryStore.categories.filter((item) => item.type === 'media_category' && item.enabled),
);
const draftTargetId = computed(() => {
  if (isEditMode.value) {
    return articleId.value;
  }

  const queryDraft = typeof route.query.draft === 'string' ? route.query.draft.trim() : '';
  return queryDraft || 'new';
});
const draftKey = computed(() =>
  buildLocalDraftKey(siteStore.currentTenant?.id || 'unknown', 'article', draftTargetId.value),
);
const draftQuery = computed(() =>
  siteStore.currentTenant?.id
    ? {
        tenantId: Number(siteStore.currentTenant.id),
        contentType: 'article' as const,
        targetId: draftTargetId.value,
      }
    : undefined,
);

const DRAFT_NAV_KEY = 'zq.desktop.draft.nav.list';
const draftNavIndex = computed(() => {
  const raw = route.query.draftNavIndex;
  const idx = typeof raw === 'string' ? parseInt(raw, 10) : NaN;
  return isNaN(idx) ? undefined : idx;
});
const draftNavList = ref<Array<{ key: string; contentType: string; targetId: string; title: string }>>([]);
const draftNavTotal = computed(() => draftNavList.value.length);

function loadDraftNavList() {
  try {
    const raw = sessionStorage.getItem(DRAFT_NAV_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        draftNavList.value = parsed;
        return;
      }
    }
  } catch { /* ignore */ }
  draftNavList.value = [];
}

function navigateToDraft(offset: number) {
  const idx = draftNavIndex.value;
  if (idx === undefined) return;

  if (saving.value) {
    Message.warning('正在提交中，请稍候...');
    return;
  }

  if (autoSaveTimer) {
    window.clearTimeout(autoSaveTimer);
    autoSaveTimer = undefined;
  }
  autoSaveReady.value = false;

  loadDraftNavList();
  const nextIdx = idx + offset;
  if (nextIdx < 0 || nextIdx >= draftNavList.value.length) return;
  const target = draftNavList.value[nextIdx];
  if (!target) return;

  const basePath = '/articles';
  const query: Record<string, string> = {
    from: 'drafts',
    draftNavIndex: String(nextIdx),
  };
  if (isVirtualLocalDraftTarget(target.targetId)) {
    query.draft = target.targetId;
  }

  const path = target.targetId === 'new' || isVirtualLocalDraftTarget(target.targetId)
    ? `${basePath}/new`
    : `${basePath}/${target.targetId}/edit`;

  void router.push({ path, query });
}

const form = reactive({
  title: '',
  slug: '',
  categoryId: undefined as number | undefined,
  mediaCategoryId: undefined as number | undefined,
  thumbnail: '',
  excerpt: '',
  content: '',
  authorId: undefined as number | undefined,
  metaTitle: '',
  metaKeywords: '',
  metaDescription: '',
  isFeatured: false,
  isPinned: false,
  isPublished: true,
  publishedAt: '',
  tagsText: '',
});

type ArticleEditorFormState = typeof form;

interface ArticleEditorDraftPayload {
  form: ArticleEditorFormState;
}

interface DraftStorageContext {
  tenantId: number | string;
  targetId: string;
  key: string;
  query?: {
    tenantId: number;
    contentType: LocalDraftContentType;
    targetId: string;
  };
}

const lastDraftContext = ref<DraftStorageContext | null>(null);

function createDraftContext(
  tenantId: number | string | null | undefined = siteStore.currentTenant?.id,
  targetId = draftTargetId.value,
): DraftStorageContext {
  const safeTenantId = tenantId || 'unknown';
  return {
    tenantId: safeTenantId,
    targetId,
    key: buildLocalDraftKey(safeTenantId, 'article', targetId),
    query: typeof tenantId === 'number'
      ? {
          tenantId,
          contentType: 'article',
          targetId,
        }
      : undefined,
  };
}

function normaliseUploadPrefix(...parts: Array<string | undefined>): string {
  for (const part of parts) {
    const normalized = (part || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
    if (normalized) {
      return normalized;
    }
  }

  return 'article';
}

function formatPublishedAt(value: string): string | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;

  // Treat null-date sentinels as unset.
  if (/^0{4}[-\/]0{2}[-\/]0{2}/.test(trimmed)) {
    return undefined;
  }

  // Normalise separators: replace slash, Chinese dash, etc. with standard dash.
  let normalised = trimmed.replace(/[\/\\．]/g, '-');

  // Split date and time parts.
  const parts = normalised.split(' ');
  const datePart = parts[0];
  const timePart = parts.length > 1 ? parts[1] : '00:00:00';

  // Try to match YYYY-MM-DD or YYYY/MM/DD with optional leading zeros.
  const dateMatch = datePart.match(/^(\d{1,4})[-\/](\d{1,2})[-\/](\d{1,2})$/);
  if (dateMatch) {
    const pad = (n: string) => String(parseInt(n, 10)).padStart(2, '0');
    const paddedDate = `${dateMatch[1]}-${pad(dateMatch[2])}-${pad(dateMatch[3])}`;

    // Normalise time part.
    const timeMatch = timePart.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?/);
    if (timeMatch) {
      const hh = String(parseInt(timeMatch[1], 10)).padStart(2, '0');
      const mm = timeMatch[2];
      const ss = timeMatch[3] || '00';
      return `${paddedDate} ${hh}:${mm}:${ss}`;
    }
    return `${paddedDate} 00:00:00`;
  }

  // ISO 8601 → YYYY-MM-DD HH:mm:ss
  const d = new Date(trimmed);
  if (!isNaN(d.getTime())) {
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  }

  // Fallback: return as-is for backend validation.
  return trimmed;
}

function buildPayload(overrides: Partial<Pick<ArticleUpsertPayload, 'thumbnail' | 'content'>> = {}): ArticleUpsertPayload {
  const rawThumbnail = String(overrides.thumbnail ?? form.thumbnail ?? '').trim();
  const rawContent = String(overrides.content ?? form.content ?? '');
  const resolvedThumbnail = isDataUrl(rawThumbnail)
    ? rawThumbnail
    : resolveTenantAssetUrl(rawThumbnail, siteStore.currentTenant?.apiBaseUrl).trim();

  return {
    title: form.title.trim(),
    slug: form.slug.trim() || undefined,
    categoryId: form.categoryId,
    thumbnail: resolvedThumbnail || undefined,
    excerpt: form.excerpt.trim() || undefined,
    content: rawContent.trim(),
    authorId: form.authorId,
    metaTitle: form.metaTitle.trim() || undefined,
    metaKeywords: form.metaKeywords.trim() || undefined,
    metaDescription: form.metaDescription.trim() || undefined,
    isFeatured: form.isFeatured,
    isPinned: form.isPinned,
    isPublished: form.isPublished,
    publishedAt: formatPublishedAt(form.publishedAt),
    tags: form.tagsText
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
  };
}

async function prepareSubmitPayload(): Promise<ArticleUpsertPayload> {
  const tenant = { apiBaseUrl: siteStore.currentTenant!.apiBaseUrl };
  const accessToken = authStore.token!.accessToken;
  const prefix = normaliseUploadPrefix(form.slug, form.title, 'article');

  let thumbnail = String(form.thumbnail || '').trim();
  if (isDataUrl(thumbnail)) {
    const item = await uploadDeferredDataUrl(tenant, accessToken, thumbnail, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'article-thumbnail',
      fileNamePrefix: `${prefix}-thumbnail`,
    });
    thumbnail = item.url;
    form.thumbnail = item.url;
  } else if (thumbnail) {
    const remoteItem = await uploadRemoteMedia(thumbnail, tenant, accessToken, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'article-thumbnail',
      fileNamePrefix: `${prefix}-thumbnail`,
    });
    if (remoteItem) {
      thumbnail = remoteItem.url;
      form.thumbnail = remoteItem.url;
    }
  }

  const latestEditorHtml = contentEditorRef.value?.getCurrentHtml();
  if (typeof latestEditorHtml === 'string') {
    form.content = latestEditorHtml;
  }

  let content = String(form.content || '');
  const contentResult = await uploadDeferredHtmlImages(content, tenant, accessToken, {
    mediaCategoryId: form.mediaCategoryId,
    uploadScene: 'article-content',
    fileNamePrefix: `${prefix}-content`,
  });
  if (contentResult.uploaded.length > 0) {
    content = contentResult.html;
    form.content = contentResult.html;
  }

  const remoteResult = await uploadRemoteHtmlImages(content, tenant, accessToken, {
    mediaCategoryId: form.mediaCategoryId,
    uploadScene: 'article-content',
    fileNamePrefix: `${prefix}-content`,
  });
  if (remoteResult.uploaded.length > 0) {
    content = remoteResult.html;
    form.content = remoteResult.html;
  }
  if (remoteResult.failedCount > 0) {
    Message.warning(`${remoteResult.failedCount} 张远程图片下载失败，已保留原链接`);
  }

  return buildPayload({ thumbnail, content });
}

function ensureDefaultMediaCategory() {
  if (!form.mediaCategoryId && mediaCategories.value.length > 0) {
    form.mediaCategoryId = Number(mediaCategories.value[0].id);
  }
}

function resetForm() {
  form.title = '';
  form.slug = '';
  form.categoryId = undefined;
  form.mediaCategoryId = undefined;
  form.thumbnail = '';
  form.excerpt = '';
  form.content = '';
  form.authorId = undefined;
  form.metaTitle = '';
  form.metaKeywords = '';
  form.metaDescription = '';
  form.isFeatured = false;
  form.isPinned = false;
  form.isPublished = true;
  form.publishedAt = '';
  form.tagsText = '';
  hasDraft.value = false;
  draftUpdatedAt.value = '';
  autoSaveError.value = '';
}

function getDraftPayload() {
  return {
    form: JSON.parse(JSON.stringify(form)) as ArticleEditorFormState,
  } satisfies ArticleEditorDraftPayload;
}

function buildEditorSnapshot() {
  return JSON.stringify(getDraftPayload());
}

const hasPendingChanges = computed(
  () => autoSaveReady.value && buildEditorSnapshot() !== initialSnapshot.value,
);
const editorStatus = computed(() => {
  if (saving.value) {
    return { text: '正在提交到服务端', tone: 'neutral' as const };
  }
  if (autoSaving.value) {
    return { text: '正在自动保存草稿', tone: 'neutral' as const };
  }
  if (autoSaveError.value) {
    return { text: autoSaveError.value, tone: 'danger' as const };
  }
  if (hasPendingChanges.value) {
    return {
      text: hasDraft.value ? '有未提交修改，草稿将自动更新' : '有未提交修改',
      tone: 'neutral' as const,
    };
  }
  if (draftUpdatedAt.value) {
    return { text: '本地草稿已保存', tone: 'neutral' as const };
  }
  return { text: '自动草稿已开启', tone: 'neutral' as const };
});

function applyDraftPayload(
  payload: ArticleEditorDraftPayload | ArticleEditorFormState,
  context = createDraftContext(),
) {
  if ('form' in payload) {
    Object.assign(form, payload.form);
    return;
  }

  Object.assign(form, payload);
}

async function restoreDraft(silent = false, context = createDraftContext()) {
  const draft = await loadLocalDraft<ArticleEditorDraftPayload | ArticleEditorFormState>(context.key, context.query);
  if (!draft) {
    hasDraft.value = false;
    draftUpdatedAt.value = '';
    if (!silent) {
      Message.warning('当前没有可恢复的本地草稿');
    }
    return false;
  }

  applyDraftPayload(draft.payload, context);
  draftUpdatedAt.value = draft.updatedAt;
  hasDraft.value = true;
  if (!silent) {
    Message.success('已恢复本地草稿');
  }
  return true;
}

async function persistDraft(silent = true, context = createDraftContext()) {
  autoSaving.value = silent;
  autoSaveError.value = '';
  try {
    const latestEditorHtml = contentEditorRef.value?.getCurrentHtml();
    if (typeof latestEditorHtml === 'string') {
      form.content = latestEditorHtml;
    }
    const saved = await saveLocalDraft(context.key, {
      tenantId: context.tenantId,
      contentType: 'article',
      targetId: draftTargetId.value,
      title: form.title.trim() || '未命名文章',
      payload: getDraftPayload(),
    });
    draftUpdatedAt.value = saved.updatedAt;
    hasDraft.value = true;
    if (!silent) {
      Message.success('本地草稿已保存');
    }
  } catch (error) {
    autoSaveError.value = '本地草稿保存失败';
    if (!silent) {
      Message.error(error instanceof Error ? error.message : '本地草稿保存失败');
    }
  } finally {
    autoSaving.value = false;
  }
}

async function clearDraft(silent = false, context = createDraftContext()) {
  await clearLocalDraft(context.key, context.query);
  hasDraft.value = false;
  draftUpdatedAt.value = '';
  autoSaveError.value = '';
  if (!silent) {
    Message.success('本地草稿已清空');
  }
}

async function loadDetail(background = false) {
  if (!isEditMode.value || !articleId.value) {
    return;
  }
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    return;
  }

  if (!background) {
    loading.value = true;
  }
  try {
    const detail = await getArticleDetail(
      {
        apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
      },
      authStore.token.accessToken,
      articleId.value,
    );
    form.title = detail.title;
    form.slug = detail.slug;
    form.categoryId = detail.categoryId ?? undefined;
    form.thumbnail = detail.thumbnail || '';
    form.excerpt = detail.excerpt || '';
    form.content = detail.content || '';
    form.authorId = detail.authorId ?? undefined;
    form.metaTitle = detail.metaTitle || '';
    form.metaKeywords = detail.metaKeywords || '';
    form.metaDescription = detail.metaDescription || '';
    form.isFeatured = detail.isFeatured;
    form.isPinned = detail.isPinned;
    form.isPublished = detail.isPublished;
    form.publishedAt = detail.publishedAt || '';
    form.tagsText = detail.tags.map((item) => item.name).join(', ');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '获取文章详情失败');
  } finally {
    if (!background) {
      loading.value = false;
    }
  }
}

async function initialiseDictionary() {
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    dictionaryStore.$patch({
      categories: [],
      tags: [],
      loaded: false,
      currentTenantId: null,
    });
    return;
  }

  await dictionaryStore.initialise(
    {
      id: siteStore.currentTenant.id,
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    },
    authStore.token.accessToken,
    false,
  );
}

async function hydrateEditorContext() {
  autoSaveReady.value = false;
  if (autoSaveTimer) {
    window.clearTimeout(autoSaveTimer);
    autoSaveTimer = undefined;
  }

  resetForm();
  const context = createDraftContext();

  await Promise.all([
    initialiseDictionary(),
    loadDetail(false),
  ]);
  ensureDefaultMediaCategory();
  await restoreDraft(true, context);

  initialSnapshot.value = buildEditorSnapshot();
  autoSaveReady.value = true;
  lastDraftContext.value = context;
}

async function handleSubmit() {
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    Message.warning('当前缺少租户上下文或登录态');
    return;
  }
  if (!form.title.trim() || !form.categoryId || !form.content.trim()) {
    Message.warning('请填写标题、分类和正文');
    return;
  }

  saving.value = true;
  try {
    const payload = await prepareSubmitPayload();
    const result = isEditMode.value
      ? await updateArticle(
          {
            apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
          },
          authStore.token.accessToken,
          articleId.value,
          payload,
        )
      : await createArticle(
          {
            apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
          },
          authStore.token.accessToken,
          payload,
        );
    const submitContext = createDraftContext();
    await clearLocalDraft(submitContext.key, submitContext.query);
    if (autoSaveTimer) {
      window.clearTimeout(autoSaveTimer);
      autoSaveTimer = undefined;
    }
    autoSaveReady.value = false;
    hasDraft.value = false;
    draftUpdatedAt.value = '';
    initialSnapshot.value = buildEditorSnapshot();
    suppressLeaveGuard.value = true;
    Message.success(isEditMode.value ? '文章已更新' : '文章已创建');
    if (!isEditMode.value) {
      void router.replace(`/articles/${result.id}`);
    } else {
      form.title = result.title;
      form.slug = result.slug;
      form.categoryId = result.categoryId ?? undefined;
      form.thumbnail = result.thumbnail || '';
      form.excerpt = result.excerpt || '';
      form.content = result.content || '';
      form.authorId = result.authorId ?? undefined;
      form.metaTitle = result.metaTitle || '';
      form.metaKeywords = result.metaKeywords || '';
      form.metaDescription = result.metaDescription || '';
      form.isFeatured = result.isFeatured;
      form.isPinned = result.isPinned;
      form.isPublished = result.isPublished;
      form.publishedAt = result.publishedAt || '';
      form.tagsText = result.tags.map((item) => item.name).join(', ');
      initialSnapshot.value = buildEditorSnapshot();
      hasDraft.value = false;
      draftUpdatedAt.value = '';
      autoSaveReady.value = true;
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '保存文章失败');
  } finally {
    saving.value = false;
  }
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasPendingChanges.value || suppressLeaveGuard.value) {
    return;
  }

  event.preventDefault();
  event.returnValue = '';
}

function confirmLeave(onConfirm: () => void) {
  if (!hasPendingChanges.value || suppressLeaveGuard.value) {
    onConfirm();
    return;
  }

  Modal.confirm({
    title: '离开当前编辑页？',
    content: hasDraft.value
      ? '当前还有未提交到服务端的修改。本地草稿已自动保存，但离开后本次编辑不会立即发布。'
      : '当前还有未保存修改，离开后将丢失本次编辑内容。',
    okText: '仍然离开',
    cancelText: '继续编辑',
    onOk: () => {
      suppressLeaveGuard.value = true;
      onConfirm();
    },
  });
}

function handleBack() {
  confirmLeave(() => {
    void router.push(openedFromDraftBox.value ? '/drafts' : '/articles');
  });
}

function handleDeleteDraft() {
  Modal.confirm({
    title: '删除当前草稿？',
    content: '删除后不可恢复，并将跳转到下一条草稿（如有）。',
    okText: '删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      try {
        const context = createDraftContext();
        await clearLocalDraft(context.key, context.query);
        hasDraft.value = false;
        draftUpdatedAt.value = '';
        Message.success('草稿已删除');

        if (autoSaveTimer) {
          window.clearTimeout(autoSaveTimer);
          autoSaveTimer = undefined;
        }
        autoSaveReady.value = false;

        const idx = draftNavIndex.value;
        const list = draftNavList.value;
        if (openedFromDraftBox.value && idx !== undefined && list.length > 0) {
          const remaining = list.filter((_, i) => i !== idx);
          sessionStorage.setItem(DRAFT_NAV_KEY, JSON.stringify(remaining));

          const sameType = remaining.filter((d) => d.contentType === 'article');
          if (sameType.length > 0) {
            const nextIdx = idx < remaining.length ? idx : remaining.length - 1;
            const target = remaining[nextIdx];
            if (target.contentType === 'article') {
              void navigateToDraft(nextIdx - idx);
              return;
            }
          }
        }

        void router.push('/drafts');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除草稿失败');
      }
    },
  });
}

watch(
  () => route.query.draft,
  (value) => {
    if (isEditMode.value) {
      return;
    }

    if (value == null) {
      return;
    }

    if (typeof value !== 'string' || !isVirtualLocalDraftTarget(value.trim())) {
      void router.replace({ path: '/articles/new', query: { ...route.query, draft: undefined } });
    }
  },
  { immediate: true },
);

onMounted(async () => {
  loadDraftNavList();
  window.addEventListener('beforeunload', handleBeforeUnload);
  await initialiseDesktopContext(siteStore, authStore);
  await hydrateEditorContext();
  contextWatchReady.value = true;
});

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload);
});

onBeforeRouteLeave((_to, _from, next) => {
  if (!hasPendingChanges.value || suppressLeaveGuard.value) {
    next();
    return;
  }

  Modal.confirm({
    title: '离开当前编辑页？',
    content: hasDraft.value
      ? '当前还有未提交到服务端的修改。本地草稿已自动保存，但离开后本次编辑不会立即发布。'
      : '当前还有未保存修改，离开后将丢失本次编辑内容。',
    okText: '仍然离开',
    cancelText: '继续编辑',
    onOk: () => {
      suppressLeaveGuard.value = true;
      next();
    },
    onCancel: () => next(false),
  });
});

watch(
  [form],
  () => {
    if (!autoSaveReady.value) {
      return;
    }
    if (autoSaveTimer) {
      window.clearTimeout(autoSaveTimer);
    }
    autoSaveTimer = window.setTimeout(() => {
      void persistDraft(true);
    }, 800);
  },
  { deep: true },
);

watch(
  () => [siteStore.currentTenant?.id ?? null, authStore.token?.accessToken ?? null, draftTargetId.value],
  async (current, previous) => {
    if (!contextWatchReady.value) {
      return;
    }

    const [currentTenantId, currentAccessToken, currentTargetId] = current;
    const [previousTenantId, previousAccessToken, previousTargetId] = previous;
    if (
      currentTenantId === previousTenantId
      && currentAccessToken === previousAccessToken
      && currentTargetId === previousTargetId
    ) {
      return;
    }

    const previousContext = lastDraftContext.value;
    const shouldPersistPreviousDraft = Boolean(previousContext && hasPendingChanges.value);
    if (shouldPersistPreviousDraft && previousContext) {
      await persistDraft(true, previousContext);
    }

    await hydrateEditorContext();

    if (shouldPersistPreviousDraft) {
      Message.info('已自动保存上一编辑上下文的本地草稿，并切换到当前租户');
    }
  },
);
</script>

<template>
  <EditorPageShell
    :title="isEditMode ? '编辑文章' : '新建文章'"
    description="当前接入真实创建与更新接口，先覆盖基础字段、标签、正文与发布状态保存。"
    :draft-updated-at="draftUpdatedAt"
    :has-draft="hasDraft"
    :saving="saving"
    :status-text="editorStatus.text"
    :status-tone="editorStatus.tone"
    :back-text="openedFromDraftBox ? '返回草稿箱' : '返回列表'"
    :submit-text="isEditMode ? '同步文章' : '上传文章'"
    :draft-nav-index="openedFromDraftBox ? draftNavIndex : undefined"
    :draft-nav-total="openedFromDraftBox ? draftNavTotal : undefined"
    @save-draft="void persistDraft(false)"
    @clear-draft="void clearDraft()"
    @back="handleBack"
    @submit="handleSubmit"
    @prev-draft="navigateToDraft(-1)"
    @next-draft="navigateToDraft(1)"
    @delete-draft="handleDeleteDraft"
  >
    <a-spin :loading="loading" style="width: 100%">
      <a-row :gutter="16">
        <a-col :span="15" class="editor-layout__main">
          <a-card title="基础内容" class="editor-card">
            <a-form :model="form" layout="vertical">
              <a-form-item label="标题" required>
                <a-input v-model="form.title" placeholder="请输入文章标题" />
              </a-form-item>
              <a-form-item label="Slug">
                <a-input v-model="form.slug" placeholder="可选，留空由服务端处理" />
              </a-form-item>
              <a-form-item label="摘要">
                <a-textarea v-model="form.excerpt" :auto-size="{ minRows: 3, maxRows: 6 }" />
              </a-form-item>
              <RichContentEditor
                ref="contentEditorRef"
                v-model="form.content"
                :tenant="siteStore.currentTenant"
                :access-token="authStore.token?.accessToken"
                :media-category-id="form.mediaCategoryId"
                upload-scene="article-content"
                label="正文"
                placeholder="请输入文章正文内容"
                :required="true"
                :min-height="420"
              />
            </a-form>
          </a-card>
        </a-col>

        <a-col :span="9" class="editor-layout__side">
          <a-card title="发布设置" class="editor-card">
            <a-form :model="form" layout="vertical">
              <a-form-item label="分类" required>
                <a-select v-model="form.categoryId" allow-clear placeholder="请选择分类">
                  <a-option
                    v-for="item in articleCategories"
                    :key="item.id"
                    :value="item.id"
                  >
                    {{ item.label }}
                  </a-option>
                </a-select>
              </a-form-item>
              <a-form-item label="作者 ID">
                <a-input-number v-model="form.authorId" :min="1" style="width: 100%" />
              </a-form-item>
              <a-form-item label="媒体分类">
                <a-select v-model="form.mediaCategoryId" allow-clear placeholder="上传时可选媒体分类">
                  <a-option v-for="item in mediaCategories" :key="item.id" :value="item.id">
                    {{ item.label }}
                  </a-option>
                </a-select>
              </a-form-item>
              <a-form-item label="缩略图">
                <MediaUploadField
                  v-model="form.thumbnail"
                  :tenant="siteStore.currentTenant"
                  :access-token="authStore.token?.accessToken"
                  :media-category-id="form.mediaCategoryId"
                  upload-scene="article-thumbnail"
                  button-text="上传缩略图"
                  placeholder="/storage/article-thumb.webp"
                  preview-alt="article-thumbnail"
                />
              </a-form-item>
              <a-form-item label="发布时间">
                <a-input v-model="form.publishedAt" placeholder="2026-04-26 10:30:00" />
              </a-form-item>
              <a-form-item label="标签">
                <a-input v-model="form.tagsText" placeholder="多个标签用英文逗号分隔" />
              </a-form-item>
              <a-form-item label="状态">
                <a-space direction="vertical" fill>
                  <a-checkbox v-model="form.isPublished">发布</a-checkbox>
                  <a-checkbox v-model="form.isFeatured">推荐</a-checkbox>
                  <a-checkbox v-model="form.isPinned">置顶</a-checkbox>
                </a-space>
              </a-form-item>
            </a-form>
          </a-card>

          <a-card title="SEO" class="editor-card">
            <a-form :model="form" layout="vertical">
              <a-form-item label="SEO 标题">
                <a-input v-model="form.metaTitle" />
              </a-form-item>
              <a-form-item label="SEO 关键词">
                <a-input v-model="form.metaKeywords" />
              </a-form-item>
              <a-form-item label="SEO 描述">
                <a-textarea v-model="form.metaDescription" :auto-size="{ minRows: 3, maxRows: 6 }" />
              </a-form-item>
            </a-form>
          </a-card>
        </a-col>
      </a-row>
    </a-spin>
  </EditorPageShell>
</template>
