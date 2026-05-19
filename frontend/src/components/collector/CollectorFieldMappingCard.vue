<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const result = ctx.result;
const collectMode = ctx.collectMode;
const hasResult = ctx.hasResult;
const canGenerateDraft = ctx.canGenerateDraft;
const draftGenerating = ctx.draftGenerating;
const draftMappingTab = ctx.draftMappingTab;
const toolMapping = ctx.toolMapping;
const articleMapping = ctx.articleMapping;
const resetMappings = ctx.resetMappings;
const createDraft = ctx.createDraft;
</script>

<template>
  <a-card title="草稿字段映射" class="page-section-card">
      <template #extra>
        <a-space>
          <a-button :disabled="!hasResult" @click="resetMappings(result)">恢复默认映射</a-button>
          <a-button
            type="primary"
            :disabled="!canGenerateDraft"
            :loading="draftGenerating === draftMappingTab"
            @click="createDraft(draftMappingTab)"
          >
            生成当前草稿
          </a-button>
        </a-space>
      </template>

      <a-tabs v-model:active-key="draftMappingTab" lazy-load>
        <a-tab-pane key="tool" title="工具草稿">
          <a-form :model="toolMapping" layout="vertical">
            <div class="collector-form-section">
              <div class="collector-form-section__title">基础信息</div>
              <a-row :gutter="16">
                <a-col id="collector-tool-title" :span="12">
                  <a-form-item label="标题">
                    <a-input v-model="toolMapping.title" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="来源链接">
                    <a-input v-model="toolMapping.url" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-tool-website" :span="12">
                  <a-form-item label="官网">
                    <a-input v-model="toolMapping.website" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-tool-tags" :span="12">
                  <a-form-item label="标签">
                    <a-input v-model="toolMapping.tagsText" placeholder="多个标签用英文逗号分隔" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-tool-icon" :span="12">
                  <a-form-item label="图标">
                    <a-input v-model="toolMapping.icon" placeholder="优先使用站点图标或页面分享图" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-tool-thumbnail" :span="12">
                  <a-form-item label="缩略图">
                    <a-input v-model="toolMapping.thumbnail" placeholder="优先页面缩略图，其次正文首图" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-tool-description" :span="24">
                  <a-form-item label="摘要">
                    <a-textarea v-model="toolMapping.description" :auto-size="{ minRows: 3, maxRows: 5 }" />
                  </a-form-item>
                </a-col>
                <a-col :span="24">
                  <a-form-item label="特性">
                    <a-textarea v-model="toolMapping.featuresText" :auto-size="{ minRows: 3, maxRows: 5 }" />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="collector-form-section">
              <div class="collector-form-section__title">SEO</div>
              <a-row :gutter="16">
                <a-col :span="8">
                  <a-form-item label="SEO 标题">
                    <a-input v-model="toolMapping.metaTitle" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item label="SEO 关键词">
                    <a-input v-model="toolMapping.metaKeywords" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item label="SEO 描述">
                    <a-input v-model="toolMapping.metaDescription" />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="collector-form-section">
              <div class="collector-form-section__title">正文</div>
              <a-row :gutter="16">
                <a-col id="collector-tool-content" :span="24">
                  <a-form-item label="正文 HTML">
                    <a-textarea
                      v-model="toolMapping.content"
                      class="collector-content-editor"
                      :auto-size="{ minRows: 12, maxRows: 12 }"
                      placeholder="这里的 HTML 会原样进入编辑器正文区域"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="article" title="文章草稿">
          <a-form :model="articleMapping" layout="vertical">
            <div class="collector-form-section">
              <div class="collector-form-section__title">基础信息</div>
              <a-row :gutter="16">
                <a-col id="collector-article-title" :span="12">
                  <a-form-item label="标题">
                    <a-input v-model="articleMapping.title" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-article-thumbnail" :span="12">
                  <a-form-item label="缩略图">
                    <a-input v-model="articleMapping.thumbnail" placeholder="可手动替换为其他图片地址" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-article-publishedAt" :span="12">
                  <a-form-item label="发布时间">
                    <a-input v-model="articleMapping.publishedAt" placeholder="例如 2026-04-28T09:30:00+08:00" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-article-tags" :span="12">
                  <a-form-item label="标签">
                    <a-input v-model="articleMapping.tagsText" placeholder="多个标签用英文逗号分隔" />
                  </a-form-item>
                </a-col>
                <a-col id="collector-article-excerpt" :span="24">
                  <a-form-item label="摘要">
                    <a-textarea v-model="articleMapping.excerpt" :auto-size="{ minRows: 3, maxRows: 5 }" />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="collector-form-section">
              <div class="collector-form-section__title">SEO</div>
              <a-row :gutter="16">
                <a-col :span="8">
                  <a-form-item label="SEO 标题">
                    <a-input v-model="articleMapping.metaTitle" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item label="SEO 关键词">
                    <a-input v-model="articleMapping.metaKeywords" />
                  </a-form-item>
                </a-col>
                <a-col :span="8">
                  <a-form-item label="SEO 描述">
                    <a-input v-model="articleMapping.metaDescription" />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="collector-form-section">
              <div class="collector-form-section__title">正文</div>
              <a-row :gutter="16">
                <a-col id="collector-article-content" :span="24">
                  <a-form-item label="正文 HTML">
                    <a-textarea
                      v-model="articleMapping.content"
                      class="collector-content-editor"
                      :auto-size="{ minRows: 12, maxRows: 12 }"
                      placeholder="这里的 HTML 会原样进入编辑器正文区域"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
            </div>
          </a-form>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </template>

<style scoped>
.collector-form-section {
  margin-bottom: 16px;
}

.collector-form-section__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1, #1d2129);
  margin-bottom: 12px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--color-border, #e5e6e8);
}

.collector-content-editor {
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
}
</style>
