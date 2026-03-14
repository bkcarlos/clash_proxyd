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
      <el-select v-model="typeFilter" placeholder="All types" clearable style="width:160px" size="small">
        <el-option v-for="t in ruleTypes" :key="t" :label="t" :value="t" />
      </el-select>
      <el-button size="small" :loading="loading" @click="loadRules">
        <el-icon><Refresh /></el-icon>Refresh
      </el-button>
      <el-button size="small" :disabled="rules.length === 0" @click="downloadRules">
        <el-icon><Download /></el-icon>Export
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="pagedRules"
      size="small"
      :max-height="tableHeight"
      stripe
    >
      <el-table-column type="index" width="60" label="#"
        :index="(i: number) => (page - 1) * pageSize + i + 1"
      />
      <el-table-column label="Type" width="160">
        <template #default="{ row }">
          <el-tag size="small" :type="ruleTagType(row.type)">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="payload" label="Payload" min-width="200" show-overflow-tooltip />
      <el-table-column label="Proxy" width="160">
        <template #default="{ row }">
          <el-tag
            size="small"
            :type="row.proxy === 'DIRECT' ? 'success' : row.proxy === 'REJECT' ? 'danger' : 'primary'"
          >
            {{ row.proxy }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-bar">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredRules.length"
        :page-sizes="[50, 100, 200, 500]"
        layout="total, sizes, prev, pager, next, jumper"
        background
        size="small"
      />
    </div>

    <div v-if="!loading && rules.length === 0" style="text-align:center;padding:40px">
      <el-empty description="No rules — mihomo may not be running or no config applied" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Search, Refresh, Download } from '@element-plus/icons-vue'
import request from '@/api/request'

const rules = ref<any[]>([])
const filter = ref('')
const typeFilter = ref('')
const loading = ref(false)
const page = ref(1)
const pageSize = ref(100)
const tableHeight = window.innerHeight - 200

// Reset page on filter change
watch([filter, typeFilter], () => { page.value = 1 })

const ruleTypes = computed(() => {
  const s = new Set(rules.value.map(r => r.type).filter(Boolean))
  return [...s].sort()
})

const filteredRules = computed(() => {
  let list = rules.value
  if (typeFilter.value) {
    list = list.filter(r => r.type === typeFilter.value)
  }
  if (filter.value.trim()) {
    const q = filter.value.toLowerCase()
    list = list.filter(r =>
      (r.type || '').toLowerCase().includes(q) ||
      (r.payload || '').toLowerCase().includes(q) ||
      (r.proxy || '').toLowerCase().includes(q)
    )
  }
  return list
})

const pagedRules = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredRules.value.slice(start, start + pageSize.value)
})

const ruleTagType = (type: string): 'primary' | 'success' | 'warning' | 'danger' | 'info' => {
  const map: Record<string, 'primary' | 'success' | 'warning' | 'danger' | 'info'> = {
    DOMAIN: 'primary',
    'DOMAIN-SUFFIX': 'primary',
    'DOMAIN-KEYWORD': 'primary',
    'DOMAIN-REGEX': 'primary',
    'IP-CIDR': 'warning',
    'IP-CIDR6': 'warning',
    'SRC-IP-CIDR': 'warning',
    GEOIP: 'success',
    GEOSITE: 'success',
    'RULE-SET': 'info',
    MATCH: 'info',
    PROCESS: 'danger',
    'PROCESS-NAME': 'danger',
  }
  return map[type] ?? 'info'
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

const downloadRules = () => {
  const lines = filteredRules.value.map(r =>
    [r.type, r.payload, r.proxy].filter(Boolean).join(',')
  )
  const content = lines.join('\n')
  const blob = new Blob([content], { type: 'text/plain' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `mihomo-rules-${Date.now()}.txt`
  a.click()
  URL.revokeObjectURL(a.href)
}

onMounted(loadRules)
</script>

<style scoped>
.rules-view { display: flex; flex-direction: column; gap: 12px; }

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  padding: 10px 16px;
}

.count { font-size: 13px; color: var(--cv-text-muted); flex: 1; }

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 4px 0;
}
</style>
