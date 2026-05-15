<script setup lang="ts">
import { inject } from 'vue';
import SiteTenantContextBar from '../SiteTenantContextBar.vue';

const ctx = inject<Record<string, any>>('collectorContext')!;

const siteStore = ctx.siteStore;
const router = ctx.router;
const filteredTenants = ctx.filteredTenants;
const currentSiteId = ctx.currentSiteId;
const currentTenantId = ctx.currentTenantId;
const syncingTenants = ctx.syncingTenants;
const collectMode = ctx.collectMode;
const sourceUrl = ctx.sourceUrl;
const collecting = ctx.collecting;
const canCollect = ctx.canCollect;
const useBrowserRender = ctx.useBrowserRender;
const batchUrlsText = ctx.batchUrlsText;
const batchDraftType = ctx.batchDraftType;
const batchStatus = ctx.batchStatus;
const batchProgress = ctx.batchProgress;
const scanning = ctx.scanning;
const scanUrl = ctx.scanUrl;
const scanMaxLinks = ctx.scanMaxLinks;
const scanSitemap = ctx.scanSitemap;
const scanFilterRules = ctx.scanFilterRules;
const scanResult = ctx.scanResult;
const scannedSelectedLinks = ctx.scannedSelectedLinks;
const collectedUrlSet = ctx.collectedUrlSet;
const hasScanSettings = ctx.hasScanSettings;
const currentScanHost = ctx.currentScanHost;
const showHistoryPanel = ctx.showHistoryPanel;
const collectedHistoryItems = ctx.collectedHistoryItems;
const historyPanelHost = ctx.historyPanelHost;
const skipCollectedUrls = ctx.skipCollectedUrls;
const collectorStore = ctx.collectorStore;
const batchResults = ctx.batchResults;

const handleSiteChange = ctx.handleSiteChange;
const handleTenantChange = ctx.handleTenantChange;
const syncCurrentSiteTenants = ctx.syncCurrentSiteTenants;
const handleCollect = ctx.handleCollect;
const handleScanSite = ctx.handleScanSite;
const handleScanReset = ctx.handleScanReset;
const toggleScannedLink = ctx.toggleScannedLink;
const selectAllScannedLinks = ctx.selectAllScannedLinks;
const sendSelectedToBatch = ctx.sendSelectedToBatch;
const handleBatchCollect = ctx.handleBatchCollect;
const handleBatchPause = ctx.handleBatchPause;
const handleBatchResume = ctx.handleBatchResume;
const handleBatchStop = ctx.handleBatchStop;
const handleBatchReset = ctx.handleBatchReset;
const addScanFilterRule = ctx.addScanFilterRule;
const removeScanFilterRule = ctx.removeScanFilterRule;
const updateScanFilterRule = ctx.updateScanFilterRule;
const toggleHistoryPanel = ctx.toggleHistoryPanel;
const forgetScanSettings = ctx.forgetScanSettings;
const SCAN_FIELD_OPTIONS = ctx.SCAN_FIELD_OPTIONS;
const SCAN_OPERATOR_OPTIONS = ctx.SCAN_OPERATOR_OPTIONS;
</script>

<template>
  <a-card title="采集入口" class="page-section-card">
    <a-space direction="vertical" fill size="large">
      <SiteTenantContextBar
        :sites="siteStore.sites"
        :tenants="filteredTenants"
        :current-site-id="currentSiteId"
        :current-tenant-id="currentTenantId"
        :syncing="syncingTenants"
        @site-change="handleSiteChange"
        @tenant-change="handleTenantChange"
        @sync-tenants="syncCurrentSiteTenants"
        @manage-sites="router.push('/sites')"
        @login-tenant="router.push('/login')"
      />

      <a-radio-group v-model="collectMode" type="button" size="small">
        <a-radio value="single">单条采集</a-radio>
        <a-radio value="batch">批量采集</a-radio>
        <a-radio value="scan">全站扫描</a-radio>
      </a-radio-group>

      <template v-if="collectMode === 'single'">
        <a-input-search
          v-model="sourceUrl"
          placeholder="请输入要采集的网页地址，例如 https://example.com/article"
          search-button
          allow-clear
          :loading="collecting"
          :disabled="!canCollect"
          @search="handleCollect"
        />
        <a-space wrap>
          <a-tag color="green">静态抓取</a-tag>
          <a-tag color="blue">支持字段映射</a-tag>
          <a-tag color="purple">可转本地草稿</a-tag>
        </a-space>
        <a-space wrap style="margin-top: 4px;">
          <a-checkbox v-model="useBrowserRender" :disabled="collecting">
            浏览器渲染（对需要JS渲染的动态页面效果更好，但会更慢）
          </a-checkbox>
        </a-space>
      </template>

      <template v-else-if="collectMode === 'scan'">
        <a-input
          v-model="scanUrl"
          placeholder="请输入要扫描的网站首页或列表页地址，例如 https://example.com"
          allow-clear
          :disabled="scanning"
          @keyup.enter="handleScanSite"
        >
          <template #suffix>
            <a-button type="primary" size="small" :loading="scanning" :disabled="scanning" @click="handleScanSite">开始扫描</a-button>
          </template>
        </a-input>
        <a-space wrap>
          <a-space>
            <span style="font-size: 13px; color: #4e5969;">最多采集：</span>
            <a-input-number v-model="scanMaxLinks" :min="1" :max="500" :step="1" size="small" style="width: 100px" :disabled="scanning" />
            <span style="font-size: 13px; color: #86909c;" />
          </a-space>
          <a-checkbox v-model="scanSitemap" :disabled="scanning">扫描 robots.txt / sitemap.xml</a-checkbox>
          <a-tag color="green">自动发现同站链接</a-tag>
          <a-tag color="blue">支持选择后批量采集</a-tag>
          <a-tag color="purple">自动保存草稿</a-tag>
          <a-tag v-if="hasScanSettings" color="orange">{{ currentScanHost }} 已保存设置</a-tag>
        </a-space>

        <a-space v-if="hasScanSettings" wrap style="margin-top: 4px;">
          <a-button size="mini" status="warning" @click="forgetScanSettings">清除 {{ currentScanHost }} 的保存设置</a-button>
        </a-space>

        <a-space v-if="currentScanHost" wrap style="margin-top: 4px;">
          <a-button size="mini" @click="toggleHistoryPanel">
            {{ showHistoryPanel ? '收起记录' : `采集记录 (${collectedHistoryItems.length})` }}
          </a-button>
        </a-space>

        <a-collapse :default-active-key="scanFilterRules.length ? ['filter'] : []" :bordered="false" size="small">
          <a-collapse-item key="filter" header="自定义过滤规则">
            <template #extra>
              <a-button size="mini" type="text" @click.stop="addScanFilterRule">+ 添加规则</a-button>
            </template>
            <a-space direction="vertical" fill size="small">
              <div v-for="(rule, ruleIndex) in scanFilterRules" :key="ruleIndex" class="scan-filter-rule-row">
                <a-select
                  :model-value="rule.field" :options="SCAN_FIELD_OPTIONS" size="small" style="width: 90px" :disabled="scanning"
                  @change="(val: any) => updateScanFilterRule(ruleIndex, 'field', String(val))"
                />
                <a-select
                  :model-value="rule.operator" :options="SCAN_OPERATOR_OPTIONS" size="small" style="width: 120px" :disabled="scanning"
                  @change="(val: any) => updateScanFilterRule(ruleIndex, 'operator', String(val))"
                />
                <a-input
                  :model-value="rule.value" placeholder="关键词或路径" size="small" style="flex: 1; min-width: 120px" :disabled="scanning"
                  @input="(val: string) => updateScanFilterRule(ruleIndex, 'value', val)"
                />
                <a-button size="mini" status="danger" :disabled="scanning" @click="removeScanFilterRule(ruleIndex)">删除</a-button>
              </div>
              <a-empty v-if="!scanFilterRules.length" description="暂无自定义规则，将扫描所有发现的同站链接" style="padding: 8px 0" />
            </a-space>
          </a-collapse-item>
        </a-collapse>

        <template v-if="scanResult">
          <a-divider style="margin: 8px 0" />
          <div class="scan-summary">
            <a-space>
              <strong>{{ scanResult.title || scanResult.finalUrl }}</strong>
              <a-tag color="arcoblue">{{ scanResult.host }}</a-tag>
              <a-tag color="green">{{ scanResult.links.length }} 个链接</a-tag>
              <a-tag
                v-if="scanResult.links.some((link: any) => collectedUrlSet.has(link.url))"
                color="orangered"
              >
                已采集过（跳过）{{ scanResult.links.filter((l: any) => collectedUrlSet.has(l.url)).length }}
              </a-tag>
              <a-tag v-if="scanResult.siteName" color="purple">{{ scanResult.siteName }}</a-tag>
              <a-tag v-if="scanResult.sitemapUrlCount > 0" color="blue">站点地图 {{ scanResult.sitemapUrlCount }}</a-tag>
              <a-tag v-if="scanResult.pageHtmlCount > 0" color="cyan">页面 {{ scanResult.pageHtmlCount }}</a-tag>
            </a-space>
          </div>
          <div class="scan-actions">
            <a-space wrap>
              <a-button size="small" @click="selectAllScannedLinks">{{ scanResult.links.every((link: any) => scannedSelectedLinks.has(link.url)) ? '取消全选' : '全选' }}</a-button>
              <a-button type="primary" size="small" :disabled="!scannedSelectedLinks.size" @click="sendSelectedToBatch">批量采集选定（{{ scannedSelectedLinks.size }}）</a-button>
              <a-button size="small" @click="handleScanReset">清空结果</a-button>
            </a-space>
          </div>
          <div class="scan-links-table">
            <div class="scan-links-table__header">
              <span class="scan-links-table__col-check">选择</span>
              <span class="scan-links-table__col-title">标题</span>
              <span class="scan-links-table__col-url">URL</span>
              <span class="scan-links-table__col-source">来源</span>
            </div>
            <div
              v-for="(link, linkIndex) in scanResult.links" :key="linkIndex"
              class="scan-links-table__row"
              :class="{ 'scan-links-table__row--selected': scannedSelectedLinks.has(link.url) }"
              @click="toggleScannedLink(link.url)"
            >
              <span class="scan-links-table__col-check">
                <a-checkbox :model-value="scannedSelectedLinks.has(link.url)" @click.stop @update:model-value="toggleScannedLink(link.url)" />
              </span>
              <span class="scan-links-table__col-title" :title="link.title">{{ link.title || '-' }}</span>
              <span class="scan-links-table__col-url" :title="link.url">{{ link.url }}</span>
              <span class="scan-links-table__col-source">
                <a-tag v-if="link.source === 'sitemap'" size="small" color="blue">站点地图</a-tag>
                <a-tag v-else size="small" color="cyan">页面</a-tag>
                <a-tag v-if="collectedUrlSet.has(link.url)" size="small" color="red">已采集过（跳过）</a-tag>
              </span>
            </div>
          </div>
        </template>
      </template>

      <template v-else>
        <a-textarea
          v-model="batchUrlsText"
          placeholder="请输入要批量采集的网页地址，每行一个，例如&#10;https://example.com/article/1&#10;https://example.com/article/2&#10;https://example.com/article/3"
          :auto-size="{ minRows: 4, maxRows: 8 }"
          :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused"
        />
        <a-space direction="vertical" fill size="small">
          <a-space>
            <span style="font-size: 13px; color: #4e5969;">草稿类型</span>
            <a-radio-group v-model="batchDraftType" type="button" size="small" :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused">
              <a-radio value="tool">仅工具</a-radio>
              <a-radio value="article">仅文章</a-radio>
              <a-radio value="both">全部</a-radio>
            </a-radio-group>
          </a-space>
          <a-space wrap>
            <a-button
              v-if="batchStatus === 'idle' || batchStatus === 'completed'"
              type="primary"
              :disabled="!canCollect"
              @click="handleBatchCollect"
            >
              开始批量采集
            </a-button>
            <a-button
              v-if="collectorStore.isBatchRunning"
              status="warning"
              @click="handleBatchPause"
            >
              暂停采集
            </a-button>
            <a-button
              v-if="collectorStore.isBatchPaused"
              type="primary"
              @click="handleBatchResume"
            >
              继续采集
            </a-button>
            <a-button
              v-if="collectorStore.isBatchRunning || collectorStore.isBatchPaused"
              status="danger"
              @click="handleBatchStop"
            >
              停止采集
            </a-button>
            <a-button
              v-if="batchStatus === 'idle' || batchStatus === 'completed'"
              :disabled="!batchResults.length"
              @click="handleBatchReset"
            >
              清空结果
            </a-button>
            <a-tag color="green">逐条采集</a-tag>
            <a-tag color="purple">自动保存草稿</a-tag>
            <a-tag color="blue">复用站点规则</a-tag>
            <a-tag v-if="useBrowserRender" color="orange">浏览器渲染</a-tag>
          </a-space>
          <a-space wrap>
            <a-checkbox v-model="skipCollectedUrls" :disabled="collectorStore.isBatchRunning || collectorStore.isBatchPaused">
              跳过已采集链接
            </a-checkbox>
            <a-button
              size="mini"
              :disabled="!historyPanelHost"
              @click="toggleHistoryPanel"
            >
              {{ showHistoryPanel ? '收起记录' : `采集记录 (${collectedHistoryItems.length})` }}
            </a-button>
          </a-space>
          <a-progress
            v-if="collectorStore.isBatchRunning || collectorStore.isBatchPaused || batchStatus === 'completed'"
            :percent="batchProgress.total > 0 ? Math.round((batchProgress.current / batchProgress.total) * 100) : 0"
            :status="batchStatus === 'completed' ? 'success' : collectorStore.isBatchPaused ? 'warning' : undefined"
          >
            <template #text>
              <span v-if="collectorStore.isBatchRunning">{{ `采集中 ${batchProgress.current}/${batchProgress.total}` }}</span>
              <span v-else-if="collectorStore.isBatchPaused">{{ `已暂停 ${batchProgress.current}/${batchProgress.total}` }}</span>
            </template>
          </a-progress>
        </a-space>
      </template>
    </a-space>
  </a-card>
</template>
