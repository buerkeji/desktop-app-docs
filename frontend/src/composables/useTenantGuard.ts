import { computed } from 'vue';
import { useAuthStore } from '../stores/auth.store';
import { useSiteStore } from '../stores/site.store';

export function useTenantGuard() {
  const authStore = useAuthStore();
  const siteStore = useSiteStore();

  const tenantReady = computed(() => {
    return Boolean(
      siteStore.currentTenant?.apiBaseUrl && authStore.token?.accessToken,
    );
  });

  const currentTenant = computed(() => siteStore.currentTenant);
  const accessToken = computed(() => authStore.token?.accessToken ?? null);

  function guardMessage(action: string): string {
    if (!currentTenant.value) {
      return `请先选择租户后再${action}`;
    }
    if (!accessToken.value) {
      return `请先登录后再${action}`;
    }
    return '';
  }

  return {
    tenantReady,
    currentTenant,
    accessToken,
    guardMessage,
  };
}
