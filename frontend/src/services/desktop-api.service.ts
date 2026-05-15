import type { AuthBootstrap, CurrentUser, TenantTokenState } from '../types/auth';
import type { TenantItem } from '../types/site';
import { ClearSavedTenantToken, ProxyDesktopApiRequest, RefreshTenantToken } from '../../wailsjs/go/main/App';
import { hasAppMethod } from './bridge';

const DESKTOP_CLIENT_TYPE = 'desktop';
const DESKTOP_CLIENT_VERSION = '0.1.0';

export interface DesktopApiContext {
  id: number;
  apiBaseUrl: string;
}

export interface DesktopApiEnvelope<T> {
  success: boolean;
  message: string;
  data: T;
  errors?: Record<string, string[]> | null;
  request_id?: string;
  error_code?: string;
  status_code?: number;
}

export interface TenantPingResult {
  ok: boolean;
  message: string;
  checkedAt: string;
  requestId?: string;
}

interface DesktopAuthSnapshot {
  user: CurrentUser | null;
  token: TenantTokenState | null;
}

interface DesktopAuthRecoveryHandlers {
  getSnapshot: () => DesktopAuthSnapshot;
  applyAuthBootstrap: (payload: AuthBootstrap) => void;
  clearAuth: () => void;
  redirectToLogin?: () => void;
}

export class DesktopApiError extends Error {
  statusCode?: number;
  errorCode?: string;
  errors?: Record<string, string[]> | null;
  requestId?: string;

  constructor(message: string, payload?: Partial<DesktopApiEnvelope<unknown>>) {
    super(message);
    this.name = 'DesktopApiError';
    this.statusCode = payload?.status_code;
    this.errorCode = payload?.error_code;
    this.errors = payload?.errors ?? null;
    this.requestId = payload?.request_id;
  }
}

let authRecoveryHandlers: DesktopAuthRecoveryHandlers | null = null;
let pendingTokenRefresh: Promise<AuthBootstrap> | null = null;

export function registerDesktopAuthRecoveryHandlers(handlers: DesktopAuthRecoveryHandlers | null): void {
  authRecoveryHandlers = handlers;
}

function createRequestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID().replace(/-/g, '');
  }

  return `${Date.now()}${Math.random().toString(16).slice(2)}`;
}

export function buildDesktopApiUrl(apiBaseUrl: string, path: string): string {
  const base = apiBaseUrl.replace(/\/+$/, '');
  const suffix = path.startsWith('/') ? path : `/${path}`;
  return `${base}${suffix}`;
}

export function getDesktopClientVersion(): string {
  return DESKTOP_CLIENT_VERSION;
}

export function getDesktopDeviceName(): string {
  if (typeof navigator === 'undefined') {
    return 'desktop-client';
  }

  const navigatorWithUAData = navigator as Navigator & {
    userAgentData?: {
      platform?: string;
    };
  };
  const platform = navigatorWithUAData.userAgentData?.platform || navigator.platform || 'desktop-client';
  return `${platform}-desktop-client`.toLowerCase();
}

function createDesktopHeaders(accessToken?: string, extraHeaders?: HeadersInit): Headers {
  const headers = new Headers(extraHeaders);
  headers.set('Accept', 'application/json');
  headers.set('X-Client-Type', DESKTOP_CLIENT_TYPE);
  headers.set('X-Client-Version', DESKTOP_CLIENT_VERSION);
  headers.set('X-Request-Id', createRequestId());

  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  return headers;
}

function toHeaderRecord(headers: Headers): Record<string, string> {
  const result: Record<string, string> = {};
  headers.forEach((value, key) => {
    result[key] = value;
  });
  return result;
}

function decodeBridgeBytes(bytes: number[]): string {
  if (typeof TextDecoder !== 'undefined') {
    return new TextDecoder().decode(new Uint8Array(bytes));
  }

  return String.fromCharCode(...bytes);
}

function createDesktopApiError(message: string, payload?: Partial<DesktopApiEnvelope<unknown>>): DesktopApiError {
  return new DesktopApiError(message, payload);
}

function flattenDesktopErrors(errors?: Record<string, string[]> | null): string[] {
  if (!errors) {
    return [];
  }

  return Object.values(errors)
    .flat()
    .map((item) => item.trim())
    .filter(Boolean);
}

function isBearerTokenFailure(message: string, errors?: Record<string, string[]> | null): boolean {
  const lowerMessage = message.toLowerCase();
  if (
    lowerMessage.includes('invalid or expired bearer token')
    || lowerMessage.includes('missing bearer token')
    || lowerMessage.includes('登录已失效')
  ) {
    return true;
  }

  return flattenDesktopErrors(errors).some((item) => {
    const lowerItem = item.toLowerCase();
    return lowerItem.includes('bearer token');
  });
}

function isDesktopAuthError(error: unknown): error is DesktopApiError {
  if (!(error instanceof DesktopApiError)) {
    return false;
  }

  if (error.statusCode === 401) {
    return true;
  }

  if (error.errorCode === 'AUTH_TOKEN_INVALID' || error.errorCode === 'AUTH_TOKEN_MISSING') {
    return true;
  }

  return isBearerTokenFailure(error.message, error.errors);
}

function isRefreshTokenFailureMessage(message: string): boolean {
  const lower = message.toLowerCase();
  return (
    lower.includes('refresh token')
    || lower.includes('refresh_token')
    || lower.includes('刷新令牌')
    || lower.includes('重新登录')
  ) && (
    lower.includes('invalid')
    || lower.includes('expired')
    || lower.includes('missing')
    || lower.includes('无效')
    || lower.includes('过期')
    || lower.includes('缺少')
    || lower.includes('重新登录')
  );
}

function shouldClearAuthAfterRefreshFailure(error: unknown): boolean {
  if (error instanceof DesktopApiError) {
    return (
      error.statusCode === 401
      || error.errorCode === 'AUTH_REFRESH_TOKEN_INVALID'
      || error.errorCode === 'AUTH_REFRESH_TOKEN_MISSING'
      || isRefreshTokenFailureMessage(error.message)
    );
  }

  return error instanceof Error && isRefreshTokenFailureMessage(error.message);
}

function clearDesktopAuthAndRedirect(): void {
  authRecoveryHandlers?.clearAuth();
  authRecoveryHandlers?.redirectToLogin?.();
}

async function refreshDesktopAccessToken(
  tenant: Pick<TenantItem, 'apiBaseUrl'> & Partial<Pick<TenantItem, 'id'>>,
  failedAccessToken?: string,
): Promise<string | null> {
  if (!authRecoveryHandlers) {
    return null;
  }

  const snapshot = authRecoveryHandlers.getSnapshot();
  if (!snapshot.token?.accessToken || !snapshot.token.refreshToken) {
    clearDesktopAuthAndRedirect();
    return null;
  }

  if (failedAccessToken && snapshot.token.accessToken !== failedAccessToken) {
    return snapshot.token.accessToken;
  }

  const tenantId = tenant.id ?? snapshot.token.tenantId;
  if (!tenant.apiBaseUrl || !tenantId) {
    return null;
  }

  if (!pendingTokenRefresh) {
    pendingTokenRefresh = (async () => {
      try {
        const result = await RefreshTenantToken({
          tenantId,
          apiBaseUrl: tenant.apiBaseUrl,
          refreshToken: snapshot.token!.refreshToken,
          deviceName: getDesktopDeviceName(),
          clientVersion: getDesktopClientVersion(),
        }) as AuthBootstrap;
        authRecoveryHandlers?.applyAuthBootstrap(result);
        return result;
      } catch (error) {
        if (shouldClearAuthAfterRefreshFailure(error)) {
          try {
            await ClearSavedTenantToken();
          } catch {
            // Ignore local cleanup failures and still clear in-memory auth state.
          }
          clearDesktopAuthAndRedirect();
        }
        throw error;
      } finally {
        pendingTokenRefresh = null;
      }
    })();
  }

  const result = await pendingTokenRefresh;
  return result.token.accessToken;
}

async function performDesktopApiRequest<T>(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  path: string,
  init: RequestInit,
  accessToken?: string,
): Promise<DesktopApiEnvelope<T>> {
  const headers = createDesktopHeaders(accessToken, init.headers);

  if (hasAppMethod('ProxyDesktopApiRequest') && !(init.body instanceof FormData)) {
    const payload = await ProxyDesktopApiRequest({
      apiBaseUrl: tenant.apiBaseUrl,
      path,
      method: init.method || 'GET',
      headers: toHeaderRecord(headers),
      body: typeof init.body === 'string' ? init.body : '',
      accessToken: accessToken || '',
    }) as DesktopApiEnvelope<unknown>;

    if (!payload?.success) {
      throw createDesktopApiError(payload?.message || '请求失败', payload);
    }

    return {
      ...payload,
      data: normaliseBridgeData<T>(payload.data),
    };
  }

  const response = await fetch(buildDesktopApiUrl(tenant.apiBaseUrl, path), {
    ...init,
    headers,
  });

  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json')
    ? ((await response.json()) as DesktopApiEnvelope<T>)
    : null;

  if (!response.ok || !payload?.success) {
    throw createDesktopApiError(payload?.message || `HTTP ${response.status}`, {
      ...payload,
      status_code: response.status,
    });
  }

  return payload;
}

function normaliseBridgeData<T>(value: unknown): T {
  if (value == null) {
    return value as T;
  }

  if (Array.isArray(value) && value.every((item) => typeof item === 'number')) {
    const decoded = decodeBridgeBytes(value);
    return decoded ? (JSON.parse(decoded) as T) : (null as T);
  }

  if (typeof value === 'string') {
    const decoded = value.trim();
    if (!decoded) {
      return null as T;
    }
    if (decoded.startsWith('{') || decoded.startsWith('[') || decoded === 'null') {
      return JSON.parse(decoded) as T;
    }
  }

  return value as T;
}

export async function desktopApiRequest<T>(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  path: string,
  init: RequestInit = {},
  accessToken?: string,
): Promise<DesktopApiEnvelope<T>> {
  try {
    return await performDesktopApiRequest<T>(tenant, path, init, accessToken);
  } catch (error) {
    if (!accessToken || !isDesktopAuthError(error)) {
      throw error;
    }

    const refreshedAccessToken = await refreshDesktopAccessToken(tenant, accessToken);
    if (!refreshedAccessToken) {
      throw error;
    }

    return performDesktopApiRequest<T>(tenant, path, init, refreshedAccessToken);
  }
}

export async function pingTenantApi(tenant: Pick<TenantItem, 'apiBaseUrl'>): Promise<TenantPingResult> {
  const checkedAt = new Date().toLocaleString();

  try {
    const payload = await desktopApiRequest<{ request_id?: string }>(tenant, '/ping');
    return {
      ok: true,
      message: payload.message || '连接正常',
      checkedAt,
      requestId: payload.request_id,
    };
  } catch (error) {
    return {
      ok: false,
      message: error instanceof Error ? error.message : '请求失败',
      checkedAt,
    };
  }
}
