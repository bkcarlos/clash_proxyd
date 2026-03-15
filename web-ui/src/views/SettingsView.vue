<template>
  <div class="settings-view">
    <h1>{{ t('settings.title') }}</h1>

    <el-card>
      <template #header>
        <span>{{ t('settings.systemSettings') }}</span>
      </template>

      <el-form label-width="200px">
        <el-form-item :label="t('settings.mihomoPath')">
          <el-input v-model="settings.mihomo_path" />
        </el-form-item>

        <el-form-item :label="t('settings.mihomoConfigDir')">
          <el-input v-model="settings.mihomo_config_dir" />
        </el-form-item>

        <el-form-item :label="t('settings.apiPort')">
          <el-input-number v-model="settings.listen_port" :min="1024" :max="65535" />
        </el-form-item>

        <el-form-item :label="t('settings.logLevel')">
          <el-select v-model="settings.log_level">
            <el-option label="Debug" value="debug" />
            <el-option label="Info" value="info" />
            <el-option label="Warn" value="warn" />
            <el-option label="Error" value="error" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.sessionTimeout')">
          <el-input-number v-model="settings.session_timeout" :min="300" :max="86400" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveSettings">{{ t('settings.saveBtn') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Terminal Proxy Config -->
    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header-row">
          <span>{{ t('settings.terminalProxy') }}</span>
          <el-tag :type="proxyStatus.running ? 'success' : 'info'" size="small">
            {{ proxyStatus.running ? t('settings.mihomoRunning', { host: proxyHost, port: proxyPort }) : t('settings.mihomoNotRunning') }}
          </el-tag>
        </div>
      </template>

      <el-form label-width="100px">
        <el-form-item :label="t('settings.host')">
          <el-select
            v-model="proxyHost"
            filterable
            allow-create
            style="width:220px"
            :placeholder="t('settings.hostPlaceholder')"
          >
            <el-option
              v-for="ip in hostOptions"
              :key="ip.value"
              :label="ip.label"
              :value="ip.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('settings.port')">
          <el-input-number v-model="proxyPort" :min="1" :max="65535" style="width:150px" />
        </el-form-item>
      </el-form>

      <el-tabs v-model="shellTab" class="proxy-tabs">
        <el-tab-pane :label="t('settings.localLinux')" name="unix">
          <div class="cmd-list">
            <div v-for="cmd in unixCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">{{ t('settings.all') }}</span>
              <code class="cmd-code" style="flex:1;white-space:normal">{{ unixAll }}</code>
              <el-button size="small" link @click="copy(unixAll)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('settings.windowsCmd')" name="cmd">
          <div class="cmd-list">
            <div v-for="cmd in winCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">{{ t('settings.all') }}</span>
              <code class="cmd-code" style="flex:1;white-space:normal">{{ winAll }}</code>
              <el-button size="small" link @click="copy(winAll)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('settings.powershell')" name="ps">
          <div class="cmd-list">
            <div v-for="cmd in psCmds" :key="cmd.label" class="cmd-row">
              <span class="cmd-label">{{ cmd.label }}</span>
              <code class="cmd-code">{{ cmd.value }}</code>
              <el-button size="small" link @click="copy(cmd.value)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
            <div class="cmd-row all">
              <span class="cmd-label">{{ t('settings.all') }}</span>
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
        <span>{{ t('settings.changePassword') }}</span>
      </template>

      <el-form :model="passwordForm" label-width="150px">
        <el-form-item :label="t('settings.currentPassword')">
          <el-input v-model="passwordForm.old_password" type="password" show-password />
        </el-form-item>

        <el-form-item :label="t('settings.newPassword')">
          <el-input v-model="passwordForm.new_password" type="password" show-password />
        </el-form-item>

        <el-form-item :label="t('settings.confirmPassword')">
          <el-input v-model="passwordForm.confirm_password" type="password" show-password />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="changePassword">{{ t('settings.changePasswordBtn') }}</el-button>
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
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
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
    ElMessage.success(t('settings.saveSuccess'))
  } catch (error: any) {
    ElMessage.error(error.message || t('settings.saveFailed'))
  }
}

const changePassword = async () => {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    ElMessage.error(t('settings.passwordMismatch'))
    return
  }

  if (passwordForm.new_password.length < 6) {
    ElMessage.error(t('settings.passwordTooShort'))
    return
  }

  try {
    await authApi.updatePassword(passwordForm.old_password, passwordForm.new_password)
    ElMessage.success(t('settings.passwordChanged'))
    Object.assign(passwordForm, {
      old_password: '',
      new_password: '',
      confirm_password: ''
    })
  } catch (error: any) {
    ElMessage.error(error.message || t('settings.passwordChangeFailed'))
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
      .then(() => ElMessage.success(t('common.copied')))
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
  ok ? ElMessage.success(t('common.copied')) : ElMessage.error(t('common.copyFailed'))
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
