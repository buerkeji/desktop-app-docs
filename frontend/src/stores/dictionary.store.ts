import { defineStore } from 'pinia';
import type { DictItem, TagItem } from '../types/dictionary';
import type { TenantItem } from '../types/site';
import { getDictionary } from '../services/dictionary.service';

let pendingInitialise:
  | {
      tenantId: number | null;
      promise: Promise<void>;
    }
  | null = null;

interface DictionaryState {
  categories: DictItem[];
  tags: TagItem[];
  loaded: boolean;
  currentTenantId: number | null;
}

export const useDictionaryStore = defineStore('dictionary', {
  state: (): DictionaryState => ({
    categories: [],
    tags: [],
    loaded: false,
    currentTenantId: null,
  }),
  actions: {
    async initialise(
      tenant?: (Pick<TenantItem, 'apiBaseUrl'> & Partial<Pick<TenantItem, 'id'>>) | null,
      accessToken?: string | null,
      force = false,
    ) {
      const tenantId = tenant?.id ?? null;
      if (this.loaded && !force && (tenant?.id === undefined || this.currentTenantId === tenant.id)) {
        return;
      }

      if (pendingInitialise && pendingInitialise.tenantId === tenantId) {
        return pendingInitialise.promise;
      }

      const task = (async () => {
        const result = await getDictionary(tenant, accessToken);
        this.categories = result.categories;
        this.tags = result.tags;
        this.loaded = true;
        this.currentTenantId = tenantId;
      })();

      pendingInitialise = {
        tenantId,
        promise: task,
      };

      try {
        await task;
      } finally {
        if (pendingInitialise?.promise === task) {
          pendingInitialise = null;
        }
      }
    },
  },
});
