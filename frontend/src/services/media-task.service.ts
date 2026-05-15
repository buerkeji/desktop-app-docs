import { ClearMediaTaskCache, DeleteMediaTask, ListMediaTasks, RetryMediaTask } from '../../wailsjs/go/main/App';
import { hasAppMethod } from './bridge';
import type { MediaTaskItem, MediaTaskListFilters } from '../types/activity';

function buildPayload(filters: MediaTaskListFilters) {
  return {
    tenantId: filters.tenantId ?? 0,
    keyword: filters.keyword?.trim() || '',
    scene: filters.scene || '',
    status: filters.status || '',
    dateFrom: filters.dateRange?.[0] || '',
    dateTo: filters.dateRange?.[1] || '',
    limit: filters.limit ?? 500,
  };
}

export async function listMediaTasks(filters: MediaTaskListFilters = {}): Promise<MediaTaskItem[]> {
  if (!hasAppMethod('ListMediaTasks')) {
    return [];
  }

  return ListMediaTasks(buildPayload(filters)) as Promise<MediaTaskItem[]>;
}

export async function retryMediaTask(taskId: number): Promise<void> {
  if (!hasAppMethod('RetryMediaTask')) {
    throw new Error('当前桌面端版本暂不支持重传媒体任务');
  }

  await RetryMediaTask({ taskId });
}

export async function clearMediaTaskCache(taskId: number): Promise<void> {
  if (!hasAppMethod('ClearMediaTaskCache')) {
    throw new Error('当前桌面端版本暂不支持清理媒体缓存');
  }

  await ClearMediaTaskCache({ taskId });
}

export async function deleteMediaTask(taskId: number): Promise<void> {
  return DeleteMediaTask(taskId);
}
