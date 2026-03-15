<template>
  <div class="profiles-view">

    <!-- Add profile bar -->
    <div class="add-bar">
      <el-input
        v-model="newUrl"
        placeholder="Paste subscription URL here..."
        clearable
        style="flex:1"
        :disabled="busy"
        @keyup.enter="addProfile"
      >
        <template #prefix><el-icon><Link /></el-icon></template>
      </el-input>
      <el-input v-model="newName" placeholder="Name (optional)" style="width:180px" :disabled="busy" />
      <el-button type="primary" :loading="busy" @click="addProfile">
        <el-icon><Plus /></el-icon>Add
      </el-button>
    </div>

    <!-- Step progress (shown while adding) -->
    <div v-if="busy" class="step-bar">
      <div v-for="step in steps" :key="step.key" class="step-item" :class="step.status">
        <el-icon v-if="step.status === 'done'"><CircleCheck /></el-icon>
        <el-icon v-else-if="step.status === 'error'"><CircleClose /></el-icon>
        <span v-else-if="step.status === 'active'" class="step-spinner" />
        <span v-else class="step-dot" />
        <span>{{ step.label }}</span>
      </div>
    </div>

    <!-- Profile grid -->
    <div class="profile-grid">
      <div
        v-for="src in sources"
        :key="src.id"
        class="profile-card"
        :class="{ disabled: !src.enabled, active: activeProfileId === src.id, loading: fetchingId === src.id }"
        @click="fetchAndApply(src)"
      >
        <!-- Active badge -->
        <div v-if="activeProfileId === src.id" class="active-badge">
          <el-icon><Select /></el-icon>
          <span>Active</span>
        </div>

        <!-- Card header: name + type -->
        <div class="card-top">
          <div class="card-avatar" :class="src.type">
            <el-icon v-if="src.type === 'http'"><Connection /></el-icon>
            <el-icon v-else><Document /></el-icon>
          </div>
          <div class="card-title-block">
            <div class="card-name">{{ src.name }}</div>
            <el-tag size="small" :type="src.type === 'http' ? 'primary' : 'success'" class="card-type-tag">
              {{ src.type }}
            </el-tag>
          </div>
        </div>

        <!-- URL -->
        <div class="card-url">{{ src.url || src.path }}</div>

        <!-- Meta row -->
        <div class="card-meta">
          <span v-if="src.last_fetch" class="meta-item">
            <el-icon><Timer /></el-icon>{{ formatRelative(src.last_fetch) }}
          </span>
          <span v-if="src.content_size" class="meta-item">
            <el-icon><Files /></el-icon>{{ formatSize(src.content_size) }}
          </span>
          <el-tag v-if="!src.last_fetch" type="warning" size="small">No cache</el-tag>
        </div>

        <!-- Progress bar while fetching -->
        <el-progress
          v-if="fetchingId === src.id"
          :percentage="100"
          status="striped"
          :striped-flow="true"
          :duration="3"
          :show-text="false"
          class="card-progress"
        />

        <!-- Actions footer -->
        <div class="card-footer" @click.stop>
          <el-tooltip content="Enable / Disable">
            <el-switch v-model="src.enabled" size="small" @change="toggleEnabled(src)" />
          </el-tooltip>
          <div class="footer-right">
            <el-tooltip content="Refresh & apply">
              <el-button link :loading="fetchingId === src.id" :disabled="busy" @click="fetchAndApply(src)">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="Delete">
              <el-button link type="danger" :disabled="busy" @click="deleteProfile(src.id)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <el-empty
        v-if="sources.length === 0 && !loading"
        description="No profiles yet. Paste a subscription URL above."
        class="grid-empty"
      />
    </div>

    <!-- Revisions -->
    <el-card class="revisions-card">
      <template #header>
        <div class="card-header-row">
          <span>历史配置版本</span>
          <el-button link @click="loadRevisions"><el-icon><Refresh /></el-icon></el-button>
        </div>
      </template>
      <el-table :data="revisions" size="small">
        <el-table-column prop="version" label="版本" width="60" />
        <el-table-column prop="created_by" label="操作人" width="90" />
        <el-table-column label="时间" width="165">
          <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="Hash" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="rev-hash">{{ row.source_hash }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" class-name="op-col">
          <template #default="{ row }">
            <div class="op-btns">
              <el-button size="small" @click="viewRevision(row)">查看</el-button>
              <el-button size="small" type="primary" plain @click="rollback(row)">应用</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Revision dialog -->
    <el-dialog v-model="revDialogVisible" title="Config Content" width="70%">
      <el-input v-model="revContent" type="textarea" :rows="25" readonly style="font-family:monospace;font-size:12px" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Refresh, Delete, Link, CircleCheck, CircleClose,
  Select, Connection, Document, Timer, Files
} from '@element-plus/icons-vue'
import * as sourceApi from '@/api/source'
import { quickApply, listRevisions, rollbackRevision } from '@/api/config'
import { useProxyStore } from '@/stores/proxy'
import type { Source } from '@/api/source'

const proxyStore = useProxyStore()

const sources = ref<any[]>([])
const loading = ref(false)
const busy = ref(false)
const fetchingId = ref<number | null>(null)
const newUrl = ref('')
const newName = ref('')
const revisions = ref<any[]>([])
const revDialogVisible = ref(false)
const revContent = ref('')

// ── Active profile selection (persisted) ─────────────────────────────────────
const ACTIVE_KEY = 'active_profile_id'
const activeProfileId = ref<number | null>(
  Number(localStorage.getItem(ACTIVE_KEY)) || null
)
const markActive = (id: number | null) => {
  activeProfileId.value = id
  if (id == null) localStorage.removeItem(ACTIVE_KEY)
  else localStorage.setItem(ACTIVE_KEY, String(id))
}

// ── Step pipeline UI ─────────────────────────────────────────────────────────
type StepStatus = 'pending' | 'active' | 'done' | 'error'
interface Step { key: string; label: string; status: StepStatus }

const steps = ref<Step[]>([])

function initSteps(labels: string[]) {
  steps.value = labels.map((label, i) => ({
    key: String(i),
    label,
    status: i === 0 ? 'active' : 'pending',
  }))
}

function advanceStep(errorMsg?: string) {
  const idx = steps.value.findIndex(s => s.status === 'active')
  if (idx < 0) return
  if (errorMsg) { steps.value[idx].status = 'error'; return }
  steps.value[idx].status = 'done'
  if (idx + 1 < steps.value.length) steps.value[idx + 1].status = 'active'
}

// ── Helpers ──────────────────────────────────────────────────────────────────
const formatSize = (bytes: number) => {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / 1024 / 1024).toFixed(1)}MB`
}

const formatRelative = (iso: string) => {
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

// ── Data loaders ─────────────────────────────────────────────────────────────
const loadSources = async () => {
  loading.value = true
  try { sources.value = await sourceApi.listSources() }
  finally { loading.value = false }
}

const loadRevisions = async () => {
  try { revisions.value = await listRevisions(10) } catch { /* non-critical */ }
}

// ── Core pipeline ────────────────────────────────────────────────────────────
const runApplyPipeline = async () => {
  const res = await quickApply()
  await loadRevisions()
  await proxyStore.fetchProxies(true)
  return res
}

// ── Add profile ───────────────────────────────────────────────────────────────
const addProfile = async () => {
  const url = newUrl.value.trim()
  if (!url) { ElMessage.warning('Please enter a URL'); return }

  busy.value = true
  initSteps(['Creating source', 'Fetching subscription', 'Applying config'])

  try {
    const src = await sourceApi.createSource({
      name: newName.value.trim() || new URL(url).hostname,
      type: 'http',
      url,
      update_interval: 3600,
      enabled: true,
      priority: 0,
    } as any)
    advanceStep()

    await sourceApi.fetchSource(src.id)
    await loadSources()
    advanceStep()

    await runApplyPipeline()
    advanceStep()

    markActive(src.id)
    newUrl.value = ''
    newName.value = ''
    ElMessage.success('Profile added and applied')
  } catch (e: any) {
    advanceStep(e.message || 'Failed')
    ElMessage.error(e.message || 'Failed to add profile')
  } finally {
    busy.value = false
    steps.value = []
  }
}

// ── Fetch & apply ─────────────────────────────────────────────────────────────
const fetchAndApply = async (src: Source) => {
  if (fetchingId.value) return
  fetchingId.value = src.id
  try {
    await sourceApi.fetchSource(src.id)
    await loadSources()
    await runApplyPipeline()
    markActive(src.id)
    ElMessage.success(`${src.name} updated and applied`)
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed')
  } finally {
    fetchingId.value = null
  }
}

// ── Other actions ─────────────────────────────────────────────────────────────
const toggleEnabled = async (src: any) => {
  try {
    await sourceApi.updateSource(src.id, src)
  } catch (e: any) {
    ElMessage.error(e.message || 'Update failed')
    src.enabled = !src.enabled
  }
}

const deleteProfile = async (id: number) => {
  try {
    await ElMessageBox.confirm('Delete this profile?', 'Confirm', { type: 'warning' })
  } catch {
    return
  }
  await sourceApi.deleteSource(id)
  if (activeProfileId.value === id) markActive(null)
  await loadSources()
  ElMessage.success('Deleted')
}

const viewRevision = (rev: any) => {
  revContent.value = rev.content
  revDialogVisible.value = true
}

const rollback = async (rev: any) => {
  try {
    await ElMessageBox.confirm(`Apply revision ${rev.version}?`, 'Confirm', { type: 'warning' })
  } catch {
    return
  }
  await rollbackRevision(rev.id)
  markActive(null)
  await proxyStore.fetchProxies(true)
  ElMessage.success('Applied')
  await loadRevisions()
}

onMounted(() => {
  loadSources()
  loadRevisions()
})
</script>

<style scoped>
.profiles-view { display: flex; flex-direction: column; gap: 16px; }

/* ── Add bar ── */
.add-bar {
  display: flex;
  gap: 8px;
  align-items: center;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 14px 16px;
}

/* ── Step bar ── */
.step-bar {
  display: flex;
  align-items: center;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 12px 20px;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--cv-text-muted);
  flex: 1;
  position: relative;
}

.step-item:not(:last-child)::after {
  content: '';
  position: absolute;
  right: 0; top: 50%;
  width: 24px; height: 1px;
  background: var(--cv-border);
  transform: translateY(-50%);
}

.step-item.active  { color: #5865f2; font-weight: 600; }
.step-item.done    { color: #67c23a; }
.step-item.error   { color: #f56c6c; }
.step-item.pending { opacity: 0.45; }

.step-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.step-spinner {
  width: 14px; height: 14px;
  border: 2px solid #5865f2;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* ── Profile grid ── */
.profile-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.grid-empty {
  grid-column: 1 / -1;
}

/* ── Profile card ── */
.profile-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 16px;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.1s;
  min-height: 160px;
}

.profile-card:hover {
  border-color: rgba(88, 101, 242, 0.5);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  transform: translateY(-1px);
}

.profile-card.disabled { opacity: 0.5; }

.profile-card.active {
  border-color: rgba(103, 194, 58, 0.7);
  box-shadow: 0 0 0 1px rgba(103, 194, 58, 0.3), 0 4px 20px rgba(103, 194, 58, 0.1);
}

/* ── Active badge ── */
.active-badge {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: #67c23a;
  background: rgba(103, 194, 58, 0.12);
  border: 1px solid rgba(103, 194, 58, 0.3);
  border-radius: 20px;
  padding: 2px 8px 2px 6px;
}

/* ── Card top: avatar + name ── */
.card-top {
  display: flex;
  align-items: center;
  gap: 12px;
  /* leave room for active badge on the right */
  padding-right: 80px;
}

.card-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--cv-radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.card-avatar.http {
  background: rgba(88, 101, 242, 0.15);
  color: #5865f2;
}

.card-avatar.file,
.card-avatar.local {
  background: rgba(103, 194, 58, 0.12);
  color: #67c23a;
}

.card-title-block {
  min-width: 0;
  flex: 1;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--cv-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}

.card-type-tag { font-size: 10px; }

/* ── URL ── */
.card-url {
  font-size: 11px;
  color: var(--cv-text-muted);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

/* ── Meta ── */
.card-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--cv-text-muted);
}

.meta-item .el-icon { font-size: 12px; }

/* ── Progress ── */
.card-progress { margin-top: 0; }

/* ── Footer ── */
.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--cv-border);
}

.footer-right {
  display: flex;
  align-items: center;
  gap: 2px;
}

/* ── Revisions ── */
.revisions-card { margin-top: 0; }

.card-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.rev-hash {
  font-family: monospace;
  font-size: 11px;
  color: var(--cv-text-muted);
}

.op-btns {
  display: flex;
  gap: 6px;
  flex-wrap: nowrap;
  align-items: center;
}
</style>
