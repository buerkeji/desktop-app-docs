import { ClearSystemLogs, DeleteSystemLog, ListSystemLogs } from '../../wailsjs/go/main/App';
import { hasAppMethod } from './bridge';
import type { SystemLogItem, SystemLogListFilters } from '../types/activity';

function buildPayload(filters: SystemLogListFilters) {
  return {
    tenantId: filters.tenantId ?? 0,
    keyword: filters.keyword?.trim() || '',
    level: filters.level || '',
    module: filters.module?.trim() || '',
    requestId: filters.requestId?.trim() || '',
    dateFrom: filters.dateRange?.[0] || '',
    dateTo: filters.dateRange?.[1] || '',
    limit: filters.limit ?? 500,
  };
}

export async function listSystemLogs(filters: SystemLogListFilters = {}): Promise<SystemLogItem[]> {
  if (!hasAppMethod('ListSystemLogs')) {
    return [];
  }

  return ListSystemLogs(buildPayload(filters)) as Promise<SystemLogItem[]>;
}

export async function deleteSystemLog(logID: number): Promise<void> {
  return DeleteSystemLog(logID);
}

export async function clearAllSystemLogs(): Promise<void> {
  return ClearSystemLogs();
}
