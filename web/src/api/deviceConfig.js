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

export const batchUpdateDeviceConfig = (data) => {
  return service({
    url: '/deviceConfig/batchUpdate',
    method: 'post',
    data
  })
}

export const getDeviceGroups = () => {
  return service({
    url: '/deviceConfig/group/list',
    method: 'get'
  })
}

export const saveDeviceGroup = (data) => {
  return service({
    url: '/deviceConfig/group/save',
    method: 'post',
    data
  })
}

export const deleteDeviceGroup = (data) => {
  return service({
    url: '/deviceConfig/group/delete',
    method: 'post',
    data
  })
}
