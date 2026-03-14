import request from './request'

export const quickApply = (): Promise<{ message: string; data: { sources: string[]; hash: string; path: string; revision: string } }> => {
  return request({
    url: '/config/quick-apply',
    method: 'POST'
  })
}

export const generateConfig = (sourceIds: number[]): Promise<any> => {
  return request({
    url: '/config/generate',
    method: 'POST',
    data: { source_ids: sourceIds }
  })
}

export const getConfig = (): Promise<any> => {
  return request({
    url: '/config',
    method: 'GET'
  })
}

export const saveConfig = (config: string, path: string): Promise<void> => {
  return request({
    url: '/config/save',
    method: 'POST',
    data: { config, path }
  })
}

export const applyConfig = (config: string, path?: string): Promise<void> => {
  return request({
    url: '/config/apply',
    method: 'POST',
    data: { config, path }
  })
}

export interface Revision {
  id: number
  version: string
  content: string
  source_hash?: string
  created_by?: string
  created_at: string
}

export const listRevisions = (limit = 50): Promise<Revision[]> => {
  return request({
    url: '/config/revisions',
    method: 'GET',
    params: { limit }
  })
}

export const getRevision = (id: number): Promise<Revision> => {
  return request({
    url: `/config/revisions/${id}`,
    method: 'GET'
  })
}

export const rollbackRevision = (id: number): Promise<{ revision: Revision; path: string }> => {
  return request({
    url: `/config/revisions/${id}/rollback`,
    method: 'POST'
  }).then((resp: any) => resp.data || resp)
}

export const deleteRevision = (id: number): Promise<void> => {
  return request({
    url: `/config/revisions/${id}`,
    method: 'DELETE'
  })
}
