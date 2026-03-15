<template>
  <div class="sidebar">
    <div class="sidebar-logo">
      <div class="logo-icon">P</div>
      <span class="logo-text">Proxyd</span>
    </div>

    <nav class="nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ active: isActive(item.path) }"
      >
        <el-icon class="nav-icon"><component :is="item.icon" /></el-icon>
        <span class="nav-label">{{ t(item.labelKey) }}</span>
      </router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Odometer, Connection, Files,
  Share, List, Tickets, Monitor, Setting
} from '@element-plus/icons-vue'

const { t } = useI18n()
const route = useRoute()

const navItems = [
  { path: '/',           labelKey: 'nav.dashboard',   icon: Odometer   },
  { path: '/proxies',    labelKey: 'nav.proxies',      icon: Connection },
  { path: '/profiles',   labelKey: 'nav.profiles',     icon: Files      },
  { path: '/connections',labelKey: 'nav.connections',  icon: Share      },
  { path: '/rules',      labelKey: 'nav.rules',        icon: List       },
  { path: '/logs',       labelKey: 'nav.logs',         icon: Tickets    },
  { path: '/mihomo',     labelKey: 'nav.mihomo',       icon: Monitor    },
  { path: '/settings',   labelKey: 'nav.settings',     icon: Setting    },
]

const isActive = (path: string) => {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}
</script>

<style scoped>
.sidebar {
  width: 200px;
  min-width: 200px;
  height: 100vh;
  background: var(--cv-sidebar);
  border-right: 1px solid var(--cv-border);
  display: flex;
  flex-direction: column;
  padding: 0 10px 20px;
  overflow: hidden;
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 10px 16px;
  border-bottom: 1px solid var(--cv-border);
  margin-bottom: 10px;
}

.logo-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--cv-accent);
  color: #fff;
  font-size: 14px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 12px rgba(88,101,242,0.4);
}

.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: var(--cv-text);
  letter-spacing: 0.5px;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--cv-radius-sm);
  color: var(--cv-text-muted);
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.nav-item:hover {
  background: rgba(255,255,255,0.05);
  color: var(--cv-text);
}

.nav-item.active {
  background: var(--cv-accent-soft);
  color: var(--cv-accent);
}

.nav-icon { font-size: 16px; flex-shrink: 0; }
.nav-label { white-space: nowrap; }
</style>
