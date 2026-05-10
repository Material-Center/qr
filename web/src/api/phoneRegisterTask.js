import service from '@/utils/request'

export const createPhoneRegisterTask = (data) => {
  return service({
    url: '/phoneRegisterTask/create',
    method: 'post',
    data
  })
}

export const submitPhoneRegisterTaskCode = (data) => {
  return service({
    url: '/phoneRegisterTask/submitCode',
    method: 'post',
    data
  })
}

export const getActivePhoneRegisterTask = () => {
  return service({
    url: '/phoneRegisterTask/active',
    method: 'get'
  })
}

export const getActivePhoneRegisterTasks = () => {
  return service({
    url: '/phoneRegisterTask/actives',
    method: 'get'
  })
}

export const getPhoneRegisterTaskList = (data) => {
  return service({
    url: '/phoneRegisterTask/list',
    method: 'post',
    data
  })
}

export const getPhoneRegisterTaskSummary = (params) => {
  return service({
    url: '/phoneRegisterTask/summary',
    method: 'get',
    params
  })
}

export const settlePhoneRegisterTaskLeader = (data) => {
  return service({
    url: '/phoneRegisterTask/settle',
    method: 'post',
    data
  })
}

export const getPhoneRegisterTaskLogs = (data) => {
  return service({
    url: '/phoneRegisterTask/logs',
    method: 'post',
    data
  })
}
