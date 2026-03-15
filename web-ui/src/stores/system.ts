import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as systemApi from '@/api/system'
import { useProxyStore } from '@/stores/proxy'

export type SystemInfo = systemApi.SystemInfo
export type SystemStatus = systemApi.SystemStatus

let ws: WebSocket | null = null
let reconnectTimer: number | null = null
let reconnectDelay = 1000

export const useSystemStore = defineStore('system', () => {
  const info = ref<SystemInfo | null>(null)
  const status = ref<SystemStatus | null>(null)
  const settings = ref<Record<string, string>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)
  const wsConnected = ref(false)

  const fetchInfo = async (): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      info.value = await systemApi.getSystemInfo()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch system info'
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchStatus = async (): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      status.value = await systemApi.getSystemStatus()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch system status'
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchSettings = async (): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      const data = await systemApi.getSettings()
      settings.value = data.reduce((acc, s) => {
        acc[s.key] = s.value
        return acc
      }, {} as Record<string, string>)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateSetting = async (key: string, value: string, description?: string): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      await systemApi.updateSetting(key, value, description)
      settings.value[key] = value
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update setting'
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateSettingsBatch = async (newSettings: Record<string, string>): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      await systemApi.updateSettingsBatch(
        Object.entries(newSettings).map(([key, value]) => ({ key, value: String(value) }))
      )
      settings.value = { ...settings.value, ...newSettings }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  const applySnapshot = (payload: any): void => {
    const statusPayload = payload?.status
    if (statusPayload) {
      status.value = {
        ...(status.value || {
          uptime: '', goroutines: 0, memory_alloc: 0, memory_sys: 0,
          heap_alloc: 0, heap_sys: 0, heap_objects: 0, gc_cycles: 0, mihomo_status: 'unknown'
        }),
        ...statusPayload
      }
      if (info.value) {
        info.value = {
          ...info.value,
          mihomo_status: String(statusPayload.mihomo_status || info.value.mihomo_status)
        }

        const wsAutoUpdate = statusPayload.last_auto_update
        if (wsAutoUpdate) {
          info.value.last_auto_update_action = wsAutoUpdate.action
          info.value.last_auto_update_details = wsAutoUpdate.details
          info.value.last_auto_update_at = wsAutoUpdate.at
        }

        const wsAlert = statusPayload.last_alert
        if (wsAlert) {
          info.value.last_alert_action = wsAlert.action
          info.value.last_alert_details = wsAlert.details
          info.value.last_alert_at = wsAlert.at
        }
      }
    }
  }

  const connectWS = (): void => {
    const token = localStorage.getItem('token')
    if (!token || ws) return

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${proto}//${window.location.host}/api/v1/system/ws?token=${encodeURIComponent(token)}`
    ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      wsConnected.value = true
      reconnectDelay = 1000
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    }

    ws.onmessage = (evt) => {
      try {
        const payload = JSON.parse(evt.data)
        if (payload?.type === 'snapshot') {
          applySnapshot(payload)
          if (payload.traffic) {
            const proxyStore = useProxyStore()
            proxyStore.applyTraffic(payload.traffic)
          }
        }
      } catch {
        // ignore invalid payload
      }
    }

    ws.onclose = () => {
      wsConnected.value = false
      ws = null
      reconnectTimer = window.setTimeout(() => {
        connectWS()
      }, reconnectDelay)
      reconnectDelay = Math.min(reconnectDelay * 2, 15000)
    }

    ws.onerror = () => {
      wsConnected.value = false
    }
  }

  const disconnectWS = (): void => {
    if (reconnectTimer) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
    wsConnected.value = false
  }

  return {
    info,
    status,
    settings,
    loading,
    error,
    wsConnected,
    fetchInfo,
    fetchStatus,
    fetchSettings,
    updateSetting,
    updateSettingsBatch,
    connectWS,
    disconnectWS
  }
})
