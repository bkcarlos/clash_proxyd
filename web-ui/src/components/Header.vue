<template>
  <div class="header">
    <div class="header-title">{{ pageTitle }}</div>
    <div class="header-right">
      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-btn">
          <div class="user-avatar">{{ userInitial }}</div>
          <span class="user-name">{{ userStore.username }}</span>
          <el-icon style="font-size:12px;color:var(--cv-text-muted)"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="settings">
              <el-icon><Setting /></el-icon>Settings
            </el-dropdown-item>
            <el-dropdown-item divided command="logout">
              <el-icon><SwitchButton /></el-icon>Logout
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

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/proxies': 'Proxies',
  '/profiles': 'Profiles',
  '/connections': 'Connections',
  '/rules': 'Rules',
  '/logs': 'Logs',
  '/mihomo': 'Mihomo',
  '/settings': 'Settings',
}

const pageTitle = computed(() => pageTitles[route.path] ?? 'Proxyd')

const userInitial = computed(() =>
  (userStore.username?.[0] ?? 'U').toUpperCase()
)

const handleCommand = async (command: string) => {
  if (command === 'settings') router.push('/settings')
  if (command === 'logout') {
    await userStore.logout()
    router.push('/login')
    ElMessage.success('Logged out')
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
