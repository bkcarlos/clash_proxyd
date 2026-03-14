<template>
  <div class="proxies-view">
    <div class="page-header">
      <h1>Proxies</h1>
      <div>
        <el-button @click="refreshProxies">
          <el-icon><Refresh /></el-icon>
          Refresh
        </el-button>
        <el-button type="primary" @click="showMihomoDialog">
          <el-icon><Setting /></el-icon>
          Mihomo Control
        </el-button>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>Proxy Groups</span>
          </template>
          <el-table :data="proxyStore.groups" border stripe>
            <el-table-column prop="name" label="Name" />
            <el-table-column prop="type" label="Type" width="120" />
            <el-table-column prop="now" label="Current" width="160" />
            <el-table-column label="Actions" width="120">
              <template #default="{ row }">
                <el-button size="small" @click="showGroupDetail(row)">Detail</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <span>Traffic</span>
          </template>
          <div class="traffic-info">
            <div class="traffic-item">
              <el-icon class="traffic-icon upload"><Top /></el-icon>
              <div>
                <p class="traffic-label">Upload</p>
                <p class="traffic-value">{{ formatBytes(proxyStore.traffic.up) }}</p>
              </div>
            </div>
            <div class="traffic-item">
              <el-icon class="traffic-icon download"><Bottom /></el-icon>
              <div>
                <p class="traffic-label">Download</p>
                <p class="traffic-value">{{ formatBytes(proxyStore.traffic.down) }}</p>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="groupDialogVisible" :title="selectedGroup?.name" width="700px">
      <el-table :data="selectedGroup?.proxies || []" border stripe>
        <el-table-column prop="name" label="Proxy" />
        <el-table-column label="Delay" width="100">
          <template #default="{ row }">
            {{ row.delay ? row.delay + 'ms' : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="testProxy(row)">Test</el-button>
            <el-button size="small" type="primary" @click="switchProxy(row)">Switch</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="groupDialogVisible = false">Close</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="mihomoDialogVisible" title="Mihomo Control" width="400px">
      <div class="mihomo-controls">
        <el-button type="success" @click="controlMihomo('start')">Start</el-button>
        <el-button type="warning" @click="controlMihomo('restart')">Restart</el-button>
        <el-button type="danger" @click="controlMihomo('stop')">Stop</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useProxyStore } from '@/stores/proxy'
import { ElMessage } from 'element-plus'
import { Refresh, Setting, Top, Bottom } from '@element-plus/icons-vue'

const proxyStore = useProxyStore()

const groupDialogVisible = ref(false)
const mihomoDialogVisible = ref(false)
const selectedGroup = ref<any>(null)

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

const refreshProxies = async () => {
  try {
    await Promise.all([
      proxyStore.fetchProxies(true),
      proxyStore.fetchTraffic(true)
    ])
    ElMessage.success('Proxies refreshed')
  } catch (error: any) {
    ElMessage.error(error.message || 'Refresh failed')
  }
}

const showGroupDetail = (group: any) => {
  selectedGroup.value = group
  groupDialogVisible.value = true
}

const testProxy = async (item: any) => {
  try {
    const result = await proxyStore.testProxy(item.name)
    item.delay = result.delay
    const suffix = result.from_cache ? ' (cached)' : ''
    ElMessage.success(`Proxy ${item.name} delay: ${result.delay}ms${suffix}`)
  } catch (error: any) {
    ElMessage.error(error.message || 'Test failed')
  }
}

const switchProxy = async (item: any) => {
  if (!selectedGroup.value?.name) return

  try {
    await proxyStore.switchProxy(selectedGroup.value.name, item.name)
    selectedGroup.value.now = item.name
    ElMessage.success(`Switched ${selectedGroup.value.name} to ${item.name}`)
    await proxyStore.fetchProxies(true)
  } catch (error: any) {
    ElMessage.error(error.message || 'Switch failed')
  }
}

const showMihomoDialog = () => {
  mihomoDialogVisible.value = true
}

const controlMihomo = async (action: 'start' | 'stop' | 'restart') => {
  try {
    await proxyStore.controlMihomo(action)
    ElMessage.success(`Mihomo ${action} successful`)
    mihomoDialogVisible.value = false
    await refreshProxies()
  } catch (error: any) {
    ElMessage.error(error.message || 'Operation failed')
  }
}

onMounted(() => {
  proxyStore.fetchProxies(true)
  proxyStore.fetchTraffic(true)
})
</script>

<style scoped>
.proxies-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.traffic-info {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 15px;
}

.traffic-icon {
  font-size: 32px;
}

.traffic-icon.upload {
  color: #67c23a;
}

.traffic-icon.download {
  color: #409eff;
}

.traffic-label {
  margin: 0 0 5px 0;
  font-size: 14px;
  color: #909399;
}

.traffic-value {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.mihomo-controls {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mihomo-controls button {
  width: 100%;
}
</style>
