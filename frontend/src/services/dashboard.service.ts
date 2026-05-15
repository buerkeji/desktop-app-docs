import type { DashboardCardItem } from '../types/dashboard';

interface BuildDashboardCardsInput {
  totalSites: number;
  totalTenants: number;
  draftCount: number;
  sessionCount: number;
  currentTenantName?: string | null;
}

export function buildDashboardCards(input: BuildDashboardCardsInput): DashboardCardItem[] {
  return [
    { key: 'sites', title: '站点数量', value: String(input.totalSites), extra: '当前可直接维护的部署站点' },
    { key: 'tenants', title: '租户数量', value: String(input.totalTenants), extra: '已保存到桌面端的租户配置数量' },
    {
      key: 'drafts',
      title: '本地草稿',
      value: String(input.draftCount),
      extra: input.currentTenantName ? `${input.currentTenantName} 的未提交草稿数量` : '当前租户未选择，暂不统计草稿',
    },
    {
      key: 'sessions',
      title: '远端会话',
      value: String(input.sessionCount),
      extra: input.currentTenantName ? `${input.currentTenantName} 当前可见的登录会话数` : '当前租户未选择，暂不统计会话',
    },
  ];
}
