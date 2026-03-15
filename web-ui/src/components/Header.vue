<template>
  <div class="header">
    <div class="header-title">{{ pageTitle }}</div>
    <div class="header-right">
      <el-button size="small" link class="lang-btn" @click="toggleLocale">
        {{ t('header.switchLang') }}
      </el-button>
      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-btn">
          <div class="user-avatar">{{ userInitial }}</div>
          <span class="user-name">{{ userStore.username }}</span>
          <el-icon style="font-size:12px;color:var(--cv-text-muted)"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="settings">
              <el-icon><Setting /></el-icon>{{ t('header.settings') }}
            </el-dropdown-item>
            <el-dropdown-item divided command="logout">
              <el-icon><SwitchButton /></el-icon>{{ t('header.logout') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import { ArrowDown, Setting, SwitchButton } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'

const { t, locale } = useI18n()
const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const pageTitleKeys: Record<string, string> = {
  '/': 'nav.dashboard',
  '/proxies': 'nav.proxies',
  '/profiles': 'nav.profiles',
  '/connections': 'nav.connections',
  '/rules': 'nav.rules',
  '/logs': 'nav.logs',
  '/mihomo': 'nav.mihomo',
  '/settings': 'nav.settings',
}

const pageTitle = computed(() => {
  const key = pageTitleKeys[route.path]
  return key ? t(key) : 'Proxyd'
})

const userInitial = computed(() =>
  (userStore.username?.[0] ?? 'U').toUpperCase()
)

const toggleLocale = () => {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}

const handleCommand = async (command: string) => {
  if (command === 'settings') router.push('/settings')
  if (command === 'logout') {
    await userStore.logout()
    router.push('/login')
    ElMessage.success(t('header.loggedOut'))
  }
}
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  height: 52px;
  background: var(--cv-sidebar);
  border-bottom: 1px solid var(--cv-border);
  flex-shrink: 0;
}

.header-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--cv-text);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.lang-btn {
  font-size: 12px;
  font-weight: 600;
  color: var(--cv-text-muted);
  padding: 4px 8px;
  border-radius: var(--cv-radius-sm);
  transition: color 0.15s, background 0.15s;
}

.lang-btn:hover {
  color: var(--cv-accent);
  background: var(--cv-accent-soft);
}

.user-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border-radius: var(--cv-radius-sm);
  cursor: pointer;
  transition: background 0.15s;
}

.user-btn:hover {
  background: rgba(255,255,255,0.06);
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--cv-accent);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.user-name {
  font-size: 13px;
  color: var(--cv-text);
}
</style>
