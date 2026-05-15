<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import type { TableRowSelection } from '@arco-design/web-vue';
import { deleteCollectedUrl } from '../../services/collector-history.service';
import PageStickyTable from '../../components/PageStickyTable.vue';
import { createTablePagination } from '../../utils/table-pagination';

interface CollectorRecord {
  url: string;
  title: string;
  host: string;
  fetchedAt: string;
}

const loading = ref(false);
const allRecords = ref<CollectorRecord[]>([]);

const keywordFilter = ref('');
const hostFilter = ref('');
const dateRange = ref<string[]>([]);
const pagination = createTablePagination();

const hostOptions = computed(() => {
  const seen = new Set<string>();
  const hosts: string[] = [];
  for (const r of allRecords.value) {
    if (!seen.has(r.host)) {
      seen.add(r.host);
      hosts.push(r.host);
    }
  }
  return hosts.sort();
});

const filteredRecords = computed(() => {
  let items = allRecords.value;
  const kw = keywordFilter.value.trim().toLowerCase();
  if (kw) {
    items = items.filter((r) => r.url.toLowerCase().includes(kw) || r.title.toLowerCase().includes(kw));
  }
  if (hostFilter.value) {
    items = items.filter((r) => r.host === hostFilter.value);
  }
  if (dateRange.value.length === 2) {
    const from = dateRange.value[0];
    const to = dateRange.value[1];
    items = items.filter((r) => r.fetchedAt >= from && r.fetchedAt <= to + 'T23:59:59');
  }
  return items;
});

const selectedKeys = ref<(string | number)[]>([]);
const batchDeleting = ref(false);

const selectedCount = computed(() => selectedKeys.value.length);
const allRowKeys = computed(() => filteredRecords.value.map((r) => r.url));
const keySet = computed(() => new Set(selectedKeys.value));
const isAllSelected = computed(() => filteredRecords.value.length > 0 && selectedCount.value === filteredRecords.value.length);

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

function handleReset() {
  keywordFilter.value = '';
  hostFilter.value = '';
  dateRange.value = [];
  selectedKeys.value = [];
}

function loadAllRecords() {
  loading.value = true;
  try {
    const all: CollectorRecord[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith('zq.desktop.collector.history.')) {
        try {
          const parsed = JSON.parse(localStorage.getItem(key) || '[]');
          if (Array.isArray(parsed)) {
            for (const item of parsed) {
              if (item && typeof item.url === 'string') {
                all.push({
                  url: item.url,
                  title: item.title || item.url,
                  host: item.host || key.replace('zq.desktop.collector.history.', ''),
                  fetchedAt: item.fetchedAt || '',
                });
              }
            }
          }
        } catch { /* ignore */ }
      }
    }
    all.sort((a, b) => b.fetchedAt.localeCompare(a.fetchedAt));
    allRecords.value = all;
  } finally {
    loading.value = false;
  }
}

function handleDeleteRecord(record: CollectorRecord) {
  Modal.confirm({
    title: '删除采集记录？',
    content: `确定删除 "${record.url}" 的采集记录？`,
    okText: '删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: () => {
      try {
        deleteCollectedUrl(record.host, record.url);
        allRecords.value = allRecords.value.filter((r) => r.url !== record.url);
        selectedKeys.value = selectedKeys.value.filter((k) => k !== record.url);
        Message.success('采集记录已删除');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除失败');
      }
    },
  });
}

function handleBatchDelete() {
  if (selectedCount.value === 0) {
    Message.warning('请先勾选要删除的采集记录');
    return;
  }

  Modal.confirm({
    title: `批量删除 ${selectedCount.value} 条采集记录？`,
    content: `已选择 ${selectedCount.value} 条采集记录，删除后不可恢复。`,
    okText: '批量删除',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: async () => {
      const items = filteredRecords.value.filter((r) => keySet.value.has(r.url));
      batchDeleting.value = true;
      let successCount = 0;
      let failCount = 0;

      for (const item of items) {
        try {
          deleteCollectedUrl(item.host, item.url);
          allRecords.value = allRecords.value.filter((r) => r.url !== item.url);
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
        Message.success(`已批量删除 ${successCount} 条采集记录`);
      }
    },
  });
}

function handleClearAll() {
  Modal.confirm({
    title: '清空所有采集记录？',
    content: '此操作将删除全部采集记录，不可恢复。',
    okText: '清空全部',
    okButtonProps: { status: 'danger' },
    cancelText: '取消',
    onOk: () => {
      try {
        const keys: string[] = [];
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && key.startsWith('zq.desktop.collector.history.')) {
            keys.push(key);
          }
        }
        for (const key of keys) {
          localStorage.removeItem(key);
        }
        allRecords.value = [];
        selectedKeys.value = [];
        Message.success('已清空所有采集记录');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '清空失败');
      }
    },
  });
}

onMounted(() => {
  loadAllRecords();
});
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>采集记录</h2>
        <p>查看所有已采集的网页记录，支持搜索、筛选和批量管理。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <span style="font-size: 13px; color: #86909c;">共 {{ allRecords.length }} 条</span>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="loading" @click="loadAllRecords">刷新列表</a-button>
          <a-button v-if="allRecords.length > 0" status="danger" @click="handleClearAll">清空全部</a-button>
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

        <div class="page-filter-form page-filter-form--separated">
          <a-input v-model="keywordFilter" allow-clear placeholder="搜索 URL 或标题" />
          <a-select v-model="hostFilter" allow-clear placeholder="选择站点">
            <a-option v-for="item in hostOptions" :key="item" :value="item">{{ item }}</a-option>
          </a-select>
          <a-range-picker v-model="dateRange" value-format="YYYY-MM-DD" />
          <a-space class="page-filter-form__actions">
            <a-button type="primary" @click="selectedKeys = []">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </div>

        <div v-if="selectedCount > 0" class="collector-batch-bar">
          <a-space wrap>
            <a-button size="small" @click="handleSelectAll">
              {{ isAllSelected ? '取消全选' : '全选' }}
            </a-button>
            <a-button size="small" status="danger" :disabled="!selectedCount" :loading="batchDeleting" @click="handleBatchDelete">
              批量删除
            </a-button>
          </a-space>
        </div>

        <PageStickyTable :data="filteredRecords" :loading="loading" :pagination="pagination" row-key="url" size="small" :row-selection="rowSelection" :selected-keys="selectedKeys" @select="handleTableSelect" @select-all="handleTableSelectAll">
            <template #empty>
              <a-empty description="暂无采集记录" />
            </template>
            <template #columns>
          <a-table-column title="URL" :width="300">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.url">{{ record.url }}</span>
            </template>
          </a-table-column>
          <a-table-column title="标题" :width="220">
            <template #cell="{ record }">
              <span class="table-cell-ellipsis" :title="record.title">{{ record.title || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="站点" :width="160">
            <template #cell="{ record }">
              {{ record.host || '-' }}
            </template>
          </a-table-column>
          <a-table-column title="采集时间" :width="180">
            <template #cell="{ record }">
              {{ record.fetchedAt || '-' }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="120" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" status="danger" @click="handleDeleteRecord(record)">删除</a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>
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

.collector-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 6px;
}
</style>
