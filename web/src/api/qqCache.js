import service from '@/utils/request'

export const getQQCacheList = (data) => {
  return service({
    url: '/qqCache/list',
    method: 'post',
    data
  })
}

export const resetQQCacheExtract = (data) => {
  return service({
    url: '/qqCache/resetExtract',
    method: 'post',
    data
  })
}

export const exportQQCacheIniZip = (ids, config = {}) => {
  return service({
    url: '/qqCache/exportIniZip',
    method: 'post',
    data: { ids },
    responseType: 'blob',
    donNotShowLoading: true,
    ...config
  })
}

export const exportPendingQQCacheIniZip = (payload, config = {}) => {
  return service({
    url: '/qqCache/exportPendingIniZip',
    method: 'post',
    data: typeof payload === 'number' ? { count: payload } : payload,
    responseType: 'blob',
    donNotShowLoading: true,
    ...config
  })
}

export const settleQQCacheBilling = () => {
  return service({
    url: '/qqCache/billing/settle',
    method: 'post'
  })
}

export const getQQCacheBillingHistory = () => {
  return service({
    url: '/qqCache/billing/history',
    method: 'get'
  })
}
