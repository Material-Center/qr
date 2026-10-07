import service from '@/utils/request'

export const getMIEnvAdminList = (data) => {
  return service({
    url: '/miEnvAdmin/list',
    method: 'post',
    data
  })
}

export const getMIEnvAdminTypes = (data) => {
  return service({
    url: '/miEnvAdmin/types',
    method: 'post',
    data,
    donNotShowLoading: true
  })
}

export const deleteAllMIEnv = (data) => {
  return service({
    url: '/miEnvAdmin/deleteAll',
    method: 'post',
    data
  })
}

export const deleteSelectedMIEnv = (data) => {
  return service({
    url: '/miEnvAdmin/deleteSelected',
    method: 'post',
    data
  })
}
