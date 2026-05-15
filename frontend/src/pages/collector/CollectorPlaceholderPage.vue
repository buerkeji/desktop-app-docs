<script setup lang="ts">
import { provide } from 'vue';
import { useCollectorPage } from '../../composables/useCollectorPage';
import CollectorToolbar from '../../components/collector/CollectorToolbar.vue';
import CollectorEntryCard from '../../components/collector/CollectorEntryCard.vue';
import CollectorAlertBanner from '../../components/collector/CollectorAlertBanner.vue';
import CollectorBatchPanel from '../../components/collector/CollectorBatchPanel.vue';
import CollectorResultCard from '../../components/collector/CollectorResultCard.vue';
import CollectorPreviewCard from '../../components/collector/CollectorPreviewCard.vue';
import CollectorFieldMappingCard from '../../components/collector/CollectorFieldMappingCard.vue';

const ctx = useCollectorPage();

const { collectMode } = ctx;

provide('collectorContext', ctx);
</script>

<template>
  <div class="page-shell">
    <CollectorToolbar />

    <div class="page-content page-content--scroll">
      <CollectorEntryCard />

      <CollectorAlertBanner />

      <CollectorBatchPanel />

      <a-row v-if="collectMode === 'single'" :gutter="16" class="collector-grid">
        <a-col :span="8" class="collector-grid__side">
          <CollectorResultCard />
        </a-col>
        <a-col :span="16" class="collector-grid__main">
          <a-space direction="vertical" fill size="large" class="collector-main-stack">
            <CollectorPreviewCard />
            <CollectorFieldMappingCard />
          </a-space>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<style scoped>
.collector-grid__side,
.collector-grid__main {
  display: flex;
  flex-direction: column;
}
.collector-grid__main .collector-main-stack {
  display: flex;
  flex-direction: column;
}
</style>
