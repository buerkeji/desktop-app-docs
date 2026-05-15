import type { SiteItem, TenantItem, DesktopBootstrap } from '../types/site';
import {
  CreateSite,
  CreateTenant,
  DeleteSite,
  DeleteTenant,
  GetDesktopBootstrap,
  SyncSiteTenants,
  UpdateSite,
  UpdateTenant,
} from '../../wailsjs/go/main/App';

export async function getDesktopBootstrap(): Promise<DesktopBootstrap> {
  return GetDesktopBootstrap() as unknown as Promise<DesktopBootstrap>;
}

export async function createSite(
  payload: Pick<SiteItem, 'name' | 'baseUrl' | 'description'>,
): Promise<SiteItem> {
  return CreateSite({
    name: payload.name,
    baseUrl: payload.baseUrl,
    description: payload.description ?? '',
  }) as Promise<SiteItem>;
}

export async function updateSite(
  payload: Pick<SiteItem, 'id' | 'name' | 'baseUrl' | 'description'>,
): Promise<SiteItem> {
  return UpdateSite({
    id: payload.id,
    name: payload.name,
    baseUrl: payload.baseUrl,
    description: payload.description ?? '',
  }) as Promise<SiteItem>;
}

export async function deleteSite(siteId: number): Promise<void> {
  return DeleteSite({ id: siteId });
}

export async function createTenant(
  payload: Pick<TenantItem, 'siteId' | 'name' | 'baseUrl' | 'apiBaseUrl' | 'tenantName' | 'tenantSlug' | 'lastUsername'>,
): Promise<TenantItem> {
  return CreateTenant({
    siteId: payload.siteId,
    name: payload.name,
    baseUrl: payload.baseUrl,
    apiBaseUrl: payload.apiBaseUrl,
    tenantName: payload.tenantName ?? '',
    tenantSlug: payload.tenantSlug ?? '',
    lastUsername: payload.lastUsername ?? '',
  }) as Promise<TenantItem>;
}

export async function updateTenant(
  payload: Pick<TenantItem, 'id' | 'siteId' | 'name' | 'baseUrl' | 'apiBaseUrl' | 'tenantName' | 'tenantSlug' | 'lastUsername'>,
): Promise<TenantItem> {
  return UpdateTenant({
    id: payload.id,
    siteId: payload.siteId,
    name: payload.name,
    baseUrl: payload.baseUrl,
    apiBaseUrl: payload.apiBaseUrl,
    tenantName: payload.tenantName ?? '',
    tenantSlug: payload.tenantSlug ?? '',
    lastUsername: payload.lastUsername ?? '',
  }) as Promise<TenantItem>;
}

export async function deleteTenant(tenantId: number): Promise<void> {
  return DeleteTenant({ id: tenantId });
}

export async function syncSiteTenants(siteId: number): Promise<TenantItem[]> {
  return SyncSiteTenants({ siteId }) as Promise<TenantItem[]>;
}
