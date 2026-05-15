import type { Id } from './common';

export interface DictItem {
  id: Id;
  type: 'tool_category' | 'article_category' | 'media_category';
  label: string;
  value: string;
  sort: number;
  enabled: boolean;
}

export interface TagItem {
  id: Id;
  name: string;
  slug: string;
  color?: string;
}
