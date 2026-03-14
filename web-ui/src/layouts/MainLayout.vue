<template>
  <div class="main-layout">
    <Sidebar />
    <div class="layout-body">
      <Header />
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import Sidebar from '@/components/Sidebar.vue'
</script>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  background: var(--cv-bg);
  overflow: hidden;
}

.layout-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content {
  flex: 1;
  padding: 20px 24px;
  overflow-y: auto;
  background: var(--cv-bg);
}
</style>
