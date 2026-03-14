<template>
  <div class="sidebar">
    <el-menu
      :default-active="activeMenu"
      class="sidebar-menu"
      @select="handleSelect"
    >
      <el-menu-item index="/">
        <el-icon><Odometer /></el-icon>
        <span>Dashboard</span>
      </el-menu-item>
      <el-menu-item index="/sources">
        <el-icon><Download /></el-icon>
        <span>Sources</span>
      </el-menu-item>
      <el-menu-item index="/config">
        <el-icon><Document /></el-icon>
        <span>Config</span>
      </el-menu-item>
      <el-menu-item index="/proxies">
        <el-icon><Connection /></el-icon>
        <span>Proxies</span>
      </el-menu-item>
      <el-menu-item index="/mihomo">
        <el-icon><Monitor /></el-icon>
        <span>Mihomo</span>
      </el-menu-item>
      <el-menu-item index="/logs">
        <el-icon><Tickets /></el-icon>
        <span>Logs</span>
      </el-menu-item>
      <el-menu-item index="/settings">
        <el-icon><Setting /></el-icon>
        <span>Settings</span>
      </el-menu-item>
    </el-menu>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Odometer, Download, Document, Connection, Setting, Monitor, Tickets } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

const activeMenu = computed(() => route.path)

const handleSelect = (key: string) => {
  router.push(key)
}
</script>

<style scoped>
.sidebar {
  width: 200px;
  height: calc(100vh - 60px);
  background: #fff;
  border-right: 1px solid #e4e7ed;
  overflow-y: auto;
}

.sidebar-menu {
  border: none;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 200px;
}
</style>
