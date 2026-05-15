<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const result = ctx.result;
const contentLength = ctx.contentLength;
const imageCount = ctx.imageCount;
const resultImages = ctx.resultImages;
const resultSuggestedTags = ctx.resultSuggestedTags;
const resultSeoKeywords = ctx.resultSeoKeywords;
</script>

<template>
  <a-card title="采集结果" class="page-section-card collector-card">
    <a-space direction="vertical" fill size="large">
      <div class="collector-summary">
        <div class="collector-summary__hero">
          <h3>{{ result?.title || '等待采集结果' }}</h3>
          <p>{{ result?.seoDescription || result?.description || result?.excerpt || '采集成功后，这里会显示页面摘要' }}</p>
        </div>

        <div class="collector-stats">
          <div class="collector-stat">
            <span class="collector-stat__label">站点</span>
            <strong class="collector-stat__value">{{ result?.siteName || '待识别' }}</strong>
          </div>
          <div class="collector-stat">
            <span class="collector-stat__label">正文</span>
            <strong class="collector-stat__value">{{ contentLength }}</strong>
          </div>
          <div class="collector-stat">
            <span class="collector-stat__label">图片</span>
            <strong class="collector-stat__value">{{ imageCount }}</strong>
          </div>
          <div class="collector-stat">
            <span class="collector-stat__label">链接</span>
            <strong class="collector-stat__value">{{ result?.finalUrl ? '已解析' : '待采集' }}</strong>
          </div>
        </div>

        <div class="collector-summary-block">
          <span class="collector-summary-block__label">最终地址</span>
          <a-typography-paragraph
            v-if="result?.finalUrl"
            class="collector-summary-block__url"
            :copyable="{ text: result.finalUrl }"
            :ellipsis="{ rows: 2, expandable: true }"
          >
            {{ result.finalUrl }}
          </a-typography-paragraph>
          <span v-else class="collector-summary-block__empty">待采集</span>
        </div>

        <div class="collector-summary-block">
          <span class="collector-summary-block__label">图标 / 缩略图</span>
          <div v-if="result?.iconUrl || result?.thumbnailUrl" class="collector-media-preview-list">
            <a
              v-if="result?.iconUrl"
              :href="result.iconUrl"
              target="_blank"
              class="collector-media-preview collector-media-preview--icon"
            >
              <img :src="result.iconUrl" alt="采集图标" />
              <span>图标</span>
            </a>
            <a
              v-if="result?.thumbnailUrl"
              :href="result.thumbnailUrl"
              target="_blank"
              class="collector-media-preview collector-media-preview--thumbnail"
            >
              <img :src="result.thumbnailUrl" alt="采集缩略图" />
              <span>缩略图</span>
            </a>
          </div>
          <span v-else class="collector-summary-block__empty">未识别到摘要信息</span>
        </div>

        <div class="collector-summary-block">
          <span class="collector-summary-block__label">建议标签</span>
          <div v-if="resultSuggestedTags.length" class="collector-chip-list">
            <a-tag v-for="tag in resultSuggestedTags" :key="tag" size="small" color="arcoblue">{{ tag }}</a-tag>
          </div>
          <span v-else class="collector-summary-block__empty">未识别到摘要信息</span>
        </div>

        <div class="collector-summary-block">
          <span class="collector-summary-block__label">SEO 关键词</span>
          <div v-if="resultSeoKeywords.length" class="collector-chip-list">
            <a-tag v-for="keyword in resultSeoKeywords" :key="keyword" size="small" color="purple">{{ keyword }}</a-tag>
          </div>
          <span v-else class="collector-summary-block__empty">未识别到摘要信息</span>
        </div>
      </div>

      <a-card title="采集到的图片资源" :bordered="false" class="collector-inner-card">
        <div v-if="resultImages.length" class="collector-image-list">
          <div v-for="image in resultImages.slice(0, 6)" :key="image.url" class="collector-image-item">
            <img :src="image.url" :alt="image.alt || result?.title || ''" />
            <span>{{ image.alt || '未命名图片' }}</span>
          </div>
        </div>
        <a-empty v-else description="当前页面没有识别到额外图片资源" />
      </a-card>
    </a-space>
  </a-card>
</template>
