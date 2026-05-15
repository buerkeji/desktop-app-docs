<script setup lang="ts">
import { inject } from 'vue';

const ctx = inject<Record<string, any>>('collectorContext')!;
const currentSiteId = ctx.currentSiteId;
const filteredTenants = ctx.filteredTenants;
const currentTenantId = ctx.currentTenantId;
</script>

<template>
  <a-alert
    v-if="!currentSiteId"
    type="warning"
    title="尚未选择站点"
    content="请先在上方选择当前站点，再同步该站点的租户列表；采集与草稿上下文都应在当前站点下完成"
  />
  <a-alert
    v-else-if="!filteredTenants.length"
    type="warning"
    title="当前站点下暂无租户"
    content="请先同步该站点的租户列表，或前往站点/租户管理页补充租户配置"
  />
  <a-alert
    v-else-if="!currentTenantId"
    type="warning"
    title="尚未选择租户"
    content="采集本身不依赖登录态，但生成草稿前需要先切换到目标租户，这样草稿箱和后续编辑页才能正确隔离"
  />
</template>
