import service from '@/utils/request'

export const getMyRegisterConfig = () => {
  return service({
    url: '/registerConfig/getMyConfig',
    method: 'get'
  })
}

export const setMyRegisterConfig = (data) => {
  return service({
    url: '/registerConfig/setMyConfig',
    method: 'put',
    data
  })
}
