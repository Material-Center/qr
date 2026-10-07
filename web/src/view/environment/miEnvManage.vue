<template>
  <div class="mi-env-page">
    <div class="app-search-box">
      <el-form :inline="true" :model="searchInfo" label-position="right">
        <el-form-item label="更新时间">
          <el-date-picker
            v-model="searchInfo.updatedAtRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 360px"
          />
        </el-form-item>
        <el-form-item label="设备ID">
          <el-input v-model="searchInfo.deviceId" clearable placeholder="请输入完整设备ID" style="width: 180px" />
        </el-form-item>
        <el-form-item label="设备分组">
          <el-select v-model="searchInfo.groupValue" clearable placeholder="全部" style="width: 180px">
            <el-option label="未分组" :value="UNGROUPED_VALUE" />
            <el-option
              v-for="item in deviceGroups"
              :key="item.ID"
              :label="item.name"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select
            v-model="searchInfo.type"
            clearable
            filterable
            allow-create
            default-first-option
            :loading="typeLoading"
            placeholder="请选择或输入类型"
            style="width: 180px"
            @visible-change="(visible) => visible && fetchTypeOptions()"
          >
            <el-option v-for="item in typeOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item label="使用次数">
          <div class="range-inputs">
            <el-input-number v-model="searchInfo.minUsage" :min="0" :controls="false" placeholder="最小" />
            <span>至</span>
            <el-input-number v-model="searchInfo.maxUsage" :min="0" :controls="false" placeholder="最大" />
          </div>
        </el-form-item>
        <el-form-item label="制作次数">
          <div class="range-inputs">
            <el-input-number v-model="searchInfo.minMadeCount" :min="0" :controls="false" placeholder="最小" />
            <span>至</span>
            <el-input-number v-model="searchInfo.maxMadeCount" :min="0" :controls="false" placeholder="最大" />
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" clearable placeholder="全部" style="width: 130px">
            <el-option label="正常" value="normal" />
            <el-option label="冻结" value="frozen" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="app-table-box">
      <div class="table-toolbar">
        <div class="total-card">
          <span class="total-label">环境总数</span>
          <span class="total-value">{{ total }}</span>
        </div>
        <div class="app-btn-list">
          <el-button icon="refresh" @click="fetchList">刷新</el-button>
          <el-button
            type="danger"
            plain
            icon="delete"
            :disabled="selectedRows.length === 0"
            @click="onDeleteSelected"
          >
            删除所选{{ selectedRows.length ? `（${selectedRows.length}）` : '' }}
          </el-button>
          <el-button type="danger" icon="delete" :disabled="total === 0" @click="onDeleteAll">
            删除全部
          </el-button>
        </div>
      </div>

      <el-table
        v-loading="loading"
        :data="tableData"
        row-key="id"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" fixed="left" />
        <el-table-column label="ID" prop="id" width="90" />
        <el-table-column label="设备ID" prop="deviceId" min-width="150" show-overflow-tooltip />
        <el-table-column label="类型" prop="type" width="110" />
        <el-table-column label="使用次数" width="110" align="center">
          <template #default="{ row }">
            {{ row.usageCount }} / {{ row.maxUsage }}
          </template>
        </el-table-column>
        <el-table-column label="制作次数" prop="madeCount" width="100" align="center" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.frozen ? 'danger' : 'success'">
              {{ row.frozen ? '冻结' : '正常' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="备份名称" prop="serialBackupName" min-width="220" show-overflow-tooltip />
        <el-table-column label="安卓ID" prop="androidId" min-width="150" show-overflow-tooltip />
        <el-table-column label="最后使用时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row.lastUsedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="170" fixed="right">
          <template #default="{ row }">
            {{ formatRecordDate(row.updatedAt) }}
          </template>
        </el-table-column>
      </el-table>

      <div class="app-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[20, 50, 100, 200]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import {
  deleteAllMIEnv,
  deleteSelectedMIEnv,
  getMIEnvAdminList,
  getMIEnvAdminTypes
} from '@/api/miEnvAdmin'
import { getDeviceGroups } from '@/api/deviceConfig'

defineOptions({
  name: 'MIEnvManage'
})

const UNGROUPED_VALUE = '__ungrouped__'

const emptySearch = () => ({
  updatedAtRange: [],
  deviceId: '',
  groupValue: '',
  type: '',
  minUsage: undefined,
  maxUsage: undefined,
  minMadeCount: undefined,
  maxMadeCount: undefined,
  status: ''
})

const searchInfo = ref(emptySearch())
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const tableData = ref([])
const selectedRows = ref([])
const deviceGroups = ref([])
const typeOptions = ref([])
const typeLoading = ref(false)
let typeFetchTimer
let typeRequestVersion = 0

const normalizedNumber = (value) => {
  return value === '' || value === null || value === undefined ? undefined : Number(value)
}

const buildDeviceScope = () => {
  const filters = {
    deviceId: String(searchInfo.value.deviceId || '').trim() || undefined
  }
  if (searchInfo.value.groupValue === UNGROUPED_VALUE) {
    filters.ungrouped = true
  } else if (searchInfo.value.groupValue) {
    filters.groupId = Number(searchInfo.value.groupValue)
  }
  return filters
}

const buildFilters = () => {
  const range = searchInfo.value.updatedAtRange || []
  return {
    updatedAtStart: range[0] || undefined,
    updatedAtEnd: range[1] || undefined,
    ...buildDeviceScope(),
    type: String(searchInfo.value.type || '').trim() || undefined,
    minUsage: normalizedNumber(searchInfo.value.minUsage),
    maxUsage: normalizedNumber(searchInfo.value.maxUsage),
    minMadeCount: normalizedNumber(searchInfo.value.minMadeCount),
    maxMadeCount: normalizedNumber(searchInfo.value.maxMadeCount),
    status: searchInfo.value.status || undefined
  }
}

const fetchDeviceGroups = async () => {
  const { data } = await getDeviceGroups()
  deviceGroups.value = data || []
}

const fetchTypeOptions = async () => {
  const requestVersion = ++typeRequestVersion
  typeLoading.value = true
  try {
    const { data } = await getMIEnvAdminTypes(buildDeviceScope())
    if (requestVersion === typeRequestVersion) {
      typeOptions.value = data || []
    }
  } finally {
    if (requestVersion === typeRequestVersion) {
      typeLoading.value = false
    }
  }
}

const validateRanges = () => {
  const filters = buildFilters()
  if (filters.minUsage !== undefined && filters.maxUsage !== undefined && filters.minUsage > filters.maxUsage) {
    ElMessage.warning('使用次数最小值不能大于最大值')
    return false
  }
  if (filters.minMadeCount !== undefined && filters.maxMadeCount !== undefined && filters.minMadeCount > filters.maxMadeCount) {
    ElMessage.warning('制作次数最小值不能大于最大值')
    return false
  }
  return true
}

const fetchList = async () => {
  if (!validateRanges()) return
  loading.value = true
  try {
    const { data } = await getMIEnvAdminList({
      page: page.value,
      pageSize: pageSize.value,
      ...buildFilters()
    })
    tableData.value = data?.list || []
    selectedRows.value = []
    total.value = Number(data?.total || 0)
    if (page.value > 1 && tableData.value.length === 0 && total.value > 0) {
      page.value = Math.ceil(total.value / pageSize.value)
      await fetchList()
    }
  } finally {
    loading.value = false
  }
}

const onSearch = async () => {
  page.value = 1
  await fetchList()
}

const resetSearch = async () => {
  searchInfo.value = emptySearch()
  page.value = 1
  await fetchList()
}

const onDeleteSelected = async () => {
  const ids = selectedRows.value.map((row) => Number(row.id)).filter((id) => id > 0)
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确认删除所选的 ${ids.length} 条环境记录？此操作会立即影响客户端环境查询。`,
      '删除所选环境',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }
  const { data } = await deleteSelectedMIEnv({ ids, confirm: true })
  ElMessage.success(`已删除 ${Number(data?.deleted || 0)} 条环境记录`)
  selectedRows.value = []
  await fetchList()
}

const onDeleteAll = async () => {
  if (!validateRanges() || total.value === 0) return
  try {
    await ElMessageBox.confirm(
      `将按当前筛选条件删除全部 ${total.value} 条环境记录。此操作会立即影响客户端环境查询，确认继续？`,
      '删除全部环境',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }
  const { data } = await deleteAllMIEnv({
    ...buildFilters(),
    confirm: true
  })
  ElMessage.success(`已删除 ${Number(data?.deleted || 0)} 条环境记录`)
  page.value = 1
  await fetchList()
}

const handleCurrentChange = (value) => {
  page.value = value
  fetchList()
}

const handleSizeChange = (value) => {
  pageSize.value = value
  page.value = 1
  fetchList()
}

const handleSelectionChange = (rows) => {
  selectedRows.value = rows
}

const formatRecordDate = (value) => {
  return value ? formatDate(value) : '-'
}

watch(
  () => [searchInfo.value.deviceId, searchInfo.value.groupValue],
  () => {
    clearTimeout(typeFetchTimer)
    typeFetchTimer = setTimeout(fetchTypeOptions, 250)
  }
)

onBeforeUnmount(() => clearTimeout(typeFetchTimer))
onMounted(() => Promise.all([fetchDeviceGroups(), fetchTypeOptions(), fetchList()]))
</script>

<style scoped>
.range-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.range-inputs :deep(.el-input-number) {
  width: 90px;
}

.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.total-card {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.total-label {
  color: var(--el-text-color-secondary);
}

.total-value {
  color: var(--el-color-primary);
  font-size: 24px;
  font-weight: 600;
}
</style>
