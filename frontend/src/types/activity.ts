import type { DateTimeString, Id } from './common';

export type SubmitContentType = 'tool' | 'article';
export type SubmitJobType = 'create' | 'update' | 'batch_delete';
export type SubmitStatus = 'success' | 'failed' | 'partial' | 'pending';
export type MediaTaskStatus = 'success' | 'failed' | 'pending';
export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface SubmitRecordItem {
  id: Id;
  jobId: Id;
  siteId: Id;
  tenantId: Id;
  tenantName: string;
  title: string;
  contentType: SubmitContentType | string;
  jobType: SubmitJobType | string;
  status: SubmitStatus | string;
  idempotencyKey?: string;
  remoteId?: Id | 0;
  remoteUrl?: string;
  matchType?: string;
  createdCount: number;
  updatedCount: number;
  failedCount: number;
  errorMessage?: string;
  payloadJson?: string;
  resultJson?: string;
  submittedAt: DateTimeString;
  createdAt: DateTimeString;
  updatedAt: DateTimeString;
}

export interface SubmitRecordListFilters {
  tenantId?: number;
  keyword?: string;
  contentType?: SubmitContentType | '';
  status?: SubmitStatus | '';
  dateRange?: string[];
  limit?: number;
}

export interface SystemLogItem {
  id: Id;
  siteId: Id;
  tenantId: Id;
  tenantName: string;
  requestId?: string;
  level: LogLevel | string;
  module: string;
  message: string;
  contextJson?: string;
  createdAt: DateTimeString;
}

export interface SystemLogListFilters {
  tenantId?: number;
  keyword?: string;
  level?: LogLevel | '';
  module?: string;
  requestId?: string;
  dateRange?: string[];
  limit?: number;
}

export interface MediaTaskItem {
  id: Id;
  siteId: Id;
  tenantId: Id;
  tenantName: string;
  fileName: string;
  originalName?: string;
  mimeType?: string;
  uploadScene?: string;
  canRetry?: boolean;
  sourceUrl?: string;
  mediaCategoryId?: number;
  draftId?: number;
  status: MediaTaskStatus | string;
  requestId?: string;
  remoteMediaId?: Id | 0;
  remoteUrl?: string;
  remotePath?: string;
  disk?: string;
  sizeBytes: number;
  width?: number;
  height?: number;
  errorMessage?: string;
  responseJson?: string;
  createdAt: DateTimeString;
  updatedAt: DateTimeString;
}

export interface MediaTaskListFilters {
  tenantId?: number;
  keyword?: string;
  scene?: string;
  status?: MediaTaskStatus | '';
  dateRange?: string[];
  limit?: number;
}
