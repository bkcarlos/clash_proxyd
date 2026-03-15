<template>
  <div class="config-view">
    <div class="page-header">
      <h1>{{ t('config.title') }}</h1>
    </div>

    <el-card class="config-card">
      <template #header>
        <span>{{ t('config.generateConfig') }}</span>
      </template>

      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('config.sources')">
          <el-select
            v-model="form.source_ids"
            multiple
            :placeholder="t('config.sourcesPlaceholder')"
            style="width: 100%"
          >
            <el-option
              v-for="source in sourceStore.sources.filter(s => s.enabled)"
              :key="source.id"
              :label="source.name"
              :value="source.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" @click="generateConfig">
            {{ t('config.generateBtn') }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="generatedConfig" class="config-card">
      <template #header>
        <div class="card-header">
          <span>{{ t('config.generatedConfig') }}</span>
          <div>
            <el-button @click="downloadConfig">{{ t('config.download') }}</el-button>
            <el-button @click="applyConfig">{{ t('config.apply') }}</el-button>
            <el-button type="primary" @click="saveConfig">{{ t('config.save') }}</el-button>
          </div>
        </div>
      </template>

      <el-input
        v-model="generatedConfig"
        type="textarea"
        :rows="20"
        class="config-textarea"
      />
    </el-card>

    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <span>{{ t('config.configRevisions') }}</span>
          <el-button size="small" @click="loadRevisions">{{ t('common.refresh') }}</el-button>
        </div>
      </template>

      <el-table :data="revisions" border stripe>
        <el-table-column prop="id" :label="t('config.colId')" width="80" />
        <el-table-column prop="version" :label="t('config.colVersion')" />
        <el-table-column prop="source_hash" :label="t('config.colHash')" min-width="220" show-overflow-tooltip />
        <el-table-column prop="created_at" :label="t('config.colCreatedAt')" width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="created_by" :label="t('config.colCreatedBy')" width="150" />
        <el-table-column :label="t('config.colActions')" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="viewRevision(row)">{{ t('config.view') }}</el-button>
            <el-button size="small" @click="rollbackRevision(row)">{{ t('config.rollback') }}</el-button>
            <el-button size="small" type="danger" @click="deleteRevision(row.id)">{{ t('config.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="configDialogVisible" :title="t('config.configDialog')" width="80%">
      <el-input
        v-model="viewConfig"
        type="textarea"
        :rows="25"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useSourceStore } from '@/stores/source'
import * as configApi from '@/api/config'
import { getSystemInfo } from '@/api/system'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const sourceStore = useSourceStore()

const loading = ref(false)
const generatedConfig = ref('')
const revisions = ref<any[]>([])
const configDialogVisible = ref(false)
const viewConfig = ref('')

const form = reactive({
  source_ids: [] as number[]
})

const runtimePath = ref('')  // loaded from backend on mount

const generateConfig = async () => {
  if (form.source_ids.length === 0) {
    ElMessage.warning(t('config.selectSource'))
    return
  }

  loading.value = true
  try {
    const result = await configApi.generateConfig(form.source_ids)
    generatedConfig.value = result.config
    ElMessage.success(t('config.generateSuccess'))
    await loadRevisions()
  } catch (error: any) {
    ElMessage.error(error.message || t('config.generateFailed'))
  } finally {
    loading.value = false
  }
}

const downloadConfig = () => {
  if (!generatedConfig.value) return

  const blob = new Blob([generatedConfig.value], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'config.yaml'
  link.click()
  URL.revokeObjectURL(url)
}

const applyConfig = async () => {
  if (!generatedConfig.value) {
    ElMessage.warning(t('config.noGeneratedConfig'))
    return
  }

  try {
    // Don't send path — let backend use its configured default
    await configApi.applyConfig(generatedConfig.value)
    ElMessage.success(t('config.applySuccess'))
  } catch (error: any) {
    ElMessage.error(error.message || t('config.applyFailed'))
  }
}

const saveConfig = async () => {
  try {
    await ElMessageBox.prompt(t('config.savePromptLabel'), t('config.savePromptTitle'), {
      inputValue: runtimePath.value,
      confirmButtonText: t('config.saveConfirmBtn'),
      cancelButtonText: t('config.saveCancelBtn')
    }).then(async ({ value }) => {
      runtimePath.value = value
      await configApi.saveConfig(generatedConfig.value, value)
      ElMessage.success(t('config.saveSuccess'))
      await loadRevisions()
    })
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || t('config.saveFailed'))
    }
  }
}

const loadRevisions = async () => {
  try {
    revisions.value = await configApi.listRevisions(20)
  } catch (error: any) {
    ElMessage.error(error.message || t('config.loadRevisionsFailed'))
  }
}

const viewRevision = async (revision: any) => {
  viewConfig.value = revision.content
  configDialogVisible.value = true
}

const rollbackRevision = async (revision: any) => {
  try {
    await ElMessageBox.confirm(
      t('config.rollbackConfirm', { version: revision.version }),
      t('config.rollbackTitle'),
      { type: 'warning' }
    )
    await configApi.rollbackRevision(revision.id)
    ElMessage.success(t('config.rollbackSuccess', { version: revision.version }))
    await loadRevisions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || t('config.rollbackFailed'))
    }
  }
}

const deleteRevision = async (id: number) => {
  try {
    await ElMessageBox.confirm(t('config.deleteConfirm'), t('config.deleteTitle'), {
      type: 'warning'
    })
    await configApi.deleteRevision(id)
    ElMessage.success(t('config.deleteSuccess'))
    await loadRevisions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || t('config.deleteFailed'))
    }
  }
}

onMounted(async () => {
  sourceStore.fetchSources()
  loadRevisions()
  // Load runtime config path from backend so Save dialog shows the correct path
  try {
    const info = await getSystemInfo()
    if (info.runtime_config_path) runtimePath.value = info.runtime_config_path
  } catch { /* non-critical */ }
})
</script>

<style scoped>
.config-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}

.page-header {
  margin-bottom: 20px;
}

.config-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.config-textarea {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}
</style>
