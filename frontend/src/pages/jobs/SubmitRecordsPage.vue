<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import type { TableRowSelection } from '@arco-design/web-vue';
import { useSiteStore } from '../../stores/site.store';
import { deleteSubmitRecord, listSubmitRecords } from '../../services/submit-record.service';
import type { SubmitRecordItem, SubmitRecordListFilters } from '../../types/activity';
import { useActivityPage } from '../../composables/useActivityPage';
import PageStickyTable from '../../components/PageStickyTable.vue';

const siteStore = useSiteStore();

const filters = reactive<SubmitRecordListFilters>({
  tenantId: undefined,
  keyword: '',
  contentType: '',
  status: '',
  dateRange: [],
  limit: 500,
});

const {
  loading,
  detailVisible,
  currentRecord,
  rows,
  pagination,
  tenantOptions,
  currentTenantName,
  load,
  handleSearch,
  handleReset,
  openDetail,
} = useActivityPage<SubmitRecordItem>({
  label: '提交记录',
  loadData: () => listSubmitRecords(filters),
  resetFilters: () => {
    filters.tenantId = siteStore.currentTenant?.id;
    filters.keyword = '';
    filters.contentType = '';
    filters.status = '';
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

function handleDeleteRecord(record: SubmitRecordItem) {
  Modal.confirm({
    title: '删除提交记录？',
    content: `确定删除任务 "${record.title || '-'}" 的提交记录？删除后不可恢复。`,
    okText: '删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      deleting.value = record.id;
      try {
        await deleteSubmitRecord(record.jobId);
        rows.value = rows.value.filter((r) => r.id !== record.id);
        selectedKeys.value = selectedKeys.value.filter((k) => k !== record.id);
        Message.success('提交记录已删除');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除提交记录失败');
      } finally {
        deleting.value = null;
      }
    },
  });
}

function handleBatchDelete() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要删除的提交记录');
    return;
  }

  Modal.confirm({
    title: `批量删除 ${selectedCount.value} 条提交记录？`,
    content: `已选择 ${selectedCount.value} 条提交记录，删除后不可恢复。`,
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
          await deleteSubmitRecord(item.jobId);
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
        Message.success(`已批量删除 ${successCount} 条提交记录`);
      }
    },
  });
}

const detailText = computed(() => {
  if (!currentRecord.value) return '';
  const sections = [
    ['请求载荷', prettyJSON(currentRecord.value.payloadJson)],
    ['返回结果', prettyJSON(currentRecord.value.resultJson)],
    ['错误信息', currentRecord.value.errorMessage || '-'],
  ];
  return sections.map(([title, content]) => `${title}\n${content}`).join('\n\n');
});

function prettyJSON(value?: string) {
  if (!value) return '-';
  try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
}

function contentTypeLabel(value: string) {
  return value === 'article' ? '文章' : value === 'tool' ? '工具' : value || '-';
}

function jobTypeLabel(value: string) {
  switch (value) {
    case 'create': return '新建';
    case 'update': return '更新';
    case 'batch_delete': return '批量删除';
    default: return value || '-';
  }
}

function statusColor(value: string) {
  switch (value) {
    case 'success': return 'green';
    case 'partial': return 'orange';
    case 'failed': return 'red';
    default: return 'gray';
  }
}

function statusLabel(value: string) {
  switch (value) {
    case 'success': return '成功';
    case 'partial': return '部分成功';
    case 'failed': return '失败';
    case 'pending': return '处理中';
    default: return value || '-';
  }
}
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>提交记录</h2>
        <p>自动汇总工具、文章等远端写入结果，便于追踪成功、失败和返回数据。</p>
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
        <a-input v-model="filters.keyword" allow-clear placeholder="搜索标题 / 幂等键 / 错误信息" />
        <a-select v-model="filters.tenantId" allow-clear placeholder="选择租户">
          <a-option v-for="item in tenantOptions" :key="item.id" :value="item.id">{{ item.name }}</a-option>
        </a-select>
        <a-select v-model="filters.contentType" allow-clear placeholder="内容类型">
          <a-option value="tool">工具</a-option>
          <a-option value="article">文章</a-option>
        </a-select>
        <a-select v-model="filters.status" allow-clear placeholder="结果状态">
          <a-option value="success">成功</a-option>
          <a-option value="partial">部分成功</a-option>
          <a-option value="failed">失败</a-option>
        </a-select>
        <a-range-picker v-model="filters.dateRange" value-format="YYYY-MM-DD" />
        <a-space class="page-filter-form__actions">
          <a-button type="primary" @click="handleSearch">查询</a-button>
          <a-button @click="handleReset">重置</a-button>
        </a-space>
      </div>

        <div v-if="selectedCount > 0" class="submit-batch-bar">
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
              <a-empty description="暂无提交记录" />
            </template>
            <template #columns>
          <a-table-column title="任务 ID" :width="80">
            <template #cell="{ record }">
              <span>{{ record.jobId }}</span>
            </template>
          </a-table-column>
          <a-table-column title="标题" :width="200">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.title">{{ record.title || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="租户" :width="120">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.tenantName">{{ record.tenantName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="内容类型" :width="80">
            <template #cell="{ record }">
              {{ contentTypeLabel(record.contentType) }}
            </template>
          </a-table-column>
          <a-table-column title="任务类型" :width="80">
            <template #cell="{ record }">
              {{ jobTypeLabel(record.jobType) }}
            </template>
          </a-table-column>
          <a-table-column title="结果" :width="80">
            <template #cell="{ record }">
              <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="结果统计" :width="140">
            <template #cell="{ record }">
              <span>{{ `新建 ${record.createdCount} / 更新 ${record.updatedCount} / 失败 ${record.failedCount}` }}</span>
            </template>
          </a-table-column>
          <a-table-column title="远端 ID" :width="80">
            <template #cell="{ record }">
              {{ record.remoteId || '-' }}
            </template>
          </a-table-column>
          <a-table-column title="错误摘要" :width="200">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.errorMessage">{{ record.errorMessage || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="提交时间" :width="170">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.submittedAt">{{ record.submittedAt || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="120" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" @click="openDetail(record)">详情</a-button>
                <a-button type="text" size="small" status="danger" :loading="deleting === record.id" @click="handleDeleteRecord(record)">删除</a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>

    <a-modal v-model:visible="detailVisible" title="提交详情" width="760px" :footer="false">
      <a-descriptions v-if="currentRecord" :column="2" bordered size="medium">
        <a-descriptions-item label="任务 ID">{{ currentRecord.jobId }}</a-descriptions-item>
        <a-descriptions-item label="幂等键">{{ currentRecord.idempotencyKey || '-' }}</a-descriptions-item>
        <a-descriptions-item label="标题">{{ currentRecord.title || '-' }}</a-descriptions-item>
        <a-descriptions-item label="远端 ID">{{ currentRecord.remoteId || '-' }}</a-descriptions-item>
        <a-descriptions-item label="内容类型">{{ contentTypeLabel(currentRecord.contentType) }}</a-descriptions-item>
        <a-descriptions-item label="操作">{{ jobTypeLabel(currentRecord.jobType) }}</a-descriptions-item>
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

.submit-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 6px;
}
</style>
