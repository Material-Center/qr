import service from '@/utils/request'

export const createRegisterTask = (data) => {
  return service({
    url: '/registerTask/create',
    method: 'post',
    data
  })
}

export const getActiveRegisterTask = () => {
  return service({
    url: '/registerTask/active',
    method: 'get'
  })
}

export const getActiveRegisterTasks = () => {
  return service({
    url: '/registerTask/actives',
    method: 'get'
  })
}

export const submitRegisterTaskStep = (data) => {
  return service({
    url: '/registerTask/step',
    method: 'post',
    data
  })
}

export const getRegisterTaskList = (data) => {
  return service({
    url: '/registerTask/list',
    method: 'post',
    data
  })
}

export const getRegisterTaskSummary = (params) => {
  return service({
    url: '/registerTask/summary',
    method: 'get',
    params
  })
}
