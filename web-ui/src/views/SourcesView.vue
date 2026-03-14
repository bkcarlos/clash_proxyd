<template>
  <div class="sources-view">
    <div class="page-header">
      <h1>Sources</h1>
      <div style="display:flex;gap:8px">
        <el-button :loading="applying" @click="applyToMihomo">
          <el-icon><Promotion /></el-icon>
          Apply to Mihomo
        </el-button>
        <el-button type="primary" @click="showCreateDialog">
          <el-icon><Plus /></el-icon>
          Add Source
        </el-button>
      </div>
    </div>

    <el-table
      v-loading="sourceStore.loading"
      :data="sourceStore.sources"
      border
      stripe
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="type" label="Type" width="100">
        <template #default="{ row }">
          <el-tag :type="getTypeTagType(row.type)">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="Status" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">
            {{ row.enabled ? 'Enabled' : 'Disabled' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="Priority" width="100" />
      <el-table-column prop="update_interval" label="Interval" width="100">
        <template #default="{ row }">
          {{ formatInterval(row.update_interval) }}
        </template>
      </el-table-column>
      <el-table-column label="Cache" width="160">
        <template #default="{ row }">
          <div v-if="row.last_fetch" style="font-size:12px">
            <el-tag type="success" size="small">{{ formatSize(row.content_size) }}</el-tag>
            <div style="color:#909399;margin-top:2px">{{ formatTime(row.last_fetch) }}</div>
          </div>
          <el-tag v-else type="warning" size="small">No cache</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Actions" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="testSource(row.id)">Test</el-button>
          <el-button size="small" @click="fetchSource(row.id)">Fetch</el-button>
          <el-button size="small" type="primary" @click="editSource(row)">Edit</el-button>
          <el-button size="small" type="danger" @click="deleteSource(row.id)">Delete</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Source Form Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? 'Edit Source' : 'Add Source'"
      width="600px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Enter source name" />
        </el-form-item>

        <el-form-item label="Type" prop="type">
          <el-select v-model="form.type" placeholder="Select type">
            <el-option label="HTTP" value="http" />
            <el-option label="File" value="file" />
            <el-option label="Local" value="local" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="URL" prop="url">
          <el-input v-model="form.url" placeholder="Enter subscription URL" />
        </el-form-item>

        <el-form-item v-if="form.type === 'file' || form.type === 'local'" label="Path" prop="path">
          <el-input v-model="form.path" placeholder="Enter file path" />
        </el-form-item>

        <el-form-item label="Update Interval">
          <el-input-number v-model="form.update_interval" :min="60" :max="86400" />
          <span style="margin-left: 10px">seconds</span>
        </el-form-item>

        <el-form-item label="Priority">
          <el-input-number v-model="form.priority" :min="0" :max="100" />
        </el-form-item>

        <el-form-item label="Enabled">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleSubmit">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useSourceStore } from '@/stores/source'
import { ElMessageBox, ElMessage } from 'element-plus'
import { Plus, Promotion } from '@element-plus/icons-vue'
import { quickApply } from '@/api/config'
import type { Source } from '@/api/source'
import type { FormInstance, FormRules } from 'element-plus'

const sourceStore = useSourceStore()

const dialogVisible = ref(false)
const isEdit = ref(false)
const applying = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<Partial<Source>>({
  name: '',
  type: 'http',
  url: '',
  path: '',
  update_interval: 3600,
  update_cron: '',
  enabled: true,
  priority: 0
})

const rules: FormRules = {
  name: [{ required: true, message: 'Name is required', trigger: 'blur' }],
  type: [{ required: true, message: 'Type is required', trigger: 'change' }],
  url: [{
    validator: (_rule, value, callback) => {
      if (form.type === 'http' && !value) {
        callback(new Error('URL is required for HTTP type'))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  }],
  path: [{
    validator: (_rule, value, callback) => {
      if ((form.type === 'file' || form.type === 'local') && !value) {
        callback(new Error('Path is required for file/local type'))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  }]
}

const getTypeTagType = (type: string) => {
  const types: Record<string, any> = {
    http: 'primary',
    file: 'success',
    local: 'info'
  }
  return types[type] || ''
}

const formatInterval = (seconds: number): string => {
  if (seconds >= 3600) return `${seconds / 3600}h`
  return `${seconds / 60}m`
}

const formatSize = (bytes: number): string => {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const formatTime = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString()
}

const showCreateDialog = () => {
  isEdit.value = false
  Object.assign(form, {
    name: '',
    type: 'http',
    url: '',
    path: '',
    update_interval: 3600,
    update_cron: '',
    enabled: true,
    priority: 0
  })
  dialogVisible.value = true
}

const editSource = (source: Source) => {
  isEdit.value = true
  Object.assign(form, source)
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    try {
      if (isEdit.value) {
        await sourceStore.updateSource(form.id!, form as Source)
        ElMessage.success('Source updated successfully')
        dialogVisible.value = false
        await sourceStore.fetchSources()
      } else {
        const res: any = await sourceStore.createSource(form as Source)
        dialogVisible.value = false
        await sourceStore.fetchSources()
        if (res?.warning) {
          ElMessage.warning('Source saved, but fetch failed: ' + res.warning)
        } else {
          // Auto apply after successful create + cache
          await doQuickApply(true)
        }
      }
    } catch (error: any) {
      ElMessage.error(error.message || 'Operation failed')
    }
  })
}

const doQuickApply = async (silent = false) => {
  applying.value = true
  try {
    const res = await quickApply()
    ElMessage.success(`Config applied — ${res.data?.sources?.length ?? 0} source(s) merged`)
  } catch (error: any) {
    if (!silent) ElMessage.error(error.message || 'Apply failed')
    else ElMessage.warning('Source saved. Apply manually when ready.')
  } finally {
    applying.value = false
  }
}

const applyToMihomo = () => doQuickApply()

const testSource = async (id: number) => {
  try {
    const result = await sourceStore.testSource(id)
    if (result.success) {
      ElMessage.success(`Test successful! Latency: ${result.latency}ms`)
    } else {
      ElMessage.error(`Test failed: ${result.error}`)
    }
  } catch (error: any) {
    ElMessage.error(error.message || 'Test failed')
  }
}

const fetchSource = async (id: number) => {
  try {
    const result = await sourceStore.fetchSource(id)
    ElMessage.success(`Source fetched! Size: ${result.size} bytes`)
  } catch (error: any) {
    ElMessage.error(error.message || 'Fetch failed')
  }
}

const deleteSource = async (id: number) => {
  try {
    await ElMessageBox.confirm('Are you sure to delete this source?', 'Warning', {
      type: 'warning'
    })
    await sourceStore.deleteSource(id)
    ElMessage.success('Source deleted successfully')
    await sourceStore.fetchSources()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || 'Delete failed')
    }
  }
}

onMounted(() => {
  sourceStore.fetchSources()
})
</script>

<style scoped>
.sources-view h1 {
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: 600;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
</style>
