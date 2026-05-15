import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ArcoVue from '@arco-design/web-vue';
import '@arco-design/web-vue/dist/arco.css';
import App from './App.vue';
import router from './router';
import { initialiseApp, setupAuthRecoveryHandlers } from './services/bootstrap.service';
import { useAuthStore } from './stores/auth.store';
import { useDictionaryStore } from './stores/dictionary.store';
import { useSiteStore } from './stores/site.store';
import { initialiseDesktopContext } from './utils/desktop-context';
import './style.css';

async function bootstrap() {
  const app = createApp(App);
  const pinia = createPinia();

  app.use(pinia);
  app.use(ArcoVue);

  const siteStore = useSiteStore(pinia);
  const authStore = useAuthStore(pinia);
  const dictionaryStore = useDictionaryStore(pinia);

  setupAuthRecoveryHandlers(authStore, router);

  // Phase 1: Quick local init (DB reads, no network).
  await initialiseDesktopContext(siteStore, authStore);

  // Phase 2: Mount UI immediately so the user sees something.
  app.use(router);
  app.mount('#app');

  // Phase 3: Deferred network init (bootstrap fetch, dictionary sync).
  if (authStore.token?.accessToken && siteStore.currentTenant?.apiBaseUrl) {
    initialiseApp(siteStore, authStore, dictionaryStore).catch(() => {
      // Background failures are non-blocking; user can retry via navigation.
    });
  }
}

void bootstrap();
