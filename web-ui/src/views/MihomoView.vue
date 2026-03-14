<template>
  <div class="mihomo-view">
    <div class="page-header">
      <h1>Mihomo Management</h1>
      <el-button :loading="statusLoading || versionsLoading" @click="refresh">
        <el-icon><Refresh /></el-icon>
        Refresh
      </el-button>
    </div>

    <!-- Setup guide: shown when binary is not installed -->
    <el-alert
      v-if="status && !status.installed && !statusLoading"
      type="warning"
      title="Mihomo not installed"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
    >
      <template #default>
        <p style="margin: 4px 0 0">Mihomo binary was not found at <code>{{ status.binary_path }}</code>. To get started:</p>
        <ol style="margin: 8px 0 0; padding-left: 20px; line-height: 1.8">
          <li>Select a version (or leave blank for latest) in the <strong>Install / Update</strong> panel below.</li>
          <li>Click <strong>Install Mihomo</strong> — the binary will be downloaded automatically.</li>
          <li>Once installed, use <strong>Config → Generate → Apply</strong> to create a runtime config.</li>
          <li>Then click <strong>Start</strong> above to launch the proxy.</li>
        </ol>
        <p style="margin: 8px 0 0; color: #909399; font-size: 12px">
          Binary path is configured in your <code>config.yaml</code> under <code>mihomo.binary_path</code>.
        </p>
      </template>
    </el-alert>

    <!-- Installation Status -->
    <el-row :gutter="20" class="status-row">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>Installation Status</span>
              <el-tag v-if="status" :type="status.installed ? 'success' : 'danger'" size="small">
                {{ status.installed ? 'Installed' : 'Not Installed' }}
              </el-tag>
            </div>
          </template>

          <el-skeleton v-if="statusLoading" :rows="4" animated />

          <template v-else-if="status">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Binary Path">
                <el-text class="mono" size="small">{{ status.binary_path || '—' }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="Current Version">
                <el-tag v-if="status.current_version" type="info">{{ status.current_version }}</el-tag>
                <el-text v-else type="info">Not detected</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="Latest Version">
                <el-tag v-if="status.latest_version" :type="status.needs_update ? 'warning' : 'success'">
                  {{ status.latest_version }}
                </el-tag>
                <el-text v-else type="info">Unknown</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="Update Available">
                <el-tag v-if="status.needs_update" type="warning">Yes</el-tag>
                <el-tag v-else-if="status.installed" type="success">Up to date</el-tag>
                <el-tag v-else type="danger">Not installed</el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </template>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>Process Status</span>
              <el-tag v-if="status" :type="status.is_running ? 'success' : 'info'" size="small">
                {{ status.is_running ? 'Running' : 'Stopped' }}
              </el-tag>
            </div>
          </template>

          <el-skeleton v-if="statusLoading" :rows="3" animated />

          <template v-else-if="status">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="State">
                <el-badge :type="status.is_running ? 'success' : 'info'" is-dot>
                  <span>{{ status.is_running ? 'Running' : 'Stopped' }}</span>
                </el-badge>
              </el-descriptions-item>
              <el-descriptions-item label="PID">
                {{ status.pid > 0 ? status.pid : '—' }}
              </el-descriptions-item>
            </el-descriptions>

            <div class="process-controls">
              <el-button
                type="success"
                :disabled="status.is_running || !status.installed"
                :loading="controlLoading === 'start'"
                @click="control('start')"
              >
                <el-icon><VideoPlay /></el-icon>
                Start
              </el-button>
              <el-button
                type="warning"
                :disabled="!status.is_running"
                :loading="controlLoading === 'restart'"
                @click="control('restart')"
              >
                <el-icon><RefreshRight /></el-icon>
                Restart
              </el-button>
              <el-button
                type="danger"
                :disabled="!status.is_running"
                :loading="controlLoading === 'stop'"
                @click="control('stop')"
              >
                <el-icon><VideoPause /></el-icon>
                Stop
              </el-button>
            </div>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <!-- GeoIP Database (MMDB) -->
    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>GeoIP Database (MMDB)</span>
          <el-tag :type="mmdb?.exists ? 'success' : 'danger'" size="small">
            {{ mmdb?.exists ? 'Installed' : 'Not Found' }}
          </el-tag>
        </div>
      </template>

      <el-skeleton v-if="mmdbLoading" :rows="2" animated />
      <template v-else-if="mmdb">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Path">
            <el-text class="mono" size="small">{{ mmdb.path }}</el-text>
          </el-descriptions-item>
          <el-descriptions-item label="Size">
            {{ mmdb.exists ? formatBytes(mmdb.size) : '—' }}
          </el-descriptions-item>
        </el-descriptions>

        <div style="margin-top:14px;display:flex;gap:10px;align-items:center">
          <el-input
            v-model="mmdbUrl"
            placeholder="Custom URL (leave empty for MetaCubeX default)"
            clearable
            style="flex:1"
            :disabled="mmdbDownloading"
          />
          <el-button
            type="primary"
            :loading="mmdbDownloading"
            @click="downloadMMDB"
          >
            <el-icon><Download /></el-icon>
            {{ mmdb.exists ? 'Re-download' : 'Download' }}
          </el-button>
        </div>
        <el-text v-if="mmdbDownloading" type="info" size="small" style="margin-top:8px;display:block">
          Downloading... this may take a few minutes.
        </el-text>
      </template>
    </el-card>

    <!-- Install / Update -->
    <el-card style="margin-top: 20px">
      <template #header>
        <span>Install / Update</span>
      </template>

      <div :class="{ 'form-disabled': formDisabled }">
        <el-form label-width="160px">
          <el-form-item label="Target Version">
            <el-select
              v-model="targetVersion"
              placeholder="Latest (auto)"
              clearable
              filterable
              style="width: 260px"
              :loading="versionsLoading"
              :disabled="formDisabled"
              @visible-change="onVersionDropdownOpen"
            >
              <el-option
                v-for="v in versionOptions"
                :key="v.value"
                :label="v.label"
                :value="v.value"
              />
            </el-select>
            <el-button
              link
              :loading="versionsLoading"
              :disabled="formDisabled"
              style="margin-left: 8px"
              @click="loadVersions"
            >
              <el-icon><Refresh /></el-icon>
            </el-button>
            <el-text type="info" size="small" style="margin-left: 6px">
              {{ status?.latest_version ? 'Latest: ' + status.latest_version : '' }}
            </el-text>
          </el-form-item>

          <el-form-item label="Force Reinstall">
            <el-switch v-model="forceInstall" :disabled="formDisabled" />
            <el-text type="info" size="small" style="margin-left: 10px">
              Install even if already at the target version
            </el-text>
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              :loading="jobRunning"
              :disabled="formDisabled || !canInstall"
              @click="install"
            >
              <el-icon><Download /></el-icon>
              {{ jobRunning ? 'Installing…' : (status?.installed ? 'Update Mihomo' : 'Install Mihomo') }}
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- Progress panel -->
      <div v-if="job" class="progress-panel">
        <div class="progress-header">
          <span class="progress-stage">{{ stageLabel }}</span>
          <el-tag :type="stageTagType" size="small">{{ job.stage }}</el-tag>
        </div>

        <el-progress
          :percentage="progressPercent"
          :status="progressStatus"
          :striped="jobRunning"
          :striped-flow="jobRunning"
          :duration="6"
          style="margin: 10px 0"
        />

        <el-text size="small" type="info">{{ job.message }}</el-text>

        <el-alert
          v-if="job.stage === 'done' && !jobRunning"
          type="success"
          :title="job.new_version ? `Installed ${job.new_version}` : 'Already up to date'"
          :description="job.new_version && job.old_version ? `Updated from ${job.old_version} → ${job.new_version}` : ''"
          show-icon
          style="margin-top: 10px"
        />
        <el-alert
          v-if="job.stage === 'error'"
          type="error"
          title="Installation failed"
          :description="job.error"
          show-icon
          style="margin-top: 10px"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh, Download, VideoPlay, VideoPause, RefreshRight
} from '@element-plus/icons-vue'
import {
  getMihomoInstallStatus,
  getMihomoVersionList,
  startInstallJob,
  getInstallProgress,
  controlMihomo as apiControlMihomo,
  type MihomoInstallStatus,
  type InstallProgress
} from '@/api/proxy'
import request from '@/api/request'

const status = ref<MihomoInstallStatus | null>(null)
const statusLoading = ref(false)
const controlLoading = ref<string | null>(null)
const targetVersion = ref('')
const forceInstall = ref(false)

const versions = ref<string[]>([])
const versionsLoading = ref(false)
const versionsLoaded = ref(false)

const job = ref<InstallProgress | null>(null)
const initializing = ref(true)
let pollTimer: ReturnType<typeof setInterval> | null = null

// MMDB
const mmdb = ref<{ exists: boolean; path: string; size: number } | null>(null)
const mmdbLoading = ref(false)
const mmdbDownloading = ref(false)
const mmdbUrl = ref('')

const loadMMDB = async () => {
  mmdbLoading.value = true
  try {
    mmdb.value = await request({ url: '/proxy/mihomo/mmdb', method: 'GET' }) as any
  } finally {
    mmdbLoading.value = false
  }
}

const downloadMMDB = async () => {
  mmdbDownloading.value = true
  try {
    await request({
      url: '/proxy/mihomo/mmdb/download',
      method: 'POST',
      data: { url: mmdbUrl.value || '' },
      timeout: 15 * 60 * 1000
    })
    ElMessage.success('MMDB downloaded successfully')
    await loadMMDB()
  } catch (e: any) {
    ElMessage.error(e.message || 'Download failed')
  } finally {
    mmdbDownloading.value = false
  }
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1048576).toFixed(1)} MB`
}

// ── computed ──────────────────────────────────────────────────────────────────

const versionOptions = computed(() => {
  const latest = status.value?.latest_version
  return versions.value.map(v => ({
    value: v,
    label: latest && v === latest ? `${v} (latest)` : v
  }))
})

const jobRunning = computed(() => job.value?.running === true)
const formDisabled = computed(() => initializing.value || jobRunning.value)
const canInstall = computed(() => status.value !== null && !formDisabled.value)

const progressPercent = computed(() => {
  if (!job.value) return 0
  if (job.value.stage === 'done') return 100
  if (job.value.stage === 'error') return job.value.percent ?? 0
  // Map stages to rough overall percent
  const stageBase: Record<string, number> = {
    fetch_release: 2,
    download: 5,    // actual download percent added below
    extract: 90,
    install: 94,
    restart: 97,
    done: 100,
  }
  const base = stageBase[job.value.stage] ?? 0
  if (job.value.stage === 'download') {
    return Math.min(base + Math.floor(job.value.percent * 0.85), 89)
  }
  return base
})

const progressStatus = computed(() => {
  if (!job.value) return undefined
  if (job.value.stage === 'error') return 'exception'
  if (job.value.stage === 'done' && !job.value.running) return 'success'
  return undefined
})

const stageLabel = computed(() => {
  const map: Record<string, string> = {
    fetch_release: 'Fetching release info',
    download:      'Downloading binary',
    extract:       'Extracting archive',
    install:       'Installing binary',
    restart:       'Restarting mihomo',
    done:          'Complete',
    error:         'Failed',
  }
  return map[job.value?.stage ?? ''] ?? job.value?.stage ?? ''
})

const stageTagType = computed(() => {
  if (job.value?.stage === 'error') return 'danger'
  if (job.value?.stage === 'done') return 'success'
  return 'primary'
})

// ── status & versions ─────────────────────────────────────────────────────────

const refresh = async () => {
  await loadStatus()
  loadMMDB()
  if (versionsLoaded.value) await loadVersions()
}

const loadStatus = async () => {
  statusLoading.value = true
  try {
    status.value = await getMihomoInstallStatus()
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to load status')
  } finally {
    statusLoading.value = false
  }
}

const loadVersions = async () => {
  versionsLoading.value = true
  try {
    const res = await getMihomoVersionList()
    versions.value = res.versions
    versionsLoaded.value = true
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to load version list')
  } finally {
    versionsLoading.value = false
  }
}

const onVersionDropdownOpen = (visible: boolean) => {
  if (visible && !versionsLoaded.value) loadVersions()
}

// ── process control ───────────────────────────────────────────────────────────

const control = async (action: 'start' | 'stop' | 'restart') => {
  controlLoading.value = action
  try {
    await apiControlMihomo(action)
    ElMessage.success(`Mihomo ${action} successful`)
    await loadStatus()
  } catch (e: any) {
    ElMessage.error(e.message || `${action} failed`)
  } finally {
    controlLoading.value = null
  }
}

// ── install job ───────────────────────────────────────────────────────────────

const pollProgress = async () => {
  try {
    const p = await getInstallProgress()
    job.value = p
    if (!p.running) {
      stopPolling()
      await loadStatus()
      if (p.stage === 'error') {
        ElMessage.error('Installation failed: ' + (p.error || p.message))
      } else if (p.stage === 'done') {
        ElMessage.success(p.new_version ? `Installed ${p.new_version}` : 'Already up to date')
      }
    }
  } catch {
    // ignore transient poll errors
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(pollProgress, 1000)
}

const stopPolling = () => {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

const install = async () => {
  try {
    await startInstallJob(targetVersion.value || undefined, forceInstall.value)
    // Seed job state immediately so UI shows the panel right away
    job.value = {
      running: true,
      stage: 'fetch_release',
      percent: 0,
      message: 'Starting…',
      error: '',
      started_at: new Date().toISOString(),
    }
    startPolling()
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to start install job')
  }
}

onMounted(async () => {
  await loadStatus()
  loadMMDB()
  // Resume progress state immediately — disables form until check completes
  const p = await getInstallProgress().catch(() => null)
  if (p) {
    job.value = p
    if (p.running) startPolling()
  }
  initializing.value = false
})

onUnmounted(stopPolling)
</script>

<style scoped>
.mihomo-view h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.status-row {
  margin-bottom: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.process-controls {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.mono {
  font-family: monospace;
  word-break: break-all;
}

.progress-panel {
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 16px;
  margin-top: 4px;
  background: #fafafa;
}

.progress-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.progress-stage {
  font-weight: 500;
  font-size: 14px;
}

.form-disabled {
  pointer-events: none;
  opacity: 0.45;
  user-select: none;
}
</style>
