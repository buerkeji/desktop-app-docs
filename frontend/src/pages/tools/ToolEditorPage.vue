<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import EditorPageShell from '../../components/EditorPageShell.vue';
import MediaUploadField from '../../components/MediaUploadField.vue';
import RichContentEditor from '../../components/RichContentEditor.vue';
import type { ToolUpsertPayload } from '../../types/content';
import { createTool, getToolDetail, updateTool } from '../../services/content.service';
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
  markDraftSubmitted,
  saveLocalDraft,
  type LocalDraftContentType,
} from '../../services/local-draft.service';
import { useAuthStore } from '../../stores/auth.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import { useSiteStore } from '../../stores/site.store';
import { initialiseDesktopContext } from '../../utils/desktop-context';
import { isValidHttpUrl, normaliseExternalHttpUrl, resolveTenantAssetUrl } from '../../utils/url';

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
const toolId = computed(() => String(route.params.id || ''));
const openedFromDraftBox = computed(() => route.query.from === 'drafts');
const toolCategories = computed(() =>
  dictionaryStore.categories.filter((item) => item.type === 'tool_category' && item.enabled),
);
const mediaCategories = computed(() =>
  dictionaryStore.categories.filter((item) => item.type === 'media_category' && item.enabled),
);
const draftTargetId = computed(() => {
  if (isEditMode.value) {
    return toolId.value;
  }

  const queryDraft = typeof route.query.draft === 'string' ? route.query.draft.trim() : '';
  return queryDraft || 'new';
});
const draftKey = computed(() =>
  buildLocalDraftKey(siteStore.currentTenant?.id || 'unknown', 'tool', draftTargetId.value),
);
const draftQuery = computed(() =>
  siteStore.currentTenant?.id
    ? {
        tenantId: Number(siteStore.currentTenant.id),
        contentType: 'tool' as const,
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

  const basePath = '/tools';
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
  url: '',
  mediaCategoryId: undefined as number | undefined,
  icon: '',
  thumbnail: '',
  categoryId: undefined as number | undefined,
  description: '',
  content: '',
  featuresText: '',
  website: '',
  isFeatured: false,
  isActive: true,
  sortOrder: 0,
  metaTitle: '',
  metaKeywords: '',
  metaDescription: '',
  tagsText: '',
});

type ToolEditorFormState = typeof form;

interface ToolEditorDraftPayload {
  form: ToolEditorFormState;
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
    key: buildLocalDraftKey(safeTenantId, 'tool', targetId),
    query: typeof tenantId === 'number'
      ? {
          tenantId,
          contentType: 'tool',
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

  return 'tool';
}

function buildPayload(overrides: Partial<Pick<ToolUpsertPayload, 'icon' | 'thumbnail' | 'content'>> = {}): ToolUpsertPayload {
  const tenantApiBaseUrl = siteStore.currentTenant?.apiBaseUrl;
  const rawIcon = String(overrides.icon ?? form.icon ?? '').trim();
  const rawThumbnail = String(overrides.thumbnail ?? form.thumbnail ?? '').trim();
  const rawContent = String(overrides.content ?? form.content ?? '');
  const resolvedIcon = isDataUrl(rawIcon) ? rawIcon : resolveTenantAssetUrl(rawIcon, tenantApiBaseUrl).trim();
  const resolvedThumbnail = isDataUrl(rawThumbnail)
    ? rawThumbnail
    : resolveTenantAssetUrl(rawThumbnail, tenantApiBaseUrl).trim();

  return {
    title: form.title.trim(),
    slug: form.slug.trim() || undefined,
    url: normaliseExternalHttpUrl(form.url),
    icon: resolvedIcon || undefined,
    thumbnail: resolvedThumbnail || undefined,
    categoryId: form.categoryId,
    description: form.description.trim(),
    content: rawContent.trim() || undefined,
    features: form.featuresText
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean),
    website: normaliseExternalHttpUrl(form.website) || undefined,
    isFeatured: form.isFeatured,
    isActive: form.isActive,
    sortOrder: Number(form.sortOrder) || 0,
    metaTitle: form.metaTitle.trim() || undefined,
    metaKeywords: form.metaKeywords.trim() || undefined,
    metaDescription: form.metaDescription.trim() || undefined,
    tags: form.tagsText
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
  };
}

async function prepareSubmitPayload(): Promise<ToolUpsertPayload> {
  const tenant = { apiBaseUrl: siteStore.currentTenant!.apiBaseUrl };
  const accessToken = authStore.token!.accessToken;
  const prefix = normaliseUploadPrefix(form.slug, form.title, 'tool');

  let icon = String(form.icon || '').trim();
  if (isDataUrl(icon)) {
    const item = await uploadDeferredDataUrl(tenant, accessToken, icon, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'tool-icon',
      fileNamePrefix: `${prefix}-icon`,
    });
    icon = item.url;
    form.icon = item.url;
  } else if (icon) {
    const remoteItem = await uploadRemoteMedia(icon, tenant, accessToken, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'tool-icon',
      fileNamePrefix: `${prefix}-icon`,
    });
    if (remoteItem) {
      icon = remoteItem.url;
      form.icon = remoteItem.url;
    }
  }

  let thumbnail = String(form.thumbnail || '').trim();
  if (isDataUrl(thumbnail)) {
    const item = await uploadDeferredDataUrl(tenant, accessToken, thumbnail, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'tool-thumbnail',
      fileNamePrefix: `${prefix}-thumbnail`,
    });
    thumbnail = item.url;
    form.thumbnail = item.url;
  } else if (thumbnail) {
    const remoteItem = await uploadRemoteMedia(thumbnail, tenant, accessToken, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'tool-thumbnail',
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
    uploadScene: 'tool-content',
    fileNamePrefix: `${prefix}-content`,
  });
  if (contentResult.uploaded.length > 0) {
    content = contentResult.html;
    form.content = contentResult.html;
  }

  if (!isEditMode.value) {
    const remoteResult = await uploadRemoteHtmlImages(content, tenant, accessToken, {
      mediaCategoryId: form.mediaCategoryId,
      uploadScene: 'tool-content',
      fileNamePrefix: `${prefix}-content`,
    });
    if (remoteResult.uploaded.length > 0) {
      content = remoteResult.html;
      form.content = remoteResult.html;
    }
    if (remoteResult.failedCount > 0) {
      Message.warning(`${remoteResult.failedCount} 张远程图片下载失败，已保留原链接`);
    }
  }

  return buildPayload({ icon, thumbnail, content });
}

function ensureDefaultMediaCategory() {
  if (!form.mediaCategoryId && mediaCategories.value.length > 0) {
    form.mediaCategoryId = Number(mediaCategories.value[0].id);
  }
}

function resetForm() {
  form.title = '';
  form.slug = '';
  form.url = '';
  form.mediaCategoryId = undefined;
  form.icon = '';
  form.thumbnail = '';
  form.categoryId = undefined;
  form.description = '';
  form.content = '';
  form.featuresText = '';
  form.website = '';
  form.isFeatured = false;
  form.isActive = true;
  form.sortOrder = 0;
  form.metaTitle = '';
  form.metaKeywords = '';
  form.metaDescription = '';
  form.tagsText = '';
  hasDraft.value = false;
  draftUpdatedAt.value = '';
  autoSaveError.value = '';
}

function getDraftPayload() {
  return {
    form: JSON.parse(JSON.stringify(form)) as ToolEditorFormState,
  } satisfies ToolEditorDraftPayload;
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
  payload: ToolEditorDraftPayload | ToolEditorFormState,
  context = createDraftContext(),
) {
  if ('form' in payload) {
    Object.assign(form, payload.form);
    return;
  }

  Object.assign(form, payload);
}

async function restoreDraft(silent = false, context = createDraftContext()) {
  const draft = await loadLocalDraft<ToolEditorDraftPayload | ToolEditorFormState>(context.key, context.query);
  if (!draft || draft.submittedAt) {
    hasDraft.value = false;
    draftUpdatedAt.value = '';
    if (!silent && !draft) {
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
    const saved = await saveLocalDraft(context.key, {
      tenantId: context.tenantId,
      contentType: 'tool',
      targetId: draftTargetId.value,
      title: form.title.trim() || '未命名工具',
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
  if (!isEditMode.value || !toolId.value) {
    return;
  }
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    return;
  }

  if (!background) {
    loading.value = true;
  }
  try {
    const detail = await getToolDetail(
      {
        apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
      },
      authStore.token.accessToken,
      toolId.value,
    );
    form.title = detail.title;
    form.slug = detail.slug;
    form.url = detail.url || '';
    form.icon = detail.icon || '';
    form.thumbnail = detail.thumbnail || '';
    form.categoryId = detail.categoryId ?? undefined;
    form.description = detail.description || '';
    form.content = detail.content || '';
    form.featuresText = detail.features.map((item) => item.feature).join('\n');
    form.website = detail.website || '';
    form.isFeatured = detail.isFeatured;
    form.isActive = detail.isActive;
    form.sortOrder = detail.sortOrder;
    form.metaTitle = detail.metaTitle || '';
    form.metaKeywords = detail.metaKeywords || '';
    form.metaDescription = detail.metaDescription || '';
    form.tagsText = detail.tags.map((item) => item.name).join(', ');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '获取工具详情失败');
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
  const restoredDraft =
    openedFromDraftBox.value
      ? await restoreDraft(true, context)
      : false;
  const backgroundDetailLoad = openedFromDraftBox.value && restoredDraft;

  await Promise.all([
    initialiseDictionary(),
    loadDetail(backgroundDetailLoad),
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
  if (!form.title.trim() || !form.url.trim() || !form.categoryId || !form.description.trim()) {
    Message.warning('请填写标题、链接、分类和简介');
    return;
  }

  const toolUrl = normaliseExternalHttpUrl(form.url);
  if (!isValidHttpUrl(toolUrl)) {
    Message.warning('工具链接必须是有效的 http 或 https 地址');
    return;
  }

  const websiteUrl = normaliseExternalHttpUrl(form.website);
  if (websiteUrl && !isValidHttpUrl(websiteUrl)) {
    Message.warning('官网地址必须是有效的 http 或 https 地址');
    return;
  }

  saving.value = true;
  try {
    const payload = await prepareSubmitPayload();
    const result = isEditMode.value
      ? await updateTool(
          {
            apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
          },
          authStore.token.accessToken,
          toolId.value,
          payload,
        )
      : await createTool(
          {
            apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
          },
          authStore.token.accessToken,
          payload,
        );
    const submitContext = createDraftContext();
    if (isEditMode.value) {
      await saveLocalDraft(submitContext.key, {
        tenantId: submitContext.tenantId,
        contentType: 'tool',
        targetId: submitContext.targetId,
        title: form.title.trim(),
        payload: getDraftPayload(),
      });
    }
    await markDraftSubmitted(submitContext.key, submitContext.query);
    if (autoSaveTimer) {
      window.clearTimeout(autoSaveTimer);
      autoSaveTimer = undefined;
    }
    autoSaveReady.value = false;
    hasDraft.value = false;
    draftUpdatedAt.value = '';
    initialSnapshot.value = buildEditorSnapshot();
    suppressLeaveGuard.value = true;
    Message.success(isEditMode.value ? '工具已更新' : '工具已创建');
    if (!isEditMode.value) {
      void router.replace(`/tools/${result.id}`);
    } else {
      autoSaveReady.value = true;
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '保存工具失败');
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
    void router.push(openedFromDraftBox.value ? '/drafts' : '/tools');
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

          const sameType = remaining.filter((d) => d.contentType === 'tool');
          if (sameType.length > 0) {
            const nextIdx = idx < remaining.length ? idx : remaining.length - 1;
            const target = remaining[nextIdx];
            if (target.contentType === 'tool') {
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
      void router.replace({ path: '/tools/new', query: { ...route.query, draft: undefined } });
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
    :title="isEditMode ? '编辑工具' : '新建工具'"
    description="当前接入真实创建与更新接口，先覆盖基础字段、标签、特性与正文保存。"
    :draft-updated-at="draftUpdatedAt"
    :has-draft="hasDraft"
    :saving="saving"
    :status-text="editorStatus.text"
    :status-tone="editorStatus.tone"
    :back-text="openedFromDraftBox ? '返回草稿箱' : '返回列表'"
    :submit-text="isEditMode ? '同步工具' : '上传工具'"
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
                <a-input v-model="form.title" placeholder="请输入工具标题" />
              </a-form-item>
              <a-form-item label="Slug">
                <a-input v-model="form.slug" placeholder="可选，留空由服务端处理" />
              </a-form-item>
              <a-form-item label="工具链接" required>
                <a-input v-model="form.url" placeholder="https://tool.example.com" />
              </a-form-item>
              <a-form-item label="简介" required>
                <a-textarea v-model="form.description" :auto-size="{ minRows: 3, maxRows: 6 }" />
              </a-form-item>
              <RichContentEditor
                ref="contentEditorRef"
                v-model="form.content"
                :tenant="siteStore.currentTenant"
                :access-token="authStore.token?.accessToken"
                :media-category-id="form.mediaCategoryId"
                upload-scene="tool-content"
                label="正文"
                placeholder="请输入工具详情、使用说明、适用场景等内容"
                :min-height="380"
              />
            </a-form>
          </a-card>
        </a-col>

        <a-col :span="9" class="editor-layout__side">
          <a-card title="发布设置" class="editor-card">
            <a-form :model="form" layout="vertical">
              <a-form-item label="分类" required>
                <a-select v-model="form.categoryId" allow-clear placeholder="请选择分类">
                  <a-option v-for="item in toolCategories" :key="item.id" :value="item.id">{{ item.label }}</a-option>
                </a-select>
              </a-form-item>
              <a-form-item label="官网地址">
                <a-input v-model="form.website" placeholder="可选" />
              </a-form-item>
              <a-form-item label="媒体分类">
                <a-select v-model="form.mediaCategoryId" allow-clear placeholder="上传时可选媒体分类">
                  <a-option v-for="item in mediaCategories" :key="item.id" :value="item.id">
                    {{ item.label }}
                  </a-option>
                </a-select>
              </a-form-item>
              <a-form-item label="图标">
                <MediaUploadField
                  v-model="form.icon"
                  :tenant="siteStore.currentTenant"
                  :access-token="authStore.token?.accessToken"
                  :media-category-id="form.mediaCategoryId"
                  upload-scene="tool-icon"
                  button-text="上传图标"
                  placeholder="/storage/icon.webp"
                  preview-alt="tool-icon"
                />
              </a-form-item>
              <a-form-item label="缩略图">
                <MediaUploadField
                  v-model="form.thumbnail"
                  :tenant="siteStore.currentTenant"
                  :access-token="authStore.token?.accessToken"
                  :media-category-id="form.mediaCategoryId"
                  upload-scene="tool-thumbnail"
                  button-text="上传缩略图"
                  placeholder="/storage/thumb.webp"
                  preview-alt="tool-thumbnail"
                />
              </a-form-item>
              <a-form-item label="标签">
                <a-input v-model="form.tagsText" placeholder="多个标签用英文逗号分隔" />
              </a-form-item>
              <a-form-item label="特性">
                <a-textarea
                  v-model="form.featuresText"
                  placeholder="每行一个特性"
                  :auto-size="{ minRows: 4, maxRows: 8 }"
                />
              </a-form-item>
              <a-form-item label="排序值">
                <a-input-number v-model="form.sortOrder" :min="0" style="width: 100%" />
              </a-form-item>
              <a-form-item label="状态">
                <a-space>
                  <a-checkbox v-model="form.isActive">上架</a-checkbox>
                  <a-checkbox v-model="form.isFeatured">推荐</a-checkbox>
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
