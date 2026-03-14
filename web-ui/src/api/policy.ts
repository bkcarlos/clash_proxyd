import request from './request'

export const generateGroups = (proxyNames: string[]): Promise<any> => {
  return request({
    url: '/policy/groups',
    method: 'POST',
    data: { proxy_names: proxyNames }
  })
}

export const generateRules = (customRules: string[]): Promise<any> => {
  return request({
    url: '/policy/rules',
    method: 'POST',
    data: { custom_rules: customRules }
  })
}

export const validateRule = (rule: string): Promise<void> => {
  return request({
    url: '/policy/validate-rule',
    method: 'POST',
    data: { rule }
  })
}

export const createCustomGroup = (config: {
  name: string
  type: string
  proxies: string[]
  url?: string
  interval?: number
}): Promise<any> => {
  return request({
    url: '/policy/custom-group',
    method: 'POST',
    data: config
  })
}
