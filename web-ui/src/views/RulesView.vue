<template>
  <div class="rules-view">
    <div class="toolbar">
      <span class="count">{{ filteredRules.length }} / {{ rules.length }} rules</span>
      <el-input
        v-model="filter"
        placeholder="Filter by type, payload or proxy..."
        clearable
        style="width:280px"
        size="small"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-button size="small" :loading="loading" @click="loadRules">
        <el-icon><Refresh /></el-icon>Refresh
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="filteredRules"
      size="small"
      :max-height="tableHeight"
      stripe
    >
      <el-table-column type="index" width="55" label="#" />
      <el-table-column label="Type" width="160">
        <template #default="{ row }">
          <el-tag size="small" :type="ruleTagType(row.type)">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="payload" label="Payload" min-width="200" show-overflow-tooltip />
      <el-table-column label="Proxy" width="160">
        <template #default="{ row }">
          <el-tag size="small" :type="row.proxy === 'DIRECT' ? 'success' : row.proxy === 'REJECT' ? 'danger' : 'primary'">
            {{ row.proxy }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="!loading && rules.length === 0" style="text-align:center;padding:40px">
      <el-empty description="No rules — mihomo may not be running or no config applied" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'
import request from '@/api/request'

const rules = ref<any[]>([])
const filter = ref('')
const loading = ref(false)
const tableHeight = window.innerHeight - 160

const filteredRules = computed(() => {
  if (!filter.value) return rules.value
  const q = filter.value.toLowerCase()
  return rules.value.filter(r =>
    (r.type || '').toLowerCase().includes(q) ||
    (r.payload || '').toLowerCase().includes(q) ||
    (r.proxy || '').toLowerCase().includes(q)
  )
})

const ruleTagType = (type: string) => {
  const map: Record<string, string> = {
    DOMAIN: 'primary', 'DOMAIN-SUFFIX': 'primary', 'DOMAIN-KEYWORD': 'primary',
    'IP-CIDR': 'warning', 'IP-CIDR6': 'warning',
    GEOIP: 'success', GEOSITE: 'success',
    MATCH: '', RULE_SET: 'info',
  }
  return (map[type] || '') as any
}

const loadRules = async () => {
  loading.value = true
  try {
    const data: any = await request({ url: '/proxy/rules', method: 'GET' })
    rules.value = data?.rules ?? []
  } catch {
    rules.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRules)
</script>

<style scoped>
.rules-view { display: flex; flex-direction: column; gap: 12px; }

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 10px 16px;
}

.count { font-size: 13px; color: var(--cv-text-muted); flex: 1; }
</style>
