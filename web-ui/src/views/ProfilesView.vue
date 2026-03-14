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
      <div
        v-for="step in steps"
        :key="step.key"
        class="step-item"
        :class="step.status"
      >
        <el-icon v-if="step.status === 'done'"><CircleCheck /></el-icon>
        <el-icon v-else-if="step.status === 'error'"><CircleClose /></el-icon>
        <span v-else-if="step.status === 'active'" class="step-spinner" />
        <span v-else class="step-dot" />
        <span>{{ step.label }}</span>
      </div>
    </div>

    <!-- Profile list -->
    <div class="profile-list">
      <div
        v-for="src in sources"
        :key="src.id"
        class="profile-card"
        :class="{ disabled: !src.enabled }"
      >
        <div class="profile-main">
          <div class="profile-info">
            <div class="profile-name">{{ src.name }}</div>
            <div class="profile-url">{{ src.url || src.path }}</div>
            <div class="profile-meta">
              <el-tag size="small" :type="src.type === 'http' ? 'primary' : 'success'">{{ src.type }}</el-tag>
              <span v-if="src.last_fetch" class="meta-text">
                {{ formatSize(src.content_size) }} · Updated {{ formatRelative(src.last_fetch) }}
              </span>
              <el-tag v-else type="warning" size="small">No cache</el-tag>
            </div>
          </div>

          <div class="profile-actions">
            <el-switch
              v-model="src.enabled"
              size="small"
              @change="toggleEnabled(src)"
            />
            <el-tooltip content="Fetch & apply">
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

        <!-- Progress bar while fetching -->
        <el-progress
          v-if="fetchingId === src.id"
          :percentage="100"
          status="striped"
          :striped-flow="true"
          :duration="3"
          :show-text="false"
          style="margin-top:8px"
        />
      </div>

      <el-empty v-if="sources.length === 0 && !loading" description="No profiles yet. Paste a subscription URL above." />
    </div>

    <!-- Revisions -->
    <el-card class="revisions-card">
      <template #header>
        <div class="card-header-row">
          <span>Config Revisions</span>
          <el-button link @click="loadRevisions"><el-icon><Refresh /></el-icon></el-button>
        </div>
      </template>
      <el-table :data="revisions" size="small">
        <el-table-column prop="version" label="Ver" width="60" />
        <el-table-column prop="created_by" label="By" width="80" />
        <el-table-column label="Time" width="160">
          <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column prop="source_hash" label="Hash" show-overflow-tooltip />
        <el-table-column label="" width="120">
          <template #default="{ row }">
            <el-button size="small" @click="viewRevision(row)">View</el-button>
            <el-button size="small" @click="rollback(row)">Apply</el-button>
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
import { Plus, Refresh, Delete, Link, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import * as sourceApi from '@/api/source'
import { quickApply, listRevisions, rollbackRevision } from '@/api/config'
import { useProxyStore } from '@/stores/proxy'
import type { Source } from '@/api/source'

const proxyStore = useProxyStore()

const sources = ref<any[]>([])
const loading = ref(false)
const busy = ref(false)       // locked while add pipeline runs
const fetchingId = ref<number | null>(null)
const newUrl = ref('')
const newName = ref('')
const revisions = ref<any[]>([])
const revDialogVisible = ref(false)
const revContent = ref('')

// ── Step pipeline UI ────────────────────────────────────────────────────────
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
  if (errorMsg) {
    steps.value[idx].status = 'error'
    return
  }
  steps.value[idx].status = 'done'
  if (idx + 1 < steps.value.length) {
    steps.value[idx + 1].status = 'active'
  }
}

// ── Helpers ─────────────────────────────────────────────────────────────────
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

// ── Core pipeline: fetch + apply + refresh proxies ───────────────────────────
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
    // Step 1: create
    const src = await sourceApi.createSource({
      name: newName.value.trim() || new URL(url).hostname,
      type: 'http',
      url,
      update_interval: 3600,
      enabled: true,
      priority: 0,
    } as any)
    advanceStep()

    // Step 2: fetch
    await sourceApi.fetchSource(src.id)
    await loadSources()
    advanceStep()

    // Step 3: apply
    await runApplyPipeline()
    advanceStep()

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

// ── Fetch & auto-apply ────────────────────────────────────────────────────────
const fetchAndApply = async (src: Source) => {
  fetchingId.value = src.id
  try {
    await sourceApi.fetchSource(src.id)
    await loadSources()
    await runApplyPipeline()
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
    await sourceApi.deleteSource(id)
    await loadSources()
    ElMessage.success('Deleted')
  } catch { /* cancel */ }
}

const viewRevision = (rev: any) => {
  revContent.value = rev.content
  revDialogVisible.value = true
}

const rollback = async (rev: any) => {
  try {
    await ElMessageBox.confirm(`Apply revision ${rev.version}?`, 'Confirm', { type: 'warning' })
    await rollbackRevision(rev.id)
    await proxyStore.fetchProxies(true)
    ElMessage.success('Applied')
    await loadRevisions()
  } catch { /* cancel */ }
}

onMounted(() => {
  loadSources()
  loadRevisions()
})
</script>

<style scoped>
.profiles-view { display: flex; flex-direction: column; gap: 16px; }

.add-bar {
  display: flex;
  gap: 8px;
  align-items: center;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 14px 16px;
}

/* Step progress bar */
.step-bar {
  display: flex;
  align-items: center;
  gap: 0;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 12px 20px;
  overflow: hidden;
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
  right: 0;
  top: 50%;
  width: 24px;
  height: 1px;
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

.profile-list { display: flex; flex-direction: column; gap: 10px; }

.profile-card {
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 14px 16px;
  transition: border-color 0.15s;
}

.profile-card:hover { border-color: rgba(88,101,242,0.4); }
.profile-card.disabled { opacity: 0.5; }

.profile-main { display: flex; align-items: center; gap: 12px; }

.profile-info { flex: 1; min-width: 0; }

.profile-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--cv-text);
  margin-bottom: 3px;
}

.profile-url {
  font-size: 12px;
  color: var(--cv-text-muted);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 6px;
}

.profile-meta { display: flex; align-items: center; gap: 8px; }

.meta-text { font-size: 12px; color: var(--cv-text-muted); }

.profile-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }

.revisions-card { margin-top: 0; }

.card-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
