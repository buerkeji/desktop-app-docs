<script setup lang="ts">
import { watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from './stores/auth.store';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

watch(
  () => authStore.isLoggedIn,
  async (isLoggedIn) => {
    if (!isLoggedIn && route.path !== '/login') {
      await router.replace('/login');
    }
  },
);
</script>

<template>
  <RouterView />
</template>
