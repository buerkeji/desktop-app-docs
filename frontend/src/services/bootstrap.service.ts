import type { Router } from 'vue-router';
import { registerDesktopAuthRecoveryHandlers } from '../services/desktop-api.service';
import { useAuthStore } from '../stores/auth.store';
import { useDictionaryStore } from '../stores/dictionary.store';
import { useSiteStore } from '../stores/site.store';
import { initialiseDesktopContext } from '../utils/desktop-context';

export function setupAuthRecoveryHandlers(
  authStore: ReturnType<typeof useAuthStore>,
  router: Router,
) {
  registerDesktopAuthRecoveryHandlers({
    getSnapshot: () => ({
      user: authStore.user,
      token: authStore.token,
    }),
    applyAuthBootstrap: (payload) => {
      authStore.applyAuthBootstrap(payload);
    },
    clearAuth: () => {
      authStore.clearAuthState();
    },
    redirectToLogin: () => {
      void router.replace('/login');
    },
  });
}

export async function initialiseApp(
  siteStore: ReturnType<typeof useSiteStore>,
  authStore: ReturnType<typeof useAuthStore>,
  dictionaryStore: ReturnType<typeof useDictionaryStore>,
) {
  await initialiseDesktopContext(siteStore, authStore);

  if (!authStore.token?.accessToken || !siteStore.currentTenant?.apiBaseUrl) {
    return;
  }

  const tenant = siteStore.currentTenant;

  try {
    await refreshCurrentTenantState(authStore, siteStore, dictionaryStore, tenant);
  } catch {
    await tryTokenRefresh(authStore, siteStore, dictionaryStore, tenant);
  }
}

async function refreshCurrentTenantState(
  authStore: ReturnType<typeof useAuthStore>,
  siteStore: ReturnType<typeof useSiteStore>,
  dictionaryStore: ReturnType<typeof useDictionaryStore>,
  tenant: NonNullable<ReturnType<typeof useSiteStore>['currentTenant']>,
) {
  const bootstrap = await authStore.refreshBootstrap({ apiBaseUrl: tenant.apiBaseUrl });
  if (bootstrap) {
    siteStore.applyTenantBootstrap(tenant.id, bootstrap);
  }

  await dictionaryStore.initialise(
    { id: tenant.id, apiBaseUrl: tenant.apiBaseUrl },
    authStore.token!.accessToken,
    true,
  );
}

async function tryTokenRefresh(
  authStore: ReturnType<typeof useAuthStore>,
  siteStore: ReturnType<typeof useSiteStore>,
  dictionaryStore: ReturnType<typeof useDictionaryStore>,
  tenant: NonNullable<ReturnType<typeof useSiteStore>['currentTenant']>,
) {
  if (!authStore.token?.refreshToken || !tenant.id) {
    return;
  }

  try {
    await authStore.refreshAuthToken({
      id: tenant.id,
      apiBaseUrl: tenant.apiBaseUrl,
    });

    const bootstrap = await authStore.refreshBootstrap({ apiBaseUrl: tenant.apiBaseUrl });
    if (bootstrap) {
      siteStore.applyTenantBootstrap(tenant.id, bootstrap);
    }

    await dictionaryStore.initialise(
      { id: tenant.id, apiBaseUrl: tenant.apiBaseUrl },
      authStore.token?.accessToken ?? null,
      true,
    );
  } catch {
    // Keep local cached state if both bootstrap and refresh are temporarily unavailable.
  }
}
