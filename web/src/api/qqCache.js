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

export const exportQQCacheAccountList = (payload, config = {}) => {
  return service({
    url: '/qqCache/exportAccountList',
    method: 'post',
    data: payload,
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

export const exportQQCacheIniZipByQQFile = (file, config = {}) => {
  const { markExtracted = false, ...requestConfig } = config || {}
  const form = new FormData()
  form.append('qqFile', file)
  if (markExtracted) {
    form.append('markExtracted', 'true')
  }
  return service({
    url: '/qqCache/exportIniZipByQQFile',
    method: 'post',
    data: form,
    responseType: 'blob',
    donNotShowLoading: true,
    ...requestConfig
  })
}

export const importQQCacheZip = (files, config = {}) => {
  const form = new FormData()
  const fileList = Array.isArray(files) ? files : [files]
  fileList.filter(Boolean).forEach((file) => {
    form.append('cacheZip', file)
  })
  return service({
    url: '/qqCache/importZip',
    method: 'post',
    data: form,
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

export const getQQCacheSalesSummary = (params = {}) => {
  return service({
    url: '/qqCache/sales/summary',
    method: 'get',
    params
  })
}

export const exportSalesQQCacheIniZip = (payload, config = {}) => {
  return service({
    url: '/qqCache/sales/extract',
    method: 'post',
    data: payload,
    responseType: 'blob',
    donNotShowLoading: true,
    ...config
  })
}

export const getQQCacheSalesHistory = (data) => {
  return service({
    url: '/qqCache/sales/history',
    method: 'post',
    data
  })
}

export const getQQCacheSalesSummaryList = (params = {}) => {
  return service({
    url: '/qqCache/sales/summaryList',
    method: 'get',
    params
  })
}

export const settleQQCacheSalesBilling = (data) => {
  return service({
    url: '/qqCache/sales/settle',
    method: 'post',
    data
  })
}

export const getQQCacheSalesSettlementHistory = (params) => {
  return service({
    url: '/qqCache/sales/settlement/history',
    method: 'get',
    params
  })
}
