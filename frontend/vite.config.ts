import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    chunkSizeWarningLimit: 900,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return;
          }

          if (id.includes('@wangeditor/editor-for-vue')) {
            return 'wangeditor-vue';
          }

          if (id.includes('@wangeditor/editor')) {
            return 'wangeditor-editor';
          }

          if (id.includes('@wangeditor/basic-modules')) {
            return 'wangeditor-basic-modules';
          }

          if (id.includes('@wangeditor/core')) {
            return 'wangeditor-core';
          }

          if (id.includes('@wangeditor/')) {
            return 'wangeditor-shared';
          }

          if (id.includes('@arco-design')) {
            return 'arco';
          }

          if (id.includes('vue-router')) {
            return 'router';
          }

          if (id.includes('pinia')) {
            return 'pinia';
          }

          if (id.includes('/vue/')) {
            return 'vue';
          }
        },
      },
    },
  },
});
