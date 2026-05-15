import { defineStore } from 'pinia';

export interface BatchCollectItem {
  url: string;
  status: 'pending' | 'collecting' | 'paused' | 'success' | 'failed';
  title: string;
  error?: string;
  ruleHost?: string;
  toolDraftId?: string;
  articleDraftId?: string;
}

export type BatchStatus = 'idle' | 'running' | 'paused' | 'completed';

interface CollectorState {
  batchUrlsText: string;
  batchResults: BatchCollectItem[];
  batchProgress: { current: number; total: number };
  batchStatus: BatchStatus;
  batchDraftType: 'tool' | 'article' | 'both';
}

export const useCollectorStore = defineStore('collector', {
  state: (): CollectorState => ({
    batchUrlsText: '',
    batchResults: [],
    batchProgress: { current: 0, total: 0 },
    batchStatus: 'idle',
    batchDraftType: 'both',
  }),

  getters: {
    isBatchRunning: (state) => state.batchStatus === 'running',
    isBatchPaused: (state) => state.batchStatus === 'paused',
    batchSuccessCount: (state) => state.batchResults.filter((item) => item.status === 'success').length,
    batchFailCount: (state) => state.batchResults.filter((item) => item.status === 'failed').length,
  },

  actions: {
    initBatch(urls: string[], draftType: 'tool' | 'article' | 'both') {
      this.batchUrlsText = urls.join('\n');
      this.batchDraftType = draftType;
      this.batchResults = urls.map((url) => ({
        url,
        status: 'pending' as const,
        title: '',
      }));
      this.batchProgress = { current: 0, total: urls.length };
      this.batchStatus = 'idle';
    },

    startBatch() {
      if (this.batchStatus === 'idle' || this.batchStatus === 'paused') {
        this.batchStatus = 'running';
        const currentItem = this.batchResults.find((item) => item.status === 'paused');
        if (currentItem) {
          currentItem.status = 'collecting';
        }
      }
    },

    pauseBatch() {
      if (this.batchStatus === 'running') {
        this.batchStatus = 'paused';
        const currentItem = this.batchResults.find((item) => item.status === 'collecting');
        if (currentItem) {
          currentItem.status = 'paused';
        }
      }
    },

    stopBatch() {
      this.batchStatus = 'idle';
    },

    completeBatch() {
      this.batchStatus = 'completed';
    },

    resetBatch() {
      this.batchStatus = 'idle';
      this.batchResults = [];
      this.batchUrlsText = '';
      this.batchProgress = { current: 0, total: 0 };
    },

    updateBatchItem(index: number, update: Partial<BatchCollectItem>) {
      if (index >= 0 && index < this.batchResults.length) {
        this.batchResults[index] = { ...this.batchResults[index], ...update };
      }
    },
  },
});
