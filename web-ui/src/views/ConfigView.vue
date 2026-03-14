<template>
  <div class="config-view">
    <div class="page-header">
      <h1>Configuration</h1>
    </div>

    <el-card class="config-card">
      <template #header>
        <span>Generate Configuration</span>
      </template>

      <el-form :model="form" label-width="120px">
        <el-form-item label="Sources">
          <el-select
            v-model="form.source_ids"
            multiple
            placeholder="Select sources"
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
            Generate Configuration
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="generatedConfig" class="config-card">
      <template #header>
        <div class="card-header">
          <span>Generated Configuration</span>
          <div>
            <el-button @click="downloadConfig">Download</el-button>
            <el-button @click="applyConfig">Apply</el-button>
            <el-button type="primary" @click="saveConfig">Save</el-button>
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
          <span>Configuration Revisions</span>
          <el-button size="small" @click="loadRevisions">Refresh</el-button>
        </div>
      </template>

      <el-table :data="revisions" border stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="version" label="Version" />
        <el-table-column prop="source_hash" label="Hash" min-width="220" show-overflow-tooltip />
        <el-table-column prop="created_at" label="Created At" width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="created_by" label="Created By" width="150" />
        <el-table-column label="Actions" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="viewRevision(row)">View</el-button>
            <el-button size="small" @click="rollbackRevision(row)">Rollback</el-button>
            <el-button size="small" type="danger" @click="deleteRevision(row.id)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="configDialogVisible" title="Configuration" width="80%">
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
    ElMessage.warning('Please select at least one source')
    return
  }

  loading.value = true
  try {
    const result = await configApi.generateConfig(form.source_ids)
    generatedConfig.value = result.config
    ElMessage.success('Configuration generated successfully')
    await loadRevisions()
  } catch (error: any) {
    ElMessage.error(error.message || 'Generation failed')
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
    ElMessage.warning('No generated configuration')
    return
  }

  try {
    // Don't send path — let backend use its configured default
    await configApi.applyConfig(generatedConfig.value)
    ElMessage.success('Configuration applied successfully')
  } catch (error: any) {
    ElMessage.error(error.message || 'Apply failed')
  }
}

const saveConfig = async () => {
  try {
    await ElMessageBox.prompt('Enter config file path:', 'Save Configuration', {
      inputValue: runtimePath.value,
      confirmButtonText: 'Save',
      cancelButtonText: 'Cancel'
    }).then(async ({ value }) => {
      runtimePath.value = value
      await configApi.saveConfig(generatedConfig.value, value)
      ElMessage.success('Configuration saved successfully')
      await loadRevisions()
    })
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || 'Save failed')
    }
  }
}

const loadRevisions = async () => {
  try {
    revisions.value = await configApi.listRevisions(20)
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to load revisions')
  }
}

const viewRevision = async (revision: any) => {
  viewConfig.value = revision.content
  configDialogVisible.value = true
}

const rollbackRevision = async (revision: any) => {
  try {
    await ElMessageBox.confirm(`Rollback to revision ${revision.version} and apply now?`, 'Confirm rollback', {
      type: 'warning'
    })
    await configApi.rollbackRevision(revision.id)
    ElMessage.success(`Revision ${revision.version} rolled back and applied`)
    await loadRevisions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || 'Rollback failed')
    }
  }
}

const deleteRevision = async (id: number) => {
  try {
    await ElMessageBox.confirm('Are you sure to delete this revision?', 'Warning', {
      type: 'warning'
    })
    await configApi.deleteRevision(id)
    ElMessage.success('Revision deleted successfully')
    await loadRevisions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || 'Delete failed')
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
