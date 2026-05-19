import service from '@/utils/request'

export const createRegisterTask = (data) => {
  return service({
    url: '/registerTask/create',
    method: 'post',
    data
  })
}

export const getActiveRegisterTask = (config = {}) => {
  return service({
    url: '/registerTask/active',
    method: 'get',
    ...config
  })
}

export const getActiveRegisterTasks = (config = {}) => {
  return service({
    url: '/registerTask/actives',
    method: 'get',
    ...config
  })
}

export const submitRegisterTaskStep = (data) => {
  return service({
    url: '/registerTask/step',
    method: 'post',
    data
  })
}

export const getRegisterTaskList = (data, config = {}) => {
  return service({
    url: '/registerTask/list',
    method: 'post',
    data,
    ...config
  })
}

export const getRegisterTaskSummary = (params) => {
  return service({
    url: '/registerTask/summary',
    method: 'get',
    params
  })
}

export const settleRegisterTaskLeader = (data) => {
  return service({
    url: '/registerTask/settle',
    method: 'post',
    data
  })
}

export const getRegisterTaskSettlementHistory = (params) => {
  return service({
    url: '/registerTask/settlement/history',
    method: 'get',
    params
  })
}

export const startRegisterTaskDebugLogin = (data) => {
  return service({
    url: '/registerTask/debug/login/start',
    method: 'post',
    data
  })
}

export const submitRegisterTaskDebugLoginCode = (data) => {
  return service({
    url: '/registerTask/debug/login/submit',
    method: 'post',
    data
  })
}

export const getRegisterTaskDebugLoginTask = (params) => {
  return service({
    url: '/registerTask/debug/login/task',
    method: 'get',
    params
  })
}

export const downloadRegisterTaskCache = (params) => {
  return service({
    url: '/registerTask/cache/download',
    method: 'get',
    params,
    responseType: 'arraybuffer',
    donNotShowLoading: true
  })
}
