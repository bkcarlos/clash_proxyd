<template>
  <div class="login-view">
    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <h1>Proxy<span class="accent">d</span></h1>
          <p>Mihomo Proxy Manager</p>
        </div>
      </template>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="80px"
        @submit.prevent="handleLogin"
      >
        <el-form-item label="Username" prop="username">
          <el-input
            v-model="form.username"
            placeholder="Enter username"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item label="Password" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Enter password"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            style="width: 100%"
            @click="handleLogin"
          >
            Login
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <p>Default credentials: admin / admin</p>
        <p class="warning">Please change the password after first login</p>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage, FormInstance, FormRules } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: 'admin',
  password: 'admin'
})

const rules: FormRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await userStore.login(form)
      ElMessage.success('Login successful')
      router.push('/')
    } catch (error: any) {
      ElMessage.error(error.response?.data?.error || 'Login failed')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.login-view {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: var(--cv-bg);
  background-image:
    radial-gradient(ellipse at 20% 50%, rgba(88,101,242,0.12) 0%, transparent 60%),
    radial-gradient(ellipse at 80% 20%, rgba(88,101,242,0.08) 0%, transparent 50%);
}

.login-card {
  width: 380px;
  background: var(--cv-surface) !important;
  border: 1px solid var(--cv-border) !important;
  border-radius: 14px !important;
  box-shadow: 0 20px 60px rgba(0,0,0,0.5) !important;
}

.card-header {
  text-align: center;
  padding: 8px 0 4px;
}

.card-header h1 {
  margin: 0 0 4px;
  font-size: 26px;
  font-weight: 700;
  color: var(--cv-text);
  letter-spacing: 1px;
}

.accent { color: var(--cv-accent); }

.card-header p {
  margin: 0;
  font-size: 13px;
  color: var(--cv-text-muted);
}

.login-footer {
  margin-top: 16px;
  text-align: center;
  font-size: 12px;
  color: var(--cv-text-muted);
}

.login-footer p { margin: 4px 0; }

.warning { color: #f87171; }
</style>
