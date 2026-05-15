import { type ComputedRef, computed, onMounted, ref, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '../stores/auth.store';
import { useSiteStore } from '../stores/site.store';
import { initialiseDesktopContext } from '../utils/desktop-context';

export interface DetailPageConfig<TDetail> {
  contentLabel: string;
  load: (apiBaseUrl: string, accessToken: string, id: string) => Promise<TDetail>;
  delete?: (apiBaseUrl: string, accessToken: string, id: string) => Promise<void>;
  deleteConfirmTitle?: string;
  onDeleted?: (id: string) => void;
  emptyMessage?: string;
}

export function useDetailPage<TDetail extends { title?: string }>(config: DetailPageConfig<TDetail>) {
  const route = useRoute();
  const authStore = useAuthStore();
  const siteStore = useSiteStore();

  const loading = ref(false);
  const deleting = ref(false);
  const detail = ref<TDetail | null>(null);
  const entityId = computed(() => String(route.params.id || ''));

  function hasTenantContext(): boolean {
    return Boolean(siteStore.currentTenant?.apiBaseUrl && authStore.token?.accessToken);
  }

  function guardMessage(): string {
    if (!entityId.value) return `未找到${config.contentLabel}标识`;
    if (!hasTenantContext()) return '当前缺少租户上下文或登录态';
    return '';
  }

  async function loadDetail(): Promise<void> {
    const guard = guardMessage();
    if (guard) {
      detail.value = null;
      Message.warning(guard);
      return;
    }

    loading.value = true;
    try {
      detail.value = await config.load(
        siteStore.currentTenant!.apiBaseUrl,
        authStore.token!.accessToken!,
        entityId.value,
      );
    } catch (error) {
      const msg = error instanceof Error ? error.message : `获取${config.contentLabel}详情失败`;
      Message.error(msg);
    } finally {
      loading.value = false;
    }
  }

  async function handleDelete(): Promise<void> {
    if (!config.delete) return;

    const guard = guardMessage();
    if (guard) {
      Message.warning(guard);
      return;
    }

    const title = detail.value?.title || entityId.value;
    if (!window.confirm(`确认删除${config.contentLabel}「${title}」吗？此操作不可撤销。`)) {
      return;
    }

    deleting.value = true;
    try {
      await config.delete(
        siteStore.currentTenant!.apiBaseUrl,
        authStore.token!.accessToken!,
        entityId.value,
      );
      Message.success(`${config.contentLabel}已删除`);
      config.onDeleted?.(entityId.value);
    } catch (error) {
      const msg = error instanceof Error ? error.message : `删除${config.contentLabel}失败`;
      Message.error(msg);
    } finally {
      deleting.value = false;
    }
  }

  onMounted(async () => {
    await initialiseDesktopContext(siteStore as any, authStore as any);
    await loadDetail();
  });

  watch(
    () => [
      siteStore.currentTenant?.id ?? null,
      authStore.token?.accessToken ?? null,
      entityId.value,
    ],
    () => {
      void loadDetail();
    },
  );

  return {
    authStore,
    siteStore,
    loading,
    deleting,
    detail,
    entityId,
    loadDetail,
    handleDelete,
  };
}
