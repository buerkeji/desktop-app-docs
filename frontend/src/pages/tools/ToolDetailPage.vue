<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { ToolDetail } from '../../types/content';
import { deleteTool, getToolDetail } from '../../services/content.service';
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
} = useDetailPage<ToolDetail>({
  contentLabel: '工具',
  load: (apiBaseUrl, accessToken, id) => getToolDetail({ apiBaseUrl }, accessToken, id),
  delete: (apiBaseUrl, accessToken, id) => deleteTool({ apiBaseUrl }, accessToken, id),
  onDeleted: () => router.push('/tools'),
  emptyMessage: '暂无工具详情数据',
});

const resolvedIcon = computed(() => resolveTenantAssetUrl(detail.value?.icon || '', siteStore.currentTenant?.apiBaseUrl));
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
        <h2>工具详情</h2>
        <p>当前对接远端 `/api/desktop/tools/:id`，用于联调单条内容读取。</p>
      </div>
      <div class="page-toolbar__actions">
        <div class="page-toolbar__meta">
          <a-tag color="arcoblue">当前租户：{{ siteStore.currentTenant?.name || '未选择' }}</a-tag>
          <a-tag v-if="detail" :color="detail.isActive ? 'green' : 'gray'">
            {{ detail.isActive ? '已上架' : '未上架' }}
          </a-tag>
        </div>
        <div class="page-toolbar__buttons">
          <a-button @click="router.push('/tools')">返回列表</a-button>
          <a-button :loading="loading" @click="loadDetail">刷新详情</a-button>
          <a-button status="danger" :loading="deleting" @click="handleDelete">删除工具</a-button>
          <a-button type="primary" @click="router.push(`/tools/${entityId}/edit`)">编辑工具</a-button>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <a-spin :loading="loading" style="width: 100%">
        <a-empty v-if="!detail" description="暂无工具详情数据" />
        <template v-else>
          <a-row :gutter="16">
            <a-col :span="10">
              <a-card title="基础信息">
              <a-descriptions :column="1" bordered size="large">
                <a-descriptions-item label="标题">{{ detail.title }}</a-descriptions-item>
                <a-descriptions-item label="Slug">{{ detail.slug }}</a-descriptions-item>
                <a-descriptions-item label="分类">{{ detail.category?.name || '-' }}</a-descriptions-item>
                <a-descriptions-item label="链接">{{ detail.url || detail.website || '-' }}</a-descriptions-item>
                <a-descriptions-item label="上架状态">{{ detail.isActive ? '已上架' : '未上架' }}</a-descriptions-item>
                <a-descriptions-item label="推荐状态">{{ detail.isFeatured ? '已推荐' : '未推荐' }}</a-descriptions-item>
                <a-descriptions-item label="更新时间">{{ detail.updatedAt }}</a-descriptions-item>
              </a-descriptions>
              </a-card>

              <a-card title="标签与特性" style="margin-top: 16px">
              <a-space direction="vertical" fill>
                <div>
                  <strong>标签：</strong>
                  <a-space wrap style="margin-left: 8px">
                    <a-tag v-for="item in detail.tags" :key="item.id">{{ item.name }}</a-tag>
                    <span v-if="!detail.tags.length">-</span>
                  </a-space>
                </div>
                <div>
                  <strong>特性：</strong>
                  <a-space direction="vertical" fill style="margin-top: 8px">
                    <a-tag v-for="item in detail.features" :key="item.feature" color="arcoblue">
                      {{ item.feature }}
                    </a-tag>
                    <span v-if="!detail.features.length">-</span>
                  </a-space>
                </div>
              </a-space>
              </a-card>

              <a-card title="媒体资源" style="margin-top: 16px">
              <a-space direction="vertical" fill size="large">
                <div>
                  <div class="media-section__title">图标</div>
                  <a-image
                    v-if="resolvedIcon"
                    :src="resolvedIcon"
                    width="120"
                    :preview="true"
                    alt="tool-icon"
                  />
                  <span v-else>-</span>
                </div>
                <div>
                  <div class="media-section__title">缩略图</div>
                  <a-image
                    v-if="resolvedThumbnail"
                    :src="resolvedThumbnail"
                    width="100%"
                    :preview="true"
                    alt="tool-thumbnail"
                  />
                  <span v-else>-</span>
                </div>
              </a-space>
              </a-card>
            </a-col>

            <a-col :span="14">
              <a-card title="描述">
              <div class="detail-text">{{ detail.description || '-' }}</div>
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
