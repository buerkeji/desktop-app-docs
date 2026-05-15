import type { CurrentUser, TenantTokenState } from './auth';
import type { DateTimeString, Id } from './common';

export interface SiteItem {
  id: Id;
  name: string;
  baseUrl: string;
  description?: string;
  isDefault?: boolean;
  createdAt: DateTimeString;
}

export interface TenantCapabilities {
  article: boolean;
  tool: boolean;
  dictionary: boolean;
  media: boolean;
}

export interface TenantLimits {
  maxUploadMb: number;
  maxBatchCount: number;
}

export interface TenantItem {
  id: Id;
  siteId: Id;
  name: string;
  baseUrl: string;
  apiBaseUrl: string;
  tenantName?: string;
  tenantSlug?: string;
  lastUsername?: string;
  status: 'enabled' | 'disabled';
  capabilities: TenantCapabilities;
  limits: TenantLimits;
  createdAt: DateTimeString;
}

export interface TenantDomainItem {
  id: Id;
  domain: string;
  isPrimary: boolean;
  isActive: boolean;
}

export interface TenantDiscoveryItem {
  slug: string;
  name: string;
  hasActiveDomain: boolean;
  recommendedBaseUrl: string;
  apiBaseUrl: string;
  loginHint: {
    mode: string;
    tenantSlug: string;
  };
}

export interface TenantDiscoveryPayload {
  site: {
    host: string;
    baseUrl: string;
    isSystemHost: boolean;
  };
  tenants: TenantDiscoveryItem[];
}

export interface TenantBootstrapPayload {
  site: {
    name: string;
    slug: string;
    tenantId: Id;
    baseUrl: string;
    version: string;
    domains: TenantDomainItem[];
  };
  user: CurrentUser;
  capabilities: {
    canPublishTool: boolean;
    canDeleteTool?: boolean;
    canPublishArticle: boolean;
    canDeleteArticle?: boolean;
    canUploadMedia: boolean;
    canUpdateRemote: boolean;
    supportsTags: boolean;
    supportsRequestId: boolean;
    supportsSourceUrl: boolean;
    supportsIdempotency: boolean;
    supportsRefreshToken: boolean;
    supportsSessionManagement: boolean;
    apiVersion: string;
    supportedClientTypes: string[];
    clientHeaders: {
      xClientType: boolean;
      xClientVersion: boolean;
      xRequestId: boolean;
    };
  };
  limits: {
    maxUploadSizeMb: number;
    maxTitleLength: number;
    maxSlugLength: number;
    maxBatchSize: number;
  };
  client: {
    requestId: string;
    clientType: string;
    clientVersion: string;
  };
}

export interface DesktopBootstrap {
  sites: SiteItem[];
  tenants: TenantItem[];
  token?: TenantTokenState | null;
  user?: CurrentUser | null;
  tenantAuths?: TenantAuthEntry[];
}

export interface TenantAuthEntry {
  tenantId: Id;
  token: TenantTokenState;
  user: CurrentUser;
}
