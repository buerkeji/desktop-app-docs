<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import { useSiteStore } from '../../stores/site.store';
import { useAuthStore } from '../../stores/auth.store';
import PageStickyTable from '../../components/PageStickyTable.vue';
import { createTablePagination } from '../../utils/table-pagination';

const router = useRouter();
const siteStore = useSiteStore();
const authStore = useAuthStore();
const visible = ref(false);
const submitting = ref(false);
const syncingSiteId = ref<number | null>(null);
const deletingSiteId = ref<number | null>(null);
const editingSiteId = ref<number | null>(null);
const loggingOut = ref(false);
const pagination = createTablePagination();
const form = reactive({
  name: '',
  baseUrl: '',
  description: '',
});

const modalTitle = computed(() => (editingSiteId.value ? '编辑站点' : '新增站点'));

onMounted(() => {
  void siteStore.initialise();
});

function resetForm() {
  editingSiteId.value = null;
  form.name = '';
  form.baseUrl = '';
  form.description = '';
}

function openCreate() {
  resetForm();
  visible.value = true;
}

function openEdit(siteId: number) {
  const site = siteStore.sites.find((item) => item.id === siteId);
  if (!site) {
    Message.warning('未找到要编辑的站点');
    return;
  }

  editingSiteId.value = site.id;
  form.name = site.name;
  form.baseUrl = site.baseUrl;
  form.description = site.description || '';
  visible.value = true;
}

function handleCancel() {
  visible.value = false;
  resetForm();
}

async function handleSubmit() {
  if (!form.name || !form.baseUrl) {
    Message.warning('请填写站点名称和地址');
    return;
  }

  submitting.value = true;
  try {
    if (editingSiteId.value) {
      await siteStore.updateSite({
        id: editingSiteId.value,
        name: form.name,
        baseUrl: form.baseUrl,
        description: form.description,
      });
      Message.success('站点信息已更新');
      handleCancel();
    } else {
      const result = await siteStore.addSite({
        name: form.name,
        baseUrl: form.baseUrl,
        description: form.description,
      });

      Message.success(`站点已添加，并同步 ${result.tenants.length} 个租户`);
      handleCancel();
    }
  } catch (error) {
    Message.error(error instanceof Error ? error.message : (editingSiteId.value ? '站点更新失败' : '站点添加失败'));
  } finally {
    submitting.value = false;
  }
}

async function handleSync(siteId: number) {
  syncingSiteId.value = siteId;
  try {
    const tenants = await siteStore.syncSiteTenants(siteId);
    Message.success(`已同步 ${tenants.length} 个租户`);
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '同步租户失败');
  } finally {
    syncingSiteId.value = null;
  }
}

function handleDelete(siteId: number) {
  const site = siteStore.sites.find((item) => item.id === siteId);
  if (!site) {
    Message.warning('未找到要删除的站点');
    return;
  }

  Modal.confirm({
    title: '删除站点？',
    content: `删除后会同时移除该站点下的租户与本地登录记录：${site.name}`,
    okText: '删除',
    okButtonProps: {
      status: 'danger',
    },
    cancelText: '取消',
    onOk: async () => {
      deletingSiteId.value = site.id;
      try {
        const result = await siteStore.deleteSite(site.id);
        if (result.authInvalidated) {
          Message.warning('站点已删除，当前登录租户已失效，已自动退出');
        } else {
          Message.success('站点已删除');
        }
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '删除站点失败');
      } finally {
        deletingSiteId.value = null;
      }
    },
  });
}

function handleLogout() {
  const currentTenant = siteStore.currentTenant;

  Modal.confirm({
    title: '注销当前登录？',
    content: currentTenant
      ? `将退出当前租户“${currentTenant.name}”，并返回登录页。`
      : '将清除当前本地登录态并返回登录页。',
    okText: '注销',
    cancelText: '取消',
    onOk: async () => {
      loggingOut.value = true;
      try {
        await authStore.logout(currentTenant ? { apiBaseUrl: currentTenant.apiBaseUrl } : null);
        Message.success('已注销并返回登录页');
        await router.replace('/login');
      } catch (error) {
        Message.error(error instanceof Error ? error.message : '注销失败');
      } finally {
        loggingOut.value = false;
      }
    },
  });
}
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>站点管理</h2>
        <p>维护不同部署环境下的站点入口，后续可在此增加连通性检查与能力同步。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__buttons">
          <a-button :loading="loggingOut" status="warning" @click="handleLogout">注销登录</a-button>
          <a-button type="primary" @click="openCreate">新增站点</a-button>
        </div>
      </div>
    </div>

    <div class="page-content">
      <a-card class="page-scroll-card">
        <PageStickyTable :data="siteStore.sites" row-key="id" :pagination="pagination">
            <template #columns>
          <a-table-column title="站点名称" data-index="name" />
          <a-table-column title="入口地址" data-index="baseUrl" />
          <a-table-column title="说明" data-index="description" />
          <a-table-column title="创建时间" data-index="createdAt" :width="180" />
          <a-table-column title="操作" :width="232" fixed="right">
            <template #cell="{ record }">
              <a-space size="mini" class="page-table-actions">
                <a-button type="text" size="small" @click="siteStore.selectSite(record.id)">切换</a-button>
                <a-button type="text" size="small" @click="openEdit(record.id)">编辑</a-button>
                <a-button type="text" size="small" :loading="syncingSiteId === record.id" @click="handleSync(record.id)">同步</a-button>
                <a-button
                  type="text"
                  size="small"
                  status="danger"
                  :loading="deletingSiteId === record.id"
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

    <a-modal v-model:visible="visible" :title="modalTitle" :confirm-loading="submitting" @ok="handleSubmit" @cancel="handleCancel">
      <a-form :model="form" layout="vertical">
        <a-form-item label="站点名称">
          <a-input v-model="form.name" placeholder="例如：华东生产站点" />
        </a-form-item>
        <a-form-item label="站点地址">
          <a-input v-model="form.baseUrl" placeholder="请输入站点后台地址" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model="form.description" placeholder="可填写环境、运营线或用途" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
