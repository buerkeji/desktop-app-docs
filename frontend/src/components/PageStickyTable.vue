<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import type { PaginationProps } from '@arco-design/web-vue';
import StickyTableScroll from './StickyTableScroll.vue';
import { createTablePagination } from '../utils/table-pagination';

defineOptions({
  inheritAttrs: false,
});

const attrs = useAttrs();

const tableAttrs = computed(() => {
  const nextAttrs = { ...attrs };
  const pagination = nextAttrs.pagination;

  if (pagination === false) {
    return nextAttrs;
  }

  if (pagination == null || pagination === true) {
    return {
      ...nextAttrs,
      pagination: createTablePagination(),
    };
  }

  return {
    ...nextAttrs,
    pagination: createTablePagination(pagination as Partial<PaginationProps>),
  };
});
</script>

<template>
  <StickyTableScroll>
    <a-table :scrollbar="false" sticky-header v-bind="tableAttrs">
      <template v-for="(_, name) in $slots" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps || {}" />
      </template>
    </a-table>
  </StickyTableScroll>
</template>
