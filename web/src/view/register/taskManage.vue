<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
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
        <el-form-item label="是否处理中">
          <el-select v-model="searchInfo.unfinished" clearable style="width: 120px">
            <el-option :value="true" label="是" />
            <el-option :value="false" label="否" />
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
        <el-table-column label="手机号" min-width="140" prop="phone" />
        <el-table-column label="当前步骤" min-width="120">
          <template #default="scope">
            {{ stepText(scope.row.currentStep) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="statusTagType(scope.row)">
              {{ statusText(scope.row) }}
            </el-tag>
          </template>
        </el-table-column>
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
        <el-table-column label="失败原因" min-width="160" prop="lastError" show-overflow-tooltip />
        <el-table-column label="完成时间" min-width="170">
          <template #default="scope">
            {{ scope.row.finishedAt ? formatDate(scope.row.finishedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column v-if="canDownloadCache" label="操作" width="130" fixed="right">
          <template #default="scope">
            <el-button
              type="primary"
              link
              :disabled="!scope.row.loginCacheIni"
              @click="downloadCacheFile(scope.row)"
            >
              下载缓存
            </el-button>
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
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { downloadRegisterTaskCache, getRegisterTaskList, getRegisterTaskSummary } from '@/api/registerTask'
import { getUserList } from '@/api/user'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'RegisterTaskManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300
const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const currentUserId = computed(() => userStore.userInfo?.ID)
const canDownloadCache = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showLeaderFilter = computed(() => currentRoleId.value === ROLE_ADMIN)
const searchInfo = ref({
  promoterId: undefined,
  leaderId: undefined,
  unfinished: undefined
})
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

const stepText = (step) => {
  if (step === 'phone_bind') return '查绑'
  if (step === 'change_password') return '改密'
  if (step === 'login') return '登录'
  return step || '-'
}

const statusText = (task) => {
  if (!task?.finishedAt) return '处理中'
  if (task?.statusCode === 0) return '成功'
  return '失败'
}

const statusTagType = (task) => {
  if (!task?.finishedAt) return 'warning'
  if (task?.statusCode === 0) return 'success'
  return 'danger'
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
  const { data } = await getRegisterTaskList({
    page: page.value,
    pageSize: pageSize.value,
    ...searchInfo.value
  })
  tableData.value = data.list || []
  total.value = data.total || 0
  counters.value = {
    success: data.successCount || 0,
    fail: data.failCount || 0,
    processing: data.processingCount || 0
  }
}

const fetchSummary = async () => {
  const { data } = await getRegisterTaskSummary({
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
    unfinished: undefined
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

const parseFilenameFromDisposition = (disposition) => {
  const source = String(disposition || '')
  if (!source) return ''
  const utfMatch = source.match(/filename\*=UTF-8''([^;]+)/i)
  if (utfMatch?.[1]) {
    try {
      return decodeURIComponent(utfMatch[1])
    } catch (e) {
      return utfMatch[1]
    }
  }
  const basicMatch = source.match(/filename="?([^";]+)"?/i)
  return basicMatch?.[1] || ''
}

const downloadCacheFile = async (row) => {
  if (!row?.ID) return
  try {
    const rsp = await downloadRegisterTaskCache({ taskId: row.ID })
    const contentType = String(rsp?.headers?.['content-type'] || rsp?.headers?.['Content-Type'] || '').toLowerCase()
    const buffer = rsp?.data
    if (contentType.includes('application/json')) {
      const text = new TextDecoder('utf-8').decode(buffer)
      try {
        const parsed = JSON.parse(text)
        ElMessage.error(parsed?.msg || '下载失败')
      } catch (e) {
        ElMessage.error(text || '下载失败')
      }
      return
    }
    const blob = new Blob([buffer], { type: 'text/plain;charset=utf-8' })
    const disposition = rsp?.headers?.['content-disposition'] || rsp?.headers?.['Content-Disposition']
    const filename = parseFilenameFromDisposition(disposition) || `register_task_${row.ID}.ini`
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    // 不能立即 revoke，部分浏览器会导致下载任务卡住在 0B。
    window.setTimeout(() => {
      window.URL.revokeObjectURL(url)
    }, 10000)
    ElMessage.success('缓存下载成功')
  } catch (e) {
    ElMessage.error(e?.message || '下载失败')
  }
}

onMounted(async () => {
  if (currentRoleId.value === ROLE_LEADER) {
    searchInfo.value.leaderId = currentUserId.value
  }
  await Promise.all([loadLeaderOptions(), loadPromoterOptions()])
  await fetchAll()
})
</script>
