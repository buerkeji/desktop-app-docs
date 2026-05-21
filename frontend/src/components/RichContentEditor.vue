﻿﻿﻿﻿﻿﻿<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, shallowRef, watch } from 'vue';
import { Message } from '@arco-design/web-vue';
import { Editor, Toolbar } from '@wangeditor/editor-for-vue';
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor/editor';
import '@wangeditor/editor/dist/css/style.css';
import { readFileAsDataUrl } from '../services/deferred-media.service';
import type { TenantItem } from '../types/site';

const props = withDefaults(
  defineProps<{
    modelValue: string;
    tenant: TenantItem | null;
    accessToken?: string;
    mediaCategoryId?: number;
    uploadScene?: string;
    label?: string;
    placeholder?: string;
    hint?: string;
    required?: boolean;
    minHeight?: number;
  }>(),
  {
    label: '正文',
    placeholder: '请输入正文内容',
    hint: '',
    required: false,
    minHeight: 360,
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

type EditorImageInsertFn = (src: string, alt: string, href: string) => void;

const imageUploading = ref(false);
const editorRef = shallowRef<IDomEditor>();
const syncingModelValue = ref(false);
let syncGeneration = 0;
const toolbarConfig: Partial<IToolbarConfig> = {};
const editorHeightStyle = computed(() => ({
  minHeight: `${props.minHeight}px`,
  maxHeight: '640px',
  overflowY: 'auto',
}));
const editorConfig: Partial<IEditorConfig> = {
  placeholder: props.placeholder,
  MENU_CONF: {
    uploadImage: {
      allowedFileTypes: ['image/*'],
      async customUpload(file: File, insertFn: EditorImageInsertFn) {
        await handleEditorImageUpload(file, insertFn);
      },
    },
  },
  customAlert(info, type) {
    const text = String(info);
    if (type === 'success') {
      Message.success(text);
      return;
    }
    if (type === 'warning') {
      Message.warning(text);
      return;
    }
    if (type === 'error') {
      Message.error(text);
      return;
    }
    Message.info(text);
  },
};

function syncEditorHtml(value: string) {
  const editor = editorRef.value;
  if (!editor || editor.isDestroyed) {
    return;
  }

  const nextValue = String(value || '');
  if (editor.getHtml() === nextValue) {
    return;
  }

  syncGeneration++;
  const currentGen = syncGeneration;
  syncingModelValue.value = true;
  editor.setHtml(nextValue);
  nextTick(() => {
    if (syncGeneration === currentGen) {
      syncingModelValue.value = false;
    }
  });
}

function handleEditorCreated(editor: IDomEditor) {
  editorRef.value = editor;
  syncEditorHtml(props.modelValue);
}

function handleEditorModelValueUpdate(value: string) {
  if (syncingModelValue.value) {
    return;
  }
  emit('update:modelValue', value);
}

async function handleEditorImageUpload(file: File, insertFn: EditorImageInsertFn) {
  imageUploading.value = true;
  try {
    const dataUrl = await readFileAsDataUrl(file);
    insertFn(dataUrl, file.name, '');
    Message.success('图片已插入，点击保存后才会上传到服务端');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '读取图片失败');
  } finally {
    imageUploading.value = false;
  }
}

function getCurrentHtml() {
  const editor = editorRef.value;
  if (editor && !editor.isDestroyed) {
    return editor.getHtml();
  }
  return String(props.modelValue || '');
}

defineExpose({
  getCurrentHtml,
});

watch(
  () => props.modelValue,
  (value) => {
    syncEditorHtml(value);
  },
);

onBeforeUnmount(() => {
  if (editorRef.value && !editorRef.value.isDestroyed) {
    editorRef.value.destroy();
  }
});
</script>

<template>
  <a-form-item :label="label" :required="required">
    <div v-if="hint" class="rich-editor__hint">{{ hint }}</div>
    <div class="rich-editor__frame">
      <Toolbar
        :editor="editorRef"
        :default-config="toolbarConfig"
        mode="default"
        class="rich-editor__toolbar"
      />
      <Editor
        :model-value="modelValue"
        :default-config="editorConfig"
        mode="default"
        :style="editorHeightStyle"
        @update:modelValue="handleEditorModelValueUpdate"
        @onCreated="handleEditorCreated"
      />
    </div>

  </a-form-item>
</template>

<style scoped>
.rich-editor__hint {
  margin-bottom: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f7f8fa;
  color: #4e5969;
  font-size: 12px;
  line-height: 1.6;
}

.rich-editor__frame {
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  background: #fff;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.rich-editor__toolbar {
  border-bottom: 1px solid var(--color-border-2);
  background: rgba(255, 255, 255, 0.96);
  position: sticky;
  top: 0;
  z-index: 2;
}

</style>
