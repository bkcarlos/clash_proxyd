<template>
  <div class="logs-view">
    <div class="page-header">
      <h1>Logs</h1>
      <div class="header-actions">
        <el-select v-model="lineCount" style="width: 110px" @change="fetchLogs">
          <el-option label="100 lines" :value="100" />
          <el-option label="200 lines" :value="200" />
          <el-option label="500 lines" :value="500" />
          <el-option label="1000 lines" :value="1000" />
        </el-select>
        <el-input
          v-model="filterText"
          placeholder="Filter..."
          clearable
          style="width: 200px"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-tooltip :content="autoRefresh ? 'Auto-refresh on (5s)' : 'Auto-refresh off'">
          <el-button
            :type="autoRefresh ? 'primary' : 'default'"
            @click="toggleAutoRefresh"
          >
            <el-icon><Timer /></el-icon>
            {{ autoRefresh ? 'Live' : 'Live' }}
          </el-button>
        </el-tooltip>
        <el-button :loading="loading" @click="fetchLogs">
          <el-icon><Refresh /></el-icon>
          Refresh
        </el-button>
        <el-button
          :disabled="!currentInfo.available"
          @click="download"
        >
          <el-icon><Download /></el-icon>
          Download
        </el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <el-tab-pane label="proxyd" name="proxyd">
        <template #label>
          <span>proxyd</span>
          <el-badge v-if="proxydInfo.available === false" value="!" type="warning" style="margin-left:4px" />
        </template>
      </el-tab-pane>
      <el-tab-pane label="Mihomo" name="mihomo">
        <template #label>
          <span>Mihomo</span>
          <el-badge v-if="mihomoInfo.available === false" value="!" type="warning" style="margin-left:4px" />
        </template>
      </el-tab-pane>
    </el-tabs>

    <!-- File info bar -->
    <div class="file-info" v-if="currentInfo.file">
      <el-icon><Document /></el-icon>
      <span class="file-path">{{ currentInfo.file }}</span>
      <span class="file-meta" v-if="currentInfo.available">
        {{ currentInfo.total }} lines shown · {{ formatBytes(currentInfo.file_size) }}
      </span>
      <el-tag v-if="currentInfo.available === false" type="warning" size="small">
        {{ currentInfo.message || 'Not available' }}
      </el-tag>
    </div>

    <!-- Log output -->
    <div ref="logContainer" class="log-container" v-loading="loading">
      <template v-if="filteredLines.length > 0">
        <div
          v-for="(line, i) in filteredLines"
          :key="i"
          :class="['log-line', levelClass(line)]"
        >
          <span class="log-text">{{ line }}</span>
        </div>
      </template>
      <div v-else-if="!loading" class="log-empty">
        <el-empty
          :description="currentInfo.available === false
            ? (currentInfo.message || 'Log file not available')
            : (filterText ? 'No matching lines' : 'No log entries')"
        />
      </div>
    </div>

    <!-- Scroll to bottom -->
    <el-tooltip content="Scroll to bottom" placement="left">
      <el-button
        class="scroll-btn"
        circle
        type="primary"
        @click="scrollToBottom"
      >
        <el-icon><ArrowDown /></el-icon>
      </el-button>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Timer, Document, ArrowDown, Download } from '@element-plus/icons-vue'
import { getLogs, downloadLog, type LogResponse } from '@/api/system'

const activeTab = ref<'proxyd' | 'mihomo'>('proxyd')
const lineCount = ref(200)
const filterText = ref('')
const loading = ref(false)
const autoRefresh = ref(false)
const logContainer = ref<HTMLElement | null>(null)

const emptyInfo: LogResponse = { source: '', file: '', lines: [], total: 0, file_size: 0, available: false }
const proxydInfo = ref<LogResponse>({ ...emptyInfo })
const mihomoInfo = ref<LogResponse>({ ...emptyInfo })

const currentInfo = computed(() =>
  activeTab.value === 'proxyd' ? proxydInfo.value : mihomoInfo.value
)

const filteredLines = computed(() => {
  const lines = currentInfo.value.lines ?? []
  if (!filterText.value.trim()) return lines
  const q = filterText.value.toLowerCase()
  return lines.filter(l => l.toLowerCase().includes(q))
})

const levelClass = (line: string): string => {
  const l = line.toLowerCase()
  if (l.includes('"level":"error"') || l.includes('error') || l.includes('fatal')) return 'level-error'
  if (l.includes('"level":"warn"') || l.includes('warn')) return 'level-warn'
  if (l.includes('"level":"debug"') || l.includes('debug')) return 'level-debug'
  return 'level-info'
}

const formatBytes = (bytes: number): string => {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await getLogs(activeTab.value, lineCount.value)
    if (activeTab.value === 'proxyd') {
      proxydInfo.value = res
    } else {
      mihomoInfo.value = res
    }
    if (!res.available && res.message) {
      // Don't show error toast for "not configured" — just show in UI
    }
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to load logs')
  } finally {
    loading.value = false
  }
}

const onTabChange = () => {
  filterText.value = ''
  fetchLogs()
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

// Auto-scroll when new lines arrive and user is near the bottom
watch(filteredLines, () => {
  if (!logContainer.value) return
  const el = logContainer.value
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  if (nearBottom) scrollToBottom()
})

let timer: ReturnType<typeof setInterval> | null = null

const download = () => {
  downloadLog(activeTab.value)
}

const toggleAutoRefresh = () => {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    timer = setInterval(fetchLogs, 5000)
  } else {
    if (timer) clearInterval(timer)
    timer = null
  }
}

onMounted(fetchLogs)

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.logs-view h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #606266;
}

.file-path {
  font-family: monospace;
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  white-space: nowrap;
  color: #909399;
  font-size: 12px;
}

.log-container {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px;
  padding: 12px;
  height: calc(100vh - 280px);
  min-height: 300px;
  overflow-y: auto;
  border: 1px solid #3c3c3c;
}

.log-line {
  padding: 1px 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.level-error { color: #f48771; }
.level-warn  { color: #dcdcaa; }
.level-debug { color: #9cdcfe; }
.level-info  { color: #d4d4d4; }

.log-empty {
  display: flex;
  justify-content: center;
  padding: 40px 0;
}

.scroll-btn {
  position: fixed;
  bottom: 40px;
  right: 40px;
  z-index: 100;
}

:deep(.el-tabs__header) {
  margin-bottom: 8px;
}
</style>
