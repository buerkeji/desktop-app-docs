import { DeleteSubmitRecord, ListSubmitRecords } from '../../wailsjs/go/main/App';
import { hasAppMethod } from './bridge';
import type { SubmitRecordItem, SubmitRecordListFilters } from '../types/activity';

function buildPayload(filters: SubmitRecordListFilters) {
  return {
    tenantId: filters.tenantId ?? 0,
    keyword: filters.keyword?.trim() || '',
    contentType: filters.contentType || '',
    status: filters.status || '',
    dateFrom: filters.dateRange?.[0] || '',
    dateTo: filters.dateRange?.[1] || '',
    limit: filters.limit ?? 500,
  };
}

export async function listSubmitRecords(filters: SubmitRecordListFilters = {}): Promise<SubmitRecordItem[]> {
  if (!hasAppMethod('ListSubmitRecords')) {
    return [];
  }

  return ListSubmitRecords(buildPayload(filters)) as Promise<SubmitRecordItem[]>;
}

export async function deleteSubmitRecord(jobID: number): Promise<void> {
  return DeleteSubmitRecord(jobID);
}
