<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="完成时间">
          <el-date-picker
            v-model="searchInfo.finishedAtRange"
            type="datetimerange"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="finishedAtShortcuts"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            style="width: 360px"
          />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input
            v-model="searchInfo.phone"
            clearable
            placeholder="请输入手机号"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item label="QQ号">
          <el-input
            v-model="searchInfo.qqNum"
            clearable
            placeholder="请输入QQ号"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item v-if="showLeaderFilter" label="团长">
          <el-select
            v-model="searchInfo.leaderId"
            clearable
            filterable
            style="width: 180px"
            placeholder="请选择团长"
            @change="onLeaderChange"
          >
            <el-option
              v-for="item in leaderOptions"
              :key="item.ID"
              :label="`${item.nickName}(${item.userName})`"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="地推">
          <el-select
            v-model="searchInfo.promoterId"
            clearable
            filterable
            style="width: 180px"
            placeholder="请选择地推"
          >
            <el-option
              v-for="item in promoterOptions"
              :key="item.ID"
              :label="`${item.nickName}(${item.userName})`"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="收码方式">
          <el-select
            v-model="searchInfo.smsReceiveMode"
            clearable
            style="width: 180px"
          >
            <el-option label="平台发码" value="PLATFORM_SEND" />
            <el-option label="用户转发到腾讯" value="USER_SENT_TO_TX" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchInfo.status"
            clearable
            style="width: 180px"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="fetchAll">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-row :gutter="12" class="mb-3">
        <el-col :span="8">
          <el-card shadow="never">成功任务：{{ counters.success }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">失败任务：{{ counters.fail }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">处理中任务：{{ counters.processing }}</el-card>
        </el-col>
      </el-row>

      <el-table :data="tableData" row-key="ID">
        <el-table-column label="任务ID" min-width="90" prop="ID" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="scope">
            {{ safeFormatDate(scope.row.CreatedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="手机号" min-width="140" prop="phone" />
        <el-table-column label="收码方式" min-width="130">
          <template #default="scope">
            {{ smsModeText(scope.row.smsReceiveMode) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="140">
          <template #default="scope">
            <el-tag :type="statusTagType(scope.row.status)">
              {{ statusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="当前设备" min-width="140">
          <template #default="scope">
            {{ scope.row.holderDeviceId || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="QQ号" min-width="120" prop="qqNum" />
        <el-table-column label="地推" min-width="120">
          <template #default="scope">
            {{ scope.row.promoter?.nickName || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="团长" min-width="120">
          <template #default="scope">
            {{ scope.row.leader?.nickName || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="失败原因" min-width="200" prop="lastError" show-overflow-tooltip />
        <el-table-column label="完成时间" min-width="170">
          <template #default="scope">
            {{ safeFormatDate(scope.row.finishedAt) }}
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>

      <template v-if="showSummary">
        <el-divider />
        <el-row :gutter="12">
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>团长汇总</template>
              <el-table :data="summary.leaders" size="small">
                <el-table-column label="团长ID" prop="leaderId" width="90" />
                <el-table-column label="团长名称" prop="leaderName" min-width="100" />
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
              </el-table>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>地推汇总</template>
              <el-table :data="summary.promoters" size="small">
                <el-table-column label="地推ID" prop="promoterId" width="90" />
                <el-table-column label="地推名称" prop="promoterName" min-width="100" />
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getUserList } from '@/api/user'
import { getPhoneRegisterTaskList, getPhoneRegisterTaskSummary } from '@/api/phoneRegisterTask'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'PhoneRegisterTaskManage'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const leaderOptions = ref([])
const promoterOptions = ref([])
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
})
const summary = ref({
  leaders: [],
  promoters: []
})
const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const currentUserId = computed(() => userStore.userInfo?.ID)
const showLeaderFilter = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showSummary = computed(() => [ROLE_SUPER, ROLE_ADMIN, ROLE_LEADER].includes(currentRoleId.value))

const searchInfo = ref({
  promoterId: undefined,
  leaderId: undefined,
  status: undefined,
  smsReceiveMode: undefined,
  finishedAtRange: [],
  phone: '',
  qqNum: ''
})

const statusOptions = [
  { label: '待领取', value: 'pending' },
  { label: '执行中', value: 'running' },
  { label: '待地推验证码', value: 'waiting_promoter_code' },
  { label: '待上传缓存', value: 'registered_wait_upload' },
  { label: '成功', value: 'succeeded' },
  { label: '失败', value: 'failed' }
]

const dayStart = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(0, 0, 0, 0)
  return d
}

const dayEnd = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(23, 59, 59, 999)
  return d
}

const shiftDay = (base, days) => {
  const d = new Date(base)
  d.setDate(d.getDate() + days)
  return d
}

const finishedAtShortcuts = [
  {
    text: '今天',
    value: () => {
      const now = new Date()
      return [dayStart(now), now]
    }
  },
  {
    text: '昨天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -1)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '前天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -2)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '近一周',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -6)), now]
    }
  },
  {
    text: '近一月',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -29)), now]
    }
  }
]

const safeFormatDate = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return formatDate(value)
}

const smsModeText = (mode) => {
  if (mode === 'PLATFORM_SEND') return '平台发码'
  if (mode === 'USER_SENT_TO_TX') return '用户转发到腾讯'
  return mode || '-'
}

const statusText = (status) => {
  const map = {
    pending: '待领取',
    running: '执行中',
    waiting_promoter_code: '待地推验证码',
    registered_wait_upload: '待上传缓存',
    succeeded: '成功',
    failed: '失败'
  }
  return map[status] || status || '-'
}

const statusTagType = (status) => {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'waiting_promoter_code') return 'warning'
  return 'info'
}

const loadLeaderOptions = async () => {
  if (!showLeaderFilter.value) {
    leaderOptions.value = []
    return
  }
  const { data } = await getUserList({
    page: 1,
    pageSize: 1000,
    authorityId: ROLE_LEADER
  })
  leaderOptions.value = data?.list || []
}

const loadPromoterOptions = async () => {
  const params = {
    page: 1,
    pageSize: 1000,
    authorityId: ROLE_PROMOTER
  }
  if (currentRoleId.value === ROLE_LEADER) {
    params.leaderId = currentUserId.value
  } else if (showLeaderFilter.value && searchInfo.value.leaderId) {
    params.leaderId = searchInfo.value.leaderId
  }
  const { data } = await getUserList(params)
  promoterOptions.value = data?.list || []
}

const onLeaderChange = async () => {
  searchInfo.value.promoterId = undefined
  await loadPromoterOptions()
}

const fetchList = async () => {
  const [finishedAtStart, finishedAtEnd] = searchInfo.value.finishedAtRange || []
  const { data } = await getPhoneRegisterTaskList({
    page: page.value,
    pageSize: pageSize.value,
    promoterId: searchInfo.value.promoterId,
    leaderId: searchInfo.value.leaderId,
    status: searchInfo.value.status || undefined,
    smsReceiveMode: searchInfo.value.smsReceiveMode || undefined,
    phone: String(searchInfo.value.phone || '').trim() || undefined,
    qqNum: String(searchInfo.value.qqNum || '').trim() || undefined,
    finishedAtStart: finishedAtStart || undefined,
    finishedAtEnd: finishedAtEnd || undefined
  })
  tableData.value = data?.list || []
  total.value = data?.total || 0
  counters.value = {
    success: data?.successCount || 0,
    fail: data?.failCount || 0,
    processing: data?.processingCount || 0
  }
}

const fetchSummary = async () => {
  if (!showSummary.value) {
    summary.value = { leaders: [], promoters: [] }
    return
  }
  const { data } = await getPhoneRegisterTaskSummary({
    leaderId: searchInfo.value.leaderId || undefined
  })
  summary.value = data || { leaders: [], promoters: [] }
}

const fetchAll = async () => {
  try {
    await Promise.all([fetchList(), fetchSummary()])
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const resetSearch = () => {
  searchInfo.value = {
    promoterId: undefined,
    leaderId: currentRoleId.value === ROLE_LEADER ? currentUserId.value : undefined,
    status: undefined,
    smsReceiveMode: undefined,
    finishedAtRange: [],
    phone: '',
    qqNum: ''
  }
  page.value = 1
  loadPromoterOptions()
  fetchAll()
}

const handleCurrentChange = (val) => {
  page.value = val
  fetchList()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  page.value = 1
  fetchList()
}

onMounted(async () => {
  if (currentRoleId.value === ROLE_LEADER) {
    searchInfo.value.leaderId = currentUserId.value
  }
  await Promise.all([loadLeaderOptions(), loadPromoterOptions()])
  await fetchAll()
})
</script>
