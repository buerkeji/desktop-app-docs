<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import type { TableRowSelection } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import {
  buildLocalDraftKey,
  clearLocalDraft,
  isVirtualLocalDraftTarget,
  listLocalDrafts,
  loadLocalDraft,
  saveLocalDraft,
  type LocalDraftListItem,
} from '../../services/local-draft.service';
import type { LocalDraftContentType } from '../../services/local-draft.service';
import { createTool, updateTool, createArticle, updateArticle } from '../../services/content.service';
import {
  uploadDeferredDataUrl,
  uploadDeferredHtmlImages,
  uploadRemoteHtmlImages,
  uploadRemoteMedia,
  isDataUrl,
} from '../../services/deferred-media.service';
import { useSiteStore } from '../../stores/site.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import { useAuthStore } from '../../stores/auth.store';
import PageStickyTable from '../../components/PageStickyTable.vue';
import { createTablePagination } from '../../utils/table-pagination';

const DRAFT_NAV_KEY = 'zq.desktop.draft.nav.list';

const router = useRouter();
const siteStore = useSiteStore();
const dictionaryStore = useDictionaryStore();
const authStore = useAuthStore();

const loading = ref(false);
const deleting = ref<string | null>(null);
const batchDeleting = ref(false);
const batchCategoryChanging = ref(false);
const batchCategoryModalVisible = ref(false);
const batchCategoryForm = reactive({ categoryId: undefined as number | undefined });
const batchUploading = ref(false);
const batchUploadModalVisible = ref(false);
const batchUploadProgress = reactive({ current: 0, total: 0 });
const drafts = ref<Array<LocalDraftListItem>>([]);
const selectedKeys = ref<string[]>([]);
const contentTypeFilter = ref<LocalDraftContentType | ''>('');
const keywordFilter = ref('');
const statusFilter = ref<'submitted' | 'unsubmitted' | ''>('');
const dateRangeFilter = ref<string[]>([]);
const pagination = createTablePagination();

const currentTenantId = computed(() => siteStore.currentTenant?.id ?? null);
const currentTenantName = computed(() => siteStore.currentTenant?.name || '未选择');

function handleResetFilters() {
  contentTypeFilter.value = '';
  keywordFilter.value = '';
  statusFilter.value = '';
  dateRangeFilter.value = [];
}

const filteredDrafts = computed(() => {
  let items = drafts.value;

  if (contentTypeFilter.value) {
    items = items.filter((d) => d.contentType === contentTypeFilter.value);
  }

  const kw = keywordFilter.value.trim().toLowerCase();
  if (kw) {
    items = items.filter((d) => (d.title || '').toLowerCase().includes(kw));
  }

  if (statusFilter.value === 'submitted') {
    items = items.filter((d) => (d as any).submittedAt);
  } else if (statusFilter.value === 'unsubmitted') {
    items = items.filter((d) => !(d as any).submittedAt);
  }

  if (dateRangeFilter.value.length === 2) {
    const from = dateRangeFilter.value[0];
    const to = dateRangeFilter.value[1];
    items = items.filter((d) => d.updatedAt >= from && d.updatedAt <= to + 'T23:59:59');
  }

  return items;
});

const selectedCount = computed(() => selectedKeys.value.length);
const allFilteredKeys = computed(() => new Set(filteredDrafts.value.map((d) => d.key)));
const isAllSelected = computed(() => filteredDrafts.value.length > 0 && selectedCount.value === filteredDrafts.value.length);

const rowSelection = reactive<TableRowSelection>({
  type: 'checkbox',
  showCheckedAll: true,
  selectedRowKeys: selectedKeys.value,
});

watch(selectedKeys, (val) => {
  rowSelection.selectedRowKeys = val;
});

function handleTableSelect(rowKeys: (string | number)[], _rowKey: string | number, _record: LocalDraftListItem) {
  selectedKeys.value = rowKeys.map(String);
}

function handleTableSelectAll(checked: boolean) {
  selectedKeys.value = checked ? [...allFilteredKeys.value] : [];
}

const warmArticleEditor = () => import('../articles/ArticleEditorPage.vue');
const warmToolEditor = () => import('../tools/ToolEditorPage.vue');

function getContentTypeLabel(type: LocalDraftContentType) {
  return type === 'tool' ? '工具' : '文章';
}

function resolveEditRoute(item: LocalDraftListItem, index: number) {
  const basePath = item.contentType === 'tool' ? '/tools' : '/articles';
  const baseQuery: Record<string, string> = {
    from: 'drafts',
    draftNavIndex: String(index),
  };

  if (isVirtualLocalDraftTarget(item.targetId)) {
    return {
      path: `${basePath}/new`,
      query: {
        ...baseQuery,
        draft: item.targetId,
      },
    };
  }

  return {
    path: item.targetId === 'new' ? `${basePath}/new` : `${basePath}/${item.targetId}/edit`,
    query: baseQuery,
  };
}

function warmEditor(contentType: LocalDraftContentType) {
  void (contentType === 'tool' ? warmToolEditor() : warmArticleEditor());
}

async function loadDraftItems() {
  if (!currentTenantId.value) {
    drafts.value = [];
    return;
  }

  loading.value = true;
  try {
    const items = await listLocalDrafts({
      tenantId: currentTenantId.value,
    });
    const seen = new Set<string>();
    drafts.value = items.filter((d) => {
      if (seen.has(d.key)) return false;
      seen.add(d.key);
      return d.targetId === 'new' || isVirtualLocalDraftTarget(d.targetId);
    });
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '获取草稿列表失败');
  } finally {
    loading.value = false;
  }
}

function openDraft(item: LocalDraftListItem) {
  const navList = filteredDrafts.value
    .filter((d) => d.contentType === item.contentType)
    .map((d) => ({
      key: d.key,
      contentType: d.contentType,
      targetId: d.targetId,
      title: d.title || '',
    }));
  sessionStorage.setItem(DRAFT_NAV_KEY, JSON.stringify(navList));
  const index = navList.findIndex((d) => d.key === item.key);
  warmEditor(item.contentType);
  void router.push(resolveEditRoute(item, index));
}

function openNewDraft(contentType: LocalDraftContentType) {
  if (!currentTenantId.value) {
    Message.warning('请先选择租户后再新建内容');
    return;
  }

  warmEditor(contentType);
  void router.push({
    path: contentType === 'tool' ? '/tools/new' : '/articles/new',
    query: {
      from: 'drafts',
    },
  });
}

function handleDelete(item: LocalDraftListItem) {
  if (!currentTenantId.value) {
    return;
  }

  Modal.confirm({
    title: '删除本地草稿？',
    content: '删除后不可恢复，本地保存的编辑内容将被清空。',
    okText: '删除',
    okButtonProps: {
      status: 'danger',
    },
    cancelText: '取消',
    onOk: async () => {
      deleting.value = item.key;
      try {
        await clearLocalDraft(buildLocalDraftKey(currentTenantId.value!, item.contentType, item.targetId), {
          tenantId: currentTenantId.value!,
          contentType: item.contentType,
          targetId: item.targetId,
        });
        drafts.value = drafts.value.filter((draft) => draft.key !== item.key);
        selectedKeys.value = selectedKeys.value.filter((k) => k !== item.key);
        Message.success('草稿已删除');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除草稿失败');
      } finally {
        deleting.value = null;
      }
    },
  });
}

function handleSelectAll() {
  selectedKeys.value = isAllSelected.value ? [] : [...allFilteredKeys.value];
}

function handleBatchDelete() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要删除的草稿');
    return;
  }

  Modal.confirm({
    title: `批量删除 ${selectedCount.value} 个草稿？`,
    content: `已选择 ${selectedCount.value} 个草稿，删除后不可恢复。`,
    okText: '批量删除',
    okButtonProps: {
      status: 'danger',
    },
    cancelText: '取消',
    onOk: async () => {
      const keySet = new Set(selectedKeys.value);
      const itemsToDelete = drafts.value.filter((d) => keySet.has(d.key));

      batchDeleting.value = true;
      let successCount = 0;
      let failCount = 0;

      for (const item of itemsToDelete) {
        try {
          const dq = buildDraftQuery(item);
          if (!dq) continue;
          await clearLocalDraft(buildLocalDraftKey(currentTenantId.value!, item.contentType, item.targetId), dq);
          drafts.value = drafts.value.filter((draft) => draft.key !== item.key);
          successCount += 1;
        } catch {
          failCount += 1;
        }
      }

      selectedKeys.value = [];
      batchDeleting.value = false;

      if (failCount > 0) {
        Message.warning(`已删除 ${successCount} 个草稿，${failCount} 个删除失败`);
      } else {
        Message.success(`已批量删除 ${successCount} 个草稿`);
      }
    },
  });
}

function setContentTypeFilter(type: LocalDraftContentType | '') {
  contentTypeFilter.value = contentTypeFilter.value === type ? '' : type;
  selectedKeys.value = [];
}

const allCategories = computed(() => dictionaryStore.categories.filter((item) => item.enabled));

const toolCategories = computed(() =>
  allCategories.value.filter((item) => item.type === 'tool_category'),
);
const articleCategories = computed(() =>
  allCategories.value.filter((item) => item.type === 'article_category'),
);

const categoryLabelMap = computed(() => {
  const map = new Map<number, string>();
  for (const cat of allCategories.value) {
    if (cat.enabled) {
      map.set(cat.id, cat.label);
    }
  }
  return map;
});

const draftCategoryName = (draft: LocalDraftListItem): string => {
  const payload = (draft as any).payload;
  if (!payload) return '';
  const categoryId = payload?.form?.categoryId ?? payload?.categoryId;
  if (categoryId === undefined || categoryId === null) return '';
  return categoryLabelMap.value.get(categoryId) || '';
};

const draftSubmittedAt = (draft: LocalDraftListItem): string => {
  return (draft as any).submittedAt || '';
};

const unSubmittedSelectedKeys = computed(() => {
  const keySet = new Set(selectedKeys.value);
  return drafts.value.filter((d) => keySet.has(d.key) && !(d as any).submittedAt);
});

const batchCategoryMode = computed<LocalDraftContentType | null>(() => {
  if (contentTypeFilter.value) return contentTypeFilter.value;
  const keySet = new Set(selectedKeys.value);
  const selected = drafts.value.filter((d) => keySet.has(d.key));
  if (!selected.length) return null;
  const allTool = selected.every((d) => d.contentType === 'tool');
  const allArticle = selected.every((d) => d.contentType === 'article');
  if (allTool) return 'tool';
  if (allArticle) return 'article';
  return null;
});

const availableCategoryOptions = computed(() => {
  if (batchCategoryMode.value === 'tool') return toolCategories.value;
  if (batchCategoryMode.value === 'article') return articleCategories.value;
  return [];
});

function buildDraftQuery(item: LocalDraftListItem): { tenantId: number; contentType: LocalDraftContentType; targetId: string } | undefined {
  if (!currentTenantId.value) return undefined;
  return {
    tenantId: currentTenantId.value,
    contentType: item.contentType,
    targetId: item.targetId,
  };
}

function openBatchCategoryModal() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要设置分类的草稿');
    return;
  }
  if (!batchCategoryMode.value) {
    Message.warning('只支持对同一类型（工具或文章）的草稿批量设置分类，请确保勾选的草稿类型一致');
    return;
  }
  batchCategoryForm.categoryId = undefined;
  batchCategoryModalVisible.value = true;
}

async function confirmBatchCategory() {
  if (batchCategoryForm.categoryId === undefined) {
    Message.warning('请选择分类');
    return;
  }

  const keySet = new Set(selectedKeys.value);
  const itemsToUpdate = drafts.value.filter((d) => keySet.has(d.key));

  batchCategoryChanging.value = true;
  let successCount = 0;
  let failCount = 0;

  for (const item of itemsToUpdate) {
    try {
      const key = buildLocalDraftKey(currentTenantId.value!, item.contentType, item.targetId);
      const query = buildDraftQuery(item);
      const existing = await loadLocalDraft<Record<string, unknown>>(key, query);
      if (existing) {
        const payload = existing.payload;
        if (payload && typeof payload === 'object') {
          if ('form' in payload && payload.form && typeof payload.form === 'object') {
            (payload.form as Record<string, unknown>).categoryId = batchCategoryForm.categoryId;
          } else {
            payload.categoryId = batchCategoryForm.categoryId;
          }
        }
        await saveLocalDraft(key, existing);
      }
      successCount += 1;
    } catch {
      failCount += 1;
    }
  }

  await loadDraftItems();
  batchCategoryModalVisible.value = false;
  batchCategoryChanging.value = false;

  if (failCount > 0) {
    Message.warning(`已为 ${successCount} 个草稿设置分类，${failCount} 个设置失败`);
  } else {
    Message.success(`已为 ${successCount} 个草稿批量设置分类`);
  }
}

function openBatchUploadModal() {
  const unsubmitted = unSubmittedSelectedKeys.value;
  if (!unsubmitted.length) {
    Message.warning('所选草稿均已上传，无需重复上传');
    return;
  }
  batchUploadProgress.current = 0;
  batchUploadProgress.total = unsubmitted.length;
  batchUploadModalVisible.value = true;
}

async function confirmBatchUpload() {
  const unsubmitted = unSubmittedSelectedKeys.value;
  if (!unsubmitted.length) return;

  const tenant = siteStore.currentTenant;
  const token = authStore.token;
  if (!tenant?.apiBaseUrl || !token?.accessToken) {
    Message.warning('缺少租户上下文或登录态，无法上传');
    return;
  }

  batchUploading.value = true;
  let successCount = 0;
  let failCount = 0;
  let totalFailedRemoteImages = 0;

  for (const item of unsubmitted) {
    batchUploadProgress.current += 1;
    try {
      const key = buildLocalDraftKey(currentTenantId.value!, item.contentType, item.targetId);
      const query = buildDraftQuery(item);
      const existing = await loadLocalDraft<{ form: Record<string, unknown> }>(key, query);
      if (!existing || !existing.payload) {
        console.warn('[BatchUpload] 草稿数据不存在', item.title || item.key);
        failCount += 1;
        continue;
      }

      const rawForm = existing.payload.form || (existing.payload as Record<string, unknown>);
      const form = { ...rawForm };

      const missingFields: string[] = [];
      if (!form.title) missingFields.push('标题');
      if (item.contentType === 'tool') {
        if (!form.url) missingFields.push('链接');
        if (!form.description) missingFields.push('简介');
      }
      if (item.contentType === 'article' && !form.content) missingFields.push('正文');
      if (!form.categoryId) missingFields.push('分类');

      if (missingFields.length > 0) {
        console.warn('[BatchUpload]', `"${item.title || '未命名草稿'}"缺少必填字段：${missingFields.join('、')}`);
        failCount += 1;
        continue;
      }

      const scenePrefix = item.contentType === 'tool' ? 'tool' : 'article';

      let icon = String(form.icon || '').trim();
      if (isDataUrl(icon)) {
        const result = await uploadDeferredDataUrl(
          { apiBaseUrl: tenant.apiBaseUrl },
          token.accessToken,
          icon,
          { uploadScene: `${scenePrefix}-icon`, fileNamePrefix: `${scenePrefix}-batch-icon` },
        );
        icon = result.url;
        form.icon = result.url;
      } else if (icon) {
        const result = await uploadRemoteMedia(icon, { apiBaseUrl: tenant.apiBaseUrl }, token.accessToken, {
          uploadScene: `${scenePrefix}-icon`,
          fileNamePrefix: `${scenePrefix}-batch-icon`,
        });
        if (result) {
          icon = result.url;
          form.icon = result.url;
        }
      }

      let thumbnail = String(form.thumbnail || '').trim();
      if (isDataUrl(thumbnail)) {
        const result = await uploadDeferredDataUrl(
          { apiBaseUrl: tenant.apiBaseUrl },
          token.accessToken,
          thumbnail,
          { uploadScene: `${scenePrefix}-thumbnail`, fileNamePrefix: `${scenePrefix}-batch-thumb` },
        );
        thumbnail = result.url;
        form.thumbnail = result.url;
      } else if (thumbnail) {
        const result = await uploadRemoteMedia(thumbnail, { apiBaseUrl: tenant.apiBaseUrl }, token.accessToken, {
          uploadScene: `${scenePrefix}-thumbnail`,
          fileNamePrefix: `${scenePrefix}-batch-thumb`,
        });
        if (result) {
          thumbnail = result.url;
          form.thumbnail = result.url;
        }
      }

      let content = String(form.content || '');
      const deferredResult = await uploadDeferredHtmlImages(content, { apiBaseUrl: tenant.apiBaseUrl }, token.accessToken, {
        uploadScene: `${scenePrefix}-content`,
        fileNamePrefix: `${scenePrefix}-batch-content`,
      });
      if (deferredResult.uploaded.length > 0) {
        content = deferredResult.html;
        form.content = deferredResult.html;
      }
      const contentResult = await uploadRemoteHtmlImages(content, { apiBaseUrl: tenant.apiBaseUrl }, token.accessToken, {
        uploadScene: `${scenePrefix}-content`,
        fileNamePrefix: `${scenePrefix}-batch-content`,
      });
      if (contentResult.uploaded.length > 0) {
        content = contentResult.html;
        form.content = contentResult.html;
      }
      totalFailedRemoteImages += contentResult.failedCount;

      const payload: Record<string, unknown> = { ...form, tags: String(form.tagsText || '').split(',').filter(Boolean) };
      if (payload.icon === '') payload.icon = undefined;
      if (payload.thumbnail === '') payload.thumbnail = undefined;

      if (item.targetId === 'new' || isVirtualLocalDraftTarget(item.targetId)) {
        if (item.contentType === 'tool') {
          await createTool(
            { apiBaseUrl: tenant.apiBaseUrl },
            token.accessToken,
            payload as any,
          );
        } else {
          await createArticle(
            { apiBaseUrl: tenant.apiBaseUrl },
            token.accessToken,
            payload as any,
          );
        }
      } else {
        if (item.contentType === 'tool') {
          await updateTool(
            { apiBaseUrl: tenant.apiBaseUrl },
            token.accessToken,
            item.targetId,
            payload as any,
          );
        } else {
          await updateArticle(
            { apiBaseUrl: tenant.apiBaseUrl },
            token.accessToken,
            item.targetId,
            payload as any,
          );
        }
      }

      await clearLocalDraft(key, query);
      successCount += 1;
    } catch (error) {
      console.error('[BatchUpload]', item.title || item.key, error);
      failCount += 1;
    }
  }

  await loadDraftItems();
  selectedKeys.value = [];
  batchUploadModalVisible.value = false;
  batchUploading.value = false;

  let message = `已上传 ${successCount} 个草稿，${failCount} 个上传失败`;
  if (totalFailedRemoteImages > 0) {
    message += `，${totalFailedRemoteImages} 张远程图片下载失败（已保留原链接）`;
  }
  if (failCount > 0) {
    message += `，请确认草稿的必填字段（工具：标题+链接+分类+简介；文章：标题+分类+正文）完整`;
    Message.warning(message);
  } else {
    Message.success(message);
  }
}

onMounted(async () => {
  await siteStore.initialise();
  await loadDraftItems();
  const tenant = siteStore.currentTenant;
  const token = authStore.token;
  if (tenant?.apiBaseUrl) {
    void dictionaryStore.initialise(
      { id: tenant.id, apiBaseUrl: tenant.apiBaseUrl },
      token?.accessToken,
      false,
    );
  }
});

watch(
  () => siteStore.currentTenant?.id,
  () => {
    selectedKeys.value = [];
    contentTypeFilter.value = '';
    void loadDraftItems();
  },
);
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>草稿箱</h2>
        <p>集中管理当前租户下的本地工具草稿与文章草稿，可继续编辑、批量设置分类、批量上传或删除。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="arcoblue">当前租户：{{ currentTenantName }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="loading" @click="loadDraftItems">刷新列表</a-button>
          <a-button
            :type="contentTypeFilter === 'tool' ? 'primary' : 'outline'"
            size="small"
            @click="setContentTypeFilter('tool')"
          >
            工具 ({{ drafts.filter(d => d.contentType === 'tool').length }})
          </a-button>
          <a-button
            :type="contentTypeFilter === 'article' ? 'primary' : 'outline'"
            size="small"
            @click="setContentTypeFilter('article')"
          >
            文章 ({{ drafts.filter(d => d.contentType === 'article').length }})
          </a-button>
          <a-button
            type="primary"
            :disabled="!currentTenantId"
            @mouseenter="warmEditor('article')"
            @click="openNewDraft('article')"
          >
            新建文章
          </a-button>
          <a-button
            type="primary"
            :disabled="!currentTenantId"
            @mouseenter="warmEditor('tool')"
            @click="openNewDraft('tool')"
          >
            新建工具
          </a-button>
        </div>
      </div>
    </div>

    <div class="page-content">
      <a-alert
        v-if="!currentTenantId"
        type="warning"
        title="尚未选择租户"
        content="请先在左侧切换到目标租户，草稿箱会按租户隔离展示本地草稿。"
      />

      <a-card v-else title="本地草稿列表" class="page-scroll-card">
        <template #extra>
          <a-space>
            <span v-if="selectedCount > 0" style="font-size: 13px; color: #4e5969;">已选 {{ selectedCount }} 条</span>
          </a-space>
        </template>

        <div class="page-filter-form page-filter-form--separated">
          <a-input v-model="keywordFilter" allow-clear placeholder="搜索草稿标题" />
          <a-select v-model="contentTypeFilter" allow-clear placeholder="内容类型">
            <a-option value="tool">工具</a-option>
            <a-option value="article">文章</a-option>
          </a-select>
          <a-select v-model="statusFilter" allow-clear placeholder="草稿状态">
            <a-option value="unsubmitted">本地草稿</a-option>
            <a-option value="submitted">已上传</a-option>
          </a-select>
          <a-range-picker v-model="dateRangeFilter" value-format="YYYY-MM-DD" />
          <a-space class="page-filter-form__actions">
            <a-button type="primary" @click="selectedKeys = []">查询</a-button>
            <a-button @click="handleResetFilters">重置</a-button>
          </a-space>
        </div>

        <div v-if="selectedCount > 0" class="draft-batch-bar">
           <a-space wrap>
             <a-button
               size="small"
               @click="handleSelectAll"
             >
               {{ isAllSelected ? '取消全选' : '全选' }}
             </a-button>
             <a-button
               size="small"
               :disabled="!selectedCount"
               :loading="batchCategoryChanging"
               @click="openBatchCategoryModal"
             >
               {{ batchCategoryMode ? '设置分类' : '批量分类' }}
             </a-button>
             <a-button
               size="small"
               type="primary"
               :disabled="!unSubmittedSelectedKeys.length"
               :loading="batchUploading"
               @click="openBatchUploadModal"
             >
               上传 ({{ unSubmittedSelectedKeys.length }})
             </a-button>
             <a-button
               size="small"
               status="danger"
               :disabled="!selectedCount"
               :loading="batchDeleting"
               @click="handleBatchDelete"
             >
               批量删除
             </a-button>
           </a-space>
         </div>
        <PageStickyTable :data="filteredDrafts" :loading="loading" :pagination="pagination" row-key="key" :row-selection="rowSelection" :selected-keys="selectedKeys" @select="handleTableSelect" @select-all="handleTableSelectAll">
            <template #empty>
              <a-empty description="当前租户下还没有本地草稿，可在工具或文章编辑页先保存草稿。" />
            </template>
            <template #columns>
          <a-table-column title="草稿标题" :width="220">
            <template #cell="{ record }">
              <div class="draft-title-cell">
                <strong>{{ record.title || '未命名草稿' }}</strong>
                <span class="section-muted">{{ record.targetId === 'new' ? '新建内容草稿' : `目标 ID：${record.targetId}` }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="内容类型" :width="108">
            <template #cell="{ record }">
              <a-tag :color="record.contentType === 'tool' ? 'green' : 'purple'">
                {{ getContentTypeLabel(record.contentType) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="分类" :width="120">
            <template #cell="{ record }">
              <span v-if="draftCategoryName(record)">{{ draftCategoryName(record) }}</span>
              <span v-else style="color: #c9cdd4;">-</span>
            </template>
          </a-table-column>
          <a-table-column title="草稿状态" :width="130">
            <template #cell="{ record }">
              <a-tag v-if="draftSubmittedAt(record)" color="green">已上传</a-tag>
              <a-tag v-else color="orange">本地草稿</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="最近更新" data-index="updatedAt" :width="180" />
          <a-table-column title="操作" :width="136" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button
                  type="text"
                  size="small"
                  @mouseenter="warmEditor(record.contentType)"
                  @click="openDraft(record)"
                >
                  编辑
                </a-button>
                <a-button
                  type="text"
                  size="small"
                  status="danger"
                  :loading="deleting === record.key"
                  @click="handleDelete(record)"
                >
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>

    <a-modal
      v-model:visible="batchCategoryModalVisible"
      :title="`批量设置分类 (${selectedCount} 个草稿)`"
      :ok-loading="batchCategoryChanging"
      ok-text="确认设置"
      cancel-text="取消"
      @ok="confirmBatchCategory"
    >
      <a-form layout="vertical" :model="batchCategoryForm">
        <a-form-item label="目标分类">
          <a-select
            v-model="batchCategoryForm.categoryId"
            :style="{ width: '100%' }"
            placeholder="请选择分类"
            allow-clear
          >
            <a-option
              v-for="item in availableCategoryOptions"
              :key="item.id"
              :value="item.id"
            >
              {{ item.label }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-alert type="info" style="margin-top: 12px;">
          将把已选中的 {{ selectedCount }} 个「{{ batchCategoryMode === 'tool' ? '工具' : '文章' }}」草稿的分类统一设置为所选分类。
          该操作会读取并修改对应草稿的本地存储内容。
        </a-alert>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="batchUploadModalVisible"
      title="批量上传草稿"
      :ok-loading="batchUploading"
      ok-text="开始上传"
      cancel-text="取消"
      @ok="confirmBatchUpload"
    >
      <template v-if="!batchUploading">
        <a-alert type="info" style="margin-bottom: 16px;">
          将上传 {{ batchUploadProgress.total }} 个未上传的草稿到服务端。
          上传前请确保草稿必填字段完整（工具：标题+链接+分类+简介；文章：标题+分类+正文）。
        </a-alert>
        <a-table
          :data="unSubmittedSelectedKeys.slice(0, 10)"
          :pagination="false"
          :scroll="{ maxHeight: 300 }"
          row-key="key"
        >
          <a-table-column title="草稿标题" data-index="title" :width="200">
            <template #cell="{ record }">
              {{ record.title || '未命名草稿' }}
            </template>
          </a-table-column>
          <a-table-column title="类型" :width="80">
            <template #cell="{ record }">
              <a-tag :color="record.contentType === 'tool' ? 'green' : 'purple'" size="small">
                {{ getContentTypeLabel(record.contentType) }}
              </a-tag>
            </template>
          </a-table-column>
        </a-table>
        <div v-if="unSubmittedSelectedKeys.length > 10" style="margin-top: 8px; color: #86909c; font-size: 13px;">
          还有 {{ unSubmittedSelectedKeys.length - 10 }} 条草稿未显示
        </div>
      </template>
      <template v-else>
        <div style="text-align: center; padding: 24px 0;">
          <a-progress
            :percent="Math.round((batchUploadProgress.current / batchUploadProgress.total) * 100)"
            style="max-width: 400px; margin: 0 auto;"
          />
          <p style="margin-top: 12px; color: #4e5969;">
            正在上传 {{ batchUploadProgress.current }}/{{ batchUploadProgress.total }}
          </p>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<style scoped>
.draft-title-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.draft-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 6px;
}
</style>
