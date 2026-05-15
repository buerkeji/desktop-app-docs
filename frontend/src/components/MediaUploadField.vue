<script setup lang="ts">
import { computed, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import type { TenantItem } from '../types/site';
import { isDataUrl, readFileAsDataUrl } from '../services/deferred-media.service';
import { validateDesktopUploadFile } from '../services/media.service';
import { useAuthStore } from '../stores/auth.store';
import { resolveTenantAssetUrl } from '../utils/url';

const props = defineProps<{
  modelValue: string;
  tenant?: Pick<TenantItem, 'apiBaseUrl'> | null;
  accessToken?: string | null;
  mediaCategoryId?: number;
  uploadScene?: string;
  placeholder?: string;
  buttonText?: string;
  previewAlt?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const authStore = useAuthStore();
const fileInputRef = ref<HTMLInputElement | null>(null);
const uploading = ref(false);
const uploadAllowed = computed(() => authStore.bootstrap?.capabilities.canUploadMedia !== false);
const uploadLimitMb = computed(() => authStore.bootstrap?.limits.maxUploadSizeMb || 10);

const resolvedPreviewUrl = computed(() => {
  if (isDataUrl(props.modelValue)) {
    return props.modelValue;
  }
  return resolveTenantAssetUrl(props.modelValue || '', props.tenant?.apiBaseUrl);
});

function handleValueChange(value: string | number | undefined) {
  emit('update:modelValue', String(value || ''));
}

function openFilePicker() {
  if (!uploadAllowed.value) {
    Message.error('当前账号没有媒体上传权限');
    return;
  }

  fileInputRef.value?.click();
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }

  uploading.value = true;
  try {
    validateDesktopUploadFile(file, uploadLimitMb.value);
    const dataUrl = await readFileAsDataUrl(file);
    emit('update:modelValue', dataUrl);
    Message.success('图片已加入待上传，保存后才会提交到服务端');
  } catch (error) {
    Message.error(error instanceof Error ? error.message : '读取图片失败');
  } finally {
    uploading.value = false;
    input.value = '';
  }
}
</script>

<template>
  <a-space direction="vertical" fill>
    <a-input
      :model-value="modelValue"
      :placeholder="placeholder || '请输入图片地址或点击上传'"
      @update:model-value="handleValueChange"
    />
    <a-space wrap>
      <a-button :loading="uploading" :disabled="!uploadAllowed" @click="openFilePicker">
        {{ buttonText || '上传图片' }}
      </a-button>
      <a-button :disabled="!modelValue" @click="emit('update:modelValue', '')">清空</a-button>
    </a-space>
    <a-image
      v-if="resolvedPreviewUrl"
      :src="resolvedPreviewUrl"
      :alt="previewAlt || 'media-preview'"
      width="100%"
      :preview="true"
    />
    <input
      ref="fileInputRef"
      type="file"
      accept=".jpg,.jpeg,.png,.gif,.webp,image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleFileChange"
    />
  </a-space>
</template>
