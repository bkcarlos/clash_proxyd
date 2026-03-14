import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/DashboardView.vue')
      },
      {
        path: 'sources',
        name: 'Sources',
        component: () => import('@/views/SourcesView.vue')
      },
      {
        path: 'config',
        name: 'Config',
        component: () => import('@/views/ConfigView.vue')
      },
      {
        path: 'proxies',
        name: 'Proxies',
        component: () => import('@/views/ProxiesView.vue')
      },
      {
        path: 'mihomo',
        name: 'Mihomo',
        component: () => import('@/views/MihomoView.vue')
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/views/LogsView.vue')
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/SettingsView.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard for authentication
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()
  const requiresAuth = to.meta.requiresAuth !== false

  if (requiresAuth && !userStore.isLoggedIn) {
    next('/login')
  } else if (to.path === '/login' && userStore.isLoggedIn) {
    next('/')
  } else {
    next()
  }
})

export default router
