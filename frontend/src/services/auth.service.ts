import type { AuthBootstrap, LoginRequest, SessionItem, TenantTokenState } from '../types/auth';
import type { DesktopBootstrap, TenantBootstrapPayload, TenantItem } from '../types/site';
import {
  GetDesktopBootstrap,
  LoginTenant,
  LogoutTenant,
  RefreshTenantToken,
  SaveTenantAuth,
} from '../../wailsjs/go/main/App';
import {
  desktopApiRequest,
  getDesktopClientVersion,
  getDesktopDeviceName,
} from './desktop-api.service';

interface LoginTenantInput {
  tenantId: number;
  apiBaseUrl: string;
  username: string;
  password: string;
  tenantSlug: string;
  deviceName: string;
  clientVersion: string;
}

interface ExchangeTicketInput {
  ticket: string;
  device_name: string;
  client_version: string;
}

interface RefreshTenantTokenInput {
  tenantId: number;
  apiBaseUrl: string;
  refreshToken: string;
  deviceName: string;
  clientVersion: string;
}

interface LogoutTenantInput {
  apiBaseUrl: string;
  accessToken: string;
}

interface RemoteBootstrapResponse {
  site: {
    name: string;
    slug: string;
    tenant_id: number;
    base_url: string;
    version: string;
    domains: Array<{
      id: number;
      domain: string;
      is_primary: boolean;
      is_active: boolean;
    }>;
  };
  user: {
    id: number;
    username: string;
    name: string;
    tenant_id: number;
    roles?: string[];
  };
  capabilities: {
    can_publish_tool: boolean;
    can_delete_tool?: boolean;
    can_publish_article: boolean;
    can_delete_article?: boolean;
    can_upload_media: boolean;
    can_update_remote: boolean;
    supports_tags: boolean;
    supports_request_id: boolean;
    supports_source_url: boolean;
    supports_idempotency: boolean;
    supports_refresh_token: boolean;
    supports_session_management: boolean;
    api_version: string;
    supported_client_types: string[];
    client_headers: {
      x_client_type: boolean;
      x_client_version: boolean;
      x_request_id: boolean;
    };
  };
  limits: {
    max_upload_size_mb: number;
    max_title_length: number;
    max_slug_length: number;
    max_batch_size: number;
  };
  client: {
    request_id: string;
    client_type: string;
    client_version: string;
  };
}

interface RemoteLoginResponse {
  access_token: string;
  refresh_token: string;
  session_id: number;
  token_type: string;
  expires_at: string;
  refresh_expires_at: string;
  user: {
    id: number;
    username: string;
    name: string;
    tenant_id: number;
    roles?: string[];
  };
}

interface RemoteSessionItem {
  id: number;
  name: string;
  device_name: string;
  client_version: string;
  tenant_id: number;
  is_current: boolean;
  last_used_at: string;
  expires_at: string;
  refresh_expires_at: string;
  created_at: string;
}

function mapBootstrapResponse(payload: RemoteBootstrapResponse): TenantBootstrapPayload {
  return {
    site: {
      name: payload.site.name,
      slug: payload.site.slug,
      tenantId: payload.site.tenant_id,
      baseUrl: payload.site.base_url,
      version: payload.site.version,
      domains: (payload.site.domains ?? []).map((item) => ({
        id: item.id,
        domain: item.domain,
        isPrimary: item.is_primary,
        isActive: item.is_active,
      })),
    },
    user: {
      id: payload.user.id,
      username: payload.user.username,
      name: payload.user.name,
      roles: payload.user.roles?.length ? payload.user.roles : ['desktop'],
    },
    capabilities: {
      canPublishTool: payload.capabilities.can_publish_tool,
      canDeleteTool: payload.capabilities.can_delete_tool ?? payload.capabilities.can_publish_tool,
      canPublishArticle: payload.capabilities.can_publish_article,
      canDeleteArticle: payload.capabilities.can_delete_article ?? payload.capabilities.can_publish_article,
      canUploadMedia: payload.capabilities.can_upload_media,
      canUpdateRemote: payload.capabilities.can_update_remote,
      supportsTags: payload.capabilities.supports_tags,
      supportsRequestId: payload.capabilities.supports_request_id,
      supportsSourceUrl: payload.capabilities.supports_source_url,
      supportsIdempotency: payload.capabilities.supports_idempotency,
      supportsRefreshToken: payload.capabilities.supports_refresh_token,
      supportsSessionManagement: payload.capabilities.supports_session_management,
      apiVersion: payload.capabilities.api_version,
      supportedClientTypes: payload.capabilities.supported_client_types,
      clientHeaders: {
        xClientType: payload.capabilities.client_headers.x_client_type,
        xClientVersion: payload.capabilities.client_headers.x_client_version,
        xRequestId: payload.capabilities.client_headers.x_request_id,
      },
    },
    limits: {
      maxUploadSizeMb: payload.limits.max_upload_size_mb,
      maxTitleLength: payload.limits.max_title_length,
      maxSlugLength: payload.limits.max_slug_length,
      maxBatchSize: payload.limits.max_batch_size,
    },
    client: {
      requestId: payload.client.request_id,
      clientType: payload.client.client_type,
      clientVersion: payload.client.client_version,
    },
  };
}

function mapSessionItem(payload: RemoteSessionItem): SessionItem {
  return {
    id: payload.id,
    name: payload.name,
    deviceName: payload.device_name,
    clientVersion: payload.client_version,
    tenantId: payload.tenant_id,
    isCurrent: payload.is_current,
    lastUsedAt: payload.last_used_at,
    expiresAt: payload.expires_at,
    refreshExpiresAt: payload.refresh_expires_at,
    createdAt: payload.created_at,
  };
}

export async function login(tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>, payload: LoginRequest): Promise<AuthBootstrap> {
  const result = await LoginTenant({
    tenantId: tenant.id,
    apiBaseUrl: tenant.apiBaseUrl,
    username: payload.username,
    password: payload.password,
    tenantSlug: payload.tenantSlug || '',
    deviceName: getDesktopDeviceName(),
    clientVersion: getDesktopClientVersion(),
  } satisfies LoginTenantInput) as AuthBootstrap;
  return result;
}

export async function loginWithTicket(
  tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>,
  ticket: string,
): Promise<AuthBootstrap> {
  const response = await desktopApiRequest<RemoteLoginResponse>(
    tenant,
    '/auth/exchange-ticket',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ticket: ticket.trim(),
        device_name: getDesktopDeviceName(),
        client_version: getDesktopClientVersion(),
      } satisfies ExchangeTicketInput),
    },
  );

  const result = await SaveTenantAuth({
    tenantId: tenant.id,
    accessToken: response.data.access_token,
    refreshToken: response.data.refresh_token,
    tokenType: response.data.token_type,
    expiresAt: response.data.expires_at,
    refreshExpiresAt: response.data.refresh_expires_at,
    sessionId: response.data.session_id,
    userId: response.data.user.id,
    username: response.data.user.username,
    name: response.data.user.name,
    roles: response.data.user.roles?.length ? response.data.user.roles : ['desktop'],
  }) as AuthBootstrap;
  return result;
}

export async function refreshToken(
  tenant: Pick<TenantItem, 'id' | 'apiBaseUrl'>,
  token: Pick<TenantTokenState, 'refreshToken'>,
): Promise<AuthBootstrap> {
  const result = await RefreshTenantToken({
    tenantId: tenant.id,
    apiBaseUrl: tenant.apiBaseUrl,
    refreshToken: token.refreshToken,
    deviceName: getDesktopDeviceName(),
    clientVersion: getDesktopClientVersion(),
  } satisfies RefreshTenantTokenInput) as AuthBootstrap;
  return result;
}

export async function fetchBootstrap(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
): Promise<TenantBootstrapPayload> {
  const response = await desktopApiRequest<RemoteBootstrapResponse>(tenant, '/bootstrap', {}, accessToken);
  return mapBootstrapResponse(response.data);
}

export async function fetchSessions(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
): Promise<SessionItem[]> {
  const response = await desktopApiRequest<RemoteSessionItem[]>(tenant, '/auth/sessions', {}, accessToken);
  return (response.data ?? []).map(mapSessionItem);
}

export async function revokeSession(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  sessionId: number,
): Promise<void> {
  await desktopApiRequest<{ revoked: boolean; session_id: number }>(
    tenant,
    `/auth/sessions/${sessionId}`,
    {
      method: 'DELETE',
    },
    accessToken,
  );
}

export async function logout(
  tenant?: Pick<TenantItem, 'apiBaseUrl'> | null,
  accessToken?: string | null,
): Promise<void> {
  await LogoutTenant({
    apiBaseUrl: tenant?.apiBaseUrl || '',
    accessToken: accessToken || '',
  } satisfies LogoutTenantInput);
}

export async function restoreSavedAuth(): Promise<Pick<DesktopBootstrap, 'token' | 'user' | 'tenantAuths'>> {
  const result = await GetDesktopBootstrap() as unknown as DesktopBootstrap;
  return {
    token: result.token ?? null,
    user: result.user ?? null,
    tenantAuths: result.tenantAuths ?? [],
  };
}
