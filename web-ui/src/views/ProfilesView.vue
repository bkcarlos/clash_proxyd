<template>
  <div class="profiles-view">
    <!-- Add profile bar -->
    <div class="add-bar">
      <el-input
        v-model="newUrl"
        placeholder="Paste subscription URL here..."
        clearable
        style="flex:1"
        @keyup.enter="addProfile"
      >
        <template #prefix><el-icon><Link /></el-icon></template>
      </el-input>
      <el-input v-model="newName" placeholder="Name (optional)" style="width:180px" />
      <el-button type="primary" :loading="adding" @click="addProfile">
        <el-icon><Plus /></el-icon>Add
      </el-button>
      <el-button :loading="applying" @click="applyAll">
        <el-icon><Promotion /></el-icon>Apply All
      </el-button>
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
            <el-tooltip content="Fetch & update cache">
              <el-button link :loading="fetchingId === src.id" @click="fetchProfile(src)">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="Delete">
              <el-button link type="danger" @click="deleteProfile(src.id)">
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
import { Plus, Refresh, Delete, Link, Promotion } from '@element-plus/icons-vue'
import * as sourceApi from '@/api/source'
import { quickApply, listRevisions, rollbackRevision } from '@/api/config'
import type { Source } from '@/api/source'

const sources = ref<any[]>([])
const loading = ref(false)
const adding = ref(false)
const applying = ref(false)
const fetchingId = ref<number | null>(null)
const newUrl = ref('')
const newName = ref('')
const revisions = ref<any[]>([])
const revDialogVisible = ref(false)
const revContent = ref('')

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

const loadSources = async () => {
  loading.value = true
  try {
    sources.value = await sourceApi.listSources()
  } finally {
    loading.value = false
  }
}

const loadRevisions = async () => {
  try {
    revisions.value = await listRevisions(10)
  } catch { /* non-critical */ }
}

const addProfile = async () => {
  const url = newUrl.value.trim()
  if (!url) { ElMessage.warning('Please enter a URL'); return }
  adding.value = true
  try {
    await sourceApi.createSource({
      name: newName.value.trim() || new URL(url).hostname,
      type: 'http',
      url,
      update_interval: 3600,
      enabled: true,
      priority: 0,
    } as any)
    newUrl.value = ''
    newName.value = ''
    await loadSources()
    await applyAll(true)
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to add profile')
  } finally {
    adding.value = false
  }
}

const fetchProfile = async (src: Source) => {
  fetchingId.value = src.id
  try {
    await sourceApi.fetchSource(src.id)
    ElMessage.success(`${src.name} updated`)
    await loadSources()
  } catch (e: any) {
    ElMessage.error(e.message || 'Fetch failed')
  } finally {
    fetchingId.value = null
  }
}

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

const applyAll = async (silent = false) => {
  applying.value = true
  try {
    const res = await quickApply()
    ElMessage.success(`Applied — ${res.data?.sources?.length ?? 0} profile(s)`)
    await loadRevisions()
  } catch (e: any) {
    if (!silent) ElMessage.error(e.message || 'Apply failed')
  } finally {
    applying.value = false
  }
}

const viewRevision = (rev: any) => {
  revContent.value = rev.content
  revDialogVisible.value = true
}

const rollback = async (rev: any) => {
  try {
    await ElMessageBox.confirm(`Apply revision ${rev.version}?`, 'Confirm', { type: 'warning' })
    await rollbackRevision(rev.id)
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
