type DesktopContextSiteStore = {
  initialise(): Promise<void>;
  currentTenantId?: number | null;
  syncCurrentTenantWithAuth(tenantId: number | null | undefined): Promise<void>;
};

type DesktopContextAuthStore = {
  restore(): Promise<void>;
  token: {
    tenantId: number;
  } | null;
  switchAuth(tenantId: number | null | undefined): void;
};

export async function initialiseDesktopContext(
  siteStore: DesktopContextSiteStore,
  authStore: DesktopContextAuthStore,
): Promise<void> {
  const savedTenantId = siteStore.currentTenantId;

  await siteStore.initialise();
  await authStore.restore();

  // restore() loads all saved tokens from DB and sets token to the
  // most recently logged tenant. If the user had switched to a different
  // tenant in the current session, re-apply that tenant's auth.
  if (savedTenantId) {
    authStore.switchAuth(savedTenantId);
  }

  await siteStore.syncCurrentTenantWithAuth(authStore.token?.tenantId);
}
