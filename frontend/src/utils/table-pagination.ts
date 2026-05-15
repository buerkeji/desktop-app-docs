import type { PaginationProps } from '@arco-design/web-vue';

export const DEFAULT_TABLE_PAGE_SIZE = 20;
export const TABLE_PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

export function createTablePagination(overrides: Partial<PaginationProps> = {}): PaginationProps {
  return {
    pageSize: DEFAULT_TABLE_PAGE_SIZE,
    showTotal: true,
    showPageSize: true,
    hideOnSinglePage: true,
    pageSizeOptions: TABLE_PAGE_SIZE_OPTIONS,
    ...overrides,
  };
}
