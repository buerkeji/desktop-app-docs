<script setup lang="ts">
import { Message } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import type { ArticleListFilters, ArticleListItem } from '../../types/content';
import { batchDeleteArticles, getArticleList } from '../../services/content.service';
import { resolveTenantAssetUrl } from '../../utils/url';
import { useListPage } from '../../composables/useListPage';
import PageStickyTable from '../../components/PageStickyTable.vue';

const router = useRouter();

const {
  authStore,
  siteStore,
  dictionaryStore,
  loading,
  batchDeleting,
  rows,
  selectedRowKeys,
  pagination,
  paginationConfig,
  filters,
  handleSearch,
  handleReset,
  handlePageChange,
  handlePageSizeChange,
  handleSelectionChange,
  invalidateAndLoad,
} = useListPage<ArticleListFilters>({
  contentLabel: '文章',
  dictLabel: '文章',
  initialFilters: { page: 1, perPage: 20, keyword: '', sort: 'latest' },
  loadPage: async () => {
    const result = await getArticleList(
      { id: siteStore.currentTenant!.id, apiBaseUrl: siteStore.currentTenant!.apiBaseUrl },
      authStore.token!.accessToken!,
      filters.value,
    );
    return result;
  },
});

function openDetail(record: ArticleListItem) {
  void router.push(`/articles/${record.id}`);
}

function openEdit(record: ArticleListItem) {
  void router.push(`/articles/${record.id}/edit`);
}

function resolvePreviewUrl(record: ArticleListItem): string {
  return resolveTenantAssetUrl(record.thumbnail || '', siteStore.currentTenant?.apiBaseUrl);
}

async function handleBatchDelete() {
  if (!siteStore.currentTenant?.apiBaseUrl || !authStore.token?.accessToken) {
    Message.warning('当前缺少租户上下文或登录态');
    return;
  }

  const targets = rows.value.filter((item: any) => selectedRowKeys.value.includes(item.id));
  if (!targets.length) {
    Message.warning('请先选择要删除的文章');
    return;
  }
  if (!window.confirm(`确认批量删除选中的 ${targets.length} 篇文章吗？此操作不可撤销。`)) {
    return;
  }

  batchDeleting.value = true;
  try {
    const result = await batchDeleteArticles(
      { apiBaseUrl: siteStore.currentTenant.apiBaseUrl },
      authStore.token.accessToken,
      targets.map((item: any, index: number) => ({
        clientId: `article-${item.id}-${index + 1}`,
        remoteId: item.id,
        slug: item.slug,
      })),
    );
    if (result.summary.failed > 0) {
      Message.warning(
        `批量删除已提交：成功 ${result.summary.deleted}，失败 ${result.summary.failed}，幂等键 ${result.idempotencyKey}`,
      );
    } else {
      Message.success(`已删除 ${result.summary.deleted} 篇文章，幂等键 ${result.idempotencyKey}`);
    }
    selectedRowKeys.value = [];
    invalidateAndLoad();
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '批量删除文章失败');
  } finally {
    batchDeleting.value = false;
  }
}
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>文章管理</h2>
        <p>对接远端 `/api/desktop/articles`，先完成列表、筛选和分页联调。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="arcoblue">当前租户：{{ siteStore.currentTenant?.name || '未选择' }}</a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button @click="invalidateAndLoad">刷新列表</a-button>
          <a-button
            status="danger"
            :disabled="!selectedRowKeys.length"
            :loading="batchDeleting"
            @click="handleBatchDelete"
          >
            批量删除
          </a-button>
          <a-button type="primary" @click="router.push('/articles/new')">新建文章</a-button>
        </div>
      </div>
    </div>

    <div class="page-content">
      <a-card class="page-section-card">
        <a-form layout="vertical" :model="filters" class="page-filter-form page-filter-form--articles">
          <a-form-item field="keyword" label="关键词">
            <a-input v-model="filters.keyword" allow-clear placeholder="标题 / slug / 摘要 / 正文" />
          </a-form-item>
          <a-form-item field="categoryId" label="分类">
            <a-select v-model="filters.categoryId" allow-clear placeholder="全部分类">
              <a-option v-for="item in dictionaryStore.categories" :key="item.id" :value="item.id">
                {{ item.label }}
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="isPublished" label="发布状态">
            <a-select v-model="filters.isPublished" allow-clear placeholder="全部状态">
              <a-option :value="true">已发布</a-option>
              <a-option :value="false">未发布</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="isFeatured" label="推荐状态">
            <a-select v-model="filters.isFeatured" allow-clear placeholder="全部状态">
              <a-option :value="true">已推荐</a-option>
              <a-option :value="false">未推荐</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="isPinned" label="置顶状态">
            <a-select v-model="filters.isPinned" allow-clear placeholder="全部状态">
              <a-option :value="true">已置顶</a-option>
              <a-option :value="false">未置顶</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="sort" label="排序">
            <a-select v-model="filters.sort">
              <a-option value="latest">最新</a-option>
              <a-option value="oldest">最早</a-option>
              <a-option value="published">发布时间</a-option>
              <a-option value="title">标题</a-option>
            </a-select>
          </a-form-item>
          <a-form-item class="page-filter-form__actions">
            <a-space>
              <a-button type="primary" :loading="loading" @click="handleSearch">查询</a-button>
              <a-button @click="handleReset">重置</a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </a-card>

      <a-card class="page-scroll-card">
        <PageStickyTable
          row-key="id"
          :data="rows"
          :loading="loading"
          :row-selection="{
            type: 'checkbox',
            showCheckedAll: true,
            selectedRowKeys,
          }"
          :pagination="{
            ...paginationConfig,
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
          }"
          @page-change="handlePageChange"
          @page-size-change="handlePageSizeChange"
          @selection-change="handleSelectionChange"
        >
          <template #columns>
            <a-table-column title="预览" :width="90" align="center">
              <template #cell="{ record }">
                <a-image
                  v-if="resolvePreviewUrl(record)"
                  :src="resolvePreviewUrl(record)"
                  width="44"
                  height="44"
                  fit="cover"
                  :preview="true"
                  alt="article-preview"
                />
                <span v-else>-</span>
              </template>
            </a-table-column>
            <a-table-column title="标题" :width="220">
              <template #cell="{ record }">
                <a-space direction="vertical" size="mini" fill>
                  <a-link @click="openDetail(record)">{{ record.title }}</a-link>
                  <a-typography-text type="secondary" style="font-size: 12px">
                    别名：{{ record.slug || '-' }}
                  </a-typography-text>
                </a-space>
              </template>
            </a-table-column>
            <a-table-column title="分类">
              <template #cell="{ record }">
                {{ record.category?.name || '-' }}
              </template>
            </a-table-column>
            <a-table-column title="发布">
              <template #cell="{ record }">
                <a-tag :color="record.isPublished ? 'green' : 'gray'">{{ record.isPublished ? '是' : '否' }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="推荐">
              <template #cell="{ record }">
                <a-tag :color="record.isFeatured ? 'orange' : 'gray'">{{ record.isFeatured ? '是' : '否' }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="置顶">
              <template #cell="{ record }">
                <a-tag :color="record.isPinned ? 'purple' : 'gray'">{{ record.isPinned ? '是' : '否' }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="更新时间" data-index="updatedAt" :width="180" />
            <a-table-column title="操作" :width="136" fixed="right">
              <template #cell="{ record }">
                <a-space size="mini" class="page-table-actions">
                  <a-button type="text" size="small" @click="openDetail(record)">详情</a-button>
                  <a-button type="text" size="small" @click="openEdit(record)">编辑</a-button>
                </a-space>
              </template>
            </a-table-column>
          </template>
        </PageStickyTable>
      </a-card>
    </div>
  </div>
</template>
