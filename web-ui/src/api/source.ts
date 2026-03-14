import request from './request'

export interface Source {
  id: number
  name: string
  type: 'http' | 'file' | 'local'
  url?: string
  path?: string
  update_interval: number
  update_cron?: string
  enabled: boolean
  priority: number
  config_override?: string
  created_at: string
  updated_at: string
}

export const listSources = (): Promise<Source[]> => {
  return request({
    url: '/sources',
    method: 'GET'
  })
}

export const getSource = (id: number): Promise<Source> => {
  return request({
    url: `/sources/${id}`,
    method: 'GET'
  })
}

export const createSource = (data: Omit<Source, 'id' | 'created_at' | 'updated_at'>): Promise<Source> => {
  return request({
    url: '/sources',
    method: 'POST',
    data
  })
}

export const updateSource = (id: number, data: Omit<Source, 'id' | 'created_at' | 'updated_at'>): Promise<Source> => {
  return request({
    url: `/sources/${id}`,
    method: 'PUT',
    data
  })
}

export const deleteSource = (id: number): Promise<void> => {
  return request({
    url: `/sources/${id}`,
    method: 'DELETE'
  })
}

export const testSource = (id: number): Promise<any> => {
  return request({
    url: `/sources/${id}/test`,
    method: 'POST'
  })
}

export const fetchSource = (id: number): Promise<{ size: number; hash: string; last_fetch: string }> => {
  return request({
    url: `/sources/${id}/fetch`,
    method: 'POST'
  })
}

