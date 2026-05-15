<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { useSiteStore } from '../../stores/site.store';
import { useAuthStore } from '../../stores/auth.store';
import { useDictionaryStore } from '../../stores/dictionary.store';
import PageStickyTable from '../../components/PageStickyTable.vue';
import { createTablePagination } from '../../utils/table-pagination';

const siteStore = useSiteStore();
const authStore = useAuthStore();
const dictionaryStore = useDictionaryStore();

const visible = ref(false);
const loginVisible = ref(false);
const syncing = ref(false);
const submitting = ref(false);
const deletingTenantId = ref<number | null>(null);
const editingTenantId = ref<number | null>(null);
const loginTenantId = ref<number | null>(null);
const pagination = createTablePagination();
const form = reactive({
  siteId: 0,
  name: '',
  baseUrl: '',
  apiBaseUrl: '',
  tenantName: '',
  tenantSlug: '',
  lastUsername: '',
});
const loginForm = reactive({
  username: '',
  password: '',
  ticket: '',
  loginMode: 'password' as 'password' | 'ticket',
});

const tableData = computed(() => siteStore.filteredTenants);
const modalTitle = computed(() => (editingTenantId.value ? '编辑租户' : '新增租户'));
const loginTenant = computed(() => siteStore.tenants.find((item) => item.id === loginTenantId.value) ?? null);
const loggedTenantId = computed(() => authStore.token?.tenantId ?? null);

function isTenantLoggedIn(tenantId: number): boolean {
  return loggedTenantId.value === tenantId;
}

function resetForm() {
  editingTenantId.value = null;
  form.siteId = siteStore.currentSiteId ?? siteStore.sites[0]?.id ?? 0;
  form.name = '';
  form.baseUrl = '';
  form.apiBaseUrl = '';
  form.tenantName = '';
  form.tenantSlug = '';
  form.lastUsername = '';
}

function resetLoginForm() {
  loginTenantId.value = null;
  loginForm.username = '';
  loginForm.password = '';
  loginForm.ticket = '';
  loginForm.loginMode = 'password';
}

function openCreate() {
  resetForm();
  visible.value = true;
}

function openEdit(tenantId: number) {
  const tenant = siteStore.tenants.find((item) => item.id === tenantId);
  if (!tenant) {
    Message.warning('未找到要编辑的租户');
    return;
  }

  editingTenantId.value = tenant.id;
  form.siteId = tenant.siteId;
  form.name = tenant.name;
  form.baseUrl = tenant.baseUrl;
  form.apiBaseUrl = tenant.apiBaseUrl;
  form.tenantName = tenant.tenantName || '';
  form.tenantSlug = tenant.tenantSlug || '';
  form.lastUsername = tenant.lastUsername || '';
  visible.value = true;
}

function openLogin(tenantId: number) {
  const tenant = siteStore.tenants.find((item) => item.id === tenantId);
  if (!tenant) return;

  loginTenantId.value = tenant.id;
  loginForm.username = tenant.lastUsername || '';
  loginForm.password = '';
  loginVisible.value = true;
}

function handleCancel() {
  visible.value = false;
  resetForm();
}

function handleLoginCancel() {
  loginVisible.value = false;
  resetLoginForm();
}

async function handleSubmit() {
  if (!form.siteId || !form.name || !form.baseUrl || !form.apiBaseUrl) {
    Message.warning('请填写完整租户信息');
    return;
  }

  submitting.value = true;
  try {
    if (editingTenantId.value) {
      await siteStore.updateTenant({
        id: editingTenantId.value,
        siteId: form.siteId,
        name: form.name,
        baseUrl: form.baseUrl,
        apiBaseUrl: form.apiBaseUrl,
        tenantName: form.tenantName,
        tenantSlug: form.tenantSlug,
        lastUsername: form.lastUsername,
      });
      Message.success('租户信息已更新');
      handleCancel();
    } else {
      await siteStore.addTenant({
        siteId: form.siteId,
        name: form.name,
        baseUrl: form.baseUrl,
        apiBaseUrl: form.apiBaseUrl,
        tenantName: form.tenantName,
        tenantSlug: form.tenantSlug,
        lastUsername: form.lastUsername,
      });

      Message.success('租户已添加并切换为当前租户');
      handleCancel();
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : (editingTenantId.value ? '租户更新失败' : '租户添加失败'));
  } finally {
    submitting.value = false;
  }
}

async function handleLogin() {
  if (loginForm.loginMode === 'ticket') {
    return handleTicketLogin();
  }

  const tenant = loginTenant.value;
  if (!tenant || !loginTenantId.value) {
    Message.warning('未找到要登录的租户');
    return;
  }
  if (!loginForm.username || !loginForm.password) {
    Message.warning('请输入账号和密码');
    return;
  }
  if (!tenant.apiBaseUrl) {
    Message.warning('当前租户未配置桌面端 API 地址');
    return;
  }

  submitting.value = true;
  try {
    siteStore.selectSite(tenant.siteId);
    siteStore.selectTenant(tenant.id);

    await authStore.login(
      { id: tenant.id, apiBaseUrl: tenant.apiBaseUrl },
      {
        tenantId: tenant.id,
        tenantSlug: tenant.tenantSlug,
        username: loginForm.username,
        password: loginForm.password,
      },
    );

    siteStore.updateTenantLastUsername(tenant.id, loginForm.username);

    const bootstrap = await authStore.refreshBootstrap({ apiBaseUrl: tenant.apiBaseUrl });
    if (bootstrap) {
      siteStore.applyTenantBootstrap(tenant.id, bootstrap);
    }

    await dictionaryStore.initialise(
      { apiBaseUrl: tenant.apiBaseUrl },
      authStore.token?.accessToken ?? null,
      true,
    );

    Message.success(`已登录租户「${tenant.name}」`);
    handleLoginCancel();
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '登录失败');
  } finally {
    submitting.value = false;
  }
}

async function handleTicketLogin() {
  const tenant = loginTenant.value;
  if (!tenant || !loginTenantId.value) {
    Message.warning('未找到要登录的租户');
    return;
  }
  if (!loginForm.ticket.trim()) {
    Message.warning('请输入一次性登录票据');
    return;
  }
  if (!tenant.apiBaseUrl) {
    Message.warning('当前租户未配置桌面端 API 地址');
    return;
  }

  submitting.value = true;
  try {
    siteStore.selectSite(tenant.siteId);
    siteStore.selectTenant(tenant.id);

    const result = await authStore.loginWithTicket(
      { id: tenant.id, apiBaseUrl: tenant.apiBaseUrl },
      loginForm.ticket,
    );

    loginForm.username = result.user.username;
    siteStore.updateTenantLastUsername(tenant.id, result.user.username);

    const bootstrap = await authStore.refreshBootstrap({ apiBaseUrl: tenant.apiBaseUrl });
    if (bootstrap) {
      siteStore.applyTenantBootstrap(tenant.id, bootstrap);
    }

    await dictionaryStore.initialise(
      { apiBaseUrl: tenant.apiBaseUrl },
      authStore.token?.accessToken ?? null,
      true,
    );

    Message.success(`已登录租户「${tenant.name}」`);
    handleLoginCancel();
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '票据登录失败，请检查票据是否已过期或已使用');
  } finally {
    submitting.value = false;
  }
}

async function handleLogout(tenantId: number) {
  const tenant = siteStore.tenants.find((item) => item.id === tenantId);
  Modal.confirm({
    title: '退出登录？',
    content: `确认退出租户「${tenant?.name || tenantId}」的登录状态？`,
    okText: '退出',
    onOk: async () => {
      try {
        await authStore.logout(tenant ? { apiBaseUrl: tenant.apiBaseUrl } : null);
        Message.success('已退出登录');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '退出登录失败');
      }
    },
  });
}

async function handleSync() {
  if (!siteStore.currentSiteId) {
    Message.warning('请先选择站点');
    return;
  }

  syncing.value = true;
  try {
    const tenants = await siteStore.syncSiteTenants(siteStore.currentSiteId);
    Message.success(`已同步 ${tenants.length} 个租户`);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '同步租户失败');
  } finally {
    syncing.value = false;
  }
}

function handleDelete(tenantId: number) {
  const tenant = siteStore.tenants.find((item) => item.id === tenantId);
  if (!tenant) {
    Message.warning('未找到要删除的租户');
    return;
  }

  Modal.confirm({
    title: '删除租户？',
    content: `删除后会清理该租户的本地登录记录：${tenant.name}`,
    okText: '删除',
    okButtonProps: {
      status: 'danger',
    },
    cancelText: '取消',
    onOk: async () => {
      deletingTenantId.value = tenant.id;
      try {
        const result = await siteStore.deleteTenant(tenant.id);
        if (result.authInvalidated) {
          Message.warning('租户已删除，当前登录态已失效，已自动退出');
        } else {
          Message.success('租户已删除');
        }
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除租户失败');
      } finally {
        deletingTenantId.value = null;
      }
    },
  });
}

onMounted(() => {
  void siteStore.initialise().then(() => {
    form.siteId = siteStore.currentSiteId ?? siteStore.sites[0]?.id ?? 0;
  });
});
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>租户管理</h2>
        <p>在当前站点下维护租户域名上下文和登录入口。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-select
            :model-value="siteStore.currentSiteId ?? undefined"
            placeholder="选择站点"
            @change="(value) => siteStore.selectSite(Number(value))"
          >
            <a-option v-for="item in siteStore.sites" :key="item.id" :value="item.id">
              {{ item.name }}
            </a-option>
          </a-select>
        </div>
        <div class="page-toolbar__buttons">
          <a-button :loading="syncing" @click="handleSync">同步租户</a-button>
          <a-button type="primary" @click="openCreate">新增租户</a-button>
        </div>
      </div>
    </div>

    <div class="page-content">
      <a-card class="page-scroll-card">
        <PageStickyTable :data="tableData" row-key="id" :pagination="pagination">
            <template #columns>
          <a-table-column title="租户名称" data-index="name" />
          <a-table-column title="域名" data-index="baseUrl" />
          <a-table-column title="API 地址" data-index="apiBaseUrl" />
          <a-table-column title="租户" data-index="tenantName" :width="150" />
          <a-table-column title="最近账号" data-index="lastUsername" />
          <a-table-column title="登录态" :width="96">
            <template #cell="{ record }">
              <a-tag v-if="isTenantLoggedIn(record.id)" color="green">已登录</a-tag>
              <a-tag v-else color="gray">未登录</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="240" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" @click="siteStore.selectTenant(record.id)">切换</a-button>
                <a-button
                  v-if="isTenantLoggedIn(record.id)"
                  type="text"
                  size="small"
                  status="warning"
                  @click="handleLogout(record.id)"
                >
                  登出
                </a-button>
                <a-button
                  v-else
                  type="text"
                  size="small"
                  status="success"
                  @click="openLogin(record.id)"
                >
                  登录
                </a-button>
                <a-button type="text" size="small" @click="openEdit(record.id)">编辑</a-button>
                <a-button
                  type="text"
                  size="small"
                  status="danger"
                  :loading="deletingTenantId === record.id"
                  @click="handleDelete(record.id)"
                >
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
            </template>
          </PageStickyTable>
      </a-card>
    </div>

    <a-modal
      v-model:visible="visible"
      :title="modalTitle"
      :confirm-loading="submitting"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="所属站点">
          <a-select v-model="form.siteId" placeholder="请选择站点">
            <a-option v-for="item in siteStore.sites" :key="item.id" :value="item.id">
              {{ item.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="租户名称">
          <a-input v-model="form.name" placeholder="请输入租户名称" />
        </a-form-item>
        <a-form-item label="租户域名">
          <a-input v-model="form.baseUrl" placeholder="请输入租户访问地址" />
        </a-form-item>
        <a-form-item label="桌面端 API 地址">
          <a-input v-model="form.apiBaseUrl" placeholder="请输入桌面端 API 地址" />
        </a-form-item>
        <a-form-item label="租户名称">
          <a-input v-model="form.tenantName" placeholder="可选" />
        </a-form-item>
        <a-form-item label="租户标识">
          <a-input v-model="form.tenantSlug" placeholder="可选" />
        </a-form-item>
        <a-form-item label="最近登录账号">
          <a-input v-model="form.lastUsername" placeholder="可选" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="loginVisible"
      title="租户登录"
      :confirm-loading="submitting"
      @ok="handleLogin"
      @cancel="handleLoginCancel"
    >
      <template #default>
        <a-form :model="loginForm" layout="vertical">
          <a-alert type="info" style="margin-bottom: 16px">
            正在登录租户：<strong>{{ loginTenant?.name || '-' }}</strong>
          </a-alert>
          <a-tabs v-model:active-key="loginForm.loginMode" lazy-load size="small">
            <a-tab-pane key="password" title="账号密码">
              <a-form-item field="username" label="账号">
                <a-input
                  v-model="loginForm.username"
                  placeholder="请输入租户管理员账号"
                  @keyup.enter="handleLogin"
                />
              </a-form-item>
              <a-form-item field="password" label="密码">
                <a-input-password
                  v-model="loginForm.password"
                  placeholder="请输入密码"
                  @keyup.enter="handleLogin"
                />
              </a-form-item>
            </a-tab-pane>
            <a-tab-pane key="ticket" title="一次性票据">
              <a-form-item field="ticket" label="登录票据">
                <a-input
                  v-model="loginForm.ticket"
                  placeholder="请输入后台生成的一次性登录票据"
                  allow-clear
                  @keyup.enter="handleLogin"
                />
              </a-form-item>
              <a-alert type="warning">
                票据通常 5 分钟内有效，且只能使用一次。请先在平台后台的租户管理里生成桌面端登录票据。
              </a-alert>
            </a-tab-pane>
          </a-tabs>
        </a-form>
      </template>
    </a-modal>
  </div>
</template>
