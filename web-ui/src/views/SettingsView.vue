<template>
  <div class="settings-view">
    <h1>Settings</h1>

    <el-card>
      <template #header>
        <span>System Settings</span>
      </template>

      <el-form label-width="200px">
        <el-form-item label="Mihomo Binary Path">
          <el-input v-model="settings.mihomo_path" />
        </el-form-item>

        <el-form-item label="Mihomo Config Directory">
          <el-input v-model="settings.mihomo_config_dir" />
        </el-form-item>

        <el-form-item label="API Port">
          <el-input-number v-model="settings.listen_port" :min="1024" :max="65535" />
        </el-form-item>

        <el-form-item label="Log Level">
          <el-select v-model="settings.log_level">
            <el-option label="Debug" value="debug" />
            <el-option label="Info" value="info" />
            <el-option label="Warn" value="warn" />
            <el-option label="Error" value="error" />
          </el-select>
        </el-form-item>

        <el-form-item label="Session Timeout (seconds)">
          <el-input-number v-model="settings.session_timeout" :min="300" :max="86400" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveSettings">Save Settings</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Terminal Proxy Config -->
    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header-row">
          <span>Terminal Proxy Config</span>
          <el-tag :type="proxyStatus.running ? 'success' : 'info'" size="small">
            {{ proxyStatus.running ? `Running · ${proxyHost}:${proxyPort}` : 'Mihomo not running' }}
          </el-tag>
        </div>
      </template>

      <el-form label-width="100px">
        <el-form-item label="Host">
          <el-select
            v-model="proxyHost"
            filterable
            allow-create
            style="width:220px"
            placeholder="Select or enter IP"
          >
            <el-option
              v-for="ip in hostOptions"
              :key="ip.value"
              :label="ip.label"
              :value="ip.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Port">
          <el-input-number v-model="proxyPort" :min="1" :max="65535" style="width:150px" />
        </el-form-item>
      </el-form>

      <el-tabs v-model="shellTab" class="proxy-tabs">
        <el-tab-pane label="Linux / macOS" name="unix">
          <div class="cmd-list">
            <div v-for="cmd in unixCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">All</span>
              <code class="cmd-code" style="flex:1;white-space:normal">{{ unixAll }}</code>
              <el-button size="small" link @click="copy(unixAll)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="Windows CMD" name="cmd">
          <div class="cmd-list">
            <div v-for="cmd in winCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">All</span>
              <code class="cmd-code" style="flex:1;white-space:normal">{{ winAll }}</code>
              <el-button size="small" link @click="copy(winAll)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="PowerShell" name="ps">
          <div class="cmd-list">
            <div v-for="cmd in psCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">All</span>
              <code class="cmd-code" style="flex:1;white-space:normal">{{ psAll }}</code>
              <el-button size="small" link @click="copy(psAll)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-card style="margin-top: 20px">
      <template #header>
        <span>Change Password</span>
      </template>

      <el-form :model="passwordForm" label-width="150px">
        <el-form-item label="Current Password">
          <el-input v-model="passwordForm.old_password" type="password" show-password />
        </el-form-item>

        <el-form-item label="New Password">
          <el-input v-model="passwordForm.new_password" type="password" show-password />
        </el-form-item>

        <el-form-item label="Confirm Password">
          <el-input v-model="passwordForm.confirm_password" type="password" show-password />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="changePassword">Change Password</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useSystemStore } from '@/stores/system'
import * as authApi from '@/api/auth'
import { ElMessage } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import request from '@/api/request'

const systemStore = useSystemStore()

const settings = reactive({
  mihomo_path: '',
  mihomo_config_dir: '',
  listen_port: 9090,
  log_level: 'info',
  session_timeout: 86400
})

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const saveSettings = async () => {
  try {
    await systemStore.updateSettingsBatch(
      Object.fromEntries(Object.entries(settings).map(([k, v]) => [k, String(v)]))
    )
    ElMessage.success('Settings saved successfully')
  } catch (error: any) {
    ElMessage.error(error.message || 'Save failed')
  }
}

const changePassword = async () => {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    ElMessage.error('Passwords do not match')
    return
  }

  if (passwordForm.new_password.length < 6) {
    ElMessage.error('Password must be at least 6 characters')
    return
  }

  try {
    await authApi.updatePassword(passwordForm.old_password, passwordForm.new_password)
    ElMessage.success('Password changed successfully')
    Object.assign(passwordForm, {
      old_password: '',
      new_password: '',
      confirm_password: ''
    })
  } catch (error: any) {
    ElMessage.error(error.message || 'Password change failed')
  }
}

// ── Terminal proxy config ────────────────────────────────────────────────
const proxyHost = ref('127.0.0.1')
const proxyPort = ref(7891)
const shellTab = ref('unix')
const proxyStatus = ref({ running: false, port: 7891 })
const hostOptions = ref([{ value: '127.0.0.1', label: '127.0.0.1 (localhost)' }])

const addr = computed(() => `${proxyHost.value}:${proxyPort.value}`)
const httpAddr = computed(() => `http://${addr.value}`)
const socksAddr = computed(() => `socks5://${addr.value}`)

const unixCmds = computed(() => [
  { label: 'http_proxy',  value: `export http_proxy=${httpAddr.value}` },
  { label: 'https_proxy', value: `export https_proxy=${httpAddr.value}` },
  { label: 'all_proxy',   value: `export all_proxy=${socksAddr.value}` },
  { label: 'no_proxy',    value: `export no_proxy=localhost,127.0.0.1` },
])
const unixAll = computed(() => unixCmds.value.map(c => c.value).join('\n'))

const winCmds = computed(() => [
  { label: 'http_proxy',  value: `set http_proxy=${httpAddr.value}` },
  { label: 'https_proxy', value: `set https_proxy=${httpAddr.value}` },
  { label: 'all_proxy',   value: `set all_proxy=${socksAddr.value}` },
  { label: 'no_proxy',    value: `set no_proxy=localhost,127.0.0.1` },
])
const winAll = computed(() => winCmds.value.map(c => c.value).join(' & '))

const psCmds = computed(() => [
  { label: 'http_proxy',  value: `$env:http_proxy="${httpAddr.value}"` },
  { label: 'https_proxy', value: `$env:https_proxy="${httpAddr.value}"` },
  { label: 'all_proxy',   value: `$env:all_proxy="${socksAddr.value}"` },
  { label: 'no_proxy',    value: `$env:no_proxy="localhost,127.0.0.1"` },
])
const psAll = computed(() => psCmds.value.map(c => c.value).join('; '))

const copy = (text: string) => {
  // Prefer clipboard API, fall back to execCommand for non-HTTPS / non-localhost
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(text)
      .then(() => ElMessage.success('Copied!'))
      .catch(() => copyFallback(text))
  } else {
    copyFallback(text)
  }
}

const copyFallback = (text: string) => {
  const el = document.createElement('textarea')
  el.value = text
  el.style.cssText = 'position:fixed;top:-9999px;left:-9999px'
  document.body.appendChild(el)
  el.focus()
  el.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(el)
  ok ? ElMessage.success('Copied!') : ElMessage.error('Copy failed')
}

onMounted(async () => {
  await systemStore.fetchSettings()
  Object.assign(settings, systemStore.settings)
  // Read actual mixed-port and running state from install-status
  try {
    const s: any = await request({ url: '/proxy/mihomo/install-status', method: 'GET' })
    proxyStatus.value.running = s.is_running
    if (s.mixed_port && s.mixed_port > 0) {
      proxyPort.value = s.mixed_port
    }
  } catch { /* non-critical */ }
  // Load local IP addresses for the host selector
  try {
    const net: any = await request({ url: '/system/network-interfaces', method: 'GET' })
    const ips: string[] = net.addresses ?? []
    hostOptions.value = ips.map(ip => ({
      value: ip,
      label: ip === '127.0.0.1' ? `${ip} (localhost)` : ip
    }))
  } catch { /* non-critical */ }
})
</script>

<style scoped>
.settings-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}

.card-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.proxy-tabs { margin-top: 4px; }

.cmd-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.cmd-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  background: var(--cv-surface2, #1e2235);
  border-radius: 6px;
}

.cmd-row.all {
  align-items: flex-start;
  padding: 8px 10px;
}

.cmd-label {
  font-size: 12px;
  color: var(--cv-text-muted, #64748b);
  width: 90px;
  flex-shrink: 0;
}

.cmd-code {
  flex: 1;
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  color: #22d3ee;
  word-break: break-all;
}
</style>
