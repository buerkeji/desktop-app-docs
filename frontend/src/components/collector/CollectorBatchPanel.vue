<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const collectMode = ctx.collectMode;
const batchResults = ctx.batchResults;
const collectorStore = ctx.collectorStore;
const batchStatus = ctx.batchStatus;
const showHistoryPanel = ctx.showHistoryPanel;
const historyPanelHost = ctx.historyPanelHost;
const collectedHistoryItems = ctx.collectedHistoryItems;
const toggleHistoryPanel = ctx.toggleHistoryPanel;
const handleDeleteHistoryItem = ctx.handleDeleteHistoryItem;
const handleClearHistory = ctx.handleClearHistory;
</script>

<template>
  <template v-if="collectMode === 'batch' && batchResults.length">
    <a-card title="批量采集结果" class="page-section-card">
      <div class="batch-progress">
        <div class="batch-progress__summary">
          <span>{{ batchResults.length }} 个链接，</span>
          <span>成功 {{ collectorStore.batchSuccessCount }} 个，</span>
          <span>失败 {{ collectorStore.batchFailCount }} </span>
          <span v-if="collectorStore.isBatchPaused" style="color: #ff7d00;">（已暂停）</span>
          <span v-if="batchStatus === 'idle'" style="color: #f53f3f;">已停止批量采集</span>
        </div>
      </div>
      <div class="batch-results-table">
        <div class="batch-results-table__header">
          <span class="batch-results-table__col-url">URL</span>
          <span class="batch-results-table__col-title">标题</span>
          <span class="batch-results-table__col-status">状态</span>
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
            <a-tag v-else-if="item.status === 'collecting'" color="arcoblue">采集中</a-tag>
            <a-tag v-else-if="item.status === 'success'" color="green">{{ item.ruleHost ? '已套用规则' : '成功' }}</a-tag>
            <a-tag v-else-if="item.status === 'failed'" color="red" :title="item.error">{{ item.error === '已采集过（跳过）' ? '已跳过' : '失败' }}</a-tag>
          </span>
          <span class="batch-results-table__col-draft">
            <template v-if="item.status === 'success'">
              <a-tag v-if="item.toolDraftId" color="purple" size="small">工具</a-tag>
              <a-tag v-if="item.articleDraftId" color="blue" size="small">文章</a-tag>
              <span class="batch-results-table__draft-hint">已保存</span>
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
        v-for="item in [...collectedHistoryItems].reverse()"
        :key="item.url"
        class="batch-results-table__row"
      >
        <span class="batch-results-table__col-url" :title="item.url" style="font-size: 12px;">{{ item.url }}</span>
        <span class="batch-results-table__col-title">{{ item.title || '-' }}</span>
        <span class="batch-results-table__col-status" style="font-size: 12px; color: #86909c;">{{ (item as any).fetchedAt?.slice(0, 10) }}</span>
        <span class="batch-results-table__col-draft">
          <a-button size="mini" status="danger" @click="handleDeleteHistoryItem(item.host, item.url)">删除</a-button>
        </span>
      </div>
    </div>
  </a-card>
</template>

<style scoped>
.batch-results-table {
  width: 100%;
}
.batch-results-table__header,
.batch-results-table__row {
  display: flex;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
}
.batch-results-table__header {
  font-weight: 600;
  border-bottom: 1px solid var(--color-border, #e5e6e8);
}
.batch-progress__summary {
  font-size: 14px;
  margin-bottom: 8px;
}
.batch-results-table__col-url { flex: 2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.batch-results-table__col-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.batch-results-table__col-status { flex: 0 0 80px; }
.batch-results-table__col-draft { flex: 0 0 100px; }
</style>
