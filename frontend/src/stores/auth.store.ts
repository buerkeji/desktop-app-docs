import { defineStore } from 'pinia';
import type { AuthBootstrap, CurrentUser, LoginRequest, SessionItem, TenantTokenState } from '../types/auth';
import type { TenantAuthEntry, TenantBootstrapPayload, TenantItem } from '../types/site';
import {
  fetchBootstrap,
  fetchSessions,
  login,
  loginWithTicket,
  logout,
  refreshToken,
  restoreSavedAuth,
  revokeSession,
} from '../services/auth.service';

export function isTokenExpired(expiresAt: string): boolean {
  if (!expiresAt) return true;
  try {
    return new Date(expiresAt).getTime() <= Date.now();
  } catch {
    return true;
  }
}

interface AuthState {
  user: CurrentUser | null;
  token: TenantTokenState | null;
  bootstrap: TenantBootstrapPayload | null;
  sessions: SessionItem[];
  tokensByTenant: Record<number, TenantTokenState>;
  usersByTenant: Record<number, CurrentUser>;
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    token: null,
    bootstrap: null,
    sessions: [],
    tokensByTenant: {},
    usersByTenant: {},
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token),
  },
  actions: {
    applyAuthBootstrap(result: AuthBootstrap) {
      this.user = result.user;
      this.token = result.token;
      if (result.token?.tenantId && result.user) {
        this.tokensByTenant[result.token.tenantId] = result.token;
        this.usersByTenant[result.token.tenantId] = result.user;
      }
    },
    clearAuthState() {
      this.user = null;
      this.token = null;
      this.bootstrap = null;
      this.sessions = [];
    },
    restoreFromAllAuths(entries: TenantAuthEntry[]) {
      for (const entry of entries) {
        this.tokensByTenant[entry.tenantId] = entry.token;
        this.usersByTenant[entry.tenantId] = entry.user;
      }
    },
    switchAuth(tenantId: number | null | undefined) {
      if (!tenantId) {
        this.token = null;
        this.user = null;
        return;
      }
      const token = this.tokensByTenant[tenantId];
      const user = this.usersByTenant[tenantId];
      if (token) {
        this.token = token;
        this.user = user ?? null;
      }
    },
    async activateTenantAuth(tenantId: number, apiBaseUrl: string): Promise<boolean> {
      this.switchAuth(tenantId);
      if (this.token && !isTokenExpired(this.token.expiresAt)) {
        return true;
      }

      const token = this.tokensByTenant[tenantId];
      if (!token?.refreshToken) {
        this.token = null;
        this.user = null;
        return false;
      }

      try {
        const result = await refreshToken(
          { id: token.tenantId, apiBaseUrl },
          { refreshToken: token.refreshToken },
        );
        this.applyAuthBootstrap(result);
        return true;
      } catch {
        this.token = null;
        this.user = null;
        return false;
      }
    },
    isTenantTokenExpired(tenantId: number): boolean {
      const token = this.tokensByTenant[tenantId];
      if (!token) return true;
      return isTokenExpired(token.expiresAt);
    },
    canRefreshTenantToken(tenantId: number): boolean {
      const token = this.tokensByTenant[tenantId];
      if (!token?.refreshToken) return false;
      return !isTokenExpired(token.refreshExpiresAt);
    },
    removeTenantAuth(tenantId: number) {
      delete this.tokensByTenant[tenantId];
      delete this.usersByTenant[tenantId];
      if (this.token?.tenantId === tenantId) {
        this.token = null;
        this.user = null;
      }
    },
    async restore() {
      const result = await restoreSavedAuth();
      this.user = result.user ?? null;
      this.token = result.token ?? null;
      if (result.tenantAuths) {
        this.restoreFromAllAuths(result.tenantAuths);
      }
    },
    async login(tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>, payload: LoginRequest) {
      const result = await login(tenant, payload);
      this.applyAuthBootstrap(result);
      return result;
    },
    async loginWithTicket(tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>, ticket: string) {
      const result = await loginWithTicket(tenant, ticket);
      this.applyAuthBootstrap(result);
      return result;
    },
    async refreshAuthToken(tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>) {
      if (!this.token?.refreshToken) {
        throw new Error('当前没有可用的刷新令牌');
      }

      const result = await refreshToken(tenant, {
        refreshToken: this.token.refreshToken,
      });
      this.applyAuthBootstrap(result);
      return result;
    },
    async refreshBootstrap(tenant: Pick<TenantItem, 'apiBaseUrl'>) {
      if (!this.token?.accessToken) {
        this.bootstrap = null;
        return null;
      }

      const result = await fetchBootstrap(tenant, this.token.accessToken);
      this.bootstrap = result;
      this.user = result.user;
      return result;
    },
    async loadSessions(tenant: Pick<TenantItem, 'apiBaseUrl'>) {
      if (!this.token?.accessToken) {
        this.sessions = [];
        return [];
      }

      const result = await fetchSessions(tenant, this.token.accessToken);
      this.sessions = result;
      return result;
    },
    async revokeSession(tenant: Pick<TenantItem, 'apiBaseUrl'>, sessionId: number) {
      if (!this.token?.accessToken) {
        throw new Error('当前没有可用的访问令牌');
      }

      await revokeSession(tenant, this.token.accessToken, sessionId);
      this.sessions = this.sessions.filter((item) => item.id !== sessionId);
    },
    async logout(tenant?: Pick<TenantItem, 'apiBaseUrl'> | null) {
      const tenantId = this.token?.tenantId;
      try {
        await logout(tenant, this.token?.accessToken);
      } finally {
        if (tenantId) {
          this.removeTenantAuth(tenantId);
        } else {
          this.clearAuthState();
        }
      }
    },
  },
});
