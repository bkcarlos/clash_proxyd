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

    <el-card>
      <template #header>
        <span>Proxy Groups</span>
      </template>
      <el-table :data="proxyStore.groups" border stripe>
        <el-table-column prop="name" label="Name" />
        <el-table-column prop="type" label="Type" width="120" />
        <el-table-column label="Current" width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag v-if="row.now" size="small" type="success">{{ row.now }}</el-tag>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120">
          <template #default="{ row }">
            <el-button size="small" @click="showGroupDetail(row)">Detail</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Group detail dialog -->
    <el-dialog v-model="groupDialogVisible" :title="selectedGroup?.name" width="740px">
      <!-- toolbar -->
      <div class="dialog-toolbar">
        <div class="dialog-meta">
          <el-tag size="small" type="info">{{ selectedGroup?.type }}</el-tag>
          <span class="current-label">Current: <strong>{{ selectedGroup?.now || '—' }}</strong></span>
        </div>
        <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
          <el-input
            v-model="proxyFilter"
            placeholder="Filter nodes..."
            clearable
            size="small"
            style="width:150px"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-input
            v-model="testUrl"
            placeholder="Test URL"
            clearable
            size="small"
            style="width:230px"
          />
          <el-button
            size="small"
            :type="sortByDelay ? 'primary' : 'default'"
            @click="sortByDelay = !sortByDelay"
            title="Sort by delay"
          >
            <el-icon><Sort /></el-icon>
            Delay
          </el-button>
          <el-button
            size="small"
            type="success"
            :loading="testingAll"
            @click="testAll"
          >
            Test All
          </el-button>
        </div>
      </div>

      <el-table :data="displayedProxies" border stripe size="small" style="margin-top:4px">
        <el-table-column prop="name" label="Proxy" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span :class="{ 'current-proxy': row.name === selectedGroup?.now }">
              {{ row.name }}
            </span>
            <el-tag v-if="row.name === selectedGroup?.now" size="small" type="success" style="margin-left:4px">✓</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Delay" width="120" align="center">
          <template #default="{ row }">
            <span v-if="testing[row.name]" class="delay-spin">
              <el-icon class="is-loading"><Loading /></el-icon>
            </span>
            <el-tag v-else-if="row.delay === undefined" size="small" type="info">—</el-tag>
            <el-tooltip
              v-else-if="row.delay === 0"
              :content="row.testErr || 'Timeout / unreachable'"
              placement="top"
              :show-after="300"
            >
              <el-tag size="small" type="danger" style="cursor:help">Timeout ⓘ</el-tag>
            </el-tooltip>
            <el-tag v-else :type="delayTagType(row.delay)" size="small">{{ row.delay }} ms</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="150" align="center">
          <template #default="{ row }">
            <el-button size="small" :loading="!!testing[row.name]" @click="testOne(row)">Test</el-button>
            <el-button
              size="small"
              type="primary"
              :disabled="row.name === selectedGroup?.now"
              @click="switchProxy(row)"
            >Switch</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <span class="footer-stat">{{ filteredProxies.length }} / {{ selectedGroup?.proxies?.length || 0 }} proxies</span>
        <el-button @click="groupDialogVisible = false">Close</el-button>
      </template>
    </el-dialog>

    <!-- Mihomo control dialog -->
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
import { ref, computed, onMounted } from 'vue'
import { useProxyStore } from '@/stores/proxy'
import { ElMessage } from 'element-plus'
import { Refresh, Setting, Search, Sort, Loading } from '@element-plus/icons-vue'

const proxyStore = useProxyStore()

const groupDialogVisible = ref(false)
const mihomoDialogVisible = ref(false)
const selectedGroup = ref<any>(null)
const testing = ref<Record<string, boolean>>({})
const testingAll = ref(false)
const proxyFilter = ref('')
const sortByDelay = ref(false)
const testUrl = ref('http://cp.cloudflare.com/generate_204')

const delayTagType = (delay: number) => {
  if (delay < 150) return 'success'
  if (delay < 300) return 'warning'
  return 'danger'
}

// Proxies filtered by search input
const filteredProxies = computed<any[]>(() => {
  const list: any[] = selectedGroup.value?.proxies || []
  const q = proxyFilter.value.trim().toLowerCase()
  return q ? list.filter(p => p.name.toLowerCase().includes(q)) : list
})

// After filter, optionally sort by delay
const displayedProxies = computed<any[]>(() => {
  if (!sortByDelay.value) return filteredProxies.value
  return [...filteredProxies.value].sort((a, b) => {
    // untested (undefined) goes to the bottom
    if (a.delay === undefined && b.delay === undefined) return 0
    if (a.delay === undefined) return 1
    if (b.delay === undefined) return -1
    // timeout (0) goes after real delays
    if (a.delay === 0 && b.delay === 0) return 0
    if (a.delay === 0) return 1
    if (b.delay === 0) return -1
    return a.delay - b.delay
  })
})

const refreshProxies = async () => {
  try {
    await proxyStore.fetchProxies(true)
    ElMessage.success('Proxies refreshed')
  } catch (error: any) {
    ElMessage.error(error.message || 'Refresh failed')
  }
}

const showGroupDetail = (group: any) => {
  // deep-copy so local delay updates don't mutate the store
  selectedGroup.value = {
    ...group,
    proxies: group.proxies.map((p: any) => ({ ...p }))
  }
  testing.value = {}
  proxyFilter.value = ''
  sortByDelay.value = false
  // keep testUrl across dialogs so user's custom URL persists
  groupDialogVisible.value = true
}

const testOne = async (item: any, silent = false): Promise<void> => {
  if (testing.value[item.name]) return
  testing.value[item.name] = true
  try {
    const url = testUrl.value || 'http://cp.cloudflare.com/generate_204'
    const result = await proxyStore.testProxy(item.name, url, 5000)
    if (result.error || result.delay === 0) {
      item.delay = 0
      // Extract the meaningful part from the error string for the tooltip
      const raw = result.error || ''
      const match = raw.match(/status (\d+): (.+)/)
      item.testErr = match ? `HTTP ${match[1]}: ${match[2]}` : (raw || 'Timeout / unreachable')
      if (!silent) ElMessage.warning(`${item.name}: ${item.testErr}`)
    } else {
      item.delay = result.delay
      item.testErr = undefined
      if (!silent) {
        const suffix = result.from_cache ? ' (cached)' : ''
        ElMessage.success(`${item.name}: ${result.delay} ms${suffix}`)
      }
    }
  } catch {
    item.delay = 0
    item.testErr = 'Test failed'
    if (!silent) ElMessage.error(`${item.name}: Test failed`)
  } finally {
    testing.value[item.name] = false
  }
}

const testAll = async () => {
  if (!selectedGroup.value?.proxies?.length || testingAll.value) return
  testingAll.value = true
  const proxies: any[] = selectedGroup.value.proxies
  // Run in parallel, concurrency capped at 10
  const BATCH = 10
  for (let i = 0; i < proxies.length; i += BATCH) {
    await Promise.allSettled(proxies.slice(i, i + BATCH).map(p => testOne(p, true)))
  }
  const ok = proxies.filter(p => p.delay && p.delay > 0).length
  const timeout = proxies.length - ok
  ElMessage.success(`Tested ${proxies.length}: ${ok} reachable, ${timeout} timeout`)
  // Auto-sort by delay after Test All
  sortByDelay.value = true
  testingAll.value = false
}

const switchProxy = async (item: any) => {
  if (!selectedGroup.value?.name) return
  try {
    await proxyStore.switchProxy(selectedGroup.value.name, item.name)
    selectedGroup.value.now = item.name
    ElMessage.success(`Switched to ${item.name}`)
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

.dialog-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.dialog-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.current-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.current-proxy {
  font-weight: 600;
  color: var(--el-color-success);
}

.delay-spin {
  color: var(--el-color-primary);
  font-size: 16px;
}

.footer-stat {
  margin-right: auto;
  font-size: 12px;
  color: var(--el-text-color-secondary);
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
