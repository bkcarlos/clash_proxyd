import request from './request'

export const getProxies = (): Promise<any> => {
  return request({
    url: '/proxy/proxies',
    method: 'GET'
  })
}

export const getProxy = (name: string): Promise<any> => {
  return request({
    url: `/proxy/proxies/${name}`,
    method: 'GET'
  })
}

export interface ProxyDelayResult {
  delay: number
  from_cache?: boolean
  error?: string
}

export const testProxy = (name: string, url?: string): Promise<ProxyDelayResult> => {
  return request({
    url: `/proxy/proxies/${name}/test`,
    method: 'POST',
    data: { url: url || 'http://www.gstatic.com/generate_204', timeout: 3000 }
  }).then((data: any) => ({ delay: data.delay ?? 0, from_cache: data.from_cache, error: data.error }))
}

export const switchProxy = (group: string, proxy: string): Promise<void> => {
  return request({
    url: `/proxy/groups/${group}`,
    method: 'PUT',
    data: { proxy }
  })
}

export const getProxyGroups = (): Promise<any> => {
  return request({
    url: '/proxy/groups',
    method: 'GET'
  })
}

export const getRules = (): Promise<any> => {
  return request({
    url: '/proxy/rules',
    method: 'GET'
  })
}

export const getTraffic = (): Promise<{ up: number; down: number }> => {
  return request({
    url: '/proxy/traffic',
    method: 'GET'
  })
}

export const getMemory = (): Promise<any> => {
  return request({
    url: '/proxy/memory',
    method: 'GET'
  })
}

export const controlMihomo = (action: 'start' | 'stop' | 'restart' | 'status'): Promise<any> => {
  return request({
    url: `/proxy/mihomo/${action}`,
    method: 'POST'
  })
}

export interface MihomoVersionInfo {
  version: string
  installed: boolean
  binary_path: string
}

export interface MihomoInstallStatus {
  installed: boolean
  current_version: string
  latest_version: string
  needs_update: boolean
  binary_path: string
  is_running: boolean
  pid: number
}

export const getMihomoVersionList = (): Promise<{ versions: string[] }> => {
  return request({
    url: '/proxy/mihomo/versions',
    method: 'GET'
  })
}

export const getMihomoInstallStatus = (): Promise<MihomoInstallStatus> => {
  return request({
    url: '/proxy/mihomo/install-status',
    method: 'GET'
  })
}

export const getMihomoVersion = (): Promise<MihomoVersionInfo> => {
  return request({
    url: '/proxy/mihomo/version',
    method: 'GET'
  })
}

export const getMihomoReleases = (): Promise<{ latest_version: string }> => {
  return request({
    url: '/proxy/mihomo/releases',
    method: 'GET'
  })
}

export interface UpdateResult {
  updated: boolean
  current_version?: string
  latest_version?: string
  old_version?: string
  new_version?: string
  downloaded_from?: string
}

export interface InstallProgress {
  running: boolean
  stage: string
  percent: number
  message: string
  error: string
  started_at: string
  finished_at?: string
  old_version?: string
  new_version?: string
  downloaded_from?: string
}

export const startInstallJob = (version?: string, force?: boolean): Promise<any> => {
  return request({
    url: '/proxy/mihomo/install-job',
    method: 'POST',
    data: { version: version || '', force: !!force }
  })
}

export const getInstallProgress = (): Promise<InstallProgress> => {
  return request({
    url: '/proxy/mihomo/install-progress',
    method: 'GET'
  })
}

export const updateMihomo = (version?: string, force?: boolean): Promise<UpdateResult> => {
  return request({
    url: '/proxy/mihomo/update',
    method: 'POST',
    data: { version: version || '', force: !!force },
    timeout: 30 * 60 * 1000  // 30 minutes — binary download can be slow
  })
}
