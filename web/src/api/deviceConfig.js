import service from '@/utils/request'

export const getDeviceConfigList = (data) => {
  return service({
    url: '/deviceConfig/list',
    method: 'post',
    data
  })
}

export const saveDeviceConfig = (data) => {
  return service({
    url: '/deviceConfig/save',
    method: 'post',
    data
  })
}

export const deleteDeviceConfig = (data) => {
  return service({
    url: '/deviceConfig/delete',
    method: 'post',
    data
  })
}
