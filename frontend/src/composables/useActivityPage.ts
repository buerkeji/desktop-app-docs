import { computed, onMounted, ref, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useAuthStore } from '../stores/auth.store';
import { useSiteStore } from '../stores/site.store';
import { initialiseDesktopContext } from '../utils/desktop-context';
import { createTablePagination } from '../utils/table-pagination';

export interface ActivityPageOptions<T> {
  label: string;
  loadData: () => Promise<T[]>;
  resetFilters: () => void;
}

export function useActivityPage<T extends { id: number | string }>(options: ActivityPageOptions<T>) {
  const authStore = useAuthStore();
  const siteStore = useSiteStore();

  const loading = ref(false);
  const detailVisible = ref(false);
  const currentRecord = ref<T | null>(null);
  const rows = ref<T[]>([]);
  const pagination = createTablePagination();

  const tenantOptions = computed(() => siteStore.tenants);
  const currentTenantName = computed(() => siteStore.currentTenant?.name || '全部租户');

  async function load() {
    loading.value = true;
    try {
      rows.value = await options.loadData();
    } catch (error) {
      const message = error instanceof Error ? error.message : `获取${options.label}失败`;
      Message.error(message);
    } finally {
      loading.value = false;
    }
  }

  function handleSearch() {
    void load();
  }

  function handleReset() {
    options.resetFilters();
    void load();
  }

  function openDetail(record: T) {
    currentRecord.value = record;
    detailVisible.value = true;
  }

  onMounted(async () => {
    await initialiseDesktopContext(siteStore as any, authStore as any);
    void load();
  });

  watch(
    () => siteStore.currentTenant?.id ?? null,
    () => {
      options.resetFilters();
      void load();
    },
  );

  return {
    authStore,
    siteStore,
    loading,
    detailVisible,
    currentRecord,
    rows,
    pagination,
    tenantOptions,
    currentTenantName,
    load,
    handleSearch,
    handleReset,
    openDetail,
  };
}
