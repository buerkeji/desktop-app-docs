<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue';

const SCROLL_TARGET_SELECTORS = ['.arco-table-body', '.arco-table-content-scroll-x'];

const containerRef = ref<HTMLElement | null>(null);
const stickyBarRef = ref<HTMLElement | null>(null);
const stickyInnerWidth = ref(0);
const showStickyBar = ref(false);

let resizeObserver: ResizeObserver | null = null;
let mutationObserver: MutationObserver | null = null;
let frameId = 0;
let syncingFromContainer = false;
let syncingFromStickyBar = false;
let scrollTarget: HTMLElement | null = null;

function resolveScrollTarget() {
  const container = containerRef.value;
  if (!container) {
    return null;
  }

  for (const selector of SCROLL_TARGET_SELECTORS) {
    const candidate = container.querySelector(selector);
    if (candidate instanceof HTMLElement) {
      return candidate;
    }
  }

  return container;
}

function bindScrollTarget() {
  const nextTarget = resolveScrollTarget();
  if (scrollTarget === nextTarget) {
    return;
  }

  if (scrollTarget) {
    scrollTarget.removeEventListener('scroll', handleContainerScroll);
  }

  scrollTarget = nextTarget;
  scrollTarget?.addEventListener('scroll', handleContainerScroll, { passive: true });
}

function updateMetrics() {
  bindScrollTarget();

  const target = scrollTarget;
  if (!target) {
    stickyInnerWidth.value = 0;
    showStickyBar.value = false;
    return;
  }

  const scrollWidth = target.scrollWidth;
  const clientWidth = target.clientWidth;
  stickyInnerWidth.value = scrollWidth;
  showStickyBar.value = scrollWidth - clientWidth > 1;

  if (stickyBarRef.value && stickyBarRef.value.scrollLeft !== target.scrollLeft) {
    stickyBarRef.value.scrollLeft = target.scrollLeft;
  }
}

function scheduleUpdate() {
  if (frameId) {
    cancelAnimationFrame(frameId);
  }

  frameId = requestAnimationFrame(() => {
    frameId = 0;
    updateMetrics();
  });
}

function handleContainerScroll() {
  const target = scrollTarget;
  const stickyBar = stickyBarRef.value;
  if (!target || !stickyBar || syncingFromStickyBar) {
    return;
  }

  syncingFromContainer = true;
  stickyBar.scrollLeft = target.scrollLeft;
  requestAnimationFrame(() => {
    syncingFromContainer = false;
  });
}

function handleStickyBarScroll() {
  const target = scrollTarget;
  const stickyBar = stickyBarRef.value;
  if (!target || !stickyBar || syncingFromContainer) {
    return;
  }

  syncingFromStickyBar = true;
  target.scrollLeft = stickyBar.scrollLeft;
  requestAnimationFrame(() => {
    syncingFromStickyBar = false;
  });
}

onMounted(async () => {
  await nextTick();
  updateMetrics();

  const container = containerRef.value;
  if (!container) {
    return;
  }

  resizeObserver = new ResizeObserver(() => {
    scheduleUpdate();
  });
  resizeObserver.observe(container);

  const content = container.firstElementChild;
  if (content instanceof HTMLElement) {
    resizeObserver.observe(content);
  }

  mutationObserver = new MutationObserver(() => {
    scheduleUpdate();
  });
  mutationObserver.observe(container, {
    childList: true,
    subtree: true,
    attributes: true,
  });

  window.addEventListener('resize', scheduleUpdate);
});

onBeforeUnmount(() => {
  if (frameId) {
    cancelAnimationFrame(frameId);
  }

  scrollTarget?.removeEventListener('scroll', handleContainerScroll);
  resizeObserver?.disconnect();
  mutationObserver?.disconnect();
  window.removeEventListener('resize', scheduleUpdate);
});
</script>

<template>
  <div class="page-table-scroll-shell">
    <div ref="containerRef" class="page-table-scroll">
      <slot />
    </div>
    <div
      v-show="showStickyBar"
      ref="stickyBarRef"
      class="page-table-scrollbar"
      @scroll="handleStickyBarScroll"
    >
      <div class="page-table-scrollbar__inner" :style="{ width: `${stickyInnerWidth}px` }" />
    </div>
  </div>
</template>
