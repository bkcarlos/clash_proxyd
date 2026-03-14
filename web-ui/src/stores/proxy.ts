import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as proxyApi from '@/api/proxy'

let proxiesInflight: Promise<void> | null = null
let trafficInflight: Promise<void> | null = null
let lastProxyFetchAt = 0
let lastTrafficFetchAt = 0
const MIN_FETCH_INTERVAL_MS = 1200

export const useProxyStore = defineStore('proxy', () => {
  const proxies = ref<Record<string, any>>({})
  const groups = ref<any[]>([])
  const rules = ref<any[]>([])
  const traffic = ref({ up: 0, down: 0 })
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchProxies = async (force = false): Promise<void> => {
    const now = Date.now()
    if (!force && now - lastProxyFetchAt < MIN_FETCH_INTERVAL_MS) {
      return
    }
    if (proxiesInflight) {
      return proxiesInflight
    }

    loading.value = true
    error.value = null

    proxiesInflight = (async () => {
      try {
        const data = await proxyApi.getProxies()
        proxies.value = data.proxies || {}

        const raw = data.proxies || {}
        groups.value = Object.entries(raw)
          .filter(([, value]) => typeof value === 'object' && value && Array.isArray((value as any).all))
          .map(([name, value]) => ({
            name,
            type: (value as any).type,
            now: (value as any).now,
            proxies: ((value as any).all || []).map((item: string) => ({ name: item }))
          }))

        lastProxyFetchAt = Date.now()
      } catch (err: any) {
        // 503 means mihomo is not running — not an error worth surfacing
        if (err?.response?.status !== 503) {
          error.value = err instanceof Error ? err.message : 'Failed to fetch proxies'
          throw err
        }
      } finally {
        loading.value = false
        proxiesInflight = null
      }
    })()

    return proxiesInflight
  }

  const fetchRules = async (): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      const data = await proxyApi.getRules()
      rules.value = data.rules || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch rules'
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchTraffic = async (force = false): Promise<void> => {
    const now = Date.now()
    if (!force && now - lastTrafficFetchAt < MIN_FETCH_INTERVAL_MS) {
      return
    }
    if (trafficInflight) {
      return trafficInflight
    }

    trafficInflight = (async () => {
      try {
        const data = await proxyApi.getTraffic()
        traffic.value = data
        lastTrafficFetchAt = Date.now()
      } catch (err: any) {
        // 503 means mihomo is not running — keep last known traffic values
        if (err?.response?.status !== 503) {
          error.value = err instanceof Error ? err.message : 'Failed to fetch traffic'
          throw err
        }
      } finally {
        trafficInflight = null
      }
    })()

    return trafficInflight
  }

  const testProxy = async (name: string, url?: string): Promise<{ delay: number; from_cache?: boolean }> => {
    try {
      return await proxyApi.testProxy(name, url)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to test proxy'
      throw err
    }
  }

  const switchProxy = async (group: string, proxy: string): Promise<void> => {
    try {
      await proxyApi.switchProxy(group, proxy)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to switch proxy'
      throw err
    }
  }

  const controlMihomo = async (action: 'start' | 'stop' | 'restart'): Promise<void> => {
    loading.value = true
    try {
      await proxyApi.controlMihomo(action)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to control mihomo'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    proxies,
    groups,
    rules,
    traffic,
    loading,
    error,
    fetchProxies,
    fetchRules,
    fetchTraffic,
    testProxy,
    switchProxy,
    controlMihomo
  }
})
