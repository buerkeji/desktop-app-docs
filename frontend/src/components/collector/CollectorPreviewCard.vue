<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const result = ctx.result;
const collectMode = ctx.collectMode;
const previewMode = ctx.previewMode;
const previewHtml = ctx.previewHtml;
const pickerEnabled = ctx.pickerEnabled;
const pickerField = ctx.pickerField;
const pickerStatus = ctx.pickerStatus;
const pickerPreviewHtml = ctx.pickerPreviewHtml;
const pickerFrameRef = ctx.pickerFrameRef;
const pickerSelectionSummary = ctx.pickerSelectionSummary;
const pickerFieldDescription = ctx.pickerFieldDescription;
const pickerBrokenImageCount = ctx.pickerBrokenImageCount;
const useBrowserRender = ctx.useBrowserRender;
const quickImageChoices = ctx.quickImageChoices;
const currentRuleFieldEntries = ctx.currentRuleFieldEntries;
const hasCurrentRule = ctx.hasCurrentRule;
const pickerLastValue = ctx.pickerLastValue;
const PICKER_FIELD_OPTIONS = ctx.PICKER_FIELD_OPTIONS;
const handlePickerFrameLoad = ctx.handlePickerFrameLoad;
const saveCurrentSiteRuleDraft = ctx.saveCurrentSiteRuleDraft;
const deleteCurrentSiteRuleDraft = ctx.deleteCurrentSiteRuleDraft;
const applyQuickImageSelection = ctx.applyQuickImageSelection;
const scrollToDraftMappingField = ctx.scrollToDraftMappingField;
const getPickerFieldLabel = ctx.getPickerFieldLabel;
</script>

<template>
  <a-space direction="vertical" fill size="large" class="collector-main-stack">
      <a-card title="采集结果预览" class="page-section-card collector-card collector-card--preview">
        <template #extra>
          <a-radio-group v-model="previewMode" type="button" size="small">
            <a-radio value="rendered">渲染预览</a-radio>
            <a-radio value="source">HTML 源码</a-radio>
            <a-radio value="picker" :disabled="!pickerEnabled">点选规则</a-radio>
          </a-radio-group>
        </template>

        <template v-if="result">
          <a-space direction="vertical" fill size="large">
            <div class="collector-preview-meta">
              <h3>{{ result.title }}</h3>
              <p>{{ result.description || result.excerpt || '未识别到摘要信息' }}</p>
            </div>

            <div v-if="previewMode === 'rendered'" class="collector-preview-body collector-scroll-panel" v-html="previewHtml" />

            <div v-else-if="previewMode === 'picker'" class="collector-picker-panel">
              <div class="collector-picker-toolbar">
                <div class="collector-picker-toolbar__header">
                  <strong>点选字段</strong>
                  <span>{{ pickerFieldDescription }}</span>
                </div>
                <div class="collector-picker-toolbar__meta">
                  <a-space wrap>
                    <a-tag :color="useBrowserRender ? 'orange' : 'arcoblue'">{{ useBrowserRender ? '浏览器渲染' : '静态源码' }}</a-tag>
                    <a-tag v-if="pickerBrokenImageCount > 0" color="red">
                      {{ `灰图/失败 ${pickerBrokenImageCount}` }}
                    </a-tag>
                  </a-space>
                </div>
                <a-radio-group v-model="pickerField" type="button" size="small" class="collector-picker-toolbar__fields">
                  <a-radio v-for="option in PICKER_FIELD_OPTIONS" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </a-radio>
                </a-radio-group>
              </div>

              <a-alert
                type="info"
                class="collector-picker-alert"
                :content="pickerStatus"
              />

              <div v-if="pickerSelectionSummary" class="collector-picker-selection">
                <div class="collector-picker-selection__header">
                  <strong>本次点选结果</strong>
                  <a-button size="mini" @click="scrollToDraftMappingField(pickerSelectionSummary.field)">
                    定位到字段
                  </a-button>
                </div>
                <div class="collector-picker-selection__grid">
                  <div class="collector-picker-selection__item">
                    <span>字段</span>
                    <strong>{{ pickerSelectionSummary.fieldLabel }}</strong>
                  </div>
                  <div class="collector-picker-selection__item">
                    <span>命中数</span>
                    <strong>{{ pickerSelectionSummary.matchCount }}</strong>
                  </div>
                  <div class="collector-picker-selection__item">
                    <span>元素</span>
                    <strong>{{ pickerSelectionSummary.targetTag }}</strong>
                  </div>
                </div>
                <code class="collector-picker-selection__selector">{{ pickerSelectionSummary.selector }}</code>
                <div class="collector-picker-selection__value">
                  {{ pickerSelectionSummary.previewValue }}
                </div>
              </div>

              <div v-if="quickImageChoices.length" class="collector-quick-image-panel">
                <div class="collector-quick-image-panel__header">
                  <strong>图片快捷选择</strong>
                  <span>{{ pickerField === 'icon' ? '当前会设为图标' : '当前会设为缩略图' }}</span>
                </div>
                <div class="collector-quick-image-grid">
                  <button
                    v-for="choice in quickImageChoices"
                    :key="choice.url"
                    type="button"
                    class="collector-quick-image-item"
                    @click="applyQuickImageSelection(choice)"
                  >
                    <div class="collector-quick-image-thumb">
                      <img :src="choice.url" :alt="choice.label" />
                      <span class="collector-quick-image-item__badge">
                        {{ choice.type === 'icon' ? '图标' : choice.type === 'thumbnail' ? '缩略图' : '正文图片' }}
                      </span>
                    </div>
                  </button>
                </div>
              </div>

              <div class="collector-picker-actions">
                <a-space wrap>
                  <a-button type="primary" :disabled="!hasCurrentRule" @click="saveCurrentSiteRuleDraft">
                    保存当前站点规则
                  </a-button>
                  <a-button status="danger" :disabled="!hasCurrentRule" @click="deleteCurrentSiteRuleDraft">
                    删除当前站点规则
                  </a-button>
                </a-space>
                <span v-if="pickerLastValue" class="collector-picker-actions__value">
                  当前提取值：{{ pickerLastValue }}
                </span>
              </div>

              <div v-if="currentRuleFieldEntries.length" class="collector-rule-summary">
                <div v-for="item in currentRuleFieldEntries" :key="item.label" class="collector-rule-summary__item">
                  <span class="collector-rule-summary__label">{{ item.label }}</span>
                  <code class="collector-rule-summary__path">{{ item.path }}</code>
                </div>
              </div>

              <iframe
                ref="pickerFrameRef"
                class="collector-picker-frame collector-scroll-panel"
                :srcdoc="pickerPreviewHtml"
                sandbox="allow-same-origin"
                @load="handlePickerFrameLoad"
              />
            </div>

            <a-typography-paragraph
              v-else
              class="collector-preview-source collector-scroll-panel"
              :copyable="{ text: previewHtml }"
            >
              <pre>{{ previewHtml }}</pre>
            </a-typography-paragraph>
          </a-space>
        </template>

        <a-empty v-else description="输入网址后开始采集，成功后会在这里显示固定尺寸预览，可滚动查看全文" />
      </a-card>
    </a-space>
  </template>
