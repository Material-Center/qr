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

export const getActivePhoneRegisterTask = (config = {}) => {
  return service({
    url: '/phoneRegisterTask/active',
    method: 'get',
    ...config
  })
}

export const getActivePhoneRegisterTasks = (config = {}) => {
  return service({
    url: '/phoneRegisterTask/actives',
    method: 'get',
    ...config
  })
}

export const getPhoneRegisterSubmitStatus = (config = {}) => {
  return service({
    url: '/phoneRegisterTask/submitStatus',
    method: 'get',
    ...config
  })
}

export const getPhoneRegisterTaskList = (data, config = {}) => {
  return service({
    url: '/phoneRegisterTask/list',
    method: 'post',
    data,
    ...config
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

export const getPhoneRegisterTaskSettlementHistory = (params) => {
  return service({
    url: '/phoneRegisterTask/settlement/history',
    method: 'get',
    params
  })
}

export const getPhoneRegisterTaskLogs = (data) => {
  return service({
    url: '/phoneRegisterTask/logs',
    method: 'post',
    data
  })
}
