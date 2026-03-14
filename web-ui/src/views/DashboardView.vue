<template>
  <div class="dashboard-view">
    <h1>Dashboard</h1>

    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#409eff"><Odometer /></el-icon>
            <div>
              <p class="stat-label">Uptime</p>
              <p class="stat-value">{{ formatUptime(systemStore.info?.uptime || 0) }}</p>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" :color="mihomoStatusColor"><CircleCheck /></el-icon>
            <div>
              <p class="stat-label">Mihomo Status</p>
              <p class="stat-value">{{ systemStore.info?.mihomo_status || 'Unknown' }}</p>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#e6a23c"><Download /></el-icon>
            <div>
              <p class="stat-label">Sources</p>
              <p class="stat-value">{{ sourceStore.sources.length }}</p>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#f56c6c"><Connection /></el-icon>
            <div>
              <p class="stat-label">Proxies</p>
              <p class="stat-value">{{ Object.keys(proxyStore.proxies).length }}</p>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Mihomo Control Panel -->
    <el-card class="control-card" style="margin-bottom: 20px;">
      <template #header>
        <div class="card-header">
          <span>Mihomo Control</span>
          <el-tag :type="mihomoStatusTagType" size="small">{{ systemStore.info?.mihomo_status || 'Unknown' }}</el-tag>
        </div>
      </template>
      <div class="control-row">
        <div class="version-info">
          <span class="version-label">Binary Version:</span>
          <el-tag type="info" size="small">{{ mihomoVersion || '—' }}</el-tag>
        </div>
        <div class="control-buttons">
          <el-button
            type="success"
            :loading="controlling === 'start'"
            :disabled="mihomoRunning || !!controlling"
            @click="doControl('start')"
          >Start</el-button>
          <el-button
            type="danger"
            :loading="controlling === 'stop'"
            :disabled="!mihomoRunning || !!controlling"
            @click="doControl('stop')"
          >Stop</el-button>
          <el-button
            type="warning"
            :loading="controlling === 'restart'"
            :disabled="!mihomoRunning || !!controlling"
            @click="doControl('restart')"
          >Restart</el-button>
          <el-divider direction="vertical" />
          <el-button
            :loading="checkingUpdate"
            :disabled="!!controlling || checkingUpdate"
            @click="doUpdate"
          >Check & Update</el-button>
          <el-button size="small" @click="fetchVersion" :loading="loadingVersion">Refresh Version</el-button>
        </div>
      </div>
    </el-card>

    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card class="info-card">
          <template #header>
            <div class="card-header">
              <span>System Information</span>
              <el-button size="small" @click="refreshSystem">Refresh</el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Version">
              {{ systemStore.info?.version || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="Go Version">
              {{ systemStore.info?.go_version || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="Database">
              {{ systemStore.info?.database || '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="auto-update-card">
            <div class="auto-update-header">
              <span>Last Auto Update</span>
              <el-tag v-if="systemStore.info?.last_auto_update_action" :type="autoUpdateTagType" size="small">
                {{ formatAutoUpdateAction(systemStore.info?.last_auto_update_action || '') }}
              </el-tag>
              <el-tag v-else type="info" size="small">No Record</el-tag>
            </div>
            <p class="auto-update-time">
              {{ formatDateTime(systemStore.info?.last_auto_update_at) }}
            </p>
            <p class="auto-update-details">
              {{ systemStore.info?.last_auto_update_details || 'No automatic update record yet.' }}
            </p>
          </div>

          <div class="auto-update-card" style="margin-top: 12px;">
            <div class="auto-update-header">
              <span>Last Alert</span>
              <el-tag v-if="systemStore.info?.last_alert_action" :type="alertTagType" size="small">
                {{ formatAlertAction(systemStore.info?.last_alert_action || '') }}
              </el-tag>
              <el-tag v-else type="info" size="small">No Record</el-tag>
            </div>
            <p class="auto-update-time">
              {{ formatDateTime(systemStore.info?.last_alert_at) }}
            </p>
            <p class="auto-update-details">
              {{ systemStore.info?.last_alert_details || 'No alert record yet.' }}
            </p>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="info-card">
          <template #header>
            <div class="card-header">
              <span>Traffic Statistics</span>
              <el-button size="small" @click="refreshTraffic">Refresh</el-button>
            </div>
          </template>
          <div class="traffic-stats">
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useSystemStore } from '@/stores/system'
import { useSourceStore } from '@/stores/source'
import { useProxyStore } from '@/stores/proxy'
import { controlMihomo, getMihomoVersion, updateMihomo } from '@/api/proxy'
import { Odometer, CircleCheck, Download, Connection, Top, Bottom } from '@element-plus/icons-vue'

const systemStore = useSystemStore()
const sourceStore = useSourceStore()
const proxyStore = useProxyStore()

const mihomoVersion = ref<string>('')
const loadingVersion = ref(false)
const controlling = ref<string>('')
const checkingUpdate = ref(false)

const mihomoRunning = computed(() => systemStore.info?.mihomo_status === 'running')

const mihomoStatusColor = computed(() => {
  const s = systemStore.info?.mihomo_status
  if (s === 'running') return '#67c23a'
  if (s === 'error') return '#f56c6c'
  return '#909399'
})

const mihomoStatusTagType = computed(() => {
  const s = systemStore.info?.mihomo_status
  if (s === 'running') return 'success'
  if (s === 'error') return 'danger'
  return 'info'
})

const autoUpdateTagType = computed(() => {
  const action = systemStore.info?.last_auto_update_action || ''
  if (action.includes('applied')) return 'success'
  if (action.includes('rolled_back')) return 'warning'
  if (action.includes('failed')) return 'danger'
  if (action.includes('skipped')) return 'info'
  return 'info'
})

const alertTagType = computed(() => {
  const action = systemStore.info?.last_alert_action || ''
  if (action.includes('abnormal') || action.includes('failed')) return 'danger'
  if (action.includes('recovered') || action.includes('ok')) return 'success'
  return 'warning'
})

const fetchVersion = async () => {
  loadingVersion.value = true
  try {
    const res = await getMihomoVersion()
    mihomoVersion.value = res.version
  } catch {
    mihomoVersion.value = 'N/A'
  } finally {
    loadingVersion.value = false
  }
}

const doControl = async (action: 'start' | 'stop' | 'restart') => {
  controlling.value = action
  try {
    await controlMihomo(action)
    ElMessage.success(`Mihomo ${action} successful`)
    await systemStore.fetchInfo()
  } catch (e: any) {
    ElMessage.error(`Failed to ${action} mihomo: ${e?.message || e}`)
  } finally {
    controlling.value = ''
  }
}

const doUpdate = async () => {
  checkingUpdate.value = true
  try {
    const res = await updateMihomo()
    if (res.updated) {
      ElMessage.success(`Updated: ${res.old_version} → ${res.new_version}`)
      await fetchVersion()
    } else {
      ElMessage.info(`Already up to date (${res.current_version})`)
    }
  } catch (e: any) {
    ElMessage.error(`Update failed: ${e?.message || e}`)
  } finally {
    checkingUpdate.value = false
  }
}

const formatAlertAction = (action: string): string => {
  if (!action) return 'Unknown'
  return action
    .replace('alert_', '')
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

const formatAutoUpdateAction = (action: string): string => {
  if (!action) return 'Unknown'
  return action
    .replace('mihomo_update_', '')
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

const formatDateTime = (value?: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  return `${hours}h ${minutes}m`
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

const refreshSystem = async () => {
  await systemStore.fetchInfo()
}

const refreshTraffic = async () => {
  await proxyStore.fetchTraffic()
}

onMounted(async () => {
  await Promise.allSettled([
    systemStore.fetchInfo(),
    systemStore.fetchStatus(),
    sourceStore.fetchSources(),
    proxyStore.fetchProxies(true),
    proxyStore.fetchTraffic(true),
    fetchVersion(),
  ])
  systemStore.connectWS()
})

onUnmounted(() => {
  systemStore.disconnectWS()
})
</script>

<style scoped>
.dashboard-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  height: 100px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 15px;
}

.stat-icon {
  font-size: 40px;
}

.stat-label {
  margin: 0 0 5px 0;
  font-size: 14px;
  color: #909399;
}

.stat-value {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.control-card .control-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}

.version-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-label {
  font-size: 14px;
  color: #606266;
}

.control-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.content-row {
  margin-bottom: 20px;
}

.info-card {
  height: 340px;
}

.auto-update-card {
  margin-top: 16px;
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background: #fafafa;
}

.auto-update-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  font-weight: 600;
}

.auto-update-time {
  margin: 0 0 6px 0;
  font-size: 12px;
  color: #909399;
}

.auto-update-details {
  margin: 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
  word-break: break-word;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.traffic-stats {
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
</style>
