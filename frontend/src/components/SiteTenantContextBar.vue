<script setup lang="ts">
import type { SiteItem, TenantItem } from '../types/site';

defineProps<{
  sites: SiteItem[];
  tenants: TenantItem[];
  currentSiteId: number | null;
  currentTenantId: number | null;
  syncing?: boolean;
}>();

defineEmits<{
  siteChange: [value: string | number | boolean | Record<string, unknown> | Array<string | number | boolean | Record<string, unknown>>];
  tenantChange: [value: string | number | boolean | Record<string, unknown> | Array<string | number | boolean | Record<string, unknown>>];
  syncTenants: [];
  manageSites: [];
  loginTenant: [];
}>();
</script>

<template>
  <div class="context-bar">
    <div class="context-bar__scroll">
      <div class="context-bar__row">
        <span class="context-bar__label">站点</span>
        <a-select
          :model-value="currentSiteId ?? undefined"
          placeholder="选择站点"
          size="small"
          class="context-bar__select"
          @change="$emit('siteChange', $event)"
        >
          <a-option v-for="item in sites" :key="item.id" :value="item.id">
            {{ item.name }}
          </a-option>
        </a-select>

        <span class="context-bar__label">租户</span>
        <a-select
          :model-value="currentTenantId ?? undefined"
          placeholder="选择租户"
          :disabled="!currentSiteId"
          size="small"
          class="context-bar__select"
          @change="$emit('tenantChange', $event)"
        >
          <a-option v-for="item in tenants" :key="item.id" :value="item.id">
            {{ item.name }}
          </a-option>
        </a-select>

        <a-button size="small" :loading="syncing" @click="$emit('syncTenants')">同步</a-button>
        <a-button size="small" @click="$emit('manageSites')">站点</a-button>
        <a-button size="small" @click="$emit('loginTenant')">登录</a-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.context-bar {
  display: flex;
  width: 100%;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
}

.context-bar__scroll {
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

.context-bar__row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: max-content;
}

.context-bar__label {
  flex: 0 0 auto;
  color: #4e5969;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
}

.context-bar__select {
  width: 160px;
  min-width: 160px;
}

.context-bar__select :deep(.arco-select-view),
.context-bar__select :deep(.arco-select-view-input),
.context-bar__select :deep(.arco-select-view-value) {
  cursor: pointer;
}

.context-bar__select :deep(.arco-select-view-input) {
  caret-color: transparent;
}
</style>
