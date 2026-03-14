<template>
  <div class="connections-view">
    <div class="toolbar">
      <div class="stats">
        <span class="stat">
          <el-icon><Connection /></el-icon>
          {{ connections.length }} active
        </span>
        <span class="stat">
          <el-icon><Top /></el-icon>
          {{ formatBytes(totalUp) }}/s
        </span>
        <span class="stat">
          <el-icon><Bottom /></el-icon>
          {{ formatBytes(totalDown) }}/s
        </span>
      </div>
      <div style="display:flex;gap:8px">
        <el-input
          v-model="filter"
          placeholder="Filter..."
          clearable
          style="width:200px"
          size="small"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button size="small" type="danger" :disabled="connections.length === 0" @click="closeAll">
          Close All
        </el-button>
        <el-button size="small" :type="live ? 'primary' : 'default'" @click="live = !live">
          <el-icon><VideoPlay /></el-icon>
          {{ live ? 'Live' : 'Paused' }}
        </el-button>
      </div>
    </div>

    <el-table
      :data="filteredConnections"
      size="small"
      :max-height="tableHeight"
      stripe
    >
      <el-table-column label="Host" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="host">{{ row.metadata?.host || row.metadata?.destinationIP || '—' }}</span>
          <span class="port">:{{ row.metadata?.destinationPort }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Network" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="row.metadata?.network === 'tcp' ? 'primary' : 'warning'">
            {{ row.metadata?.network?.toUpperCase() }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Rule" width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.rule }} {{ row.rulePayload ? `(${row.rulePayload})` : '' }}</template>
      </el-table-column>
      <el-table-column label="Proxy" width="130" show-overflow-tooltip>
        <template #default="{ row }">
          <el-tag size="small" :type="row.chains?.[0] === 'DIRECT' ? 'success' : 'info'">
            {{ row.chains?.[0] || '—' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="↑" width="90" align="right">
        <template #default="{ row }">{{ formatBytes(row.upload) }}</template>
      </el-table-column>
      <el-table-column label="↓" width="90" align="right">
        <template #default="{ row }">{{ formatBytes(row.download) }}</template>
      </el-table-column>
      <el-table-column label="Time" width="70" align="right">
        <template #default="{ row }">{{ formatAge(row.start) }}</template>
      </el-table-column>
      <el-table-column width="60" align="center">
        <template #default="{ row }">
          <el-button link size="small" type="danger" @click="closeConn(row.id)">
            <el-icon><Close /></el-icon>
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="!mihomoRunning" class="offline-tip">
      <el-empty description="Mihomo is not running" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, Search, VideoPlay, Top, Bottom, Close } from '@element-plus/icons-vue'
import request from '@/api/request'

const connections = ref<any[]>([])
const filter = ref('')
const live = ref(true)
const mihomoRunning = ref(true)
const tableHeight = window.innerHeight - 180

const totalUp = computed(() => connections.value.reduce((s, c) => s + (c.upload || 0), 0))
const totalDown = computed(() => connections.value.reduce((s, c) => s + (c.download || 0), 0))

const filteredConnections = computed(() => {
  if (!filter.value) return connections.value
  const q = filter.value.toLowerCase()
  return connections.value.filter(c =>
    (c.metadata?.host || '').toLowerCase().includes(q) ||
    (c.metadata?.destinationIP || '').toLowerCase().includes(q) ||
    (c.rule || '').toLowerCase().includes(q) ||
    (c.chains?.[0] || '').toLowerCase().includes(q)
  )
})

const formatBytes = (b: number) => {
  if (!b) return '0B'
  if (b < 1024) return `${b}B`
  if (b < 1048576) return `${(b / 1024).toFixed(1)}K`
  return `${(b / 1048576).toFixed(1)}M`
}

const formatAge = (start: string) => {
  if (!start) return ''
  const s = Math.floor((Date.now() - new Date(start).getTime()) / 1000)
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m`
  return `${Math.floor(s / 3600)}h`
}

const fetchConnections = async () => {
  if (!live.value) return
  try {
    const data: any = await request({ url: '/proxy/connections', method: 'GET' })
    connections.value = data?.connections ?? []
    mihomoRunning.value = true
  } catch {
    mihomoRunning.value = false
    connections.value = []
  }
}

const closeConn = async (id: string) => {
  try {
    await request({ url: `/proxy/connections/${id}`, method: 'DELETE' })
    connections.value = connections.value.filter(c => c.id !== id)
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed')
  }
}

const closeAll = async () => {
  try {
    await ElMessageBox.confirm('Close all connections?', 'Confirm', { type: 'warning' })
    await request({ url: '/proxy/connections', method: 'DELETE' })
    connections.value = []
    ElMessage.success('All connections closed')
  } catch { /* cancel */ }
}

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  fetchConnections()
  timer = setInterval(fetchConnections, 1000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.connections-view { display: flex; flex-direction: column; gap: 12px; }

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 10px 16px;
}

.stats { display: flex; gap: 20px; }

.stat {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  color: var(--cv-text-muted);
}

.host { color: var(--cv-text); font-size: 13px; }
.port { color: var(--cv-text-muted); font-size: 12px; }

.offline-tip {
  display: flex;
  justify-content: center;
  padding: 40px;
}
</style>
