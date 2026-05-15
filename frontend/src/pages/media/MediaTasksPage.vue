<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import type { TableRowSelection } from '@arco-design/web-vue';
import { clearMediaTaskCache, deleteMediaTask, listMediaTasks, retryMediaTask } from '../../services/media-task.service';
import type { MediaTaskItem, MediaTaskListFilters } from '../../types/activity';
import { useSiteStore } from '../../stores/site.store';
import { useActivityPage } from '../../composables/useActivityPage';
import PageStickyTable from '../../components/PageStickyTable.vue';

const siteStore = useSiteStore();

const filters = reactive<MediaTaskListFilters>({
  tenantId: undefined,
  keyword: '',
  scene: '',
  status: '',
  dateRange: [],
  limit: 500,
});

const retryingTaskId = ref<number | null>(null);
const clearingCacheTaskId = ref<number | null>(null);

const {
  loading,
  detailVisible,
  currentRecord: currentTask,
  rows,
  pagination,
  tenantOptions,
  currentTenantName,
  load,
  handleSearch,
  handleReset,
  openDetail,
} = useActivityPage<MediaTaskItem>({
  label: '媒体任务',
  loadData: () => listMediaTasks(filters),
  resetFilters: () => {
    filters.tenantId = siteStore.currentTenant?.id;
    filters.keyword = '';
    filters.scene = '';
    filters.status = '';
    filters.dateRange = [];
    filters.limit = 500;
  },
});

const selectedKeys = ref<(string | number)[]>([]);
const batchDeleting = ref(false);
const deleting = ref<number | null>(null);

const selectedCount = computed(() => selectedKeys.value.length);
const allRowKeys = computed(() => rows.value.map((r) => r.id));
const keySet = computed(() => new Set(selectedKeys.value));
const isAllSelected = computed(() => rows.value.length > 0 && selectedCount.value === rows.value.length);

const rowSelection = reactive<TableRowSelection>({
  type: 'checkbox',
  showCheckedAll: true,
  selectedRowKeys: selectedKeys.value,
});

watch(selectedKeys, (val) => {
  rowSelection.selectedRowKeys = val;
});

function handleTableSelect(rowKeys: (string | number)[]) {
  selectedKeys.value = rowKeys;
}

function handleTableSelectAll(checked: boolean) {
  selectedKeys.value = checked ? [...allRowKeys.value] : [];
}

function handleSelectAll() {
  selectedKeys.value = isAllSelected.value ? [] : [...allRowKeys.value];
}

function handleDeleteRecord(record: MediaTaskItem) {
  Modal.confirm({
    title: '删除媒体任务？',
    content: `确定删除任务 "${record.fileName || record.originalName || '-'}" 的媒体任务记录？`,
    okText: '删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      deleting.value = record.id;
      try {
        await deleteMediaTask(record.id);
        rows.value = rows.value.filter((r) => r.id !== record.id);
        selectedKeys.value = selectedKeys.value.filter((k) => k !== record.id);
        Message.success('媒体任务已删除');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除媒体任务失败');
      } finally {
        deleting.value = null;
      }
    },
  });
}

function handleBatchDelete() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要删除的媒体任务');
    return;
  }

  Modal.confirm({
    title: `批量删除 ${selectedCount.value} 条媒体任务？`,
    content: `已选择 ${selectedCount.value} 条媒体任务，删除后不可恢复。`,
    okText: '批量删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      const items = rows.value.filter((r) => keySet.value.has(r.id));
      batchDeleting.value = true;
      let successCount = 0;
      let failCount = 0;

      for (const item of items) {
        try {
          await deleteMediaTask(item.id);
          rows.value = rows.value.filter((r) => r.id !== item.id);
          successCount += 1;
        } catch {
          failCount += 1;
        }
      }

      selectedKeys.value = [];
      batchDeleting.value = false;

      if (failCount > 0) {
        Message.warning(`已删除 ${successCount} 条，${failCount} 条删除失败`);
      } else {
        Message.success(`已批量删除 ${successCount} 条媒体任务`);
      }
    },
  });
}

const detailText = computed(() => {
  if (!currentTask.value?.responseJson) {
    return '-';
  }

  try {
    return JSON.stringify(JSON.parse(currentTask.value.responseJson), null, 2);
  } catch {
    return currentTask.value.responseJson;
  }
});

function statusColor(status: string) {
  switch (status) {
    case 'success':
      return 'green';
    case 'failed':
      return 'red';
    default:
      return 'orange';
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'success':
      return '成功';
    case 'failed':
      return '失败';
    case 'pending':
      return '处理中';
    default:
      return status || '-';
  }
}

function sceneLabel(scene?: string) {
  switch (scene) {
    case 'tool-content':
      return '工具正文';
    case 'tool-icon':
      return '工具图标';
    case 'tool-thumbnail':
      return '工具缩略图';
    case 'article-content':
      return '文章正文';
    case 'article-thumbnail':
      return '文章缩略图';
    case 'editor-content':
      return '正文图片';
    case 'field':
      return '表单上传';
    default:
      return scene?.trim() || '-';
  }
}

function formatFileSize(sizeBytes: number) {
  if (!sizeBytes) {
    return '-';
  }
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  if (sizeBytes < 1024 * 1024) {
    return `${(sizeBytes / 1024).toFixed(1)} KB`;
  }
  return `${(sizeBytes / 1024 / 1024).toFixed(2)} MB`;
}

async function copyText(value: string, successMessage: string) {
  if (!value.trim()) {
    Message.warning('当前没有可复制内容');
    return;
  }

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
      Message.success(successMessage);
      return;
    }
  } catch {
    // Fallback to legacy copy flow below.
  }

  const input = document.createElement('input');
  input.value = value;
  document.body.appendChild(input);
  input.select();
  document.execCommand('copy');
  document.body.removeChild(input);
  Message.success(successMessage);
}

function openRemoteUrl(url?: string) {
  const nextUrl = url?.trim();
  if (!nextUrl) {
    Message.warning('当前没有远端地址');
    return;
  }
  window.open(nextUrl, '_blank', 'noopener');
}

async function handleRetry(record: MediaTaskItem) {
  if (!record.id || !record.canRetry) {
    Message.warning('当前任务缺少可用缓存文件，无法重传');
    return;
  }

  retryingTaskId.value = record.id;
  try {
    await retryMediaTask(record.id);
    Message.success('媒体任务已重新提交');
    await load();
    if (currentTask.value?.id === record.id) {
      currentTask.value = {
        ...currentTask.value,
        canRetry: false,
      };
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '媒体任务重传失败');
  } finally {
    retryingTaskId.value = null;
  }
}

async function handleClearCache(record: MediaTaskItem) {
  if (!record.id || !record.canRetry) {
    Message.warning('当前任务没有可清理的本地缓存文件');
    return;
  }

  const confirmed = window.confirm('清理后将无法直接使用该任务的本地缓存进行重传，确认继续吗？');
  if (!confirmed) {
    return;
  }

  clearingCacheTaskId.value = record.id;
  try {
    await clearMediaTaskCache(record.id);
    Message.success('本地缓存文件已清理');
    await load();
    if (currentTask.value?.id === record.id) {
      currentTask.value = {
        ...currentTask.value,
        canRetry: false,
      };
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '清理本地缓存失败');
  } finally {
    clearingCacheTaskId.value = null;
  }
}
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>媒体任务</h2>
        <p>自动记录编辑页中的图片、封面、图标等上传结果，便于排查失败与复用远端地址。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="green">{{ currentTenantName }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="loading" @click="load">刷新列表</a-button>
        </div>
      </div>
    </div>

    <div class="page-content">
      <a-card class="activity-card page-scroll-card">
        <template #extra>
          <a-space>
            <span v-if="selectedCount > 0" style="font-size: 13px; color: #4e5969;">已选 {{ selectedCount }} 条</span>
          </a-space>
        </template>

        <div class="page-filter-form page-filter-form--activity page-filter-form--separated">
        <a-input v-model="filters.keyword" allow-clear placeholder="搜索文件名 / 远端地址 / 错误信息" />
        <a-select v-model="filters.tenantId" allow-clear placeholder="选择租户">
          <a-option v-for="item in tenantOptions" :key="item.id" :value="item.id">{{ item.name }}</a-option>
        </a-select>
        <a-select v-model="filters.status" allow-clear placeholder="任务状态">
          <a-option value="success">成功</a-option>
          <a-option value="failed">失败</a-option>
          <a-option value="pending">处理中</a-option>
        </a-select>
        <a-select v-model="filters.scene" allow-clear placeholder="上传来源">
          <a-option value="tool-content">工具正文</a-option>
          <a-option value="tool-icon">工具图标</a-option>
          <a-option value="tool-thumbnail">工具缩略图</a-option>
          <a-option value="article-content">文章正文</a-option>
          <a-option value="article-thumbnail">文章缩略图</a-option>
        </a-select>
        <a-range-picker v-model="filters.dateRange" value-format="YYYY-MM-DD" />
        <a-space class="page-filter-form__actions">
          <a-button type="primary" @click="handleSearch">查询</a-button>
          <a-button @click="handleReset">重置</a-button>
        </a-space>
      </div>

        <div v-if="selectedCount > 0" class="media-batch-bar">
          <a-space wrap>
            <a-button size="small" @click="handleSelectAll">
              {{ isAllSelected ? '取消全选' : '全选' }}
            </a-button>
            <a-button size="small" status="danger" :disabled="!selectedCount" :loading="batchDeleting" @click="handleBatchDelete">
              批量删除
            </a-button>
          </a-space>
        </div>

        <PageStickyTable :data="rows" :loading="loading" :pagination="pagination" row-key="id" size="small" :row-selection="rowSelection" :selected-keys="selectedKeys" @select="handleTableSelect" @select-all="handleTableSelectAll">
            <template #empty>
              <a-empty description="暂无媒体任务" />
            </template>
            <template #columns>
          <a-table-column title="时间" :width="170">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.createdAt">{{ record.createdAt || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="80">
            <template #cell="{ record }">
              <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="原始文件名" :width="200">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.originalName">{{ record.originalName || record.fileName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="来源" :width="100">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="sceneLabel(record.uploadScene)">{{ sceneLabel(record.uploadScene) }}</span>
            </template>
          </a-table-column>
          <a-table-column title="租户" :width="120">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.tenantName">{{ record.tenantName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="类型" :width="90">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.mimeType">{{ record.mimeType || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="尺寸 / 大小" :width="140">
            <template #cell="{ record }">
              <span>{{ `${record.width || 0}x${record.height || 0} / ${formatFileSize(record.sizeBytes)}` }}</span>
            </template>
          </a-table-column>
          <a-table-column title="远端地址" :width="260">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.remoteUrl">{{ record.remoteUrl || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="错误摘要" :width="200">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.errorMessage">{{ record.errorMessage || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="200" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" @click="openDetail(record)">详情</a-button>
                <a-button
                  v-if="record.status === 'failed'"
                  type="text"
                  size="small"
                  :disabled="!record.canRetry"
                  :loading="retryingTaskId === record.id"
                  @click="void handleRetry(record)"
                >
                  重传
                </a-button>
                <a-button
                  v-if="record.canRetry"
                  type="text"
                  size="small"
                  status="warning"
                  :loading="clearingCacheTaskId === record.id"
                  @click="void handleClearCache(record)"
                >
                  清缓存
                </a-button>
                <a-button size="mini" @click="void copyText(record.remoteUrl || '', '远端地址已复制')">复制地址</a-button>
                <a-button size="mini" @click="openRemoteUrl(record.remoteUrl)">打开</a-button>
                <a-button type="text" size="small" status="danger" :loading="deleting === record.id" @click="handleDeleteRecord(record)">删除</a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>

    <a-modal v-model:visible="detailVisible" title="媒体任务详情" width="760px" :footer="false">
      <a-descriptions v-if="currentTask" :column="2" bordered size="medium">
        <a-descriptions-item label="原始文件名">{{ currentTask.originalName || currentTask.fileName || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态">{{ statusLabel(currentTask.status) }}</a-descriptions-item>
        <a-descriptions-item label="上传来源">{{ sceneLabel(currentTask.uploadScene) }}</a-descriptions-item>
        <a-descriptions-item label="类型">{{ currentTask.mimeType || '-' }}</a-descriptions-item>
        <a-descriptions-item label="请求 ID">{{ currentTask.requestId || '-' }}</a-descriptions-item>
        <a-descriptions-item label="远端媒体 ID">{{ currentTask.remoteMediaId || '-' }}</a-descriptions-item>
        <a-descriptions-item label="来源地址">{{ currentTask.sourceUrl || '-' }}</a-descriptions-item>
        <a-descriptions-item label="草稿 ID">{{ currentTask.draftId || '-' }}</a-descriptions-item>
        <a-descriptions-item label="本地缓存">{{ currentTask.canRetry ? '可用' : '已清理 / 不存在' }}</a-descriptions-item>
        <a-descriptions-item label="尺寸">{{ `${currentTask.width || 0}x${currentTask.height || 0}` }}</a-descriptions-item>
        <a-descriptions-item label="文件大小">{{ formatFileSize(currentTask.sizeBytes) }}</a-descriptions-item>
        <a-descriptions-item :span="2" label="远端地址">{{ currentTask.remoteUrl || '-' }}</a-descriptions-item>
        <a-descriptions-item :span="2" label="错误信息">{{ currentTask.errorMessage || '-' }}</a-descriptions-item>
      </a-descriptions>
      <pre class="activity-json-preview">{{ detailText }}</pre>
    </a-modal>
  </div>
</template>

<style scoped>
.activity-card {
  border-radius: 14px;
}

.table-cell-ellipsis {
  display: block;
  width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.activity-json-preview {
  margin: 16px 0 0;
  padding: 12px;
  max-height: 420px;
  overflow: auto;
  border-radius: 10px;
  background: var(--color-fill-2);
  white-space: pre-wrap;
  word-break: break-word;
}

.media-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 6px;
}
</style>
