export interface MediaUploadPayload {
  file: File;
  originalName?: string;
  mediaCategoryId?: number;
  sourceUrl?: string;
  draftId?: number;
  uploadScene?: string;
}

export interface MediaItem {
  mediaItemId: number;
  disk: string;
  path: string;
  url: string;
  mime: string;
  size: number;
  width?: number | null;
  height?: number | null;
  sourceUrl?: string | null;
  draftId?: number | null;
}
