export type LocalDraftContentType = 'tool' | 'article';

import {
  DeleteLocalDraft,
  GetLocalDraft,
  ListLocalDrafts,
  SaveLocalDraft,
} from '../../wailsjs/go/main/App';
import { hasAppMethod } from './bridge';

export interface LocalDraftEnvelope<T> {
  version: 1;
  tenantId: number | string;
  contentType: LocalDraftContentType;
  targetId: string;
  title?: string;
  payload: T;
  updatedAt: string;
  submittedAt?: string;
}

const LOCAL_DRAFT_PREFIX = 'zq.desktop.draft';

export function buildLocalDraftKey(
  tenantId: number | string,
  contentType: LocalDraftContentType,
  targetId: string,
): string {
  return `${LOCAL_DRAFT_PREFIX}.${tenantId}.${contentType}.${targetId}`;
}

export function isVirtualLocalDraftTarget(targetId: string): boolean {
  return targetId.startsWith('local:');
}

interface LocalDraftBridgeQuery {
  tenantId: number;
  contentType: LocalDraftContentType;
  targetId: string;
}

interface LocalDraftBridgeListQuery {
  tenantId: number;
  contentType?: LocalDraftContentType;
}

interface LocalDraftBridgeSaveInput extends LocalDraftBridgeQuery {
  title?: string;
  payloadJson: string;
  submittedAt?: string;
}

interface LocalDraftBridgeItem {
  id: number;
  tenantId: number;
  contentType: LocalDraftContentType;
  targetId: string;
  title?: string;
  payloadJson: string;
  updatedAt: string;
  submittedAt?: string;
}

export interface LocalDraftListItem<T = unknown> extends LocalDraftEnvelope<T> {
  id?: number;
  key: string;
}

function canUseDraftBridge(): boolean {
  return hasAppMethod('SaveLocalDraft') && hasAppMethod('GetLocalDraft') && hasAppMethod('DeleteLocalDraft');
}

function canUseDraftListBridge(): boolean {
  return hasAppMethod('ListLocalDrafts');
}

function readLocalStorageDraft<T>(key: string): LocalDraftEnvelope<T> | null {
  const raw = localStorage.getItem(key);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as LocalDraftEnvelope<T>;
  } catch {
    localStorage.removeItem(key);
    return null;
  }
}

function isStorageQuotaError(error: unknown): boolean {
  if (!(error instanceof DOMException)) {
    return false;
  }

  return error.name === 'QuotaExceededError' || error.name === 'NS_ERROR_DOM_QUOTA_REACHED';
}

function cacheLocalStorageDraft<T>(key: string, draft: LocalDraftEnvelope<T>, throwOnQuota = false): void {
  try {
    localStorage.setItem(key, JSON.stringify(draft));
  } catch (error) {
    if (!isStorageQuotaError(error)) {
      throw error;
    }
    if (throwOnQuota) {
      throw error;
    }
  }
}

function writeLocalStorageDraft<T>(
  key: string,
  draft: Omit<LocalDraftEnvelope<T>, 'version' | 'updatedAt'>,
): LocalDraftEnvelope<T> {
  const payload: LocalDraftEnvelope<T> = {
    version: 1,
    updatedAt: new Date().toISOString(),
    ...draft,
  };
  cacheLocalStorageDraft(key, payload, true);
  return payload;
}

function removeLocalStorageDraft(key: string): void {
  localStorage.removeItem(key);
}

function listLocalStorageDrafts<T>(
  tenantId: number | string,
  contentType?: LocalDraftContentType,
): Array<LocalDraftListItem<T>> {
  const prefix = `${LOCAL_DRAFT_PREFIX}.${tenantId}.`;
  const items: Array<LocalDraftListItem<T>> = [];

  for (let index = 0; index < localStorage.length; index += 1) {
    const key = localStorage.key(index);
    if (!key || !key.startsWith(prefix)) {
      continue;
    }

    const draft = readLocalStorageDraft<T>(key);
    if (!draft) {
      continue;
    }

    if (contentType && draft.contentType !== contentType) {
      continue;
    }

    items.push({
      ...draft,
      key,
    });
  }

  return items.sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
}

function mapBridgeDraft<T>(payload: LocalDraftBridgeItem): LocalDraftEnvelope<T> | null {
  try {
    return {
      version: 1,
      tenantId: payload.tenantId,
      contentType: payload.contentType,
      targetId: payload.targetId,
      title: payload.title,
      payload: JSON.parse(payload.payloadJson) as T,
      updatedAt: payload.updatedAt,
      submittedAt: payload.submittedAt,
    };
  } catch {
    return null;
  }
}

function mapBridgeDraftListItem<T>(payload: LocalDraftBridgeItem): LocalDraftListItem<T> | null {
  const draft = mapBridgeDraft<T>(payload);
  if (!draft) {
    return null;
  }

  return {
    ...draft,
    id: payload.id,
    key: buildLocalDraftKey(payload.tenantId, payload.contentType, payload.targetId),
  };
}

export async function loadLocalDraft<T>(
  key: string,
  query?: LocalDraftBridgeQuery,
): Promise<LocalDraftEnvelope<T> | null> {
  if (query && canUseDraftBridge()) {
    const result = await GetLocalDraft(query) as LocalDraftBridgeItem | null;
    const draft = result ? mapBridgeDraft<T>(result) : null;
    if (draft) {
      cacheLocalStorageDraft(key, draft);
      return draft;
    }
  }

  return readLocalStorageDraft<T>(key);
}

export async function saveLocalDraft<T>(
  key: string,
  draft: Omit<LocalDraftEnvelope<T>, 'version' | 'updatedAt'>,
): Promise<LocalDraftEnvelope<T>> {
  if (typeof draft.tenantId === 'number' && canUseDraftBridge()) {
    try {
      const result = await SaveLocalDraft({
        tenantId: draft.tenantId,
        contentType: draft.contentType,
        targetId: draft.targetId,
        title: draft.title ?? '',
        payloadJson: JSON.stringify(draft.payload),
        submittedAt: draft.submittedAt || '',
      } satisfies LocalDraftBridgeSaveInput) as LocalDraftBridgeItem;
      const bridgeDraft = mapBridgeDraft<T>(result);
      if (bridgeDraft) {
        cacheLocalStorageDraft(key, bridgeDraft);
        return bridgeDraft;
      }
    } catch {
      // Fall through to localStorage-only persistence when bridge storage is unavailable.
    }
  }

  return writeLocalStorageDraft(key, draft);
}

export async function clearLocalDraft(key: string, query?: LocalDraftBridgeQuery): Promise<void> {
  removeLocalStorageDraft(key);

  if (query && canUseDraftBridge()) {
    await DeleteLocalDraft(query);
  }
}

export async function listLocalDrafts<T>(
  query: LocalDraftBridgeListQuery,
): Promise<Array<LocalDraftListItem<T>>> {
  if (canUseDraftListBridge()) {
    const result = await ListLocalDrafts(query) as LocalDraftBridgeItem[];
    return result
      .map((item) => mapBridgeDraftListItem<T>(item))
      .filter((item): item is LocalDraftListItem<T> => Boolean(item));
  }

  return listLocalStorageDrafts<T>(query.tenantId, query.contentType);
}
