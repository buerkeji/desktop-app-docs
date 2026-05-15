import type { DateTimeString, Id } from './common';

export interface PaginationState {
  currentPage: number;
  perPage: number;
  total: number;
  lastPage: number;
  hasMorePages: boolean;
}

export interface ContentCategory {
  id: Id;
  name: string;
  slug: string;
}

export interface ContentTag {
  id: Id;
  name: string;
  slug: string;
}

export interface ToolFeatureItem {
  feature: string;
}

export interface ContentTagDraft {
  name: string;
  slug?: string;
}

export type ContentTagInput = Id | string | ContentTagDraft;

export interface ToolListItem {
  id: Id;
  title: string;
  slug: string;
  url?: string | null;
  icon?: string | null;
  thumbnail?: string | null;
  categoryId?: Id | null;
  description?: string | null;
  content?: string | null;
  features: ToolFeatureItem[];
  website?: string | null;
  isFeatured: boolean;
  isActive: boolean;
  sortOrder: number;
  metaTitle?: string | null;
  metaKeywords?: string | null;
  metaDescription?: string | null;
  category?: ContentCategory | null;
  tags: ContentTag[];
  createdAt: DateTimeString;
  updatedAt: DateTimeString;
}

export interface ArticleListItem {
  id: Id;
  title: string;
  slug: string;
  thumbnail?: string | null;
  excerpt?: string | null;
  content?: string | null;
  categoryId?: Id | null;
  authorId?: Id | null;
  metaTitle?: string | null;
  metaKeywords?: string | null;
  metaDescription?: string | null;
  isFeatured: boolean;
  isPinned: boolean;
  isPublished: boolean;
  publishedAt?: DateTimeString | null;
  category?: ContentCategory | null;
  tags: ContentTag[];
  createdAt: DateTimeString;
  updatedAt: DateTimeString;
}

export interface ToolListFilters {
  page: number;
  perPage: number;
  keyword?: string;
  categoryId?: number;
  tagId?: number;
  isActive?: boolean;
  isFeatured?: boolean;
  sort?: 'default' | 'latest' | 'oldest' | 'title';
}

export interface ArticleListFilters {
  page: number;
  perPage: number;
  keyword?: string;
  categoryId?: number;
  tagId?: number;
  isPublished?: boolean;
  isFeatured?: boolean;
  isPinned?: boolean;
  sort?: 'latest' | 'oldest' | 'published' | 'title';
}

export interface ToolListResponse {
  items: ToolListItem[];
  pagination: PaginationState;
}

export interface ArticleListResponse {
  items: ArticleListItem[];
  pagination: PaginationState;
}

export type ToolDetail = ToolListItem;

export type ArticleDetail = ArticleListItem;

export interface ToolUpsertPayload {
  title?: string;
  slug?: string;
  url?: string;
  icon?: string;
  thumbnail?: string;
  categoryId?: Id;
  description?: string;
  content?: string;
  features?: Array<string | ToolFeatureItem>;
  website?: string;
  isFeatured?: boolean;
  isActive?: boolean;
  sortOrder?: number;
  metaTitle?: string;
  metaKeywords?: string;
  metaDescription?: string;
  tags?: ContentTagInput[];
}

export interface ArticleUpsertPayload {
  title?: string;
  slug?: string;
  categoryId?: Id;
  thumbnail?: string;
  excerpt?: string;
  content?: string;
  authorId?: Id;
  metaTitle?: string;
  metaKeywords?: string;
  metaDescription?: string;
  isFeatured?: boolean;
  isPinned?: boolean;
  isPublished?: boolean;
  publishedAt?: string;
  tags?: ContentTagInput[];
}

export interface BatchDeleteItemInput {
  clientId?: string;
  remoteId?: Id;
  slug?: string;
}

export interface BatchDeleteSummary {
  total: number;
  deleted: number;
  failed: number;
}

export interface BatchDeleteItemResult {
  index: number;
  clientId?: string | null;
  success: boolean;
  action: string;
  remoteId?: Id | string | null;
  errors?: Record<string, string[]> | null;
  errorCode?: string | null;
}

export interface BatchDeleteResponse {
  summary: BatchDeleteSummary;
  items: BatchDeleteItemResult[];
  idempotencyKey: string;
}
