<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const currentSite = ctx.currentSite;
const currentTenant = ctx.currentTenant;
const result = ctx.result;
const canGenerateDraft = ctx.canGenerateDraft;
const draftGenerating = ctx.draftGenerating;
const handleReset = ctx.handleReset;
const createDraft = ctx.createDraft;
</script>

<template>
  <div class="page-toolbar">
    <div class="page-toolbar__title">
      <h2>本地采集</h2>
      <p>当前先接入静态网页采集，支持抓取标题、摘要、正文和图片，并按预览结果直接生成工具或文章草稿</p>
    </div>
    <div class="page-toolbar__actions">
      <div class="page-toolbar__meta">
        <a-space wrap>
          <a-tag color="purple">{{ currentSite?.name || '未选站点' }}</a-tag>
          <a-tag color="arcoblue">{{ currentTenant?.name || '未选租户' }}</a-tag>
        </a-space>
      </div>
      <div class="page-toolbar__buttons">
        <a-button :disabled="!result" @click="handleReset">清空结果</a-button>
        <a-button
          type="primary"
          :disabled="!canGenerateDraft"
          :loading="draftGenerating === 'article'"
          @click="createDraft('article')"
        >
          生成文章草稿
        </a-button>
        <a-button
          type="primary"
          :disabled="!canGenerateDraft"
          :loading="draftGenerating === 'tool'"
          @click="createDraft('tool')"
        >
          生成工具草稿
        </a-button>
      </div>
    </div>
  </div>
</template>
