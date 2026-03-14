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
      { path: '',           name: 'Dashboard',   component: () => import('@/views/DashboardView.vue') },
      { path: 'proxies',    name: 'Proxies',     component: () => import('@/views/ProxiesView.vue') },
      { path: 'profiles',   name: 'Profiles',    component: () => import('@/views/ProfilesView.vue') },
      { path: 'connections',name: 'Connections', component: () => import('@/views/ConnectionsView.vue') },
      { path: 'rules',      name: 'Rules',       component: () => import('@/views/RulesView.vue') },
      { path: 'logs',       name: 'Logs',        component: () => import('@/views/LogsView.vue') },
      { path: 'mihomo',     name: 'Mihomo',      component: () => import('@/views/MihomoView.vue') },
      { path: 'settings',   name: 'Settings',    component: () => import('@/views/SettingsView.vue') },
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()
  const requiresAuth = to.meta.requiresAuth !== false
  if (requiresAuth && !userStore.isLoggedIn) next('/login')
  else if (to.path === '/login' && userStore.isLoggedIn) next('/')
  else next()
})

export default router
