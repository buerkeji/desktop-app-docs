<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { listLocalDrafts } from '../../services/local-draft.service';
import { useSiteStore } from '../../stores/site.store';
import { initialiseDesktopContext } from '../../utils/desktop-context';
import { useAuthStore } from '../../stores/auth.store';
import { createTablePagination } from '../../utils/table-pagination';
import PageStickyTable from '../../components/PageStickyTable.vue';

const siteStore = useSiteStore();
const authStore = useAuthStore();
const sessionLoading = ref(false);
const refreshing = ref(false);
const draftLoading = ref(false);
const draftCount = ref(0);
const pagination = createTablePagination();

const dashboardCards = computed(() => [
  { key: 'sites', title: '站点', value: siteStore.sites.length },
  { key: 'tenants', title: '租户', value: siteStore.tenants.length },
  { key: 'drafts', title: '草稿', value: draftCount.value, loading: draftLoading.value },
  { key: 'sessions', title: '会话', value: authStore.sessions.length, loading: sessionLoading.value },
]);
const currentUserText = computed(() => authStore.user?.username || '未登录');
const currentTenantText = computed(() => siteStore.currentTenant?.name || '未选择租户');
const currentSiteText = computed(() => siteStore.currentSite?.name || '未选择站点');
const currentSessionLabel = computed(() => authStore.token?.sessionId || '-');
const tokenExpiresLabel = computed(() => authStore.token?.expiresAt || '-');
const refreshExpiresLabel = computed(() => authStore.token?.refreshExpiresAt || '-');

async function loadDraftCount() {
  if (!siteStore.currentTenant?.id) {
    draftCount.value = 0;
    return;
  }

  draftLoading.value = true;
  try {
    const drafts = await listLocalDrafts({
      tenantId: siteStore.currentTenant.id,
    });
    draftCount.value = drafts.length;
  } catch (error) {
    draftCount.value = 0;
    Message.warning(error instanceof Error ? error.message : '获取本地草稿数量失败');
  } finally {
    draftLoading.value = false;
  }
}

async function loadSessions() {
  siteStore.syncCurrentTenantWithAuth(authStore.token?.tenantId);

  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    authStore.sessions = [];
    return;
  }

  sessionLoading.value = true;
  try {
    await authStore.loadSessions({
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
  } catch (error) {
    Message.warning(error instanceof Error ? error.message : '获取会话列表失败');
  } finally {
    sessionLoading.value = false;
  }
}

async function handleRefreshToken() {
  if (!siteStore.currentTenant?.apiBaseUrl || !siteStore.currentTenant?.id) {
    Message.warning('当前没有可刷新的租户上下文');
    return;
  }

  refreshing.value = true;
  try {
    await authStore.refreshAuthToken({
      id: siteStore.currentTenant.id,
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });

    await authStore.refreshBootstrap({
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
    await loadSessions();
    Message.success('令牌刷新成功');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '令牌刷新失败');
  } finally {
    refreshing.value = false;
  }
}

async function handleRevokeSession(sessionId: number) {
  if (!siteStore.currentTenant?.apiBaseUrl) {
    Message.warning('当前没有可用的租户 API 地址');
    return;
  }

  try {
    await authStore.revokeSession(
      {
        apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
      },
      sessionId,
    );
    Message.success('会话已吊销');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '会话吊销失败');
  }
}

onMounted(async () => {
  await initialiseDesktopContext(siteStore, authStore);
  await loadDraftCount();
  await loadSessions();
});

watch(
  () => siteStore.currentTenant?.id ?? null,
  () => {
    void loadDraftCount();
    void loadSessions();
  },
);
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>工作台</h2>
        <p>集中查看当前站点、租户、草稿和会话概况，处理高频登录态操作。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="purple">{{ currentSiteText }}</a-tag>
          <a-tag color="arcoblue">{{ currentUserText }}</a-tag>
          <a-tag color="green">{{ currentTenantText }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="sessionLoading" @click="loadSessions">刷新会话</a-button>
          <a-button type="primary" :loading="refreshing" @click="handleRefreshToken">刷新令牌</a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <div class="stat-grid dashboard-stat-grid">
        <a-card v-for="item in dashboardCards" :key="item.key" :title="item.title" class="dashboard-stat-card">
          <div class="dashboard-card__value">{{ item.value }}</div>
          <a-spin v-if="item.loading" :size="12" />
        </a-card>
      </div>

      <div class="dashboard-grid">
        <div class="dashboard-grid__side">
          <a-card title="当前状态" class="dashboard-panel-card">
          <a-descriptions :column="1" bordered size="large">
            <a-descriptions-item label="站点">{{ currentSiteText }}</a-descriptions-item>
            <a-descriptions-item label="租户">{{ currentTenantText }}</a-descriptions-item>
            <a-descriptions-item label="用户">{{ currentUserText }}</a-descriptions-item>
            <a-descriptions-item label="令牌">{{ authStore.token?.tokenType || '-' }}</a-descriptions-item>
          </a-descriptions>
          </a-card>

          <a-card title="会话信息" class="dashboard-panel-card">
          <a-descriptions :column="1" bordered size="large">
            <a-descriptions-item label="当前会话">{{ currentSessionLabel }}</a-descriptions-item>
            <a-descriptions-item label="访问到期">{{ tokenExpiresLabel }}</a-descriptions-item>
            <a-descriptions-item label="刷新到期">{{ refreshExpiresLabel }}</a-descriptions-item>
          </a-descriptions>
          </a-card>
        </div>

        <div class="dashboard-grid__main">
          <a-card title="远端会话" class="dashboard-panel-card">
          <PageStickyTable
            :data="authStore.sessions"
            :loading="sessionLoading"
            :pagination="pagination"
            row-key="id"
            size="small"
            class="dashboard-session-table"
          >
            <template #empty>
              <a-empty description="暂无会话" />
            </template>
            <template #columns>
              <a-table-column title="设备" :width="200">
                <template #cell="{ record }">
                  <span class="table-cell-ellipsis" :title="record.deviceName">{{ record.deviceName || '-' }}</span>
                </template>
              </a-table-column>
              <a-table-column title="版本" :width="120">
                <template #cell="{ record }">
                  <span class="table-cell-ellipsis" :title="record.clientVersion">{{ record.clientVersion || '-' }}</span>
                </template>
              </a-table-column>
              <a-table-column title="最后使用" :width="180">
                <template #cell="{ record }">
                  <span class="table-cell-ellipsis" :title="record.lastUsedAt">{{ record.lastUsedAt || '-' }}</span>
                </template>
              </a-table-column>
              <a-table-column title="到期" :width="180">
                <template #cell="{ record }">
                  <span class="table-cell-ellipsis" :title="record.expiresAt">{{ record.expiresAt || '-' }}</span>
                </template>
              </a-table-column>
              <a-table-column title="当前" :width="96">
                <template #cell="{ record }">
                  <a-tag :color="record.isCurrent ? 'green' : 'gray'">{{ record.isCurrent ? '是' : '否' }}</a-tag>
                </template>
              </a-table-column>
              <a-table-column title="操作" :width="88">
                <template #cell="{ record }">
                  <a-button type="text" size="small" status="danger" :disabled="record.isCurrent" @click="handleRevokeSession(record.id)">
                    吊销
                  </a-button>
                </template>
              </a-table-column>
            </template>
          </PageStickyTable>
          </a-card>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-stat-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.dashboard-stat-card :deep(.arco-card-header) {
  padding-bottom: 8px;
}

.dashboard-stat-card :deep(.arco-card-header-title) {
  font-size: 14px;
  white-space: nowrap;
}

.dashboard-stat-card :deep(.arco-card-body) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding-top: 0;
}

.dashboard-card__value {
  font-size: 28px;
  line-height: 1.2;
  font-weight: 700;
  color: #1d2129;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(0, 1.6fr);
  gap: 16px;
  margin-top: 16px;
}

.dashboard-grid__side,
.dashboard-grid__main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.dashboard-panel-card {
  border-radius: 14px;
}

.dashboard-session-table :deep(.arco-table-td) {
  vertical-align: middle;
}

.table-cell-ellipsis {
  display: block;
  width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
