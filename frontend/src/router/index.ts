import { createRouter, createWebHashHistory } from 'vue-router';
import { useAuthStore } from '../stores/auth.store';
import AdminLayout from '../layouts/AdminLayout.vue';

const LoginPage = () => import('../pages/login/LoginPage.vue');
const DashboardPage = () => import('../pages/dashboard/DashboardPage.vue');
const SiteManagementPage = () => import('../pages/sites/SiteManagementPage.vue');
const TenantListPage = () => import('../pages/sites/TenantListPage.vue');
const TenantDataPage = () => import('../pages/sites/TenantDataPage.vue');
const ToolListPage = () => import('../pages/tools/ToolListPage.vue');
const ToolDetailPage = () => import('../pages/tools/ToolDetailPage.vue');
const ToolEditorPage = () => import('../pages/tools/ToolEditorPage.vue');
const ArticleListPage = () => import('../pages/articles/ArticleListPage.vue');
const ArticleDetailPage = () => import('../pages/articles/ArticleDetailPage.vue');
const ArticleEditorPage = () => import('../pages/articles/ArticleEditorPage.vue');
const CollectorPlaceholderPage = () => import('../pages/collector/CollectorPlaceholderPage.vue');
const CollectorRecordsPage = () => import('../pages/collector/CollectorRecordsPage.vue');
const DraftBoxPage = () => import('../pages/drafts/DraftBoxPage.vue');
const SubmitRecordsPage = () => import('../pages/jobs/SubmitRecordsPage.vue');
const MediaTasksPage = () => import('../pages/media/MediaTasksPage.vue');
const SessionManagementPage = () => import('../pages/sessions/SessionManagementPage.vue');
const SystemLogsPage = () => import('../pages/logs/SystemLogsPage.vue');

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      component: LoginPage,
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        {
          path: '/dashboard',
          component: DashboardPage,
        },
        {
          path: '/sites',
          component: SiteManagementPage,
        },
        {
          path: '/tenants',
          component: TenantListPage,
        },
        {
          path: '/tenants/data',
          component: TenantDataPage,
        },
        {
          path: '/collector',
          component: CollectorPlaceholderPage,
        },
        {
          path: '/collector/records',
          component: CollectorRecordsPage,
        },
        {
          path: '/drafts',
          component: DraftBoxPage,
        },
        {
          path: '/submit-records',
          component: SubmitRecordsPage,
        },
        {
          path: '/media-tasks',
          component: MediaTasksPage,
        },
        {
          path: '/sessions',
          component: SessionManagementPage,
        },
        {
          path: '/system-logs',
          component: SystemLogsPage,
        },
        {
          path: '/tools',
          component: ToolListPage,
        },
        {
          path: '/tools/new',
          component: ToolEditorPage,
        },
        {
          path: '/tools/:id',
          component: ToolDetailPage,
        },
        {
          path: '/tools/:id/edit',
          component: ToolEditorPage,
        },
        {
          path: '/articles',
          component: ArticleListPage,
        },
        {
          path: '/articles/new',
          component: ArticleEditorPage,
        },
        {
          path: '/articles/:id',
          component: ArticleDetailPage,
        },
        {
          path: '/articles/:id/edit',
          component: ArticleEditorPage,
        },
      ],
    },
  ],
});

router.beforeEach((to) => {
  const authStore = useAuthStore();
  const hasToken = Boolean(authStore.token?.accessToken);

  if (!hasToken && to.path !== '/login') {
    return '/login';
  }

  if (hasToken && to.path === '/login') {
    return '/dashboard';
  }

  return true;
});

export default router;
