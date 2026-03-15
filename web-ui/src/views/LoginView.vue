<template>
  <div class="login-view">
    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <h1>Proxy<span class="accent">d</span></h1>
          <p>{{ t('login.title') }}</p>
        </div>
      </template>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="80px"
        @submit.prevent="handleLogin"
      >
        <el-form-item :label="t('login.username')" prop="username">
          <el-input
            v-model="form.username"
            :placeholder="t('login.usernamePlaceholder')"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item :label="t('login.password')" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('login.passwordPlaceholder')"
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
            {{ t('login.loginBtn') }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <p>{{ t('login.defaultCredentials') }}</p>
        <p class="warning">{{ t('login.changePasswordWarning') }}</p>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: 'admin',
  password: 'admin'
})

const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('login.usernameRequired'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('login.passwordRequired'), trigger: 'blur' }
  ]
}))

const handleLogin = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await userStore.login(form)
      ElMessage.success(t('login.loginSuccess'))
      router.push('/')
    } catch (error: any) {
      ElMessage.error(error.response?.data?.error || t('login.loginFailed'))
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
