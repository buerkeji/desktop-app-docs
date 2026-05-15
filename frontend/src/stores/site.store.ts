import { defineStore } from 'pinia';
import type { SiteItem, TenantBootstrapPayload, TenantItem } from '../types/site';
import { createSite, createTenant, deleteSite, deleteTenant, getDesktopBootstrap, syncSiteTenants as syncSiteTenantsBridge, updateSite, updateTenant } from '../services/site.service';
import { useAuthStore } from './auth.store';

interface SiteState {
  sites: SiteItem[];
  tenants: TenantItem[];
  currentSiteId: number | null;
  currentTenantId: number | null;
}

export const useSiteStore = defineStore('site', {
  state: (): SiteState => ({
    sites: [],
    tenants: [],
    currentSiteId: null,
    currentTenantId: null,
  }),
  getters: {
    currentSite: (state) => state.sites.find((item) => item.id === state.currentSiteId) ?? null,
    currentTenant: (state) => state.tenants.find((item) => item.id === state.currentTenantId) ?? null,
    filteredTenants: (state) => {
      if (!state.currentSiteId) {
        return state.tenants;
      }

      return state.tenants.filter((item) => item.siteId === state.currentSiteId);
    },
  },
  actions: {
    async initialise() {
      const bootstrap = await getDesktopBootstrap();
      this.sites = bootstrap.sites;
      this.tenants = bootstrap.tenants;
      this.currentSiteId = this.currentSiteId ?? this.sites[0]?.id ?? null;
      this.currentTenantId = this.currentTenantId ?? this.filteredTenants[0]?.id ?? this.tenants[0]?.id ?? null;
    },
    async addSite(payload: Pick<SiteItem, 'name' | 'baseUrl' | 'description'>) {
      const site = await createSite(payload);
      this.sites = [site, ...this.sites];
      this.currentSiteId = site.id;
      const tenants = await this.syncSiteTenants(site.id);
      return {
        site,
        tenants,
      };
    },
    async updateSite(payload: Pick<SiteItem, 'id' | 'name' | 'baseUrl' | 'description'>) {
      const site = await updateSite(payload);
      this.sites = this.sites.map((item) => (item.id === site.id ? site : item));
      if (this.currentSiteId === site.id) {
        this.currentSiteId = site.id;
      }
      return site;
    },
    async deleteSite(siteId: number) {
      const authStore = useAuthStore();
      const tokenTenantId = authStore.token?.tenantId ?? null;
      const authTenant = tokenTenantId
        ? (this.tenants.find((item) => item.id === tokenTenantId) ?? null)
        : null;
      const shouldLogout = authTenant?.siteId === siteId;

      await deleteSite(siteId);

      this.sites = this.sites.filter((item) => item.id !== siteId);
      this.tenants = this.tenants.filter((item) => item.siteId !== siteId);

      if (this.currentSiteId === siteId) {
        this.currentSiteId = this.sites[0]?.id ?? null;
      }

      if (!this.currentTenantId || !this.tenants.some((item) => item.id === this.currentTenantId)) {
        this.currentTenantId = this.filteredTenants[0]?.id ?? this.tenants[0]?.id ?? null;
      }

      if (shouldLogout) {
        await authStore.logout(authTenant ? { apiBaseUrl: authTenant.apiBaseUrl } : null);
      }

      return {
        authInvalidated: shouldLogout,
      };
    },
    async addTenant(payload: Pick<TenantItem, 'siteId' | 'name' | 'baseUrl' | 'apiBaseUrl' | 'tenantName' | 'tenantSlug' | 'lastUsername'>) {
      const tenant = await createTenant(payload);
      const index = this.tenants.findIndex((item) => item.id === tenant.id);
      if (index >= 0) {
        this.tenants.splice(index, 1, tenant);
      } else {
        this.tenants = [tenant, ...this.tenants];
      }
      this.currentSiteId = tenant.siteId;
      this.currentTenantId = tenant.id;
    },
    async updateTenant(payload: Pick<TenantItem, 'id' | 'siteId' | 'name' | 'baseUrl' | 'apiBaseUrl' | 'tenantName' | 'tenantSlug' | 'lastUsername'>) {
      const tenant = await updateTenant(payload);
      this.tenants = this.tenants.map((item) => (item.id === tenant.id ? tenant : item));
      if (this.currentTenantId === tenant.id) {
        this.currentSiteId = tenant.siteId;
        this.currentTenantId = tenant.id;
      }
      return tenant;
    },
    async deleteTenant(tenantId: number) {
      const tenant = this.tenants.find((item) => item.id === tenantId) ?? null;
      const authStore = useAuthStore();
      const shouldLogout = authStore.token?.tenantId === tenantId;

      await deleteTenant(tenantId);

      this.tenants = this.tenants.filter((item) => item.id !== tenantId);

      if (this.currentTenantId === tenantId) {
        const nextTenant = this.tenants.find((item) => item.siteId === tenant?.siteId)
          ?? this.tenants[0]
          ?? null;
        this.currentTenantId = nextTenant?.id ?? null;
        this.currentSiteId = nextTenant?.siteId ?? this.sites[0]?.id ?? null;
      } else if (!this.currentSiteId || !this.sites.some((item) => item.id === this.currentSiteId)) {
        this.currentSiteId = this.sites[0]?.id ?? null;
      }

      if (shouldLogout) {
        await authStore.logout(tenant ? { apiBaseUrl: tenant.apiBaseUrl } : null);
      }

      return {
        authInvalidated: shouldLogout,
      };
    },
    async syncSiteTenants(siteId: number) {
      const site = this.sites.find((item) => item.id === siteId);
      if (!site) {
        throw new Error('未找到要同步的站点');
      }

      const syncedTenants = await syncSiteTenantsBridge(site.id);

      const syncedIds = new Set(syncedTenants.map((item) => item.id));
      const retainedTenants = this.tenants.filter((item) => !syncedIds.has(item.id));
      this.tenants = [...syncedTenants, ...retainedTenants];
      this.currentSiteId = site.id;
      this.currentTenantId = this.tenants.find((item) => item.siteId === site.id)?.id ?? this.currentTenantId;

      return syncedTenants;
    },
    selectSite(id: number) {
      this.currentSiteId = id;
      this.currentTenantId = this.filteredTenants[0]?.id ?? null;
      const authStore = useAuthStore();
      authStore.switchAuth(this.currentTenantId);
    },
    async selectTenant(id: number) {
      this.currentTenantId = id;
      const authStore = useAuthStore();
      const tenant = this.tenants.find((t) => t.id === id);
      if (tenant?.apiBaseUrl) {
        await authStore.activateTenantAuth(id, tenant.apiBaseUrl);
      } else {
        authStore.switchAuth(id);
      }
    },
    async syncCurrentTenantWithAuth(tenantId: number | null | undefined) {
      if (!tenantId) {
        return;
      }

      const tenant = this.tenants.find((item) => item.id === tenantId);
      if (!tenant) {
        return;
      }

      this.currentSiteId = tenant.siteId;
      this.currentTenantId = tenant.id;
      const authStore = useAuthStore();
      if (tenant.apiBaseUrl) {
        await authStore.activateTenantAuth(tenant.id, tenant.apiBaseUrl);
      } else {
        authStore.switchAuth(tenant.id);
      }
    },
    applyTenantBootstrap(tenantId: number, payload: TenantBootstrapPayload) {
      this.tenants = this.tenants.map((item) => {
        if (item.id !== tenantId) {
          return item;
        }

        return {
          ...item,
          name: item.name || payload.site.name,
          baseUrl: payload.site.baseUrl,
          tenantName: payload.site.name,
          tenantSlug: payload.site.slug,
          capabilities: {
            article: payload.capabilities.canPublishArticle,
            tool: payload.capabilities.canPublishTool,
            dictionary: payload.capabilities.supportsTags,
            media: payload.capabilities.canUploadMedia,
          },
          limits: {
            maxUploadMb: payload.limits.maxUploadSizeMb,
            maxBatchCount: payload.limits.maxBatchSize,
          },
        };
      });
    },
    updateTenantLastUsername(tenantId: number, username: string) {
      this.tenants = this.tenants.map((item) => (item.id === tenantId ? { ...item, lastUsername: username } : item));
    },
  },
});
