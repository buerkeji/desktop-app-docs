import type { DictItem, TagItem } from '../types/dictionary';
import type { TenantItem } from '../types/site';
import { desktopApiRequest } from './desktop-api.service';

const categories: DictItem[] = [];

const tags: TagItem[] = [];

interface RemoteCategoryItem {
  id: number;
  name: string;
  slug: string;
  parent_id: number;
  sort_order: number;
  is_active: boolean;
}

interface RemoteTagItem {
  id: number;
  name: string;
  slug: string;
}

function isFulfilledResult<T>(
  result: PromiseSettledResult<T>,
): result is PromiseFulfilledResult<T> {
  return result.status === 'fulfilled';
}

function mapCategories(type: DictItem['type'], items: RemoteCategoryItem[]): DictItem[] {
  return (items ?? []).map((item) => ({
    id: item.id,
    type,
    label: item.name,
    value: item.slug,
    sort: item.sort_order,
    enabled: item.is_active,
  }));
}

export async function getDictionary(
  tenant?: (Pick<TenantItem, 'apiBaseUrl'> & Partial<Pick<TenantItem, 'id'>>) | null,
  accessToken?: string | null,
) {
  if (!tenant?.apiBaseUrl || !accessToken) {
    return {
      categories,
      tags,
    };
  }

  const [toolCategories, articleCategories, mediaCategories, remoteTags] = await Promise.allSettled([
    desktopApiRequest<RemoteCategoryItem[]>(tenant, '/categories', {}, accessToken),
    desktopApiRequest<RemoteCategoryItem[]>(tenant, '/article-categories', {}, accessToken),
    desktopApiRequest<RemoteCategoryItem[]>(tenant, '/media-categories', {}, accessToken),
    desktopApiRequest<RemoteTagItem[]>(tenant, '/tags', {}, accessToken),
  ]);

  return {
    categories: [
      ...(
        isFulfilledResult(toolCategories)
          ? mapCategories('tool_category', toolCategories.value.data)
          : categories.filter((item) => item.type === 'tool_category')
      ),
      ...(
        isFulfilledResult(articleCategories)
          ? mapCategories('article_category', articleCategories.value.data)
          : categories.filter((item) => item.type === 'article_category')
      ),
      ...(
        isFulfilledResult(mediaCategories)
          ? mapCategories('media_category', mediaCategories.value.data)
          : categories.filter((item) => item.type === 'media_category')
      ),
    ],
    tags: isFulfilledResult(remoteTags)
      ? (remoteTags.value.data ?? []).map((item) => ({
        id: item.id,
        name: item.name,
        slug: item.slug,
      }))
      : tags,
  };
}
