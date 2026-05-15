<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import {
  IconApps,
  IconStorage,
  IconDesktop,
  IconSafe,
  IconFolder,
  IconFile,
  IconCloudDownload,
  IconImage,
  IconCommon,
  IconBug,
  IconMenuFold,
  IconMenuUnfold,
} from '@arco-design/web-vue/es/icon';
import TitleBar from './TitleBar.vue';
import { useAppStore } from '../stores/app.store';
import { useSiteStore } from '../stores/site.store';

const router = useRouter();
const route = useRoute();
const appStore = useAppStore();
const siteStore = useSiteStore();

const siderCollapsed = ref(false);
const selectedKeys = computed(() => [route.path]);
const openKeys = ['overview', 'content', 'collector', 'system'];
const currentTenantName = computed(() => siteStore.currentTenant?.name || '未选择租户');

const menuGroups = [
  {
    key: 'overview',
    label: '概览',
    icon: IconApps,
    children: [
      { key: '/dashboard', label: '工作台', icon: IconApps },
      { key: '/tenants/data', label: '当前租户数据', icon: IconSafe },
    ],
  },
  {
    key: 'content',
    label: '内容',
    icon: IconFolder,
    children: [
      { key: '/tools', label: '工具管理', icon: IconApps },
      { key: '/articles', label: '文章管理', icon: IconSafe },
      { key: '/drafts', label: '草稿箱', icon: IconFolder },
      { key: '/submit-records', label: '提交记录', icon: IconFile },
    ],
  },
  {
    key: 'collector',
    label: '采集与媒体',
    icon: IconCloudDownload,
    children: [
      { key: '/collector', label: '本地采集', icon: IconCloudDownload },
      { key: '/collector/records', label: '采集记录', icon: IconCloudDownload },
      { key: '/media-tasks', label: '媒体任务', icon: IconImage },
    ],
  },
  {
    key: 'system',
    label: '系统',
    icon: IconCommon,
    children: [
      { key: '/sites', label: '站点管理', icon: IconStorage },
      { key: '/tenants', label: '租户管理', icon: IconDesktop },
      { key: '/sessions', label: '会话管理', icon: IconCommon },
      { key: '/system-logs', label: '系统日志', icon: IconBug },
    ],
  },
];

watch(
  () => route.path,
  (path) => {
    appStore.setCurrentMenu(path);
  },
  { immediate: true },
);

function handleMenuClick(key: string) {
  router.push(key);
}

function toggleSiderCollapsed() {
  siderCollapsed.value = !siderCollapsed.value;
}
</script>

<template>
  <div class="admin-shell">
    <TitleBar />
    <a-layout class="admin-shell__body">
      <a-layout-sider
        :width="220"
        :collapsed="siderCollapsed"
        :collapsed-width="68"
        collapsible
        hide-trigger
      >
        <div class="admin-shell__logo" :class="{ 'admin-shell__logo--collapsed': siderCollapsed }">
          <div class="admin-shell__logo-mark">ZQ</div>
          <div v-if="!siderCollapsed" class="admin-shell__logo-copy">
            <strong>内容管理桌面端</strong>
            <p>{{ currentTenantName }}</p>
          </div>
          <a-button class="admin-shell__collapse-btn" type="text" size="mini" @click="toggleSiderCollapsed">
            <template #icon>
              <IconMenuFold v-if="!siderCollapsed" />
              <IconMenuUnfold v-else />
            </template>
          </a-button>
        </div>
        <a-menu
          :selected-keys="selectedKeys"
          :default-open-keys="openKeys"
          @menu-item-click="handleMenuClick"
        >
          <a-sub-menu v-for="group in menuGroups" :key="group.key">
            <template #icon>
              <component :is="group.icon" />
            </template>
            <template #title>{{ group.label }}</template>
            <a-menu-item v-for="item in group.children" :key="item.key">
              <template #icon>
                <component :is="item.icon" />
              </template>
              {{ item.label }}
            </a-menu-item>
          </a-sub-menu>
        </a-menu>
      </a-layout-sider>
      <a-layout-content class="admin-shell__content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </div>
</template>

<style scoped>
.admin-shell {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.admin-shell__body {
  flex: 1;
  overflow: hidden;
}

.admin-shell__logo {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 12px;
  border-bottom: 1px solid #e5e6eb;
  min-height: 64px;
}

.admin-shell__logo--collapsed {
  flex-direction: column;
  justify-content: center;
  padding: 12px 8px;
  gap: 10px;
}

.admin-shell :deep(.arco-layout-sider-children) {
  display: flex;
  flex-direction: column;
}

.admin-shell :deep(.arco-menu) {
  padding-top: 8px;
}

.admin-shell :deep(.arco-menu-collapsed) {
  width: 68px;
}

.admin-shell__logo-mark {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: linear-gradient(135deg, #4080ff, #165dff);
  display: grid;
  place-items: center;
  color: #fff;
  font-weight: 700;
  flex-shrink: 0;
}

.admin-shell__logo-copy {
  min-width: 0;
  flex: 1;
}

.admin-shell__logo strong {
  display: block;
  color: #1d2129;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.admin-shell__logo p {
  margin: 4px 0 0;
  color: #86909c;
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.admin-shell__collapse-btn {
  margin-left: auto;
  flex-shrink: 0;
  color: #165dff;
  padding: 0;
}

.admin-shell__collapse-btn:hover {
  color: #0e42d2;
  background: transparent;
}

.admin-shell__logo--collapsed .admin-shell__collapse-btn {
  margin-left: 0;
}

.admin-shell__content {
  height: 100%;
  display: flex;
  overflow: hidden;
  padding: 20px;
  background: #f2f3f5;
}

.admin-shell__content > * {
  flex: 1;
  min-width: 0;
  min-height: 0;
}
</style>
