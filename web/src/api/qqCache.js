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
