import type { TenantItem } from '../types/site';
import type {
  ArticleDetail,
  BatchDeleteItemInput,
  BatchDeleteResponse,
  ArticleListFilters,
  ArticleListItem,
  ArticleListResponse,
  ArticleUpsertPayload,
  ContentTagInput,
  ContentCategory,
  ContentTag,
  PaginationState,
  ToolDetail,
  ToolFeatureItem,
  ToolListFilters,
  ToolListItem,
  ToolListResponse,
  ToolUpsertPayload,
} from '../types/content';
import { desktopApiRequest } from './desktop-api.service';
import { normaliseTenantHtmlAssets, resolveTenantAssetUrl } from '../utils/url';

interface RemotePaginationState {
  current_page: number;
  per_page: number;
  total: number;
  last_page: number;
  has_more_pages: boolean;
}

interface RemoteToolItem {
  id: number;
  title: string;
  slug: string;
  url?: string | null;
  icon?: string | null;
  thumbnail?: string | null;
  category_id?: number | null;
  description?: string | null;
  content?: string | null;
  features?: Array<{ feature: string }>;
  website?: string | null;
  is_featured: boolean;
  is_active: boolean;
  sort_order: number;
  meta_title?: string | null;
  meta_keywords?: string | null;
  meta_description?: string | null;
  category?: ContentCategory | null;
  tags?: ContentTag[];
  created_at: string;
  updated_at: string;
}

interface RemoteArticleItem {
  id: number;
  title: string;
  slug: string;
  thumbnail?: string | null;
  excerpt?: string | null;
  content?: string | null;
  category_id?: number | null;
  author_id?: number | null;
  meta_title?: string | null;
  meta_keywords?: string | null;
  meta_description?: string | null;
  is_featured: boolean;
  is_pinned: boolean;
  is_published: boolean;
  published_at?: string | null;
  category?: ContentCategory | null;
  tags?: ContentTag[];
  created_at: string;
  updated_at: string;
}

interface RemoteListResponse<T> {
  items: T[] | null;
  pagination: RemotePaginationState;
}

interface RemoteMetaPaginationState {
  current_page?: number;
  per_page?: number;
  total?: number;
  last_page?: number;
  has_more_pages?: boolean;
}

interface RemoteBatchDeleteResponse {
  summary: {
    total: number;
    deleted: number;
    failed: number;
  };
  items: Array<{
    index: number;
    client_id?: string | null;
    success: boolean;
    action: string;
    remote_id?: number | string | null;
    errors?: Record<string, string[]> | null;
    error_code?: string | null;
  }>;
}

function appendBoolean(params: URLSearchParams, key: string, value?: boolean): void {
  if (typeof value === 'boolean') {
    params.set(key, value ? '1' : '0');
  }
}

function appendNumber(params: URLSearchParams, key: string, value?: number): void {
  if (typeof value === 'number' && value > 0) {
    params.set(key, String(value));
  }
}

function mapPagination(payload: RemotePaginationState): PaginationState {
  return {
    currentPage: payload.current_page,
    perPage: payload.per_page,
    total: payload.total,
    lastPage: payload.last_page,
    hasMorePages: payload.has_more_pages,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function toRemotePaginationState(payload: unknown, fallbackCount = 0): RemotePaginationState {
  const source = isRecord(payload) ? payload as Record<string, unknown> : {};
  const currentPage = typeof source.current_page === 'number' ? source.current_page : 1;
  const perPage = typeof source.per_page === 'number' ? source.per_page : fallbackCount;
  const total = typeof source.total === 'number' ? source.total : fallbackCount;
  const lastPage = typeof source.last_page === 'number' ? source.last_page : (perPage > 0 ? Math.max(1, Math.ceil(total / perPage)) : 1);
  const hasMorePages = typeof source.has_more_pages === 'boolean' ? source.has_more_pages : currentPage < lastPage;

  return {
    current_page: currentPage,
    per_page: perPage || fallbackCount || 1,
    total,
    last_page: lastPage,
    has_more_pages: hasMorePages,
  };
}

function normaliseListResponse<T>(payload: unknown): RemoteListResponse<T> {
  if (Array.isArray(payload)) {
    return {
      items: payload as T[],
      pagination: toRemotePaginationState({}, payload.length),
    };
  }

  if (!isRecord(payload)) {
    return {
      items: [],
      pagination: toRemotePaginationState({}, 0),
    };
  }

  if (Array.isArray(payload.items)) {
    const items = payload.items as T[];
    const paginationSource = isRecord(payload.pagination) ? payload.pagination : payload;
    return {
      items,
      pagination: toRemotePaginationState(paginationSource, items.length),
    };
  }

  if (payload.items == null) {
    const paginationSource = isRecord(payload.pagination) ? payload.pagination : payload;
    return {
      items: [],
      pagination: toRemotePaginationState(paginationSource, 0),
    };
  }

  if (Array.isArray(payload.data)) {
    const items = payload.data as T[];
    const meta = isRecord(payload.meta) ? payload.meta : payload;
    return {
      items,
      pagination: toRemotePaginationState(meta, items.length),
    };
  }

  return {
    items: [],
    pagination: toRemotePaginationState(payload, 0),
  };
}

function mapToolItem(payload: RemoteToolItem, apiBaseUrl?: string | null): ToolListItem {
  return {
    id: payload.id,
    title: payload.title,
    slug: payload.slug,
    url: payload.url,
    icon: resolveTenantAssetUrl(payload.icon || '', apiBaseUrl) || null,
    thumbnail: resolveTenantAssetUrl(payload.thumbnail || '', apiBaseUrl) || null,
    categoryId: payload.category_id,
    description: payload.description,
    content: payload.content ? normaliseTenantHtmlAssets(payload.content, apiBaseUrl) : payload.content,
    features: payload.features ?? [],
    website: payload.website,
    isFeatured: payload.is_featured,
    isActive: payload.is_active,
    sortOrder: payload.sort_order,
    metaTitle: payload.meta_title,
    metaKeywords: payload.meta_keywords,
    metaDescription: payload.meta_description,
    category: payload.category ?? null,
    tags: payload.tags ?? [],
    createdAt: payload.created_at,
    updatedAt: payload.updated_at,
  };
}

function mapArticleItem(payload: RemoteArticleItem, apiBaseUrl?: string | null): ArticleListItem {
  return {
    id: payload.id,
    title: payload.title,
    slug: payload.slug,
    thumbnail: resolveTenantAssetUrl(payload.thumbnail || '', apiBaseUrl) || null,
    excerpt: payload.excerpt,
    content: payload.content ? normaliseTenantHtmlAssets(payload.content, apiBaseUrl) : payload.content,
    categoryId: payload.category_id,
    authorId: payload.author_id,
    metaTitle: payload.meta_title,
    metaKeywords: payload.meta_keywords,
    metaDescription: payload.meta_description,
    isFeatured: payload.is_featured,
    isPinned: payload.is_pinned,
    isPublished: payload.is_published,
    publishedAt: payload.published_at,
    category: payload.category ?? null,
    tags: payload.tags ?? [],
    createdAt: payload.created_at,
    updatedAt: payload.updated_at,
  };
}

function mapTagInput(tag: ContentTagInput): number | string | { name: string; slug?: string } {
  if (typeof tag === 'number' || typeof tag === 'string') {
    return tag;
  }

  return {
    name: tag.name,
    ...(tag.slug ? { slug: tag.slug } : {}),
  };
}

function mapFeatureInput(feature: string | ToolFeatureItem): string | { feature: string } {
  if (typeof feature === 'string') {
    return feature;
  }

  return {
    feature: feature.feature,
  };
}

function createIdempotencyKey(prefix: string): string {
  const suffix =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID().replace(/-/g, '')
      : `${Date.now()}${Math.random().toString(16).slice(2)}`;
  return `${prefix}-${suffix}`;
}

function mapBatchDeleteItems(items: BatchDeleteItemInput[]): Array<Record<string, unknown>> {
  return items.map((item, index) => ({
    ...(item.clientId ? { client_id: item.clientId } : { client_id: `desktop-delete-${index + 1}` }),
    ...(item.remoteId !== undefined ? { remote_id: item.remoteId } : {}),
    ...(item.slug ? { slug: item.slug } : {}),
  }));
}

function mapBatchDeleteResponse(
  payload: RemoteBatchDeleteResponse,
  idempotencyKey: string,
): BatchDeleteResponse {
  return {
    summary: {
      total: payload.summary.total,
      deleted: payload.summary.deleted,
      failed: payload.summary.failed,
    },
    items: (payload.items ?? []).map((item) => ({
      index: item.index,
      clientId: item.client_id,
      success: item.success,
      action: item.action,
      remoteId: item.remote_id,
      errors: item.errors ?? null,
      errorCode: item.error_code ?? null,
    })),
    idempotencyKey,
  };
}

function mapToolPayload(payload: ToolUpsertPayload): Record<string, unknown> {
  return {
    ...(payload.title !== undefined ? { title: payload.title } : {}),
    ...(payload.slug !== undefined ? { slug: payload.slug } : {}),
    ...(payload.url !== undefined ? { url: payload.url } : {}),
    ...(payload.icon !== undefined ? { icon: payload.icon } : {}),
    ...(payload.thumbnail !== undefined ? { thumbnail: payload.thumbnail } : {}),
    ...(payload.categoryId !== undefined ? { category_id: payload.categoryId } : {}),
    ...(payload.description !== undefined ? { description: payload.description } : {}),
    ...(payload.content !== undefined ? { content: payload.content } : {}),
    ...(payload.features !== undefined ? { features: payload.features.map(mapFeatureInput) } : {}),
    ...(payload.website !== undefined ? { website: payload.website } : {}),
    ...(payload.isFeatured !== undefined ? { is_featured: payload.isFeatured } : {}),
    ...(payload.isActive !== undefined ? { is_active: payload.isActive } : {}),
    ...(payload.sortOrder !== undefined ? { sort_order: payload.sortOrder } : {}),
    ...(payload.metaTitle !== undefined ? { meta_title: payload.metaTitle } : {}),
    ...(payload.metaKeywords !== undefined ? { meta_keywords: payload.metaKeywords } : {}),
    ...(payload.metaDescription !== undefined ? { meta_description: payload.metaDescription } : {}),
    ...(payload.tags !== undefined ? { tags: payload.tags.map(mapTagInput) } : {}),
  };
}

function mapArticlePayload(payload: ArticleUpsertPayload): Record<string, unknown> {
  return {
    ...(payload.title !== undefined ? { title: payload.title } : {}),
    ...(payload.slug !== undefined ? { slug: payload.slug } : {}),
    ...(payload.categoryId !== undefined ? { category_id: payload.categoryId } : {}),
    ...(payload.thumbnail !== undefined ? { thumbnail: payload.thumbnail } : {}),
    ...(payload.excerpt !== undefined ? { excerpt: payload.excerpt } : {}),
    ...(payload.content !== undefined ? { content: payload.content } : {}),
    ...(payload.authorId !== undefined ? { author_id: payload.authorId } : {}),
    ...(payload.metaTitle !== undefined ? { meta_title: payload.metaTitle } : {}),
    ...(payload.metaKeywords !== undefined ? { meta_keywords: payload.metaKeywords } : {}),
    ...(payload.metaDescription !== undefined ? { meta_description: payload.metaDescription } : {}),
    ...(payload.isFeatured !== undefined ? { is_featured: payload.isFeatured } : {}),
    ...(payload.isPinned !== undefined ? { is_pinned: payload.isPinned } : {}),
    ...(payload.isPublished !== undefined ? { is_published: payload.isPublished } : {}),
    ...normaliseOptionalDate('published_at', payload.publishedAt),
    ...(payload.tags !== undefined ? { tags: payload.tags.map(mapTagInput) } : {}),
  };
}

/** Normalise a date string to YYYY-MM-DD HH:mm:ss, omitting if unparseable. */
function normaliseOptionalDate(key: string, value: string | undefined): Record<string, string> {
  if (value === undefined) return {};
  const trimmed = value.trim();
  if (!trimmed) return {};

  // Null-date sentinel → omit so backend preserves existing value.
  if (/^0{4}[-\/]0{2}[-\/]0{2}/.test(trimmed)) {
    return {};
  }

  // Already in expected format.
  if (/^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}$/.test(trimmed)) {
    return { [key]: trimmed };
  }

  // Parse via Date (handles ISO 8601, etc.).
  const d = new Date(trimmed);
  if (!isNaN(d.getTime())) {
    const pad = (n: number) => String(n).padStart(2, '0');
    return { [key]: `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}` };
  }

  // Unrecognised → omit to avoid validation errors.
  return {};
}

export async function getToolList(
  tenant: Pick<TenantItem, 'apiBaseUrl'> & Partial<Pick<TenantItem, 'id'>>,
  accessToken: string,
  filters: ToolListFilters,
): Promise<ToolListResponse> {
  const params = new URLSearchParams({
    page: String(filters.page),
    per_page: String(filters.perPage),
  });
  if (filters.keyword?.trim()) {
    params.set('keyword', filters.keyword.trim());
  }
  appendNumber(params, 'category_id', filters.categoryId);
  appendNumber(params, 'tag_id', filters.tagId);
  appendBoolean(params, 'is_active', filters.isActive);
  appendBoolean(params, 'is_featured', filters.isFeatured);
  if (filters.sort) {
    params.set('sort', filters.sort);
  }
  if (tenant.id) {
    params.set('tenant_id', String(tenant.id));
  }
  params.set('_', String(Date.now()));

  const response = await desktopApiRequest<unknown>(
    tenant,
    `/tools?${params.toString()}`,
    {},
    accessToken,
  );
  const list = normaliseListResponse<RemoteToolItem>(response.data);

  return {
    items: (list.items ?? []).map((item) => mapToolItem(item, tenant.apiBaseUrl)),
    pagination: mapPagination(list.pagination),
  };
}

export async function getArticleList(
  tenant: Pick<TenantItem, 'apiBaseUrl'> & Partial<Pick<TenantItem, 'id'>>,
  accessToken: string,
  filters: ArticleListFilters,
): Promise<ArticleListResponse> {
  const params = new URLSearchParams({
    page: String(filters.page),
    per_page: String(filters.perPage),
  });
  if (filters.keyword?.trim()) {
    params.set('keyword', filters.keyword.trim());
  }
  appendNumber(params, 'category_id', filters.categoryId);
  appendNumber(params, 'tag_id', filters.tagId);
  appendBoolean(params, 'is_published', filters.isPublished);
  appendBoolean(params, 'is_featured', filters.isFeatured);
  appendBoolean(params, 'is_pinned', filters.isPinned);
  if (filters.sort) {
    params.set('sort', filters.sort);
  }
  if (tenant.id) {
    params.set('tenant_id', String(tenant.id));
  }
  params.set('_', String(Date.now()));

  const response = await desktopApiRequest<unknown>(
    tenant,
    `/articles?${params.toString()}`,
    {},
    accessToken,
  );
  const list = normaliseListResponse<RemoteArticleItem>(response.data);

  return {
    items: (list.items ?? []).map((item) => mapArticleItem(item, tenant.apiBaseUrl)),
    pagination: mapPagination(list.pagination),
  };
}

export async function getToolDetail(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
): Promise<ToolDetail> {
  const response = await desktopApiRequest<RemoteToolItem>(
    tenant,
    `/tools/${encodeURIComponent(String(id))}`,
    {},
    accessToken,
  );
  return mapToolItem(response.data, tenant.apiBaseUrl);
}

export async function getArticleDetail(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
): Promise<ArticleDetail> {
  const response = await desktopApiRequest<RemoteArticleItem>(
    tenant,
    `/articles/${encodeURIComponent(String(id))}`,
    {},
    accessToken,
  );
  return mapArticleItem(response.data, tenant.apiBaseUrl);
}

export async function createTool(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  payload: ToolUpsertPayload,
): Promise<ToolDetail> {
  const response = await desktopApiRequest<RemoteToolItem>(
    tenant,
    '/tools',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(mapToolPayload(payload)),
    },
    accessToken,
  );
  return mapToolItem(response.data, tenant.apiBaseUrl);
}

export async function updateTool(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
  payload: ToolUpsertPayload,
): Promise<ToolDetail> {
  const response = await desktopApiRequest<RemoteToolItem>(
    tenant,
    `/tools/${encodeURIComponent(String(id))}`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(mapToolPayload(payload)),
    },
    accessToken,
  );
  return mapToolItem(response.data, tenant.apiBaseUrl);
}

export async function createArticle(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  payload: ArticleUpsertPayload,
): Promise<ArticleDetail> {
  const response = await desktopApiRequest<RemoteArticleItem>(
    tenant,
    '/articles',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(mapArticlePayload(payload)),
    },
    accessToken,
  );
  return mapArticleItem(response.data, tenant.apiBaseUrl);
}

export async function updateArticle(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
  payload: ArticleUpsertPayload,
): Promise<ArticleDetail> {
  const response = await desktopApiRequest<RemoteArticleItem>(
    tenant,
    `/articles/${encodeURIComponent(String(id))}`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(mapArticlePayload(payload)),
    },
    accessToken,
  );
  return mapArticleItem(response.data, tenant.apiBaseUrl);
}

export async function deleteTool(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
): Promise<void> {
  await desktopApiRequest<null>(
    tenant,
    `/tools/${encodeURIComponent(String(id))}`,
    {
      method: 'DELETE',
    },
    accessToken,
  );
}

export async function batchDeleteTools(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  items: BatchDeleteItemInput[],
): Promise<BatchDeleteResponse> {
  const idempotencyKey = createIdempotencyKey('tools-batch-delete');
  const response = await desktopApiRequest<RemoteBatchDeleteResponse>(
    tenant,
    '/tools/batch-delete',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify({
        items: mapBatchDeleteItems(items),
      }),
    },
    accessToken,
  );
  return mapBatchDeleteResponse(response.data, idempotencyKey);
}

export async function deleteArticle(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  id: string | number,
): Promise<void> {
  await desktopApiRequest<null>(
    tenant,
    `/articles/${encodeURIComponent(String(id))}`,
    {
      method: 'DELETE',
    },
    accessToken,
  );
}

export async function batchDeleteArticles(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  items: BatchDeleteItemInput[],
): Promise<BatchDeleteResponse> {
  const idempotencyKey = createIdempotencyKey('articles-batch-delete');
  const response = await desktopApiRequest<RemoteBatchDeleteResponse>(
    tenant,
    '/articles/batch-delete',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify({
        items: mapBatchDeleteItems(items),
      }),
    },
    accessToken,
  );
  return mapBatchDeleteResponse(response.data, idempotencyKey);
}
