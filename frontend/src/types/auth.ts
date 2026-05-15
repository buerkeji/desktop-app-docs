import type { DateTimeString, Id } from './common';

export interface CurrentUser {
  id: Id;
  name: string;
  username: string;
  roles: string[];
}

export interface LoginRequest {
  tenantId: Id;
  tenantSlug?: string;
  username: string;
  password: string;
}

export interface TenantTokenState {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresAt: DateTimeString;
  refreshExpiresAt: DateTimeString;
  sessionId: Id;
  tenantId: Id;
}

export interface AuthBootstrap {
  token: TenantTokenState;
  user: CurrentUser;
}

export interface SessionItem {
  id: Id;
  name: string;
  deviceName: string;
  clientVersion: string;
  tenantId: Id;
  lastUsedAt: DateTimeString;
  expiresAt: DateTimeString;
  refreshExpiresAt: DateTimeString;
  createdAt: DateTimeString;
  isCurrent: boolean;
}
