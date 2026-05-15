import type { TenantItem } from '../types/site';
import type { MediaItem } from '../types/media';
import { uploadMedia, validateDesktopUploadFile } from './media.service';
import { DownloadRemoteMedia } from '../../wailsjs/go/main/App';

const DATA_URL_PREFIX = 'data:';
const HTTP_URL_PREFIX = 'http';

function normaliseMimeExtension(mime: string): string {
  switch (mime.toLowerCase()) {
    case 'image/jpeg':
      return 'jpg';
    case 'image/png':
      return 'png';
    case 'image/gif':
      return 'gif';
    case 'image/webp':
      return 'webp';
    case 'image/svg+xml':
      return 'svg';
    case 'image/x-icon':
    case 'image/vnd.microsoft.icon':
      return 'ico';
    default:
      return 'bin';
  }
}

function decodeBase64(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function buildUploadFileName(prefix: string, mime: string, index?: number): string {
  const ext = normaliseMimeExtension(mime);
  const suffix = typeof index === 'number' ? `-${String(index).padStart(2, '0')}` : '';
  return `${prefix}${suffix}.${ext}`;
}

export interface DeferredUploadOptions {
  mediaCategoryId?: number;
  uploadScene?: string;
  fileNamePrefix?: string;
  index?: number;
}

export function isDataUrl(value: string | null | undefined): boolean {
  return typeof value === 'string' && value.trim().startsWith(DATA_URL_PREFIX);
}

function isHttpUrl(value: string): boolean {
  return value.startsWith(HTTP_URL_PREFIX) && value.includes('://');
}

function isSameOriginAsApi(url: string, apiBaseUrl: string): boolean {
  try {
    const urlOrigin = new URL(url).origin;
    const apiOrigin = new URL(apiBaseUrl).origin;
    return urlOrigin === apiOrigin;
  } catch {
    return false;
  }
}

export async function downloadRemoteMedia(url: string, referer?: string): Promise<{ fileBase64: string; mimeType: string; fileName: string }> {
  const origin = referer || (() => { try { return new URL(url).origin + '/'; } catch { return ''; } })();
  return DownloadRemoteMedia({ url, referer: origin });
}

export async function uploadRemoteMedia(
  url: string,
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  options: DeferredUploadOptions & { referer?: string } = {},
): Promise<MediaItem | null> {
  const trimmed = url.trim();
  if (!trimmed || isDataUrl(trimmed)) return null;

  let normalised = trimmed;
  if (normalised.startsWith('//')) {
    normalised = `https:${normalised}`;
  }
  if (!isHttpUrl(normalised)) return null;

  if (isSameOriginAsApi(normalised, tenant.apiBaseUrl)) {
    return null;
  }

  try {
    const remote = await downloadRemoteMedia(normalised, options.referer);
    const dataUrl = `data:${remote.mimeType};base64,${remote.fileBase64}`;
    const file = dataUrlToFile(dataUrl, remote.fileName);
    validateDesktopUploadFile(file);

    return uploadMedia(tenant, accessToken, {
      file,
      originalName: remote.fileName,
      mediaCategoryId: options.mediaCategoryId,
      uploadScene: options.uploadScene,
    });
  } catch {
    return null;
  }
}

export async function uploadRemoteHtmlImages(
  html: string,
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  options: DeferredUploadOptions & { referer?: string } = {},
): Promise<{ html: string; uploaded: MediaItem[]; failedCount: number }> {
  const container = document.createElement('div');
  container.innerHTML = html;

  const uploaded: MediaItem[] = [];
  const cache = new Map<string, MediaItem>();
  const attrs = ['src', 'data-src', 'data-original', 'data-lazy-src'];
  let uploadIndex = 1;
  let failedCount = 0;

  function getImageReferer(imageUrl: string): string {
    if (options.referer) return options.referer;
    try {
      return new URL(imageUrl).origin + '/';
    } catch {
      return '';
    }
  }

  async function processUrl(value: string, setUrl: (url: string) => void) {
    let normalised = value.trim();
    if (normalised.startsWith('//')) {
      normalised = `https:${normalised}`;
    }
    if (!isHttpUrl(normalised) || isDataUrl(normalised)) {
      return;
    }

    if (isSameOriginAsApi(normalised, tenant.apiBaseUrl)) {
      return;
    }

    let item = cache.get(normalised);
    if (!item) {
      try {
        const remote = await DownloadRemoteMedia({ url: normalised, referer: getImageReferer(normalised) });
        const dataUrl = `data:${remote.mimeType};base64,${remote.fileBase64}`;
        const file = dataUrlToFile(dataUrl, remote.fileName);
        validateDesktopUploadFile(file);

        item = await uploadMedia(tenant, accessToken, {
          file,
          originalName: remote.fileName,
          mediaCategoryId: options.mediaCategoryId,
          uploadScene: options.uploadScene,
        });
        cache.set(normalised, item);
        uploaded.push(item);
        uploadIndex += 1;
      } catch {
        failedCount += 1;
        return;
      }
    }

    setUrl(item.url);
  }

  for (const image of Array.from(container.querySelectorAll('img'))) {
    for (const attr of attrs) {
      const value = image.getAttribute(attr)?.trim() || '';
      if (!value) continue;
      await processUrl(value, (url) => image.setAttribute(attr, url));
    }

    const srcset = image.getAttribute('srcset') || image.getAttribute('data-srcset');
    if (srcset) {
      const candidates = srcset.split(',').map((c) => c.trim()).filter(Boolean);
      const resolvedCandidates: string[] = [];
      for (const candidate of candidates) {
        const [urlPart, ...descriptors] = candidate.split(/\s+/);
        let resolvedUrl = urlPart;
        await processUrl(urlPart, (url) => { resolvedUrl = url; });
        resolvedCandidates.push(descriptors.length > 0 ? `${resolvedUrl} ${descriptors.join(' ')}` : resolvedUrl);
      }
      image.setAttribute('srcset', resolvedCandidates.join(', '));
    }
  }

  for (const source of Array.from(container.querySelectorAll('picture source'))) {
    for (const attr of attrs) {
      const value = source.getAttribute(attr)?.trim() || '';
      if (!value) continue;
      await processUrl(value, (url) => source.setAttribute(attr, url));
    }
  }

  const resultHtml = container.innerHTML;

  return {
    html: resultHtml,
    uploaded,
    failedCount,
  };
}

export function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(reader.error || new Error('读取文件失败'));
    reader.readAsDataURL(file);
  });
}

export function dataUrlToFile(dataUrl: string, fileName: string): File {
  const trimmed = dataUrl.trim();
  const match = /^data:([^;,]+);base64,(.+)$/i.exec(trimmed);
  if (!match) {
    throw new Error('图片数据格式无效');
  }

  const mime = match[1] || 'application/octet-stream';
  const bytes = decodeBase64(match[2]);
  const buffer = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(buffer).set(bytes);
  return new File([buffer], fileName, { type: mime });
}

export async function uploadDeferredDataUrl(
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  dataUrl: string,
  options: DeferredUploadOptions = {},
): Promise<MediaItem> {
  const trimmed = dataUrl.trim();
  const match = /^data:([^;,]+);base64,/i.exec(trimmed);
  if (!match) {
    throw new Error('图片数据格式无效');
  }

  const mime = match[1] || 'application/octet-stream';
  const basePrefix = options.fileNamePrefix?.trim() || 'media';
  const fileName = buildUploadFileName(basePrefix, mime, options.index);
  const file = dataUrlToFile(trimmed, fileName);
  validateDesktopUploadFile(file);

  return uploadMedia(tenant, accessToken, {
    file,
    originalName: file.name,
    mediaCategoryId: options.mediaCategoryId,
    uploadScene: options.uploadScene,
  });
}

export async function uploadDeferredHtmlImages(
  html: string,
  tenant: Pick<TenantItem, 'apiBaseUrl'>,
  accessToken: string,
  options: DeferredUploadOptions = {},
): Promise<{ html: string; uploaded: MediaItem[] }> {
  if (!html.includes(DATA_URL_PREFIX)) {
    return { html, uploaded: [] };
  }

  const container = document.createElement('div');
  container.innerHTML = html;

  const uploaded: MediaItem[] = [];
  const cache = new Map<string, MediaItem>();
  const attrs = ['src', 'data-src', 'data-original', 'data-lazy-src'];
  let uploadIndex = 1;

  for (const image of Array.from(container.querySelectorAll('img'))) {
    for (const attr of attrs) {
      const value = image.getAttribute(attr)?.trim() || '';
      if (!isDataUrl(value)) {
        continue;
      }

      let item = cache.get(value);
      if (!item) {
        item = await uploadDeferredDataUrl(tenant, accessToken, value, {
          ...options,
          index: uploadIndex,
        });
        cache.set(value, item);
        uploaded.push(item);
        uploadIndex += 1;
      }

      image.setAttribute(attr, item.url);
    }
  }

  return {
    html: container.innerHTML,
    uploaded,
  };
}
