<template>
  <div class="dashboard-view">
    <h1>{{ t('dashboard.title') }}</h1>

    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#409eff"><Odometer /></el-icon>
            <div>
              <p class="stat-label">{{ t('dashboard.uptime') }}</p>
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
              <p class="stat-label">{{ t('dashboard.mihomoStatus') }}</p>
              <p class="stat-value">{{ systemStore.info?.mihomo_status || t('common.unknown') }}</p>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#e6a23c"><Download /></el-icon>
            <div>
              <p class="stat-label">{{ t('dashboard.sources') }}</p>
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
              <p class="stat-label">{{ t('dashboard.proxies') }}</p>
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
          <span>{{ t('dashboard.mihomoControl') }}</span>
          <el-tag :type="mihomoStatusTagType" size="small">{{ systemStore.info?.mihomo_status || t('common.unknown') }}</el-tag>
        </div>
      </template>
      <div class="control-row">
        <div class="version-info">
          <span class="version-label">{{ t('dashboard.binaryVersion') }}:</span>
          <el-tag type="info" size="small">{{ mihomoVersion || '—' }}</el-tag>
        </div>
        <div class="control-buttons">
          <el-button
            type="success"
            :loading="controlling === 'start'"
            :disabled="mihomoRunning || !!controlling"
            @click="doControl('start')"
          >{{ t('dashboard.start') }}</el-button>
          <el-button
            type="danger"
            :loading="controlling === 'stop'"
            :disabled="!mihomoRunning || !!controlling"
            @click="doControl('stop')"
          >{{ t('dashboard.stop') }}</el-button>
          <el-button
            type="warning"
            :loading="controlling === 'restart'"
            :disabled="!mihomoRunning || !!controlling"
            @click="doControl('restart')"
          >{{ t('dashboard.restart') }}</el-button>
          <el-divider direction="vertical" />
          <el-button
            :loading="checkingUpdate"
            :disabled="!!controlling || checkingUpdate"
            @click="doUpdate"
          >{{ t('dashboard.checkUpdate') }}</el-button>
          <el-button size="small" @click="fetchVersion" :loading="loadingVersion">{{ t('dashboard.refreshVersion') }}</el-button>
        </div>
      </div>
    </el-card>

    <!-- Network Speed: full-width row -->
    <el-card class="speed-card" style="margin-bottom: 20px;">
      <template #header>
        <div class="card-header">
          <span>{{ t('dashboard.networkSpeed') }} <span class="chart-window-label">2 min</span></span>
          <div style="display:flex;gap:16px;font-size:13px">
            <span style="color:#5865f2">↑ {{ formatRate(upRate) }}</span>
            <span style="color:#22d3ee">↓ {{ formatRate(downRate) }}</span>
          </div>
        </div>
      </template>
      <div class="speed-chart-wrap">
        <svg width="100%" :viewBox="`0 0 ${chartW} ${chartH}`" preserveAspectRatio="none" class="speed-chart">
          <!-- Horizontal grid lines -->
          <line v-for="n in 3" :key="n"
            x1="0" :y1="(chartH / 4) * n"
            :x2="chartW" :y2="(chartH / 4) * n"
            stroke="rgba(255,255,255,0.06)" stroke-width="1"
          />
          <!-- Vertical grid lines at 25% intervals -->
          <line v-for="n in 3" :key="`v${n}`"
            :x1="(chartW / 4) * n" y1="0"
            :x2="(chartW / 4) * n" :y2="chartH"
            stroke="rgba(255,255,255,0.04)" stroke-width="1"
          />
          <!-- Up fill -->
          <polygon :points="upFill" fill="rgba(88,101,242,0.2)" />
          <!-- Down fill -->
          <polygon :points="downFill" fill="rgba(34,211,238,0.12)" />
          <!-- Up line -->
          <polyline
            :points="upPoints"
            fill="none"
            stroke="#5865f2"
            stroke-width="2"
            stroke-linejoin="round"
            stroke-linecap="round"
          />
          <!-- Down line -->
          <polyline
            :points="downPoints"
            fill="none"
            stroke="#22d3ee"
            stroke-width="2"
            stroke-linejoin="round"
            stroke-linecap="round"
          />
          <!-- Max label -->
          <text v-if="chartMax > 0"
            x="4" y="12"
            font-size="9" fill="rgba(255,255,255,0.3)"
          >{{ formatRate(chartMax) }}</text>
        </svg>
        <!-- Time axis: real clock times, scroll with data -->
        <div class="chart-time-axis">
          <span v-for="label in timeAxisLabels" :key="label">{{ label }}</span>
        </div>
        <div class="chart-labels">
          <span style="color:#5865f2">↑ {{ formatBytes(proxyStore.traffic.upTotal) }} {{ t('dashboard.total') }}</span>
          <span style="color:#22d3ee">↓ {{ formatBytes(proxyStore.traffic.downTotal) }} {{ t('dashboard.total') }}</span>
        </div>
      </div>
    </el-card>

    <el-row :gutter="20" class="content-row">
      <el-col :span="24">
        <el-card class="info-card">
          <template #header>
            <div class="card-header">
              <span>{{ t('dashboard.systemInfo') }}</span>
              <el-button size="small" @click="refreshSystem">{{ t('common.refresh') }}</el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('dashboard.version')">
              {{ systemStore.info?.version || '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('dashboard.goVersion')">
              {{ systemStore.info?.go_version || '-' }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('dashboard.database')">
              {{ systemStore.info?.database || '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="auto-update-card">
            <div class="auto-update-header">
              <span>{{ t('dashboard.lastAutoUpdate') }}</span>
              <el-tag v-if="systemStore.info?.last_auto_update_action" :type="autoUpdateTagType" size="small">
                {{ formatAutoUpdateAction(systemStore.info?.last_auto_update_action || '') }}
              </el-tag>
              <el-tag v-else type="info" size="small">{{ t('dashboard.noRecord') }}</el-tag>
            </div>
            <p class="auto-update-time">
              {{ formatDateTime(systemStore.info?.last_auto_update_at) }}
            </p>
            <p class="auto-update-details">
              {{ systemStore.info?.last_auto_update_details || t('dashboard.noAutoUpdateRecord') }}
            </p>
          </div>

          <div class="auto-update-card" style="margin-top: 12px;">
            <div class="auto-update-header">
              <span>{{ t('dashboard.lastAlert') }}</span>
              <el-tag v-if="systemStore.info?.last_alert_action" :type="alertTagType" size="small">
                {{ formatAlertAction(systemStore.info?.last_alert_action || '') }}
              </el-tag>
              <el-tag v-else type="info" size="small">{{ t('dashboard.noRecord') }}</el-tag>
            </div>
            <p class="auto-update-time">
              {{ formatDateTime(systemStore.info?.last_alert_at) }}
            </p>
            <p class="auto-update-details">
              {{ systemStore.info?.last_alert_details || t('dashboard.noAlertRecord') }}
            </p>
          </div>
        </el-card>
      </el-col>

    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useSystemStore } from '@/stores/system'
import { useSourceStore } from '@/stores/source'
import { useProxyStore } from '@/stores/proxy'
import { controlMihomo, getMihomoVersion, updateMihomo } from '@/api/proxy'
import { Odometer, CircleCheck, Download, Connection } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const systemStore = useSystemStore()
const sourceStore = useSourceStore()
const proxyStore = useProxyStore()

const mihomoVersion = ref<string>('')
const loadingVersion = ref(false)
const controlling = ref<string>('')
const checkingUpdate = ref(false)

// ── Speed chart ────────────────────────────────────────────────────────────
const HISTORY = 120          // 2 minutes at 1 sample/s
const chartW = 340
const chartH = 100
const upHistory = ref<number[]>(Array(HISTORY).fill(0))
const downHistory = ref<number[]>(Array(HISTORY).fill(0))
const nowTs = ref(Date.now())

// 5 evenly-spaced time labels across the 2-minute window
const TIME_AXIS_OFFSETS = [120, 90, 60, 30, 0] // seconds before "now"
const fmtTime = (ts: number) => {
  const d = new Date(ts)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${hh}:${mm}:${ss}`
}
const timeAxisLabels = computed(() =>
  TIME_AXIS_OFFSETS.map(s => fmtTime(nowTs.value - s * 1000))
)

const upRate = computed(() => upHistory.value[upHistory.value.length - 1] ?? 0)
const downRate = computed(() => downHistory.value[downHistory.value.length - 1] ?? 0)

// Dynamic scale: use the last 30 samples (30s) so the chart adapts quickly
// after a big download ends. Clamp ratio to 1 so older spikes don't go off-chart.
const SCALE_WINDOW = 30
const chartMax = computed(() => {
  const tail = (arr: number[]) => arr.slice(-SCALE_WINDOW)
  return Math.max(...tail(upHistory.value), ...tail(downHistory.value), 1024)
})

const toPoints = (data: number[], maxVal: number) => {
  const pad = 6 // vertical padding so lines don't touch top/bottom edge
  return data.map((v, i) => {
    const x = (i / (HISTORY - 1)) * chartW
    const ratio = maxVal > 0 ? Math.min(v / maxVal, 1) : 0
    const y = chartH - pad - ratio * (chartH - pad * 2)
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

const toFill = (points: string) => {
  return `0,${chartH} ${points} ${chartW},${chartH}`
}

const upPoints = computed(() => toPoints(upHistory.value, chartMax.value))
const downPoints = computed(() => toPoints(downHistory.value, chartMax.value))
const upFill = computed(() => toFill(upPoints.value))
const downFill = computed(() => toFill(downPoints.value))

const formatRate = (bps: number) => {
  if (!bps || !isFinite(bps) || bps < 0) return '0 B/s'
  if (bps < 1024) return `${bps} B/s`
  if (bps < 1048576) return `${(bps / 1024).toFixed(1)} KB/s`
  return `${(bps / 1048576).toFixed(1)} MB/s`
}

const tickTraffic = () => {
  nowTs.value = Date.now()
  // mihomo /traffic already returns instantaneous rates (bytes/s);
  // push them directly — no delta calculation needed.
  const { up, down } = proxyStore.traffic
  upHistory.value = [...upHistory.value.slice(1), Math.max(0, up)]
  downHistory.value = [...downHistory.value.slice(1), Math.max(0, down)]
}

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
    mihomoVersion.value = t('common.na')
  } finally {
    loadingVersion.value = false
  }
}

const doControl = async (action: 'start' | 'stop' | 'restart') => {
  controlling.value = action
  try {
    await controlMihomo(action)
    ElMessage.success(t('dashboard.mihomoActionSuccess', { action }))
    await systemStore.fetchInfo()
  } catch (e: any) {
    ElMessage.error(t('dashboard.mihomoActionFailed', { action, error: e?.message || e }))
  } finally {
    controlling.value = ''
  }
}

const doUpdate = async () => {
  checkingUpdate.value = true
  try {
    const res = await updateMihomo()
    if (res.updated) {
      ElMessage.success(t('dashboard.updatedMsg', { old: res.old_version, new: res.new_version }))
      await fetchVersion()
    } else {
      ElMessage.info(t('dashboard.alreadyUpToDate', { version: res.current_version }))
    }
  } catch (e: any) {
    ElMessage.error(t('dashboard.updateFailed', { error: e?.message || e }))
  } finally {
    checkingUpdate.value = false
  }
}

const formatAlertAction = (action: string): string => {
  if (!action) return t('common.unknown')
  return action
    .replace('alert_', '')
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

const formatAutoUpdateAction = (action: string): string => {
  if (!action) return t('common.unknown')
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
  // Seed the chart with the initial traffic snapshot
  tickTraffic()
})

// Drive the speed chart from WS-pushed traffic (via proxyStore.traffic)
watch(() => proxyStore.traffic, () => tickTraffic(), { deep: true })

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

.info-card {}

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

.speed-chart-wrap {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.speed-chart {
  display: block;
  width: 100%;
  height: 120px;
  background: var(--cv-surface2);
  border-radius: var(--cv-radius-sm);
}

.chart-time-axis {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: rgba(144, 147, 153, 0.7);
  margin-top: -4px;
}

.chart-labels {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.chart-window-label {
  font-size: 11px;
  font-weight: 400;
  color: #909399;
  margin-left: 6px;
  background: var(--el-fill-color-light, #f5f7fa);
  padding: 1px 6px;
  border-radius: 8px;
}
</style>
