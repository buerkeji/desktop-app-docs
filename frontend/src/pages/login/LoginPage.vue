<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import TitleBar from '../../layouts/TitleBar.vue';
import { useAuthStore } from '../../stores/auth.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import { useSiteStore } from '../../stores/site.store';
import { setWorkspaceWindow } from '../../utils/window';

const router = useRouter();
const authStore = useAuthStore();
const dictionaryStore = useDictionaryStore();
const siteStore = useSiteStore();
const submitting = ref(false);
const syncing = ref(false);
const creatingSite = ref(false);
const showSiteNameField = ref(false);
const loginMode = ref<'password' | 'ticket'>('password');

const form = reactive({
  tenantId: siteStore.currentTenantId ?? siteStore.tenants[0]?.id ?? 0,
  username: '',
  password: '',
  ticket: '',
});
const siteForm = reactive({
  name: '',
  baseUrl: '',
});

const filteredTenants = computed(() => siteStore.filteredTenants);
const currentSiteSummary = computed(() => {
  const siteName = siteStore.currentSite?.name || '未选择站点';
  const tenantTotal = filteredTenants.value.length;
  return `${siteName} · ${tenantTotal} 个租户`;
});

onMounted(async () => {
  await siteStore.initialise();
  if (!siteStore.currentSiteId && siteStore.sites[0]?.id) {
    siteStore.selectSite(siteStore.sites[0].id);
  }
  if (!form.tenantId) {
    form.tenantId = filteredTenants.value[0]?.id ?? siteStore.tenants[0]?.id ?? 0;
  }
  if (siteStore.currentSite?.baseUrl && !siteForm.baseUrl) {
    siteForm.baseUrl = siteStore.currentSite.baseUrl;
  }
  if (siteForm.name.trim()) {
    showSiteNameField.value = true;
  }
});

watch(() => siteStore.currentSiteId, () => {
  form.tenantId = filteredTenants.value[0]?.id ?? 0;
  const currentSite = siteStore.currentSite;
  if (currentSite?.baseUrl) {
    siteForm.baseUrl = currentSite.baseUrl;
  }
  if (currentSite?.name) {
    siteForm.name = currentSite.name;
    showSiteNameField.value = true;
  }
});

watch(() => form.tenantId, (tenantId) => {
  const tenant = siteStore.tenants.find((item) => item.id === tenantId);
  if (tenant?.lastUsername) {
    form.username = tenant.lastUsername;
  }
});

function normaliseSiteBaseUrl(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }
  return /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;
}

function suggestSiteName(baseUrl: string): string {
  try {
    const hostname = new URL(baseUrl).hostname.replace(/^www\./i, '');
    return hostname || '我的站点';
  } catch {
    return '我的站点';
  }
}

async function handleCreateSite() {
  const baseUrl = normaliseSiteBaseUrl(siteForm.baseUrl);
  if (!baseUrl) {
    Message.warning('请输入站点地址');
    return;
  }

  creatingSite.value = true;
  try {
    const site = await siteStore.addSite({
      name: siteForm.name.trim() || suggestSiteName(baseUrl),
      baseUrl,
      description: '',
    });
    siteForm.baseUrl = site.site.baseUrl;
    siteForm.name = site.site.name;
    form.tenantId = site.tenants[0]?.id ?? 0;
    Message.success(site.tenants.length ? '站点已添加，并已同步租户' : '站点已添加，请点击“同步租户”获取租户列表');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '添加站点失败');
  } finally {
    creatingSite.value = false;
  }
}

async function syncCurrentSiteTenants() {
  if (!siteStore.currentSiteId) {
    Message.warning('请先选择站点');
    return;
  }

  syncing.value = true;
  try {
    const tenants = await siteStore.syncSiteTenants(siteStore.currentSiteId);
    if (!tenants.length) {
      Message.warning('当前站点未发现可用租户');
      return;
    }

    form.tenantId = tenants[0]?.id ?? form.tenantId;
    Message.success(`已同步 ${tenants.length} 个租户`);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '同步租户失败');
  } finally {
    syncing.value = false;
  }
}

async function handleSubmit() {
  if (!form.tenantId) {
    Message.warning('请先选择要登录的租户');
    return;
  }

  const tenant = siteStore.tenants.find((item) => item.id === form.tenantId);
  if (!tenant?.apiBaseUrl) {
    Message.warning('当前租户未配置可用的桌面端 API 地址');
    return;
  }

  submitting.value = true;
  try {
    siteStore.selectSite(tenant.siteId);
    siteStore.selectTenant(tenant.id);

    await authStore.login(
      {
        id: tenant.id,
        apiBaseUrl: tenant.apiBaseUrl,
      },
      {
        tenantId: tenant.id,
        tenantSlug: tenant.tenantSlug,
        username: form.username,
        password: form.password,
      },
    );
    await finishLogin(tenant.id, tenant.apiBaseUrl, form.username);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '登录失败，请检查租户地址或账号密码');
  } finally {
    submitting.value = false;
  }
}

async function handleTicketSubmit() {
  if (!form.tenantId) {
    Message.warning('请先选择要登录的租户');
    return;
  }
  if (!form.ticket.trim()) {
    Message.warning('请输入一次性登录票据');
    return;
  }

  const tenant = siteStore.tenants.find((item) => item.id === form.tenantId);
  if (!tenant?.apiBaseUrl) {
    Message.warning('当前租户未配置可用的桌面端 API 地址');
    return;
  }

  submitting.value = true;
  try {
    siteStore.selectSite(tenant.siteId);
    siteStore.selectTenant(tenant.id);

    const result = await authStore.loginWithTicket(
      {
        id: tenant.id,
        apiBaseUrl: tenant.apiBaseUrl,
      },
      form.ticket,
    );

    form.username = result.user.username;
    await finishLogin(tenant.id, tenant.apiBaseUrl, result.user.username);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '票据登录失败，请检查票据是否已过期或已使用');
  } finally {
    submitting.value = false;
  }
}

async function finishLogin(tenantId: number, apiBaseUrl: string, username: string) {
  siteStore.updateTenantLastUsername(tenantId, username);
  const bootstrap = await authStore.refreshBootstrap({
    apiBaseUrl,
  });
  if (bootstrap) {
    siteStore.applyTenantBootstrap(tenantId, bootstrap);
  }

  await dictionaryStore.initialise(
    {
      apiBaseUrl,
    },
    authStore.token?.accessToken ?? null,
    true,
  );

  Message.success('登录成功，已同步启动配置与基础字典');
  router.push('/dashboard');
  setWorkspaceWindow();
}
</script>

<template>
  <div class="login-page">
    <TitleBar />
    <div class="login-page__body">
      <section class="login-page__panel">
        <div class="login-page__panel-header">
          <div class="login-page__brand">
            <span class="login-page__badge">ZQ</span>
            <div>
              <h1>登录工作台</h1>
              <p>站点、租户、账号三步完成登录</p>
            </div>
          </div>
          <span class="login-page__summary">{{ currentSiteSummary }}</span>
        </div>

        <a-form :model="siteForm" layout="vertical" class="login-page__section">
          <a-form-item field="quickSiteBaseUrl" label="站点地址">
            <a-space fill>
              <a-input v-model="siteForm.baseUrl" placeholder="例如 http://localhost/ 或 https://你的管理后台域名" allow-clear />
              <a-button type="primary" :loading="creatingSite" @click="handleCreateSite">添加站点</a-button>
            </a-space>
          </a-form-item>
          <a-button type="text" size="small" class="login-page__toggle" @click="showSiteNameField = !showSiteNameField">
            {{ showSiteNameField ? '收起站点名称' : '填写站点名称（可选）' }}
          </a-button>
          <a-form-item v-if="showSiteNameField" field="quickSiteName" label="站点名称">
            <a-input v-model="siteForm.name" placeholder="可选，不填则自动用域名生成" allow-clear />
          </a-form-item>
        </a-form>

        <a-form :model="form" layout="vertical" class="login-page__section" @submit.prevent="loginMode === 'password' ? handleSubmit() : handleTicketSubmit()">
          <a-form-item field="siteId" label="站点">
            <a-space fill>
              <a-select
                :model-value="siteStore.currentSiteId ?? undefined"
                placeholder="请选择站点"
                @change="(value) => siteStore.selectSite(Number(value))"
              >
                <a-option v-for="item in siteStore.sites" :key="item.id" :value="item.id">
                  {{ item.name }} / {{ item.baseUrl }}
                </a-option>
              </a-select>
              <a-button :loading="syncing" @click="syncCurrentSiteTenants">同步租户</a-button>
            </a-space>
          </a-form-item>
          <a-form-item field="tenantId" label="登录租户">
            <a-select v-model="form.tenantId" placeholder="请选择租户">
              <a-option v-for="item in filteredTenants" :key="item.id" :value="item.id">
                {{ item.name }} / {{ item.baseUrl }}
              </a-option>
            </a-select>
          </a-form-item>
          <a-tabs v-model:active-key="loginMode" lazy-load size="small">
            <a-tab-pane key="password" title="账号密码">
              <a-form-item field="username" label="账号">
                <a-input v-model="form.username" placeholder="请输入租户管理员账号" />
              </a-form-item>
              <a-form-item field="password" label="密码">
                <a-input-password v-model="form.password" placeholder="请输入密码" />
              </a-form-item>
            </a-tab-pane>
            <a-tab-pane key="ticket" title="一次性票据">
              <a-form-item field="ticket" label="登录票据">
                <a-input
                  v-model="form.ticket"
                  placeholder="请输入后台生成的一次性登录票据"
                  allow-clear
                />
              </a-form-item>
              <a-alert type="warning">
                票据通常 5 分钟内有效，且只能使用一次。请先在平台后台的租户管理里生成桌面端登录票据。
              </a-alert>
            </a-tab-pane>
          </a-tabs>
          <a-space direction="vertical" fill>
            <a-button
              type="primary"
              long
              size="large"
              :loading="submitting"
              @click="loginMode === 'password' ? handleSubmit() : handleTicketSubmit()"
            >
              {{ loginMode === 'password' ? '登录并进入后台' : '使用票据登录' }}
            </a-button>
          </a-space>
        </a-form>
      </section>
    </div>
  </div>
</template>


<style scoped>
.login-page {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(circle at top left, rgba(64, 128, 255, 0.22), transparent 32%),
    radial-gradient(circle at bottom right, rgba(22, 93, 255, 0.14), transparent 28%),
    #f2f3f5;
}

.login-page__body {
  flex: 1;
  display: grid;
  place-items: center;
  padding: 14px;
  overflow: auto;
}

.login-page__panel {
  width: min(560px, 100%);
  padding: 16px 16px 14px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 12px 24px rgba(29, 33, 41, 0.08);
}

.login-page__panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.login-page__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.login-page__panel-header h1 {
  margin: 0 0 2px;
  font-size: 20px;
  color: #1d2129;
}

.login-page__panel-header p {
  margin: 0;
  color: #86909c;
  font-size: 12px;
}

.login-page__summary {
  padding: 3px 8px;
  border-radius: 999px;
  background: #f2f3f5;
  color: #4e5969;
  font-size: 12px;
  white-space: nowrap;
}

.login-page__badge {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 700;
  color: #fff;
  background: linear-gradient(135deg, #4080ff, #165dff);
}

.login-page__section + .login-page__section {
  margin-top: 8px;
}

.login-page__toggle {
  margin-top: -6px;
  margin-bottom: 4px;
  padding-left: 0;
}

.login-page :deep(.arco-form-item) {
  margin-bottom: 12px;
}

.login-page :deep(.arco-alert) {
  padding: 8px 12px;
}

.login-page :deep(.arco-tabs-header) {
  margin-bottom: 12px;
}

@media (max-width: 960px) {
  .login-page__body {
    padding: 12px;
  }

  .login-page__panel {
    padding: 14px 14px 12px;
  }
}
</style>
