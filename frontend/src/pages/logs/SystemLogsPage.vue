<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import type { TableRowSelection } from '@arco-design/web-vue';
import { useSiteStore } from '../../stores/site.store';
import { clearAllSystemLogs, deleteSystemLog, listSystemLogs } from '../../services/system-log.service';
import type { SystemLogItem, SystemLogListFilters } from '../../types/activity';
import { useActivityPage } from '../../composables/useActivityPage';
import PageStickyTable from '../../components/PageStickyTable.vue';

const siteStore = useSiteStore();

const filters = reactive<SystemLogListFilters>({
  tenantId: undefined,
  keyword: '',
  level: '',
  module: '',
  requestId: '',
  dateRange: [],
  limit: 500,
});

const {
  loading,
  detailVisible,
  currentRecord: currentLog,
  rows,
  pagination,
  tenantOptions,
  currentTenantName,
  load,
  handleSearch,
  handleReset,
  openDetail,
} = useActivityPage<SystemLogItem>({
  label: '系统日志',
  loadData: () => listSystemLogs(filters),
  resetFilters: () => {
    filters.tenantId = siteStore.currentTenant?.id;
    filters.keyword = '';
    filters.level = '';
    filters.module = '';
    filters.requestId = '';
    filters.dateRange = [];
    filters.limit = 500;
  },
});

const selectedKeys = ref<(string | number)[]>([]);
const batchDeleting = ref(false);
const deleting = ref<number | null>(null);

const filteredRows = computed(() => rows.value);
const selectedCount = computed(() => selectedKeys.value.length);
const allRowKeys = computed(() => filteredRows.value.map((r) => r.id));
const keySet = computed(() => new Set(selectedKeys.value));
const isAllSelected = computed(() => filteredRows.value.length > 0 && selectedCount.value === filteredRows.value.length);

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

function handleDeleteLog(record: SystemLogItem) {
  Modal.confirm({
    title: '删除日志？',
    content: `确定删除 ID 为 ${record.id} 的日志？删除后不可恢复。`,
    okText: '删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      deleting.value = record.id;
      try {
        await deleteSystemLog(record.id);
        rows.value = rows.value.filter((r) => r.id !== record.id);
        selectedKeys.value = selectedKeys.value.filter((k) => k !== record.id);
        Message.success('日志已删除');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除日志失败');
      } finally {
        deleting.value = null;
      }
    },
  });
}

function handleBatchDelete() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要删除的日志');
    return;
  }

  Modal.confirm({
    title: `批量删除 ${selectedCount.value} 条日志？`,
    content: `已选择 ${selectedCount.value} 条日志，删除后不可恢复。`,
    okText: '批量删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      const items = filteredRows.value.filter((r) => keySet.value.has(r.id));
      batchDeleting.value = true;
      let successCount = 0;
      let failCount = 0;

      for (const item of items) {
        try {
          await deleteSystemLog(item.id);
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
        Message.success(`已批量删除 ${successCount} 条日志`);
      }
    },
  });
}

function handleClearAll() {
  Modal.confirm({
    title: '清空所有日志？',
    content: '此操作将删除全部系统日志记录，删除后不可恢复。',
    okText: '清空全部',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      try {
        await clearAllSystemLogs();
        rows.value = [];
        selectedKeys.value = [];
        Message.success('已清空全部系统日志');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '清空日志失败');
      }
    },
  });
}

const moduleOptions = ['content.tool', 'content.article', 'media.upload', 'auth', 'session', 'bootstrap', 'desktop.api'];

const detailText = computed(() => {
  if (!currentLog.value?.contextJson) {
    return '-';
  }
  try {
    return JSON.stringify(JSON.parse(currentLog.value.contextJson), null, 2);
  } catch {
    return currentLog.value.contextJson;
  }
});

function levelColor(level: string) {
  switch (level) {
    case 'error': return 'red';
    case 'warn': return 'orange';
    case 'debug': return 'gray';
    default: return 'arcoblue';
  }
}

function levelLabel(level: string) {
  switch (level) {
    case 'error': return '错误';
    case 'warn': return '警告';
    case 'debug': return '调试';
    default: return '信息';
  }
}
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>系统日志</h2>
        <p>记录远端接口调用、媒体上传和失败诊断信息，支持按租户、模块、请求 ID 快速筛查。</p>
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

        <div class="page-filter-form page-filter-form--logs page-filter-form--separated">
        <a-input v-model="filters.keyword" allow-clear placeholder="搜索日志消息或上下文" />
        <a-select v-model="filters.tenantId" allow-clear placeholder="选择租户">
          <a-option v-for="item in tenantOptions" :key="item.id" :value="item.id">{{ item.name }}</a-option>
        </a-select>
        <a-select v-model="filters.level" allow-clear placeholder="日志级别">
          <a-option value="info">信息</a-option>
          <a-option value="warn">警告</a-option>
          <a-option value="error">错误</a-option>
          <a-option value="debug">调试</a-option>
        </a-select>
        <a-select v-model="filters.module" allow-clear placeholder="模块">
          <a-option v-for="item in moduleOptions" :key="item" :value="item">{{ item }}</a-option>
        </a-select>
        <a-input v-model="filters.requestId" allow-clear placeholder="请求 ID / 幂等键" />
        <a-range-picker v-model="filters.dateRange" value-format="YYYY-MM-DD" />
        <a-space class="page-filter-form__actions">
          <a-button type="primary" @click="handleSearch">查询</a-button>
          <a-button @click="handleReset">重置</a-button>
        </a-space>
      </div>

        <div v-if="selectedCount > 0" class="log-batch-bar">
          <a-space wrap>
            <a-button size="small" @click="handleSelectAll">
              {{ isAllSelected ? '取消全选' : '全选' }}
            </a-button>
            <a-button size="small" status="danger" :disabled="!selectedCount" :loading="batchDeleting" @click="handleBatchDelete">
              批量删除
            </a-button>
            <a-button size="small" status="danger" @click="handleClearAll">
              清空全部
            </a-button>
          </a-space>
        </div>

        <PageStickyTable :data="rows" :loading="loading" :pagination="pagination" row-key="id" size="small" :row-selection="rowSelection" :selected-keys="selectedKeys" @select="handleTableSelect" @select-all="handleTableSelectAll">
            <template #empty>
              <a-empty description="暂无系统日志" />
            </template>
            <template #columns>
          <a-table-column title="时间" :width="170">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.createdAt">{{ record.createdAt || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="级别" :width="80">
            <template #cell="{ record }">
              <a-tag :color="levelColor(record.level)">{{ levelLabel(record.level) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="模块" :width="140">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.module">{{ record.module || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="请求 ID" :width="170">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.requestId">{{ record.requestId || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="租户" :width="120">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.tenantName">{{ record.tenantName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="日志消息" :width="360">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.message">{{ record.message || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="120" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" @click="openDetail(record)">详情</a-button>
                <a-button type="text" size="small" status="danger" :loading="deleting === record.id" @click="handleDeleteLog(record)">删除</a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>

    <a-modal v-model:visible="detailVisible" title="日志详情" width="760px" :footer="false">
      <a-descriptions v-if="currentLog" :column="2" bordered size="medium">
        <a-descriptions-item label="时间">{{ currentLog.createdAt }}</a-descriptions-item>
        <a-descriptions-item label="级别">{{ levelLabel(currentLog.level) }}</a-descriptions-item>
        <a-descriptions-item label="模块">{{ currentLog.module }}</a-descriptions-item>
        <a-descriptions-item label="请求 ID">{{ currentLog.requestId || '-' }}</a-descriptions-item>
        <a-descriptions-item :span="2" label="消息">{{ currentLog.message }}</a-descriptions-item>
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

.log-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 6px;
}
</style>
