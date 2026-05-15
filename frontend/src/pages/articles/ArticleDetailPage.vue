<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { ArticleDetail } from '../../types/content';
import { deleteArticle, getArticleDetail } from '../../services/content.service';
import { normaliseTenantHtmlAssets, resolveTenantAssetUrl } from '../../utils/url';
import { useDetailPage } from '../../composables/useDetailPage';

const router = useRouter();

const {
  siteStore,
  loading,
  deleting,
  detail,
  entityId,
  loadDetail,
  handleDelete,
} = useDetailPage<ArticleDetail>({
  contentLabel: '文章',
  load: (apiBaseUrl, accessToken, id) => getArticleDetail({ apiBaseUrl }, accessToken, id),
  delete: (apiBaseUrl, accessToken, id) => deleteArticle({ apiBaseUrl }, accessToken, id),
  onDeleted: () => router.push('/articles'),
  emptyMessage: '暂无文章详情数据',
});

const resolvedThumbnail = computed(() => (
  resolveTenantAssetUrl(detail.value?.thumbnail || '', siteStore.currentTenant?.apiBaseUrl)
));
const resolvedContent = computed(() => (
  normaliseTenantHtmlAssets(detail.value?.content || '', siteStore.currentTenant?.apiBaseUrl)
));
</script>

<template>
  <div class="page-shell">
    <div class="page-toolbar">
      <div class="page-toolbar__title">
        <h2>文章详情</h2>
        <p>当前对接远端 `/api/desktop/articles/:id`，用于联调单条文章读取。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="arcoblue">当前租户：{{ siteStore.currentTenant?.name || '未选择' }}</a-tag>
          <a-tag v-if="detail" :color="detail.isPublished ? 'green' : 'gray'">
            {{ detail.isPublished ? '已发布' : '未发布' }}
          </a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button @click="router.push('/articles')">返回列表</a-button>
          <a-button :loading="loading" @click="loadDetail">刷新详情</a-button>
          <a-button status="danger" :loading="deleting" @click="handleDelete">删除文章</a-button>
          <a-button type="primary" @click="router.push(`/articles/${entityId}/edit`)">编辑文章</a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-spin :loading="loading" style="width: 100%">
        <a-empty v-if="!detail" description="暂无文章详情数据" />
        <template v-else>
          <a-row :gutter="16">
            <a-col :span="10">
              <a-card title="基础信息">
              <a-descriptions :column="1" bordered size="large">
                <a-descriptions-item label="标题">{{ detail.title }}</a-descriptions-item>
                <a-descriptions-item label="Slug">{{ detail.slug }}</a-descriptions-item>
                <a-descriptions-item label="分类">{{ detail.category?.name || '-' }}</a-descriptions-item>
                <a-descriptions-item label="发布时间">{{ detail.publishedAt || '-' }}</a-descriptions-item>
                <a-descriptions-item label="发布状态">{{ detail.isPublished ? '已发布' : '未发布' }}</a-descriptions-item>
                <a-descriptions-item label="推荐状态">{{ detail.isFeatured ? '已推荐' : '未推荐' }}</a-descriptions-item>
                <a-descriptions-item label="置顶状态">{{ detail.isPinned ? '已置顶' : '未置顶' }}</a-descriptions-item>
                <a-descriptions-item label="更新时间">{{ detail.updatedAt }}</a-descriptions-item>
              </a-descriptions>
              </a-card>

              <a-card title="标签" style="margin-top: 16px">
              <a-space wrap>
                <a-tag v-for="item in detail.tags" :key="item.id">{{ item.name }}</a-tag>
                <span v-if="!detail.tags.length">-</span>
              </a-space>
              </a-card>

              <a-card title="媒体资源" style="margin-top: 16px">
              <div class="media-section__title">缩略图</div>
              <a-image
                v-if="resolvedThumbnail"
                :src="resolvedThumbnail"
                width="100%"
                :preview="true"
                alt="article-thumbnail"
              />
              <span v-else>-</span>
              </a-card>
            </a-col>

            <a-col :span="14">
              <a-card title="摘要">
              <div class="detail-text">{{ detail.excerpt || '-' }}</div>
              </a-card>

              <a-card title="正文预览" style="margin-top: 16px">
              <div class="html-preview" v-html="resolvedContent || '<p>-</p>'"></div>
              </a-card>
            </a-col>
          </a-row>
        </template>
      </a-spin>
    </div>
  </div>
</template>

<style scoped>
.detail-text {
  white-space: pre-wrap;
  line-height: 1.7;
  color: #1d2129;
}

.media-section__title {
  margin-bottom: 8px;
  font-weight: 600;
  color: #1d2129;
}

.html-preview {
  min-height: 160px;
  line-height: 1.8;
  color: #1d2129;
}
</style>
