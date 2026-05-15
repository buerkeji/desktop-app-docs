<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../../stores/auth.store';
import { useSiteStore } from '../../stores/site.store';
import { initialiseDesktopContext } from '../../utils/desktop-context';
import { createTablePagination } from '../../utils/table-pagination';
import PageStickyTable from '../../components/PageStickyTable.vue';
import type { SessionItem } from '../../types/auth';

const router = useRouter();
const authStore = useAuthStore();
const siteStore = useSiteStore();

const loading = ref(false);
const refreshingToken = ref(false);
const pagination = createTablePagination();
const sessionFilter = ref<'all' | 'current' | 'other'>('all');
const sessionSearch = ref('');
const sessionRefreshSummary = ref('');
const lastSessionRefreshAt = ref('');
const recentlyUpdatedSessionIds = ref<number[]>([]);
const lastExpiryReminderKey = ref('');
let highlightResetTimer: ReturnType<typeof setTimeout> | null = null;

const hasTenantContext = computed(() => Boolean(siteStore.currentTenant?.apiBaseUrl && siteStore.currentTenant?.id));
const hasLoginState = computed(() => Boolean(authStore.token?.accessToken));
const currentBootstrap = computed(() => {
  if (!siteStore.currentTenant?.id || !authStore.bootstrap) {
    return null;
  }
  return authStore.bootstrap.site.tenantId === siteStore.currentTenant.id ? authStore.bootstrap : null;
});
const supportsSessionManagement = computed(() => currentBootstrap.value?.capabilities.supportsSessionManagement ?? false);
const supportsRefreshToken = computed(() => currentBootstrap.value?.capabilities.supportsRefreshToken ?? Boolean(authStore.token?.refreshToken));
const currentSession = computed(() => (
  authStore.sessions.find((item) => item.isCurrent)
  ?? authStore.sessions.find((item) => item.id === authStore.token?.sessionId)
  ?? null
));
const sortedSessions = computed(() => [...authStore.sessions].sort((left, right) => {
  if (left.isCurrent !== right.isCurrent) {
    return left.isCurrent ? -1 : 1;
  }
  return new Date(right.lastUsedAt).getTime() - new Date(left.lastUsedAt).getTime();
}));
const sessionTableData = computed(() => sortedSessions.value.filter((item) => {
  const keyword = sessionSearch.value.trim().toLowerCase();
  if (sessionFilter.value === 'current') {
    if (!item.isCurrent) {
      return false;
    }
  }
  if (sessionFilter.value === 'other') {
    if (item.isCurrent) {
      return false;
    }
  }
  if (!keyword) {
    return true;
  }
  return buildSessionSearchText(item).includes(keyword);
}));
const activeSessionCount = computed(() => authStore.sessions.length);
const otherSessionCount = computed(() => authStore.sessions.filter((item) => !item.isCurrent).length);
const currentSessionCount = computed(() => authStore.sessions.filter((item) => item.isCurrent).length);
const accessTokenStatus = computed(() => buildExpiryState(authStore.token?.expiresAt));
const refreshTokenStatus = computed(() => buildExpiryState(authStore.token?.refreshExpiresAt));
const hasUrgentExpiry = computed(() => accessTokenStatus.value.urgent || refreshTokenStatus.value.urgent);
const urgentExpiryMessages = computed(() => {
  let messages: string[] = [];
  if (accessTokenStatus.value.urgent) {
    messages = [...messages, `访问令牌${accessTokenStatus.value.text}`];
  }
  if (refreshTokenStatus.value.urgent) {
    messages = [...messages, `刷新令牌${refreshTokenStatus.value.text}`];
  }
  return messages;
});

function parseDateTime(value?: string | null): number | null {
  if (!value) {
    return null;
  }
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function buildExpiryState(value?: string | null) {
  const timestamp = parseDateTime(value);
  if (!timestamp) {
    return {
      color: 'gray',
      text: '未知',
      expired: false,
      urgent: false,
    } as const;
  }

  const diffMs = timestamp - Date.now();
  if (diffMs <= 0) {
    return {
      color: 'red',
      text: '已过期',
      expired: true,
      urgent: true,
    } as const;
  }

  const diffHours = diffMs / (1000 * 60 * 60);
  if (diffHours <= 24) {
    return {
      color: 'orange',
      text: '24 小时内到期',
      expired: false,
      urgent: true,
    } as const;
  }

  if (diffHours <= 72) {
    return {
      color: 'gold',
      text: '3 天内到期',
      expired: false,
      urgent: false,
    } as const;
  }

  return {
    color: 'green',
    text: '有效',
    expired: false,
    urgent: false,
  } as const;
}

function buildSessionSearchText(item: SessionItem): string {
  return [
    item.id,
    item.name,
    item.deviceName,
    item.clientVersion,
    item.lastUsedAt,
    item.createdAt,
  ]
    .map((value) => String(value || '').trim().toLowerCase())
    .join(' ');
}

function hasSessionChanged(previous: SessionItem | undefined, next: SessionItem): boolean {
  if (!previous) {
    return true;
  }
  return previous.isCurrent !== next.isCurrent
    || previous.deviceName !== next.deviceName
    || previous.name !== next.name
    || previous.clientVersion !== next.clientVersion
    || previous.lastUsedAt !== next.lastUsedAt
    || previous.expiresAt !== next.expiresAt
    || previous.refreshExpiresAt !== next.refreshExpiresAt
    || previous.createdAt !== next.createdAt;
}

function clearSessionHighlight() {
  if (highlightResetTimer) {
    clearTimeout(highlightResetTimer);
    highlightResetTimer = null;
  }
}

function markUpdatedSessions(sessionIds: number[]) {
  clearSessionHighlight();
  recentlyUpdatedSessionIds.value = sessionIds;
  if (!sessionIds.length) {
    return;
  }
  highlightResetTimer = setTimeout(() => {
    recentlyUpdatedSessionIds.value = [];
    highlightResetTimer = null;
  }, 12000);
}

async function ensureCurrentTenantBootstrap() {
  if (!siteStore.currentTenant?.apiBaseUrl || !siteStore.currentTenant?.id || !authStore.token?.accessToken) {
    authStore.bootstrap = null;
    return null;
  }

  if (currentBootstrap.value) {
    return currentBootstrap.value;
  }

  const bootstrap = await authStore.refreshBootstrap({
    apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
  });
  if (bootstrap) {
    siteStore.applyTenantBootstrap(siteStore.currentTenant.id, bootstrap);
  }
  return bootstrap;
}

async function loadSessions(options?: { silent?: boolean; source?: 'initial' | 'manual' | 'token-refresh' | 'tenant-change' }) {
  siteStore.syncCurrentTenantWithAuth(authStore.token?.tenantId);

  if (!siteStore.currentTenant?.apiBaseUrl || !hasLoginState.value) {
    authStore.bootstrap = null;
    authStore.sessions = [];
    sessionRefreshSummary.value = '';
    lastSessionRefreshAt.value = '';
    markUpdatedSessions([]);
    return;
  }

  loading.value = true;
  try {
    await ensureCurrentTenantBootstrap();
    if (!supportsSessionManagement.value) {
      authStore.sessions = [];
      sessionRefreshSummary.value = '';
      lastSessionRefreshAt.value = '';
      markUpdatedSessions([]);
      return;
    }
    const previousSessions = new Map(authStore.sessions.map((item) => [item.id, item]));
    const sessions = await authStore.loadSessions({
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
    const changedSessionIds = sessions
      .filter((item) => hasSessionChanged(previousSessions.get(item.id), item))
      .map((item) => item.id);
    markUpdatedSessions(changedSessionIds);
    lastSessionRefreshAt.value = new Date().toLocaleString();
    if (changedSessionIds.length) {
      sessionRefreshSummary.value = `本次同步更新了 ${changedSessionIds.length} 条会话记录`;
    } else if (sessions.length) {
      sessionRefreshSummary.value = '本次同步未发现会话变化';
    } else {
      sessionRefreshSummary.value = '当前租户暂无会话记录';
    }
    if (!options?.silent && options?.source === 'manual') {
      Message.success('会话列表已刷新');
    }
  } catch (error) {
    if (!authStore.token?.accessToken) {
      await router.replace('/login');
      return;
    }
    Message.error(error instanceof Error ? error.message : '获取会话列表失败');
  } finally {
    loading.value = false;
  }
}

async function handleRefreshToken() {
  if (!siteStore.currentTenant?.apiBaseUrl || !siteStore.currentTenant?.id) {
    Message.warning('当前没有可刷新的租户上下文');
    return;
  }
  if (!supportsRefreshToken.value) {
    Message.warning('当前租户未启用刷新令牌能力');
    return;
  }

  refreshingToken.value = true;
  try {
    await authStore.refreshAuthToken({
      id: siteStore.currentTenant.id,
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
    const bootstrap = await authStore.refreshBootstrap({
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
    if (bootstrap) {
      siteStore.applyTenantBootstrap(siteStore.currentTenant.id, bootstrap);
    }
    await loadSessions({ silent: true, source: 'token-refresh' });
    Message.success('令牌刷新成功');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '令牌刷新失败');
  } finally {
    refreshingToken.value = false;
  }
}

async function handleRevokeSession(sessionId: number) {
  if (!siteStore.currentTenant?.apiBaseUrl) {
    Message.warning('当前没有可用的租户 API 地址');
    return;
  }
  if (!supportsSessionManagement.value) {
    Message.warning('当前租户未启用会话管理能力');
    return;
  }

  const targetSession = authStore.sessions.find((item) => item.id === sessionId);
  Modal.confirm({
    title: '吊销该会话？',
    content: targetSession
      ? `会话吊销后，该设备需要重新登录：${targetSession.deviceName || targetSession.name || `会话 #${sessionId}`}`
      : `会话吊销后，该设备需要重新登录：#${sessionId}`,
    okText: '确认吊销',
    okButtonProps: {
      status: 'danger',
    },
    cancelText: '取消',
    onOk: async () => {
      try {
        await authStore.revokeSession(
          {
            apiBaseUrl: siteStore.currentTenant!.apiBaseUrl,
          },
          sessionId,
        );
        Message.success('会话已吊销');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '会话吊销失败');
      }
    },
  });
}

onMounted(async () => {
  await initialiseDesktopContext(siteStore, authStore);
  await loadSessions({ silent: true, source: 'initial' });
});

onBeforeUnmount(() => {
  clearSessionHighlight();
});

watch(
  () => siteStore.currentTenant?.id,
  () => {
    sessionSearch.value = '';
    sessionFilter.value = 'all';
    sessionRefreshSummary.value = '';
    lastSessionRefreshAt.value = '';
    void loadSessions({ silent: true, source: 'tenant-change' });
  },
);

watch(
  () => [authStore.token?.tenantId ?? null, accessTokenStatus.value.text, refreshTokenStatus.value.text].join('|'),
  () => {
    if (!hasUrgentExpiry.value) {
      lastExpiryReminderKey.value = '';
      return;
    }
    const reminderKey = `${authStore.token?.tenantId ?? 'none'}:${accessTokenStatus.value.text}:${refreshTokenStatus.value.text}`;
    if (lastExpiryReminderKey.value === reminderKey) {
      return;
    }
    lastExpiryReminderKey.value = reminderKey;
    Message.warning(`登录态提醒：${urgentExpiryMessages.value.join('；')}，建议尽快刷新令牌或重新登录`);
  },
  { immediate: true },
);
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>会话管理</h2>
        <p>查看当前账号在该租户下的全部登录会话，支持刷新令牌并吊销旧设备会话。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="purple">{{ authStore.user?.username || '未登录' }}</a-tag>
          <a-tag color="green">当前租户：{{ siteStore.currentTenant?.name || '未选择' }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="loading" :disabled="currentBootstrap ? !supportsSessionManagement : false" @click="loadSessions({ source: 'manual' })">
            刷新会话
          </a-button>
          <a-button type="primary" :loading="refreshingToken" :disabled="currentBootstrap ? !supportsRefreshToken : false" @click="handleRefreshToken">
            刷新令牌
          </a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-alert
        v-if="!hasTenantContext"
        type="warning"
        title="尚未选择可用租户"
        content="请先配置并选中带有桌面端 API 地址的租户。"
      />

      <a-alert
        v-else-if="!hasLoginState"
        type="warning"
        title="当前租户未登录"
        content="请先完成登录，再查看该租户下的会话列表。"
      />

      <a-row v-else :gutter="16">
        <a-col :span="9">
          <a-card title="当前登录态">
          <a-space direction="vertical" fill>
            <a-alert
              v-if="hasUrgentExpiry"
              type="warning"
              title="登录态即将失效"
              :content="urgentExpiryMessages.join('；')"
            />
            <a-descriptions :column="1" bordered size="large">
              <a-descriptions-item label="当前用户">{{ authStore.user?.username || '-' }}</a-descriptions-item>
              <a-descriptions-item label="访问令牌到期">
                <div class="session-expiry-cell">
                  <span>{{ authStore.token?.expiresAt || '-' }}</span>
                  <a-tag :color="accessTokenStatus.color">{{ accessTokenStatus.text }}</a-tag>
                </div>
              </a-descriptions-item>
              <a-descriptions-item label="刷新令牌到期">
                <div class="session-expiry-cell">
                  <span>{{ authStore.token?.refreshExpiresAt || '-' }}</span>
                  <a-tag :color="refreshTokenStatus.color">{{ refreshTokenStatus.text }}</a-tag>
                </div>
              </a-descriptions-item>
              <a-descriptions-item label="当前会话 ID">{{ authStore.token?.sessionId || '-' }}</a-descriptions-item>
              <a-descriptions-item label="当前设备">{{ currentSession?.deviceName || currentSession?.name || '-' }}</a-descriptions-item>
              <a-descriptions-item label="客户端版本">{{ currentSession?.clientVersion || '-' }}</a-descriptions-item>
              <a-descriptions-item label="最近使用">{{ currentSession?.lastUsedAt || '-' }}</a-descriptions-item>
              <a-descriptions-item label="登录时间">{{ currentSession?.createdAt || '-' }}</a-descriptions-item>
            </a-descriptions>
            <div class="session-summary-grid">
              <div class="session-summary-stat">
                <span class="section-muted">会话总数</span>
                <strong>{{ activeSessionCount }}</strong>
              </div>
              <div class="session-summary-stat">
                <span class="section-muted">当前设备</span>
                <strong>{{ currentSessionCount }}</strong>
              </div>
              <div class="session-summary-stat">
                <span class="section-muted">其他设备</span>
                <strong>{{ otherSessionCount }}</strong>
              </div>
            </div>
            <a-button :loading="refreshingToken" :disabled="!supportsRefreshToken" @click="handleRefreshToken">
              刷新访问令牌
            </a-button>
          </a-space>
          </a-card>
        </a-col>
        <a-col :span="15">
          <a-card title="会话列表">
            <template #extra>
              <a-space wrap>
                <a-input
                  v-model="sessionSearch"
                  allow-clear
                  size="small"
                  class="session-search-input"
                  placeholder="搜索设备名、版本、会话 ID"
                />
                <a-radio-group v-model="sessionFilter" type="button" size="small">
                  <a-radio value="all">全部</a-radio>
                  <a-radio value="current">当前设备</a-radio>
                  <a-radio value="other">其他设备</a-radio>
                </a-radio-group>
                <a-tag :color="supportsSessionManagement ? 'green' : 'gray'">
                  {{ supportsSessionManagement ? '已启用会话管理' : '未启用会话管理' }}
                </a-tag>
              </a-space>
            </template>
            <a-alert
              v-if="currentBootstrap && !supportsSessionManagement"
              class="session-inline-alert"
              type="info"
              title="当前租户未启用会话管理"
              content="服务端未开放会话列表与会话吊销接口，当前页面仍可查看本地保存的登录态信息。"
            />
            <div v-if="lastSessionRefreshAt" class="session-refresh-meta">
              <span>{{ sessionRefreshSummary || '会话状态已同步' }}</span>
              <span class="section-muted">刷新时间：{{ lastSessionRefreshAt }}</span>
            </div>
            <PageStickyTable :data="sessionTableData" :loading="loading" :pagination="pagination" row-key="id">
            <template #empty>
              <a-empty :description="supportsSessionManagement ? '暂无会话数据，可先点击刷新重新拉取。' : '当前租户未开放会话列表能力。'" />
            </template>
            <a-table-column title="会话 ID" data-index="id" :width="90" />
            <a-table-column title="设备">
              <template #cell="{ record }">
                <div class="session-device-cell">
                  <div class="session-device-header">
                    <strong>{{ record.deviceName || record.name || '未知设备' }}</strong>
                    <a-tag v-if="recentlyUpdatedSessionIds.includes(record.id)" color="arcoblue" size="small">已更新</a-tag>
                  </div>
                  <span class="section-muted">版本：{{ record.clientVersion || '-' }}</span>
                  <span class="section-muted">创建：{{ record.createdAt || '-' }}</span>
                </div>
              </template>
            </a-table-column>
            <a-table-column title="最近使用" data-index="lastUsedAt" :width="180" />
            <a-table-column title="访问令牌到期" :width="180">
              <template #cell="{ record }">
                <div class="session-expiry-cell">
                  <span>{{ record.expiresAt || '-' }}</span>
                  <a-tag :color="buildExpiryState(record.expiresAt).color">{{ buildExpiryState(record.expiresAt).text }}</a-tag>
                </div>
              </template>
            </a-table-column>
            <a-table-column title="刷新令牌到期" :width="180">
              <template #cell="{ record }">
                <div class="session-expiry-cell">
                  <span>{{ record.refreshExpiresAt || '-' }}</span>
                  <a-tag :color="buildExpiryState(record.refreshExpiresAt).color">{{ buildExpiryState(record.refreshExpiresAt).text }}</a-tag>
                </div>
              </template>
            </a-table-column>
            <a-table-column title="当前会话" :width="96">
              <template #cell="{ record }">
                <a-tag :color="record.isCurrent ? 'green' : 'gray'">{{ record.isCurrent ? '当前' : '其他设备' }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="操作" :width="88">
              <template #cell="{ record }">
                <a-button type="text" size="small" status="danger" :disabled="record.isCurrent" @click="handleRevokeSession(record.id)">
                  吊销
                </a-button>
              </template>
            </a-table-column>
            </PageStickyTable>
          </a-card>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<style scoped>
.session-inline-alert {
  margin-bottom: 16px;
}

.session-search-input {
  width: 220px;
}

.session-refresh-meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 8px 16px;
  margin-bottom: 12px;
  font-size: 12px;
  color: #4e5969;
}

.session-expiry-cell {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.session-summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.session-summary-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid #e5e6eb;
  border-radius: 12px;
  background: #f7f8fa;
}

.session-summary-stat strong {
  font-size: 20px;
  line-height: 1.2;
}

.session-device-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.session-device-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
</style>
