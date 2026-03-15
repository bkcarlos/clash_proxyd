<template>
  <div class="mihomo-view">
    <div class="page-header">
      <h1>{{ t('mihomo.title') }}</h1>
      <el-button :loading="statusLoading || versionsLoading" @click="refresh">
        <el-icon><Refresh /></el-icon>
        {{ t('common.refresh') }}
      </el-button>
    </div>

    <!-- Setup guide: shown when binary is not installed -->
    <el-alert
      v-if="status && !status.installed && !statusLoading"
      type="warning"
      :title="t('mihomo.notInstalled')"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
    >
      <template #default>
        <p style="margin: 4px 0 0">{{ t('mihomo.notInstalledDesc') }} <code>{{ status.binary_path }}</code>. {{ t('common.loading') }}</p>
        <ol style="margin: 8px 0 0; padding-left: 20px; line-height: 1.8">
          <li>{{ t('mihomo.setupStep1') }}</li>
          <li>{{ t('mihomo.setupStep2') }}</li>
          <li>{{ t('mihomo.setupStep3') }}</li>
          <li>{{ t('mihomo.setupStep4') }}</li>
        </ol>
        <p style="margin: 8px 0 0; color: #909399; font-size: 12px">
          {{ t('mihomo.setupBinaryPath') }}
        </p>
      </template>
    </el-alert>

    <!-- Installation Status -->
    <el-row :gutter="20" class="status-row">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>{{ t('mihomo.installStatus') }}</span>
              <el-tag v-if="status" :type="status.installed ? 'success' : 'danger'" size="small">
                {{ status.installed ? t('mihomo.installed') : t('mihomo.notInstalledTag') }}
              </el-tag>
            </div>
          </template>

          <el-skeleton v-if="statusLoading" :rows="4" animated />

          <template v-else-if="status">
            <el-descriptions :column="1" border>
              <el-descriptions-item :label="t('mihomo.binaryPath')">
                <el-text class="mono" size="small">{{ status.binary_path || '—' }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item :label="t('mihomo.currentVersion')">
                <el-tag v-if="status.current_version" type="info">{{ status.current_version }}</el-tag>
                <el-text v-else type="info">{{ t('mihomo.notDetected') }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item :label="t('mihomo.latestVersion')">
                <el-tag v-if="status.latest_version" :type="status.needs_update ? 'warning' : 'success'">
                  {{ status.latest_version }}
                </el-tag>
                <el-text v-else type="info">{{ t('mihomo.unknownVersion') }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item :label="t('mihomo.updateAvailable')">
                <el-tag v-if="status.needs_update" type="warning">{{ t('mihomo.updateYes') }}</el-tag>
                <el-tag v-else-if="status.installed" type="success">{{ t('mihomo.upToDate') }}</el-tag>
                <el-tag v-else type="danger">{{ t('mihomo.notInstalledTag') }}</el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </template>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>{{ t('mihomo.processStatus') }}</span>
              <el-tag v-if="status" :type="status.is_running ? 'success' : 'info'" size="small">
                {{ status.is_running ? t('common.running') : t('common.stopped') }}
              </el-tag>
            </div>
          </template>

          <el-skeleton v-if="statusLoading" :rows="3" animated />

          <template v-else-if="status">
            <el-descriptions :column="1" border>
              <el-descriptions-item :label="t('mihomo.state')">
                <el-badge :type="status.is_running ? 'success' : 'info'" is-dot>
                  <span>{{ status.is_running ? t('common.running') : t('common.stopped') }}</span>
                </el-badge>
              </el-descriptions-item>
              <el-descriptions-item :label="t('mihomo.pid')">
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
                {{ t('mihomo.start') }}
              </el-button>
              <el-button
                type="warning"
                :disabled="!status.is_running"
                :loading="controlLoading === 'restart'"
                @click="control('restart')"
              >
                <el-icon><RefreshRight /></el-icon>
                {{ t('mihomo.restart') }}
              </el-button>
              <el-button
                type="danger"
                :disabled="!status.is_running"
                :loading="controlLoading === 'stop'"
                @click="control('stop')"
              >
                <el-icon><VideoPause /></el-icon>
                {{ t('mihomo.stop') }}
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
          <span>{{ t('mihomo.geoipDb') }}</span>
          <el-tag :type="mmdb?.exists ? 'success' : 'danger'" size="small">
            {{ mmdb?.exists ? t('mihomo.geoipInstalled') : t('mihomo.geoipNotFound') }}
          </el-tag>
        </div>
      </template>

      <el-skeleton v-if="mmdbLoading" :rows="2" animated />
      <template v-else-if="mmdb">
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="t('mihomo.path')">
            <el-text class="mono" size="small">{{ mmdb.path }}</el-text>
          </el-descriptions-item>
          <el-descriptions-item :label="t('mihomo.size')">
            {{ mmdb.exists ? formatBytes(mmdb.size) : '—' }}
          </el-descriptions-item>
        </el-descriptions>

        <el-tabs v-model="mmdbTab" style="margin-top:14px">
          <!-- Download from URL -->
          <el-tab-pane :label="t('mihomo.downloadUrl')" name="url">
            <div style="display:flex;gap:10px;align-items:center;padding:4px 0">
              <el-input
                v-model="mmdbUrl"
                :placeholder="t('mihomo.downloadUrlPlaceholder')"
                clearable
                style="flex:1"
                :disabled="mmdbDownloading"
              />
              <el-button type="primary" :loading="mmdbDownloading" @click="downloadMMDB">
                <el-icon><Download /></el-icon>
                {{ mmdb.exists ? t('mihomo.redownload') : t('mihomo.downloadBtn') }}
              </el-button>
            </div>
            <el-text v-if="mmdbDownloading" type="info" size="small" style="margin-top:6px;display:block">
              {{ t('mihomo.downloading') }}
            </el-text>
          </el-tab-pane>

          <!-- Upload local file -->
          <el-tab-pane :label="t('mihomo.uploadFile')" name="upload">
            <div style="padding:4px 0">
              <el-upload
                drag
                :auto-upload="false"
                :limit="1"
                accept=".mmdb"
                :on-change="onMMDBFileChange"
                :on-remove="() => mmdbFile = null"
                style="width:100%"
              >
                <el-icon style="font-size:40px;color:var(--cv-text-muted)"><Upload /></el-icon>
                <div style="margin-top:8px;color:var(--cv-text-muted)">
                  {{ t('mihomo.uploadDrag') }} <em>{{ t('mihomo.uploadClick') }}</em>
                </div>
                <div style="font-size:12px;color:var(--cv-text-muted);margin-top:4px">
                  {{ t('mihomo.uploadSupports') }}
                </div>
              </el-upload>
              <el-button
                type="primary"
                :loading="mmdbUploading"
                :disabled="!mmdbFile"
                style="margin-top:10px"
                @click="uploadMMDB"
              >
                <el-icon><Upload /></el-icon>
                {{ t('mihomo.uploadToServer') }}
              </el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-card>

    <!-- Install / Update -->
    <el-card style="margin-top: 20px">
      <template #header>
        <span>{{ t('mihomo.installUpdate') }}</span>
      </template>

      <div :class="{ 'form-disabled': formDisabled }">
        <el-form label-width="160px">
          <el-form-item :label="t('mihomo.targetVersion')">
            <el-select
              v-model="targetVersion"
              :placeholder="t('mihomo.latestAuto')"
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
              {{ status?.latest_version ? t('mihomo.latestLabel', { version: status.latest_version }) : '' }}
            </el-text>
          </el-form-item>

          <el-form-item :label="t('mihomo.forceReinstall')">
            <el-switch v-model="forceInstall" :disabled="formDisabled" />
            <el-text type="info" size="small" style="margin-left: 10px">
              {{ t('mihomo.forceReinstallDesc') }}
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
              {{ jobRunning ? t('mihomo.installing') : (status?.installed ? t('mihomo.updateMihomo') : t('mihomo.installMihomo')) }}
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
          :title="job.new_version ? t('mihomo.installedVersion', { version: job.new_version }) : t('mihomo.alreadyUpToDate')"
          :description="job.new_version && job.old_version ? t('mihomo.updatedFrom', { old: job.old_version, new: job.new_version }) : ''"
          show-icon
          style="margin-top: 10px"
        />
        <el-alert
          v-if="job.stage === 'error'"
          type="error"
          :title="t('mihomo.installFailed')"
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
  Refresh, Download, VideoPlay, VideoPause, RefreshRight, Upload
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
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
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
const mmdbUploading = ref(false)
const mmdbUrl = ref('')
const mmdbTab = ref('url')
const mmdbFile = ref<File | null>(null)

const onMMDBFileChange = (uploadFile: any) => {
  mmdbFile.value = uploadFile.raw ?? null
}

const uploadMMDB = async () => {
  if (!mmdbFile.value) return
  mmdbUploading.value = true
  try {
    const form = new FormData()
    form.append('file', mmdbFile.value)
    const token = localStorage.getItem('token') || ''
    const res = await fetch('/api/v1/proxy/mihomo/mmdb/upload', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || t('mihomo.uploadFailed'))
    ElMessage.success(t('mihomo.uploadSuccess'))
    mmdbFile.value = null
    await loadMMDB()
  } catch (e: any) {
    ElMessage.error(e.message || t('mihomo.uploadFailed'))
  } finally {
    mmdbUploading.value = false
  }
}

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
    ElMessage.success(t('mihomo.downloadSuccess'))
    await loadMMDB()
  } catch (e: any) {
    ElMessage.error(e.message || t('mihomo.downloadFailed'))
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
    fetch_release: t('mihomo.fetchRelease'),
    download:      t('mihomo.downloadBinary'),
    extract:       t('mihomo.extractArchive'),
    install:       t('mihomo.installingBinary'),
    restart:       t('mihomo.restartingMihomo'),
    done:          t('mihomo.complete'),
    error:         t('mihomo.failed'),
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
    ElMessage.error(e.message || t('mihomo.loadStatusFailed'))
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
    ElMessage.error(e.message || t('mihomo.loadVersionsFailed'))
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
    ElMessage.success(t('mihomo.actionSuccess', { action }))
    await loadStatus()
  } catch (e: any) {
    ElMessage.error(e.message || t('mihomo.actionFailed', { action }))
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
        ElMessage.error(t('mihomo.installProgressFailed') + (p.error || p.message))
      } else if (p.stage === 'done') {
        ElMessage.success(p.new_version ? t('mihomo.installProgressSuccess', { version: p.new_version }) : t('mihomo.installProgressUpToDate'))
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
    ElMessage.error(e.message || t('mihomo.installFailedStart'))
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
