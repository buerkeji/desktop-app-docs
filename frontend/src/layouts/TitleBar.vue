<script setup lang="ts">
import { computed } from 'vue';
import { IconMinus, IconClose, IconFullscreen } from '@arco-design/web-vue/es/icon';
import { Quit, WindowMinimise, WindowToggleMaximise } from '../../wailsjs/runtime/runtime';
import { useAppStore } from '../stores/app.store';

const appStore = useAppStore();
const title = computed(() => appStore.title);
</script>

<template>
  <header class="title-bar">
    <div class="title-bar__drag">
      <div class="title-bar__brand">
        <strong>{{ title }}</strong>
        <span>桌面管理台</span>
      </div>
      <span class="title-bar__hint">按住此区域可移动窗口</span>
    </div>
    <div class="title-bar__actions">
      <a-button size="mini" type="text" @click="WindowMinimise()">
        <template #icon>
          <IconMinus />
        </template>
      </a-button>
      <a-button size="mini" type="text" @click="WindowToggleMaximise()">
        <template #icon>
          <IconFullscreen />
        </template>
      </a-button>
      <a-button size="mini" type="text" status="danger" @click="Quit()">
        <template #icon>
          <IconClose />
        </template>
      </a-button>
    </div>
  </header>
</template>

<style scoped>
.title-bar {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px 0 16px;
  border-bottom: 1px solid #e5e6eb;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(10px);
}

.title-bar__drag {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex: 1;
  min-width: 0;
  margin-right: 12px;
  padding-right: 12px;
  color: #4e5969;
  --wails-draggable: drag;
}

.title-bar__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.title-bar__drag strong {
  color: #1d2129;
  font-size: 14px;
}

.title-bar__brand span,
.title-bar__hint {
  font-size: 12px;
  color: #86909c;
}

.title-bar__hint {
  white-space: nowrap;
}

.title-bar__actions {
  display: flex;
  gap: 4px;
  --wails-draggable: no-drag;
}
</style>
