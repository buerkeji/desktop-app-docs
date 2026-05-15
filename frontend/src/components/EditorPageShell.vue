<script setup lang="ts">
defineProps<{
  title: string;
  description: string;
  draftUpdatedAt?: string;
  hasDraft: boolean;
  saving: boolean;
  statusText?: string;
  statusTone?: 'neutral' | 'success' | 'warning' | 'danger';
  submitText: string;
  backText?: string;
  draftNavIndex?: number;
  draftNavTotal?: number;
}>();

defineEmits<{
  saveDraft: [];
  clearDraft: [];
  back: [];
  submit: [];
  prevDraft: [];
  nextDraft: [];
  deleteDraft: [];
}>();
</script>

<template>
  <div class="page-shell">
    <div class="editor-shell__topbar">
      <div class="page-toolbar">
        <div class="page-toolbar__title">
          <h2>{{ title }}</h2>
          <p>{{ description }}</p>
        </div>
        <div class="page-toolbar__actions editor-shell__actions">
          <div class="page-toolbar__meta">
            <span
              class="editor-shell__badge"
              :class="`editor-shell__badge--${statusTone || 'neutral'}`"
            >
              {{ statusText || (hasDraft ? '已检测到草稿' : '自动草稿已开启') }}
            </span>
            <span v-if="draftNavTotal && draftNavTotal > 1" class="editor-shell__nav-hint">
              <a-space size="small">
                <a-button size="mini" :disabled="draftNavIndex === 0 || saving" @click="$emit('prevDraft')">
                  ◀ 上一篇
                </a-button>
                <span style="font-size: 13px; color: #4e5969;">{{ (draftNavIndex ?? 0) + 1 }} / {{ draftNavTotal }}</span>
                <a-button size="mini" :disabled="(draftNavIndex ?? 0) >= (draftNavTotal ?? 1) - 1 || saving" @click="$emit('nextDraft')">
                  下一篇 ▶
                </a-button>
              </a-space>
            </span>
          </div>
          <div class="page-toolbar__buttons">
            <a-button :disabled="saving" @click="$emit('saveDraft')">保存草稿</a-button>
            <a-button status="danger" :disabled="!hasDraft" @click="$emit('clearDraft')">清空草稿</a-button>
            <a-button v-if="draftNavTotal && draftNavTotal > 0" status="danger" @click="$emit('deleteDraft')">删除草稿</a-button>
            <a-button @click="$emit('back')">{{ backText || '返回' }}</a-button>
            <a-button type="primary" :loading="saving" @click="$emit('submit')">{{ submitText }}</a-button>
          </div>
        </div>
      </div>
    </div>

    <div class="page-content page-content--scroll">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.editor-shell__topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  padding: 14px 16px 16px;
  border: 1px solid rgba(229, 230, 235, 0.9);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  box-shadow: 0 8px 24px rgba(29, 33, 41, 0.06);
}

.editor-shell__actions {
  --wails-draggable: no-drag;
}

.editor-shell__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  min-width: 180px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: #f2f3f5;
  color: #4e5969;
  font-size: 12px;
  white-space: nowrap;
}

.editor-shell__badge--neutral {
  background: #f2f3f5;
  color: #4e5969;
}

.editor-shell__badge--success {
  background: #e8ffea;
  border-color: #b7eb8f;
  color: #237b38;
}

.editor-shell__badge--warning {
  background: #fff7e8;
  border-color: #ffd591;
  color: #ad6800;
}

.editor-shell__badge--danger {
  background: #fff1f0;
  border-color: #ffccc7;
  color: #cf1322;
}

@media (max-width: 1400px) {
  .editor-shell__topbar {
    position: static;
  }

  .page-toolbar {
    align-items: flex-start;
  }
}
</style>
