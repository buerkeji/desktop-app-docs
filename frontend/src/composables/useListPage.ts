import { onMounted, ref, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useAuthStore } from '../stores/auth.store';
import { useDictionaryStore } from '../stores/dictionary.store';
import { useSiteStore } from '../stores/site.store';
import { initialiseDesktopContext } from '../utils/desktop-context';
import { createTablePagination } from '../utils/table-pagination';

export interface ListPageLoadResult<T> {
  items: T[];
  pagination: { currentPage: number; perPage: number; total: number };
}

export interface ListPageConfig<TFilters> {
  contentLabel: string;
  dictLabel: string;
  initialFilters: TFilters;
  loadPage: () => Promise<ListPageLoadResult<any>>;
}

export function useListPage<TFilters extends Record<string, any>>(config: ListPageConfig<TFilters>) {
  const authStore = useAuthStore();
  const siteStore = useSiteStore();
  const dictionaryStore = useDictionaryStore();

  const loading = ref(false);
  const batchDeleting = ref(false);
  const rows = ref<any[]>([]);
  const selectedRowKeys = ref<Array<string | number>>([]);
  const contextReady = ref(false);
  const activeLoadKey = ref('');
  const pagination = ref({ current: 1, pageSize: 20, total: 0 });
  const filters = ref<TFilters>({ ...config.initialFilters });

  const paginationConfig = createTablePagination();

  function hasTenantContext(): boolean {
    return Boolean(siteStore.currentTenant?.apiBaseUrl && authStore.token?.accessToken);
  }

  function resetState() {
    rows.value = [];
    selectedRowKeys.value = [];
    pagination.value = { current: 1, pageSize: pagination.value.pageSize, total: 0 };
  }

  function buildRequestKey(extra: Record<string, unknown> = {}): string {
    return JSON.stringify({
      tenantId: siteStore.currentTenant?.id ?? null,
      token: authStore.token?.accessToken ?? null,
      ...filters.value,
      ...extra,
    });
  }

  function skipLoad(key?: string): boolean {
    const requestKey = key ?? buildRequestKey();
    if (loading.value || activeLoadKey.value === requestKey) {
      return true;
    }
    activeLoadKey.value = requestKey;
    return false;
  }

  function applyLoadResult(result: ListPageLoadResult<any>) {
    rows.value = result.items;
    pagination.value = {
      current: result.pagination.currentPage,
      pageSize: result.pagination.perPage,
      total: result.pagination.total,
    };
    selectedRowKeys.value = [];
  }

  function handleLoadError(error: unknown) {
    const message = error instanceof Error ? error.message : `获取${config.contentLabel}列表失败`;
    if (/too many attempts/i.test(message)) {
      Message.warning(`${config.contentLabel}列表请求过于频繁，已被服务端限流，请稍后点击"刷新列表"重试`);
      return;
    }
    Message.error(message);
  }

  function invalidateAndLoad() {
    activeLoadKey.value = '';
    void loadWithGuard();
  }

  function handleSearch() {
    filters.value.page = 1;
    invalidateAndLoad();
  }

  function handleReset() {
    filters.value = { ...config.initialFilters, page: 1, perPage: (config.initialFilters as any).perPage ?? 20 } as TFilters;
    invalidateAndLoad();
  }

  function handlePageChange(page: number) {
    filters.value.page = page;
    invalidateAndLoad();
  }

  function handlePageSizeChange(pageSize: number) {
    filters.value.page = 1;
    filters.value.perPage = pageSize;
    invalidateAndLoad();
  }

  function handleSelectionChange(rowKeys: Array<string | number>) {
    selectedRowKeys.value = rowKeys;
  }

  async function loadWithGuard() {
    if (!hasTenantContext()) {
      resetState();
      return;
    }
    if (skipLoad()) return;

    loading.value = true;
    try {
      const result = await config.loadPage();
      applyLoadResult(result);
    } catch (error) {
      handleLoadError(error);
    } finally {
      loading.value = false;
    }
  }

  async function initialiseDictionary() {
    if (!hasTenantContext()) {
      dictionaryStore.$patch({ categories: [], tags: [], loaded: false, currentTenantId: null });
      return;
    }
    await dictionaryStore.initialise(
      { id: siteStore.currentTenant!.id, apiBaseUrl: siteStore.currentTenant!.apiBaseUrl },
      authStore.token!.accessToken!,
      false,
    );
  }

  function refreshDictionaryInBackground() {
    void initialiseDictionary().catch((error) => {
      Message.warning(error instanceof Error ? error.message : `加载${config.dictLabel}分类和标签失败`);
    });
  }

  onMounted(async () => {
    await initialiseDesktopContext(siteStore as any, authStore as any);
    await loadWithGuard();
    contextReady.value = true;
    refreshDictionaryInBackground();
  });

  watch(
    () => [siteStore.currentTenant?.id ?? null, authStore.token?.accessToken ?? null],
    async (current, previous) => {
      if (!contextReady.value) return;
      const [ct, ca] = current;
      const [pt, pa] = previous as Array<number | string | null>;
      if (ct === pt && ca === pa) return;
      invalidateAndLoad();
      refreshDictionaryInBackground();
    },
  );

  return {
    authStore,
    siteStore,
    dictionaryStore,
    loading,
    batchDeleting,
    rows,
    selectedRowKeys,
    contextReady,
    activeLoadKey,
    pagination,
    paginationConfig,
    filters,
    resetState,
    hasTenantContext,
    loadWithGuard,
    invalidateAndLoad,
    handleSearch,
    handleReset,
    handlePageChange,
    handlePageSizeChange,
    handleSelectionChange,
    refreshDictionaryInBackground,
    initialiseDictionary,
  };
}
