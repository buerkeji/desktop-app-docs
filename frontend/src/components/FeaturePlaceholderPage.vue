<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description: string;
    tag?: string;
    tips?: string[];
  }>(),
  {
    tag: '规划中',
    tips: () => [],
  },
);
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>{{ title }}</h2>
        <p>{{ description }}</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="orange">{{ tag }}</a-tag>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-card class="page-section-card">
        <a-empty description="该模块正在按开发文档逐步接入，当前先保留独立入口和布局占位。" />
      </a-card>

      <a-card v-if="tips.length" title="后续接入点" class="page-section-card">
        <a-space direction="vertical" fill>
          <a-alert v-for="tip in tips" :key="tip" type="info">{{ tip }}</a-alert>
        </a-space>
      </a-card>
    </div>
  </div>
</template>
