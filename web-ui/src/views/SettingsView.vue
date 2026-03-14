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
import { reactive, onMounted } from 'vue'
import { useSystemStore } from '@/stores/system'
import * as authApi from '@/api/auth'
import { ElMessage } from 'element-plus'

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

onMounted(async () => {
  await systemStore.fetchSettings()
  Object.assign(settings, systemStore.settings)
})
</script>

<style scoped>
.settings-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}
</style>
