import request from './request'

export interface LastAutoUpdateSummary {
  action: string
  details: string
  at: string
}

export interface LastAlertSummary {
  action: string
  details: string
  at: string
}

export interface SystemInfo {
  version: string
  go_version: string
  uptime: number
  mihomo_status: string
  database: string
  start_time: string
  runtime_config_path: string
  last_auto_update_action?: string
  last_auto_update_details?: string
  last_auto_update_at?: string
  last_alert_action?: string
  last_alert_details?: string
  last_alert_at?: string
}

export interface SystemStatus {
  uptime: string
  goroutines: number
  memory_alloc: number
  memory_sys: number
  heap_alloc: number
  heap_sys: number
  heap_objects: number
  gc_cycles: number
  mihomo_status: string
  mihomo_pid?: number
  mihomo_port?: number
  mihomo_uptime?: number
  mihomo_memory?: number
  last_auto_update?: LastAutoUpdateSummary
  last_alert?: LastAlertSummary
}

export const getSystemInfo = (): Promise<SystemInfo> => {
  return request({
    url: '/system/info',
    method: 'GET'
  })
}

export const getSystemStatus = (): Promise<SystemStatus> => {
  return request({
    url: '/system/status',
    method: 'GET'
  })
}

export interface Setting {
  key: string
  value: string
  description?: string
}

export const getSettings = (): Promise<Setting[]> => {
  return request({
    url: '/system/settings',
    method: 'GET'
  })
}

export const updateSetting = (key: string, value: string, description?: string): Promise<void> => {
  return request({
    url: '/system/settings',
    method: 'PUT',
    data: { key, value, description }
  })
}

export const updateSettingsBatch = (settings: Setting[]): Promise<void> => {
  return request({
    url: '/system/settings/batch',
    method: 'PUT',
    data: { settings }
  })
}

export interface LogResponse {
  source: string
  file: string
  lines: string[]
  total: number
  file_size: number
  available: boolean
  message?: string
}

export const getLogs = (source: 'proxyd' | 'mihomo', lines = 200): Promise<LogResponse> => {
  return request({
    url: '/system/logs',
    method: 'GET',
    params: { source, lines }
  })
}

export const downloadLog = (source: 'proxyd' | 'mihomo'): void => {
  const token = localStorage.getItem('token')
  const url = `/api/v1/system/logs/download?source=${source}`
  const a = document.createElement('a')
  a.href = url
  a.download = `${source}.log`
  if (token) {
    // Fetch with auth header then trigger download via blob
    fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.blob())
      .then(blob => {
        a.href = URL.createObjectURL(blob)
        a.click()
        URL.revokeObjectURL(a.href)
      })
  } else {
    a.click()
  }
}

export const getAuditLogs = (page = 1, limit = 50): Promise<any> => {
  return request({
    url: '/system/audit-logs',
    method: 'GET',
    params: { page, limit }
  })
}
