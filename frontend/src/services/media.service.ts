import type { TenantItem } from '../types/site';
import type { MediaItem, MediaUploadPayload } from '../types/media';
import { UploadDesktopMedia } from '../../wailsjs/go/main/App';
import { desktopApiRequest } from './desktop-api.service';
import { hasAppMethod } from './bridge';
import { useAuthStore } from '../stores/auth.store';

const DEFAULT_MAX_UPLOAD_SIZE_MB = 10;
const ALLOWED_IMAGE_MIME_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
const ALLOWED_IMAGE_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif', '.webp'];

interface RemoteMediaItem {
  media_item_id: number;
  disk: string;
  path: string;
  url: string;
  mime: string;
  size: number;
  width?: number | null;
  height?: number | null;
  source_url?: string | null;
  draft_id?: number | null;
}

function mapMediaItem(payload: RemoteMediaItem): MediaItem {
  return {
    mediaItemId: payload.media_item_id,
    disk: payload.disk,
    path: payload.path,
    url: payload.url,
    mime: payload.mime,
    size: payload.size,
    width: payload.width,
    height: payload.height,
    sourceUrl: payload.source_url,
    draftId: payload.draft_id,
  };
}

function resolveUploadLimitMb(): number {
  const authStore = useAuthStore();
  return authStore.bootstrap?.limits.maxUploadSizeMb || DEFAULT_MAX_UPLOAD_SIZE_MB;
}

function ensureMediaUploadAllowed() {
  const authStore = useAuthStore();
  if (authStore.bootstrap && !authStore.bootstrap.capabilities.canUploadMedia) {
    throw new Error('当前账号没有媒体上传权限，请联系管理员开通');
  }
}

export function validateDesktopUploadFile(file: File, maxUploadSizeMb = DEFAULT_MAX_UPLOAD_SIZE_MB) {
  const mime = String(file.type || '').toLowerCase();
  const lowerName = file.name.toLowerCase();
  const hasAllowedExtension = ALLOWED_IMAGE_EXTENSIONS.some((ext) => lowerName.endsWith(ext));
  if (!ALLOWED_IMAGE_MIME_TYPES.includes(mime) || !hasAllowedExtension) {
    throw new Error('仅支持 JPG、PNG、GIF、WebP 图片');
  }

  const maxBytes = Math.max(1, maxUploadSizeMb) * 1024 * 1024;
  if (file.size > maxBytes) {
    throw new Error(`图片大小不能超过 ${maxUploadSizeMb}MB`);
  }
}

export async function uploadMedia(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  payload: MediaUploadPayload,
): Promise<MediaItem> {
  ensureMediaUploadAllowed();
  validateDesktopUploadFile(payload.file, resolveUploadLimitMb());

  if (hasAppMethod('UploadDesktopMedia')) {
    const bytes = new Uint8Array(await payload.file.arrayBuffer());
    let binary = '';
    for (const byte of bytes) {
      binary += String.fromCharCode(byte);
    }

    const response = await UploadDesktopMedia({
      apiBaseUrl: tenant.apiBaseUrl,
      accessToken,
      fileName: payload.file.name,
      mimeType: payload.file.type || 'application/octet-stream',
      fileBase64: btoa(binary),
      originalName: payload.originalName || '',
      mediaCategoryId: payload.mediaCategoryId || 0,
      sourceUrl: payload.sourceUrl || '',
      draftId: payload.draftId || 0,
      uploadScene: payload.uploadScene || '',
    }) as { data: RemoteMediaItem; success: boolean; message: string };

    if (!response?.success) {
      throw new Error(response?.message || '上传失败');
    }

    return mapMediaItem(response.data);
  }

  const formData = new FormData();
  formData.append('file', payload.file);

  if (payload.originalName) {
    formData.append('original_name', payload.originalName);
  }
  if (payload.mediaCategoryId) {
    formData.append('media_category_id', String(payload.mediaCategoryId));
  }
  if (payload.sourceUrl) {
    formData.append('source_url', payload.sourceUrl);
  }
  if (payload.draftId) {
    formData.append('draft_id', String(payload.draftId));
  }

  const response = await desktopApiRequest<RemoteMediaItem>(
    tenant,
    '/media/upload',
    {
      method: 'POST',
      body: formData,
    },
    accessToken,
  );

  return mapMediaItem(response.data);
}
