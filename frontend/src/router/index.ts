import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/pages/HomePage.vue') },
    { path: '/admin', name: 'admin', component: () => import('@/pages/AdminPage.vue') },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

export default router
