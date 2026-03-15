<template>
  <div class="proxies-view">

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="proxyFilter"
          :placeholder="t('proxies.filterPlaceholder')"
          size="small"
          clearable
          style="width:160px"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input
          v-model="testUrl"
          :placeholder="t('proxies.testUrlPlaceholder')"
          size="small"
          clearable
          style="width:270px"
        />
      </div>
      <div class="toolbar-right">
        <el-tooltip :content="t('proxies.sortByDelay')" placement="bottom">
          <el-button
            size="small"
            :type="sortByDelay ? 'primary' : ''"
            @click="sortByDelay = !sortByDelay"
          >
            <el-icon><Sort /></el-icon>
          </el-button>
        </el-tooltip>
        <el-button size="small" type="success" :loading="testingAll" @click="testAll">
          {{ t('proxies.testAll') }}
        </el-button>
        <el-button size="small" :loading="refreshing" @click="refreshProxies">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- Two-panel layout -->
    <div class="panels">

      <!-- Left: group list -->
      <div class="group-panel">
        <div class="group-panel-header">
          {{ t('proxies.groups') }}
          <span class="group-count">{{ proxyStore.groups.length }}</span>
        </div>
        <div
          v-for="group in proxyStore.groups"
          :key="group.name"
          class="group-item"
          :class="{ active: selectedGroup?.name === group.name }"
          @click="selectGroup(group)"
        >
          <div class="gi-name">{{ group.name }}</div>
          <div class="gi-meta">
            <span class="gi-type">{{ group.type }}</span>
            <span class="gi-current">{{ group.now || '—' }}</span>
          </div>
        </div>
        <div v-if="proxyStore.groups.length === 0" class="group-empty">
          <el-empty :image-size="48" :description="t('proxies.noGroups')" />
        </div>
      </div>

      <!-- Right: node cards -->
      <div class="node-panel">
        <template v-if="selectedGroup">
          <!-- Group title bar -->
          <div class="node-panel-header">
            <div class="nph-left">
              <span class="nph-name">{{ selectedGroup.name }}</span>
              <span class="nph-type">{{ selectedGroup.type }}</span>
              <span class="nph-current">
                <el-icon style="font-size:11px;margin-right:2px"><Select /></el-icon>
                {{ selectedGroup.now || '—' }}
              </span>
            </div>
            <span class="nph-count">
              {{ displayedProxies.length }} / {{ selectedGroup.proxies?.length || 0 }}
            </span>
          </div>

          <!-- Node grid -->
          <div class="node-grid">
            <div
              v-for="node in displayedProxies"
              :key="node.name"
              class="node-card"
              :class="{
                selected: node.name === selectedGroup.now,
                dimmed: node.name !== selectedGroup.now && selectedGroup.now
              }"
              @click="handleNodeClick(node)"
            >
              <!-- Selected corner badge -->
              <div v-if="node.name === selectedGroup.now" class="selected-corner">
                <el-icon><Select /></el-icon>
              </div>

              <!-- Type + test btn -->
              <div class="nc-top">
                <span class="nc-type" :class="node.proxyType?.toLowerCase()">
                  {{ typeAbbr(node.proxyType) }}
                </span>
                <button
                  class="nc-test-btn"
                  :class="{ spinning: testing[node.name] }"
                  @click.stop="testOne(node.name)"
                  :title="`Test ${node.name}`"
                >
                  <el-icon v-if="testing[node.name]" class="is-loading"><Loading /></el-icon>
                  <el-icon v-else><Connection /></el-icon>
                </button>
              </div>

              <!-- Name -->
              <div class="nc-name">{{ node.name }}</div>

              <!-- Delay pill -->
              <div class="nc-bottom">
                <span class="delay-pill" :class="delayClass(node)">
                  {{ delayLabel(node) }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="displayedProxies.length === 0" class="node-no-result">
            <el-empty :image-size="60" :description="t('proxies.noMatchingNodes')" />
          </div>
        </template>

        <div v-else class="node-placeholder">
          <el-empty :image-size="80" :description="t('proxies.selectGroup')" />
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useProxyStore } from '@/stores/proxy'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Sort, Loading, Connection, Select } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const proxyStore = useProxyStore()

const selectedGroup   = ref<any>(null)
const testing         = ref<Record<string, boolean>>({})
const testingAll      = ref(false)
const proxyFilter     = ref('')
const sortByDelay     = ref(false)
const testUrl         = ref('http://cp.cloudflare.com/generate_204')
const refreshing      = ref(false)
const delays          = ref<Record<string, { delay: number; error?: string }>>({})

// ── Helpers ──────────────────────────────────────────────────────────────────

const getProxyType = (name: string): string =>
  (proxyStore.proxies[name] as any)?.type || ''

const typeAbbr = (t: string): string => {
  const map: Record<string, string> = {
    vmess: 'VM', vless: 'VL', trojan: 'TR', shadowsocks: 'SS',
    ss: 'SS', ssr: 'SSR', http: 'HTTP', socks5: 'S5',
    direct: 'DT', reject: 'RJ', selector: 'SEL', urltest: 'UT',
    fallback: 'FB', loadbalance: 'LB',
  }
  return t ? (map[t.toLowerCase()] ?? t.slice(0, 3).toUpperCase()) : '?'
}

const initDelaysFromHistory = () => {
  for (const [name, info] of Object.entries(proxyStore.proxies)) {
    const history = (info as any)?.history
    if (history?.length > 0) {
      const last = history[history.length - 1]
      if (last.delay !== undefined && delays.value[name] === undefined) {
        delays.value[name] = { delay: last.delay }
      }
    }
  }
}

// ── Group selection ───────────────────────────────────────────────────────────

const selectGroup = (group: any) => {
  selectedGroup.value = group
  proxyFilter.value = ''
}

// Keep selectedGroup in sync when store refreshes
watch(() => proxyStore.groups, (groups) => {
  if (selectedGroup.value) {
    const updated = groups.find((g: any) => g.name === selectedGroup.value.name)
    if (updated) selectedGroup.value = updated
  } else if (groups.length > 0) {
    selectGroup(groups[0])
  }
})

// ── Node computed list ────────────────────────────────────────────────────────

const enrichedProxies = computed(() => {
  if (!selectedGroup.value) return []
  return (selectedGroup.value.proxies || []).map((p: any) => ({
    name: p.name,
    proxyType: getProxyType(p.name),
    delay: delays.value[p.name]?.delay,
    error: delays.value[p.name]?.error,
  }))
})

const filteredProxies = computed(() => {
  const q = proxyFilter.value.trim().toLowerCase()
  return q
    ? enrichedProxies.value.filter((p: any) => p.name.toLowerCase().includes(q))
    : enrichedProxies.value
})

const displayedProxies = computed(() => {
  if (!sortByDelay.value) return filteredProxies.value
  return [...filteredProxies.value].sort((a: any, b: any) => {
    if (a.delay === undefined && b.delay === undefined) return 0
    if (a.delay === undefined) return 1
    if (b.delay === undefined) return -1
    if (a.delay === 0 && b.delay === 0) return 0
    if (a.delay === 0) return 1
    if (b.delay === 0) return -1
    return a.delay - b.delay
  })
})

// ── Delay display ─────────────────────────────────────────────────────────────

const delayClass = (node: any): string => {
  if (testing.value[node.name]) return 'testing'
  if (node.delay === undefined)  return 'untested'
  if (node.delay === 0)          return 'timeout'
  if (node.delay < 150)          return 'fast'
  if (node.delay < 300)          return 'medium'
  return 'slow'
}

const delayLabel = (node: any): string => {
  if (testing.value[node.name]) return '…'
  if (node.delay === undefined)  return '—'
  if (node.delay === 0)          return node.error ? '✕' : t('proxies.timeout')
  return `${node.delay} ms`
}

// ── Actions ───────────────────────────────────────────────────────────────────

const testOne = async (name: string, silent = false) => {
  if (testing.value[name]) return
  testing.value[name] = true
  try {
    const result = await proxyStore.testProxy(name, testUrl.value, 5000)
    if (result.error || result.delay === 0) {
      const raw = result.error || ''
      const match = raw.match(/status (\d+): (.+)/)
      const msg = match ? `HTTP ${match[1]}: ${match[2]}` : (raw || t('proxies.timeout'))
      delays.value[name] = { delay: 0, error: msg }
      if (!silent) ElMessage.warning(`${name}: ${msg}`)
    } else {
      delays.value[name] = { delay: result.delay }
      if (!silent) ElMessage.success(`${name}: ${result.delay} ms`)
    }
  } catch {
    delays.value[name] = { delay: 0, error: 'Failed' }
    if (!silent) ElMessage.error(`${name}: test failed`)
  } finally {
    testing.value[name] = false
  }
}

const testAll = async () => {
  if (!selectedGroup.value?.proxies?.length || testingAll.value) return
  testingAll.value = true
  const names: string[] = selectedGroup.value.proxies.map((p: any) => p.name)
  const BATCH = 10
  for (let i = 0; i < names.length; i += BATCH) {
    await Promise.allSettled(names.slice(i, i + BATCH).map(n => testOne(n, true)))
  }
  const ok = names.filter(n => (delays.value[n]?.delay ?? 0) > 0).length
  ElMessage.success(t('proxies.tested', { total: names.length, ok, failed: names.length - ok }))
  sortByDelay.value = true
  testingAll.value = false
}

const handleNodeClick = async (node: any) => {
  if (!selectedGroup.value || node.name === selectedGroup.value.now) return
  try {
    await proxyStore.switchProxy(selectedGroup.value.name, node.name)
    selectedGroup.value.now = node.name
    const sg = proxyStore.groups.find((g: any) => g.name === selectedGroup.value.name)
    if (sg) sg.now = node.name
    ElMessage.success(t('proxies.switchedTo', { name: node.name }))
  } catch (e: any) {
    ElMessage.error(e?.message || t('proxies.switchFailed'))
  }
}

const refreshProxies = async () => {
  refreshing.value = true
  try {
    await proxyStore.fetchProxies(true)
    initDelaysFromHistory()
    ElMessage.success(t('proxies.refreshed'))
  } catch (e: any) {
    ElMessage.error(e?.message || t('proxies.refreshFailed'))
  } finally {
    refreshing.value = false
  }
}

onMounted(async () => {
  await proxyStore.fetchProxies(true)
  initDelaysFromHistory()
  if (proxyStore.groups.length > 0) selectGroup(proxyStore.groups[0])
})
</script>

<style scoped>
/* ── Root: fill the content pane, vertical grid ── */
.proxies-view {
  height: 100%;
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 14px;
}

/* ── Toolbar ── */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 10px 16px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ── Panels container ── */
.panels {
  display: flex;
  gap: 14px;
  overflow: hidden;
  min-height: 0;
}

/* ── Left: group panel ── */
.group-panel {
  width: 220px;
  flex-shrink: 0;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.group-panel-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 14px 10px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--cv-text-muted);
  border-bottom: 1px solid var(--cv-border);
  flex-shrink: 0;
}

.group-count {
  background: var(--cv-surface2);
  border-radius: 10px;
  padding: 0 6px;
  font-size: 10px;
}

.group-item {
  padding: 9px 14px;
  cursor: pointer;
  border-left: 2px solid transparent;
  transition: background 0.12s, border-color 0.12s;
}

.group-item + .group-item { border-top: 1px solid var(--cv-border); }

.group-item:hover { background: var(--cv-surface2); }

.group-item.active {
  background: rgba(88, 101, 242, 0.08);
  border-left-color: var(--cv-accent);
}

.gi-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--cv-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}

.group-item.active .gi-name { color: var(--cv-accent); }

.gi-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.gi-type {
  background: var(--cv-accent-soft);
  color: var(--cv-accent);
  border-radius: 4px;
  padding: 0 5px;
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  flex-shrink: 0;
}

.gi-current {
  color: var(--cv-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.group-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── Right: node panel ── */
.node-panel {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius-sm);
  flex-shrink: 0;
}

.nph-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.nph-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--cv-text);
}

.nph-type {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--cv-accent);
  background: var(--cv-accent-soft);
  padding: 1px 7px;
  border-radius: 4px;
}

.nph-current {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: #67c23a;
}

.nph-count {
  font-size: 11px;
  color: var(--cv-text-muted);
}

/* ── Node grid ── */
.node-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(158px, 1fr));
  gap: 10px;
}

.node-no-result,
.node-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

/* ── Node card ── */
.node-card {
  position: relative;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius-sm);
  padding: 11px 11px 9px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-height: 106px;
  transition: border-color 0.12s, background 0.12s, box-shadow 0.12s;
  overflow: hidden;
}

.node-card:hover {
  border-color: rgba(88, 101, 242, 0.45);
  background: var(--cv-surface2);
}

.node-card.selected {
  border-color: #67c23a;
  background: rgba(103, 194, 58, 0.05);
  box-shadow: 0 0 0 1px rgba(103, 194, 58, 0.2);
}

.node-card.dimmed { opacity: 0.75; }
.node-card.dimmed:hover { opacity: 1; }

/* Selected corner badge */
.selected-corner {
  position: absolute;
  top: 0;
  right: 0;
  width: 20px;
  height: 20px;
  background: #67c23a;
  border-radius: 0 var(--cv-radius-sm) 0 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 10px;
}

/* Type row */
.nc-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.nc-type {
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--cv-text-muted);
}

/* Per-type colors */
.nc-type.vmess       { background: rgba(88,101,242,0.2);   color: #818cf8; }
.nc-type.vless       { background: rgba(139,92,246,0.2);   color: #a78bfa; }
.nc-type.trojan      { background: rgba(239,68,68,0.15);   color: #f87171; }
.nc-type.ss,
.nc-type.shadowsocks { background: rgba(16,185,129,0.15);  color: #34d399; }
.nc-type.ssr         { background: rgba(20,184,166,0.15);  color: #2dd4bf; }
.nc-type.http        { background: rgba(245,158,11,0.15);  color: #fbbf24; }
.nc-type.socks5      { background: rgba(59,130,246,0.15);  color: #60a5fa; }
.nc-type.direct      { background: rgba(34,197,94,0.15);   color: #4ade80; }
.nc-type.reject      { background: rgba(239,68,68,0.1);    color: #f87171; }

/* Test button */
.nc-test-btn {
  background: none;
  border: none;
  padding: 2px;
  cursor: pointer;
  color: var(--cv-text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 13px;
  line-height: 1;
  transition: color 0.1s, background 0.1s;
}

.nc-test-btn:hover {
  color: var(--cv-accent);
  background: var(--cv-accent-soft);
}

/* Node name */
.nc-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--cv-text);
  flex: 1;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  line-height: 1.45;
}

/* Delay */
.nc-bottom {
  display: flex;
  justify-content: flex-end;
  margin-top: auto;
}

.delay-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 20px;
  letter-spacing: 0.02em;
}

.delay-pill.untested { background: rgba(255,255,255,0.05); color: rgba(255,255,255,0.25); }
.delay-pill.testing  { background: rgba(88,101,242,0.15);  color: #818cf8; }
.delay-pill.fast     { background: rgba(34,197,94,0.2);    color: #4ade80; }
.delay-pill.medium   { background: rgba(234,179,8,0.2);    color: #facc15; }
.delay-pill.slow     { background: rgba(249,115,22,0.18);  color: #fb923c; }
.delay-pill.timeout  { background: rgba(239,68,68,0.15);   color: #f87171; }
</style>
