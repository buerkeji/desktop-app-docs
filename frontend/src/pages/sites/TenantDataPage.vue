<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { TenantPingResult } from '../../services/desktop-api.service';
import { pingTenantApi } from '../../services/desktop-api.service';
import { getArticleList, getToolList } from '../../services/content.service';
import { useAuthStore } from '../../stores/auth.store';
import { useSiteStore } from '../../stores/site.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import PageStickyTable from '../../components/PageStickyTable.vue';
import { createTablePagination } from '../../utils/table-pagination';

interface TenantDataStats {
  toolCount: number;
  articleCount: number;
  categoryCount: number;
  tagCount: number;
  refreshedAt: string;
}

const authStore = useAuthStore();
const siteStore = useSiteStore();
const dictionaryStore = useDictionaryStore();
const stats = ref<TenantDataStats | null>(null);
const pingResult = ref<TenantPingResult | null>(null);
const pingLoading = ref(false);
const loading = ref(false);
const loadError = ref('');
const activeLoadKey = ref('');
const pagination = createTablePagination();

const isReady = computed(() => Boolean(siteStore.currentTenant?.apiBaseUrl && authStore.token?.accessToken));
const currentBootstrap = computed(() => authStore.bootstrap);
const currentUser = computed(() => authStore.user);
const domainItems = computed(() => currentBootstrap.value?.site.domains ?? []);
const primaryDomainText = computed(() => {
  const primary = domainItems.value.find((item) => item.isPrimary);
  return primary?.domain || siteStore.currentTenant?.baseUrl || '-';
});
const activeDomainCount = computed(() => domainItems.value.filter((item) => item.isActive).length);
const capabilityItems = computed(() => {
  const capabilities = currentBootstrap.value?.capabilities;
  if (!capabilities) {
    return [];
  }

  return [
    { label: '工具管理', enabled: capabilities.canPublishTool },
    { label: '文章管理', enabled: capabilities.canPublishArticle },
    { label: '媒体上传', enabled: capabilities.canUploadMedia },
    { label: '分类标签', enabled: capabilities.supportsTags },
    { label: '会话管理', enabled: capabilities.supportsSessionManagement },
    { label: '刷新令牌', enabled: capabilities.supportsRefreshToken },
    { label: '请求追踪', enabled: capabilities.supportsRequestId },
    { label: '幂等提交', enabled: capabilities.supportsIdempotency },
  ];
});
const supportedClientTypesText = computed(() => {
  const clientTypes = currentBootstrap.value?.capabilities.supportedClientTypes ?? [];
  return clientTypes.length ? clientTypes.join(', ') : '-';
});

function formatLoadError(error: unknown): string {
  const message = error instanceof Error ? error.message : '加载当前租户真实数据失败';
  if (/too many attempts/i.test(message)) {
    return '当前租户数据请求过于频繁，已被服务端限流，请稍后点击“刷新真实数据”重试。';
  }
  return message;
}

async function loadTenantData(force = false) {
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    stats.value = null;
    pingResult.value = null;
    loadError.value = '';
    return;
  }

  const requestKey = `${siteStore.currentTenant.id}:${authStore.token.accessToken}`;
  if (!force && (loading.value || activeLoadKey.value === requestKey)) {
    return;
  }

  activeLoadKey.value = requestKey;
  loading.value = true;
  loadError.value = '';
  try {
    if (!authStore.bootstrap || authStore.bootstrap.site.tenantId !== siteStore.currentTenant.id) {
      const bootstrap = await authStore.refreshBootstrap({
        apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
      });
      if (bootstrap && siteStore.currentTenant?.id) {
        siteStore.applyTenantBootstrap(siteStore.currentTenant.id, bootstrap);
      }
    }

    const [toolResult, articleResult] = await Promise.all([
      getToolList(
        { id: siteStore.currentTenant.id, apiBaseUrl: siteStore.currentTenant.apiBaseUrl },
        authStore.token.accessToken,
        { page: 1, perPage: 1, sort: 'latest' },
      ),
      getArticleList(
        { id: siteStore.currentTenant.id, apiBaseUrl: siteStore.currentTenant.apiBaseUrl },
        authStore.token.accessToken,
        { page: 1, perPage: 1, sort: 'latest' },
      ),
      dictionaryStore.initialise(
        { id: siteStore.currentTenant.id, apiBaseUrl: siteStore.currentTenant.apiBaseUrl },
        authStore.token.accessToken,
      ),
    ]);

    stats.value = {
      toolCount: toolResult.pagination.total,
      articleCount: articleResult.pagination.total,
      categoryCount: dictionaryStore.categories.length,
      tagCount: dictionaryStore.tags.length,
      refreshedAt: new Date().toLocaleString(),
    };
  } catch (error) {
    stats.value = null;
    loadError.value = formatLoadError(error);
  } finally {
    loading.value = false;
  }
}

async function checkConnection() {
  if (!siteStore.currentTenant?.apiBaseUrl) {
    pingResult.value = {
      ok: false,
      message: '当前租户未配置桌面端 API 地址',
      checkedAt: new Date().toLocaleString(),
    };
    return;
  }

  pingLoading.value = true;
  try {
    pingResult.value = await pingTenantApi({
      apiBaseUrl: siteStore.currentTenant.apiBaseUrl,
    });
  } finally {
    pingLoading.value = false;
  }
}

watch(
  () => [siteStore.currentTenant?.id ?? null, authStore.token?.accessToken ?? null],
  () => {
    void loadTenantData();
  },
  { immediate: true },
);
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>当前租户数据</h2>
        <p>查看当前站点和租户的内容统计、域名信息、能力开关与会话状态。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="green">{{ siteStore.currentSite?.name || '未选择站点' }}</a-tag>
          <a-tag color="arcoblue">{{ siteStore.currentTenant?.name || '未选择租户' }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="loading" @click="loadTenantData(true)">刷新</a-button>
          <a-button type="primary" :loading="pingLoading" @click="checkConnection">检测接口</a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-alert
        v-if="!isReady"
        type="warning"
        title="暂无数据"
        content="请先选择租户并完成登录。"
      />

      <a-alert
        v-if="loadError"
        type="warning"
        title="加载失败"
        :content="loadError"
      />

      <a-alert
        v-if="pingResult"
        :type="pingResult.ok ? 'success' : 'warning'"
        :title="pingResult.ok ? '接口正常' : '接口异常'"
        :content="`${pingResult.message}，检测时间：${pingResult.checkedAt}${pingResult.requestId ? `，request_id：${pingResult.requestId}` : ''}`"
      />

      <div class="stat-grid site-data-stat-grid">
        <a-card title="工具" :loading="loading" class="site-data-stat-card">
        <a-statistic :value="stats?.toolCount ?? 0" />
      </a-card>
      <a-card title="文章" :loading="loading" class="site-data-stat-card">
        <a-statistic :value="stats?.articleCount ?? 0" />
      </a-card>
      <a-card title="分类" :loading="loading" class="site-data-stat-card">
        <a-statistic :value="stats?.categoryCount ?? 0" />
      </a-card>
      <a-card title="标签" :loading="loading" class="site-data-stat-card">
        <a-statistic :value="stats?.tagCount ?? 0" />
      </a-card>
      <a-card title="域名" :loading="loading" class="site-data-stat-card">
        <a-statistic :value="domainItems.length" />
      </a-card>
        <a-card title="启用" :loading="loading" class="site-data-stat-card">
          <a-statistic :value="activeDomainCount" />
        </a-card>
      </div>

      <div class="tenant-data-grid">
        <div class="tenant-data-grid__main">
          <a-card title="站点信息" :loading="loading" class="tenant-data-card">
          <a-descriptions :column="2" bordered size="large" layout="inline-horizontal">
            <a-descriptions-item label="所属站点">{{ siteStore.currentSite?.name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="租户名称">{{ siteStore.currentTenant?.name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="站点标识">{{ currentBootstrap?.site.slug || '-' }}</a-descriptions-item>
            <a-descriptions-item label="站点版本">{{ currentBootstrap?.site.version || '-' }}</a-descriptions-item>
            <a-descriptions-item label="前台主域名">{{ primaryDomainText }}</a-descriptions-item>
            <a-descriptions-item label="API 地址">{{ siteStore.currentTenant?.apiBaseUrl || '-' }}</a-descriptions-item>
            <a-descriptions-item label="支持客户端">{{ supportedClientTypesText }}</a-descriptions-item>
            <a-descriptions-item label="最近刷新">{{ stats?.refreshedAt || '-' }}</a-descriptions-item>
          </a-descriptions>
          </a-card>

          <a-card title="会话信息" :loading="loading" class="tenant-data-card">
          <a-descriptions :column="2" bordered size="large" layout="inline-horizontal">
            <a-descriptions-item label="登录用户">{{ currentUser?.name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="登录账号">{{ currentUser?.username || '-' }}</a-descriptions-item>
            <a-descriptions-item label="会话 ID">{{ authStore.token?.sessionId || '-' }}</a-descriptions-item>
            <a-descriptions-item label="访问令牌到期">{{ authStore.token?.expiresAt || '-' }}</a-descriptions-item>
            <a-descriptions-item label="刷新令牌到期">{{ authStore.token?.refreshExpiresAt || '-' }}</a-descriptions-item>
          </a-descriptions>
          </a-card>

          <a-card title="域名信息" :loading="loading" class="tenant-data-card">
          <PageStickyTable
            :data="domainItems"
            :pagination="pagination"
            row-key="id"
            size="small"
          >
            <template #empty>
              <a-empty description="当前 bootstrap 未返回域名列表" />
            </template>
            <template #columns>
              <a-table-column title="域名" data-index="domain" />
              <a-table-column title="主域名" :width="120">
                <template #cell="{ record }">
                  <a-tag :color="record.isPrimary ? 'green' : 'gray'">{{ record.isPrimary ? '是' : '否' }}</a-tag>
                </template>
              </a-table-column>
              <a-table-column title="状态" :width="96">
                <template #cell="{ record }">
                  <a-tag :color="record.isActive ? 'green' : 'red'">{{ record.isActive ? '启用' : '停用' }}</a-tag>
                </template>
              </a-table-column>
            </template>
          </PageStickyTable>
          </a-card>
        </div>

        <div class="tenant-data-grid__side">
          <a-card title="能力概览" :loading="loading" class="tenant-data-card">
          <div class="tenant-capability-list">
            <a-tag v-for="item in capabilityItems" :key="item.label" :color="item.enabled ? 'green' : 'gray'">
              {{ item.label }}：{{ item.enabled ? '已启用' : '未启用' }}
            </a-tag>
          </div>
          </a-card>

          <a-card title="限制与客户端" :loading="loading" class="tenant-data-card">
          <a-descriptions :column="1" bordered size="large">
            <a-descriptions-item label="最大上传">{{ currentBootstrap?.limits.maxUploadSizeMb ?? '-' }} MB</a-descriptions-item>
            <a-descriptions-item label="标题长度">{{ currentBootstrap?.limits.maxTitleLength ?? '-' }}</a-descriptions-item>
            <a-descriptions-item label="Slug 长度">{{ currentBootstrap?.limits.maxSlugLength ?? '-' }}</a-descriptions-item>
            <a-descriptions-item label="批量上限">{{ currentBootstrap?.limits.maxBatchSize ?? '-' }}</a-descriptions-item>
            <a-descriptions-item label="请求 ID">{{ currentBootstrap?.client.requestId || '-' }}</a-descriptions-item>
            <a-descriptions-item label="客户端类型">{{ currentBootstrap?.client.clientType || '-' }}</a-descriptions-item>
            <a-descriptions-item label="客户端版本">{{ currentBootstrap?.client.clientVersion || '-' }}</a-descriptions-item>
          </a-descriptions>
          </a-card>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tenant-data-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.65fr) minmax(320px, 0.95fr);
  gap: 16px;
}

.site-data-stat-grid {
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.site-data-stat-card :deep(.arco-card-header) {
  padding-bottom: 8px;
}

.site-data-stat-card :deep(.arco-card-header-title) {
  font-size: 14px;
  white-space: nowrap;
}

.site-data-stat-card :deep(.arco-card-body) {
  padding-top: 0;
}

.site-data-stat-card :deep(.arco-statistic-content) {
  font-size: 28px;
  white-space: nowrap;
}

.tenant-data-grid__main,
.tenant-data-grid__side {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.tenant-data-card {
  border-radius: 14px;
}

.tenant-capability-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.tenant-capability-list :deep(.arco-tag) {
  padding: 6px 10px;
}
</style>
