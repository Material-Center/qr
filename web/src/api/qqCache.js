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
