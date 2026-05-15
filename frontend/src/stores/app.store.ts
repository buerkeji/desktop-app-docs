import { defineStore } from 'pinia';

export const useAppStore = defineStore('app', {
  state: () => ({
    title: 'ZQ 内容管理桌面端',
    currentMenu: '/dashboard',
  }),
  actions: {
    setCurrentMenu(value: string) {
      this.currentMenu = value;
    },
  },
});
