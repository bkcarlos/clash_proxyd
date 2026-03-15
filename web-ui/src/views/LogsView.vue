<template>
  <div class="logs-view">
    <div class="toolbar">
      <el-tabs v-model="mode" class="mode-tabs" @tab-change="onModeChange">
        <el-tab-pane :label="t('logs.liveModeTab')" name="live" />
        <el-tab-pane :label="t('logs.proxydTab')" name="proxyd" />
        <el-tab-pane :label="t('logs.mihomoTab')" name="mihomo" />
      </el-tabs>

      <div class="toolbar-right">
        <!-- Live mode controls -->
        <template v-if="mode === 'live'">
          <el-select v-model="logLevel" style="width:110px" size="small" @change="reconnectWS">
            <el-option label="debug" value="debug" />
            <el-option label="info"  value="info" />
            <el-option label="warn"  value="warning" />
            <el-option label="error" value="error" />
          </el-select>
          <el-tag :type="wsConnected ? 'success' : 'danger'" size="small">
            {{ wsConnected ? t('logs.connected') : t('logs.disconnected') }}
          </el-tag>
          <el-button size="small" @click="liveLines = []">{{ t('logs.clear') }}</el-button>
        </template>

        <!-- File mode controls -->
        <template v-else>
          <el-select v-model="lineCount" style="width:110px" size="small" @change="fetchLogs">
            <el-option :label="t('logs.lines100')" :value="100" />
            <el-option :label="t('logs.lines200')" :value="200" />
            <el-option :label="t('logs.lines500')" :value="500" />
            <el-option :label="t('logs.lines1000')" :value="1000" />
          </el-select>
          <el-button
            :type="autoRefresh ? 'primary' : 'default'"
            size="small"
            @click="toggleAutoRefresh"
          >
            {{ autoRefresh ? t('logs.liveBtnOn') : t('logs.liveBtnOff') }}
          </el-button>
          <el-button size="small" :loading="loading" @click="fetchLogs">
            <el-icon><Refresh /></el-icon>
          </el-button>
          <el-button
            size="small"
            :disabled="!currentInfo.available"
            @click="download"
          >
            <el-icon><Download /></el-icon>
          </el-button>
        </template>

        <el-input
          v-model="filterText"
          :placeholder="t('logs.filterPlaceholder')"
          clearable
          size="small"
          style="width:180px"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>
    </div>

    <!-- File info bar (file mode) -->
    <div class="file-info" v-if="mode !== 'live' && currentInfo.file">
      <el-icon><Document /></el-icon>
      <span class="file-path">{{ currentInfo.file }}</span>
      <span class="file-meta" v-if="currentInfo.available">
        {{ currentInfo.total }} lines · {{ formatBytes(currentInfo.file_size) }}
      </span>
      <el-tag v-else type="warning" size="small">{{ currentInfo.message || t('logs.notAvailable') }}</el-tag>
    </div>

    <!-- Log output -->
    <div ref="logContainer" class="log-container" v-loading="loading">
      <template v-if="filteredLines.length > 0">
        <div
          v-for="(line, i) in filteredLines"
          :key="i"
          :class="['log-line', levelClass(line)]"
        >{{ line }}</div>
      </template>
      <div v-else-if="!loading" class="log-empty">
        <el-empty :description="mode === 'live' ? (wsConnected ? t('logs.waitingLogs') : t('logs.notConnected')) : t('logs.noEntries')" />
      </div>
    </div>

    <el-tooltip :content="t('logs.scrollToBottom')" placement="left">
      <el-button class="scroll-btn" circle type="primary" @click="scrollToBottom">
        <el-icon><ArrowDown /></el-icon>
      </el-button>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Document, ArrowDown, Download } from '@element-plus/icons-vue'
import { getLogs, downloadLog, type LogResponse } from '@/api/system'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
type Mode = 'live' | 'proxyd' | 'mihomo'

const mode = ref<Mode>('live')
const filterText = ref('')
const logContainer = ref<HTMLElement | null>(null)

// ── Live WebSocket mode ────────────────────────────────────────────────────
const liveLines = ref<string[]>([])
const wsConnected = ref(false)
const logLevel = ref('info')
const MAX_LIVE = 2000
let ws: WebSocket | null = null

const connectWS = () => {
  if (ws) ws.close()
  const token = localStorage.getItem('token') || ''
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${protocol}://${location.host}/api/v1/proxy/mihomo/log-stream?level=${logLevel.value}&token=${token}`
  ws = new WebSocket(url)
  ws.onopen = () => { wsConnected.value = true }
  ws.onclose = () => { wsConnected.value = false }
  ws.onerror = () => { wsConnected.value = false }
  ws.onmessage = (e) => {
    try {
      const d = JSON.parse(e.data)
      if (d.type === 'error') { ElMessage.warning(d.payload); return }
      const line = `${d.time || ''} [${(d.type || '').toUpperCase()}] ${d.payload || ''}`
      liveLines.value.push(line)
      if (liveLines.value.length > MAX_LIVE) liveLines.value.shift()
      autoScroll()
    } catch {
      liveLines.value.push(e.data)
      autoScroll()
    }
  }
}

const disconnectWS = () => { ws?.close(); ws = null; wsConnected.value = false }

const reconnectWS = () => {
  liveLines.value = []
  connectWS()
}

// ── File mode ──────────────────────────────────────────────────────────────
const loading = ref(false)
const lineCount = ref(200)
const autoRefresh = ref(false)
const emptyInfo: LogResponse = { source: '', file: '', lines: [], total: 0, file_size: 0, available: false }
const proxydInfo = ref<LogResponse>({ ...emptyInfo })
const mihomoInfo = ref<LogResponse>({ ...emptyInfo })
let fileTimer: ReturnType<typeof setInterval> | null = null

const currentInfo = computed(() =>
  mode.value === 'proxyd' ? proxydInfo.value : mihomoInfo.value
)

const fetchLogs = async () => {
  if (mode.value === 'live') return
  loading.value = true
  try {
    const res = await getLogs(mode.value as 'proxyd' | 'mihomo', lineCount.value)
    if (mode.value === 'proxyd') proxydInfo.value = res
    else mihomoInfo.value = res
  } catch (e: any) {
    ElMessage.error(e.message || t('logs.loadFailed'))
  } finally {
    loading.value = false
  }
}

const toggleAutoRefresh = () => {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) fileTimer = setInterval(fetchLogs, 5000)
  else { if (fileTimer) clearInterval(fileTimer); fileTimer = null }
}

const download = () => downloadLog(mode.value as 'proxyd' | 'mihomo')

// ── Shared ─────────────────────────────────────────────────────────────────
const fileLines = computed(() =>
  mode.value === 'proxyd' ? proxydInfo.value.lines : mihomoInfo.value.lines
)

const filteredLines = computed(() => {
  const lines = mode.value === 'live' ? liveLines.value : (fileLines.value ?? [])
  if (!filterText.value.trim()) return lines
  const q = filterText.value.toLowerCase()
  return lines.filter(l => l.toLowerCase().includes(q))
})

const levelClass = (line: string) => {
  const l = line.toLowerCase()
  if (l.includes('error') || l.includes('fatal')) return 'level-error'
  if (l.includes('warn')) return 'level-warn'
  if (l.includes('debug')) return 'level-debug'
  return 'level-info'
}

const formatBytes = (bytes: number) => {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1048576).toFixed(1)} MB`
}

const autoScroll = () => {
  nextTick(() => {
    const el = logContainer.value
    if (!el) return
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 100
    if (nearBottom) el.scrollTop = el.scrollHeight
  })
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
  })
}

const onModeChange = () => {
  filterText.value = ''
  if (mode.value === 'live') {
    connectWS()
  } else {
    disconnectWS()
    fetchLogs()
  }
}

onMounted(() => { connectWS() })
onUnmounted(() => {
  disconnectWS()
  if (fileTimer) clearInterval(fileTimer)
})
</script>

<style scoped>
.logs-view { display: flex; flex-direction: column; height: calc(100vh - 80px); gap: 8px; }

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 6px 14px;
}

.mode-tabs { flex: 1; margin-bottom: -8px; }

:deep(.mode-tabs .el-tabs__header) { margin: 0; }
:deep(.mode-tabs .el-tabs__nav-wrap::after) { display: none; }

.toolbar-right { display: flex; gap: 8px; align-items: center; }

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius-sm);
  font-size: 12px;
  color: var(--cv-text-muted);
}

.file-path { font-family: monospace; color: var(--cv-text); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-meta { white-space: nowrap; font-size: 11px; }

.log-container {
  flex: 1;
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
  background: #0d0f16;
  color: #d4d4d4;
  border-radius: var(--cv-radius);
  padding: 12px;
  overflow-y: auto;
  border: 1px solid var(--cv-border);
}

.log-line { padding: 1px 0; white-space: pre-wrap; word-break: break-all; }
.log-line:hover { background: rgba(255,255,255,0.04); }
.level-error { color: #f87171; }
.level-warn  { color: #fbbf24; }
.level-debug { color: #7dd3fc; }
.level-info  { color: #d4d4d4; }

.log-empty { display: flex; justify-content: center; padding: 40px 0; }

.scroll-btn { position: fixed; bottom: 40px; right: 40px; z-index: 100; }
</style>
