<template>
  <div class="device-config-page">
    <div class="app-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备ID">
          <el-input v-model="searchInfo.deviceId" clearable placeholder="请输入设备ID" />
        </el-form-item>
        <el-form-item label="账号类型">
          <el-select v-model="searchInfo.accountType" clearable style="width: 150px">
            <el-option
              v-for="item in accountTypes"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="设备分组">
          <el-select v-model="searchInfo.groupValue" clearable style="width: 180px">
            <el-option label="未分组" :value="UNGROUPED_VALUE" />
            <el-option
              v-for="item in deviceGroups"
              :key="item.ID"
              :label="item.name"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="app-table-box">
      <div class="app-btn-list">
        <el-button type="primary" icon="plus" @click="openDialog()">新增设备</el-button>
        <el-button type="primary" plain icon="edit" :disabled="!selectedRows.length" @click="openBatchDialog">批量设置</el-button>
        <el-button type="primary" plain icon="folder" @click="openGroupManage">分组管理</el-button>
        <el-button icon="refresh" @click="fetchAll">刷新</el-button>
      </div>

      <el-table ref="tableRef" :data="tableData" row-key="ID" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column align="left" label="ID" prop="ID" width="90" />
        <el-table-column align="left" label="设备ID" prop="deviceId" min-width="180" show-overflow-tooltip />
        <el-table-column align="left" label="账号类型" width="140">
          <template #default="{ row }">
            {{ accountTypeLabel(row.accountType) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="设备分组" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ groupLabel(row) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="备注" prop="remark" min-width="180" show-overflow-tooltip />
        <el-table-column align="left" label="创建时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row, 'createdAt') }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="更新时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row, 'updatedAt') }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link icon="edit" @click="openDialog(row)">编辑</el-button>
            <el-button type="danger" link icon="delete" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="app-pagination">
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
    </div>

    <el-dialog v-model="dialogVisible" :title="form.ID ? '编辑设备配置' : '新增设备配置'" width="520px">
      <el-form ref="formRef" :model="form" label-width="90px">
        <el-form-item label="设备ID" required>
          <el-input v-model="form.deviceId" :disabled="!!form.ID" clearable placeholder="请输入设备ID" />
        </el-form-item>
        <el-form-item label="账号类型" required>
          <el-select v-model="form.accountType" style="width: 100%">
            <el-option
              v-for="item in accountTypes"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="设备分组">
          <el-select v-model="form.groupId" clearable style="width: 100%" placeholder="未分组">
            <el-option
              v-for="item in deviceGroups"
              :key="item.ID"
              :label="item.name"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" maxlength="255" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchDialogVisible" title="批量设置设备" width="520px">
      <el-form :model="batchForm" label-width="90px">
        <el-form-item label="已选设备">
          <span>{{ selectedRows.length }} 个</span>
        </el-form-item>
        <el-form-item label="账号类型">
          <div class="batch-field">
            <el-checkbox v-model="batchForm.updateAccountType">更新</el-checkbox>
            <el-select v-model="batchForm.accountType" :disabled="!batchForm.updateAccountType" style="width: 220px">
              <el-option
                v-for="item in accountTypes"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="设备分组">
          <div class="batch-field">
            <el-checkbox v-model="batchForm.updateGroup">更新</el-checkbox>
            <el-select v-model="batchForm.groupValue" :disabled="!batchForm.updateGroup" style="width: 220px">
              <el-option label="未分组" :value="CLEAR_GROUP_VALUE" />
              <el-option
                v-for="item in deviceGroups"
                :key="item.ID"
                :label="item.name"
                :value="item.ID"
              />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="备注">
          <div class="batch-field batch-field-vertical">
            <el-checkbox v-model="batchForm.updateRemark">更新</el-checkbox>
            <el-input
              v-model="batchForm.remark"
              :disabled="!batchForm.updateRemark"
              type="textarea"
              :rows="3"
              maxlength="255"
              show-word-limit
              placeholder="留空将清空备注"
            />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onBatchSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="groupDialogVisible" title="分组管理" width="680px">
      <div class="app-btn-list group-actions">
        <el-button type="primary" icon="plus" @click="openGroupForm()">新增分组</el-button>
        <el-button icon="refresh" @click="fetchDeviceGroups">刷新</el-button>
      </div>
      <el-table :data="deviceGroups" row-key="ID" size="small">
        <el-table-column label="ID" prop="ID" width="90" />
        <el-table-column label="分组名称" prop="name" min-width="160" show-overflow-tooltip />
        <el-table-column label="备注" prop="remark" min-width="180" show-overflow-tooltip />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row, 'createdAt') }}
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">
            {{ formatRecordDate(row, 'updatedAt') }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link icon="edit" @click="openGroupForm(row)">编辑</el-button>
            <el-button type="danger" link icon="delete" @click="onDeleteGroup(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="groupFormDialogVisible" :title="groupForm.ID ? '编辑分组' : '新增分组'" width="460px">
      <el-form :model="groupForm" label-width="90px">
        <el-form-item label="分组名称" required>
          <el-input v-model="groupForm.name" clearable maxlength="64" show-word-limit placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="groupForm.remark" type="textarea" :rows="3" maxlength="255" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupFormDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onSaveGroup">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { getQQCacheAccountTypes } from '@/api/qqCache'
import {
  batchUpdateDeviceConfig,
  deleteDeviceConfig,
  deleteDeviceGroup,
  getDeviceConfigList,
  getDeviceGroups,
  saveDeviceConfig,
  saveDeviceGroup
} from '@/api/deviceConfig'

defineOptions({
  name: 'DeviceConfig'
})

const UNGROUPED_VALUE = '__ungrouped__'
const CLEAR_GROUP_VALUE = '__clear_group__'

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const tableRef = ref()
const selectedRows = ref([])
const accountTypes = ref([])
const deviceGroups = ref([])
const dialogVisible = ref(false)
const batchDialogVisible = ref(false)
const groupDialogVisible = ref(false)
const groupFormDialogVisible = ref(false)
const form = ref({
  ID: 0,
  deviceId: '',
  accountType: 'default',
  groupId: undefined,
  remark: ''
})
const batchForm = ref({
  updateAccountType: false,
  accountType: 'default',
  updateGroup: false,
  groupValue: CLEAR_GROUP_VALUE,
  updateRemark: false,
  remark: ''
})
const groupForm = ref({
  ID: 0,
  name: '',
  remark: ''
})
const searchInfo = ref({
  deviceId: '',
  accountType: '',
  groupValue: ''
})

const accountTypeLabel = (value) => {
  return accountTypes.value.find((item) => item.value === value)?.label || '默认账号'
}

const groupLabel = (row) => {
  if (!row?.groupId) return '未分组'
  return row.group?.name || deviceGroups.value.find((item) => Number(item.ID) === Number(row.groupId))?.name || `ID ${row.groupId}`
}

const formatRecordDate = (row, key) => {
  const pascalKey = key === 'createdAt' ? 'CreatedAt' : 'UpdatedAt'
  const value = row?.[key] || row?.[pascalKey]
  return value ? formatDate(value) : '-'
}

const buildGroupFilter = () => {
  if (searchInfo.value.groupValue === UNGROUPED_VALUE) {
    return { ungrouped: true }
  }
  if (searchInfo.value.groupValue) {
    return { groupId: Number(searchInfo.value.groupValue) }
  }
  return {}
}

const fetchAccountTypes = async () => {
  const { data } = await getQQCacheAccountTypes()
  accountTypes.value = data || []
  if (!form.value.accountType && accountTypes.value.length) {
    form.value.accountType = accountTypes.value[0].value
  }
}

const fetchDeviceGroups = async () => {
  const { data } = await getDeviceGroups()
  deviceGroups.value = data || []
}

const fetchList = async () => {
  try {
    const { data } = await getDeviceConfigList({
      page: page.value,
      pageSize: pageSize.value,
      deviceId: searchInfo.value.deviceId || undefined,
      accountType: searchInfo.value.accountType || undefined,
      ...buildGroupFilter()
    })
    tableData.value = data?.list || []
    total.value = data?.total || 0
    selectedRows.value = []
    tableRef.value?.clearSelection?.()
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const fetchAll = async () => {
  try {
    await Promise.all([fetchDeviceGroups(), fetchList()])
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const openDialog = (row) => {
  form.value = {
    ID: row?.ID || 0,
    deviceId: row?.deviceId || '',
    accountType: row?.accountType || 'default',
    groupId: row?.groupId || undefined,
    remark: row?.remark || ''
  }
  dialogVisible.value = true
}

const onSelectionChange = (rows) => {
  selectedRows.value = rows || []
}

const openBatchDialog = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择设备')
    return
  }
  batchForm.value = {
    updateAccountType: false,
    accountType: accountTypes.value[0]?.value || 'default',
    updateGroup: false,
    groupValue: CLEAR_GROUP_VALUE,
    updateRemark: false,
    remark: ''
  }
  batchDialogVisible.value = true
}

const onBatchSave = async () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择设备')
    return
  }
  if (!batchForm.value.updateAccountType && !batchForm.value.updateGroup && !batchForm.value.updateRemark) {
    ElMessage.warning('请选择要更新的字段')
    return
  }
  if (batchForm.value.updateAccountType && !batchForm.value.accountType) {
    ElMessage.warning('请选择账号类型')
    return
  }
  const payload = {
    ids: selectedRows.value.map((row) => row.ID),
    updateAccountType: batchForm.value.updateAccountType,
    accountType: batchForm.value.updateAccountType ? batchForm.value.accountType : undefined,
    updateGroup: batchForm.value.updateGroup,
    groupId: batchForm.value.updateGroup && batchForm.value.groupValue !== CLEAR_GROUP_VALUE ? Number(batchForm.value.groupValue) : null,
    updateRemark: batchForm.value.updateRemark,
    remark: batchForm.value.updateRemark ? batchForm.value.remark : undefined
  }
  await batchUpdateDeviceConfig(payload)
  ElMessage.success('批量设置成功')
  batchDialogVisible.value = false
  await fetchList()
}

const onSave = async () => {
  if (!String(form.value.deviceId || '').trim()) {
    ElMessage.warning('请输入设备ID')
    return
  }
  if (!form.value.accountType) {
    ElMessage.warning('请选择账号类型')
    return
  }
  await saveDeviceConfig({
    id: form.value.ID,
    deviceId: form.value.deviceId,
    accountType: form.value.accountType,
    groupId: form.value.groupId || undefined,
    remark: form.value.remark
  })
  ElMessage.success('保存成功')
  dialogVisible.value = false
  await fetchList()
}

const onDelete = async (row) => {
  if (!row?.ID) return
  await ElMessageBox.confirm('确认删除该设备配置？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await deleteDeviceConfig({ id: row.ID })
  ElMessage.success('删除成功')
  await fetchList()
}

const openGroupManage = async () => {
  await fetchDeviceGroups()
  groupDialogVisible.value = true
}

const openGroupForm = (row) => {
  groupForm.value = {
    ID: row?.ID || 0,
    name: row?.name || '',
    remark: row?.remark || ''
  }
  groupFormDialogVisible.value = true
}

const onSaveGroup = async () => {
  if (!String(groupForm.value.name || '').trim()) {
    ElMessage.warning('请输入分组名称')
    return
  }
  await saveDeviceGroup({
    id: groupForm.value.ID,
    name: groupForm.value.name,
    remark: groupForm.value.remark
  })
  ElMessage.success('保存成功')
  groupFormDialogVisible.value = false
  await fetchDeviceGroups()
  await fetchList()
}

const onDeleteGroup = async (row) => {
  if (!row?.ID) return
  await ElMessageBox.confirm('确认删除该设备分组？有关联设备时将无法删除。', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await deleteDeviceGroup({ id: row.ID })
  ElMessage.success('删除成功')
  await fetchDeviceGroups()
  await fetchList()
}

const onSearch = async () => {
  page.value = 1
  await fetchList()
}

const resetSearch = () => {
  searchInfo.value = {
    deviceId: '',
    accountType: '',
    groupValue: ''
  }
  page.value = 1
  fetchList()
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
  await fetchAccountTypes()
  await fetchAll()
})
</script>

<style scoped>
.group-actions {
  margin-bottom: 12px;
}

.batch-field {
  align-items: center;
  display: flex;
  gap: 12px;
  width: 100%;
}

.batch-field-vertical {
  align-items: flex-start;
}

.batch-field-vertical :deep(.el-textarea) {
  flex: 1;
}

@media (max-width: 900px) {
  .device-config-page :deep(.app-search-box .el-form) {
    display: flex;
    flex-direction: column;
  }

  .device-config-page :deep(.app-search-box .el-form-item) {
    align-items: stretch;
    display: flex;
    margin-right: 0;
  }

  .device-config-page :deep(.app-search-box .el-form-item__label) {
    flex: 0 0 72px;
    justify-content: flex-start;
  }

  .device-config-page :deep(.app-search-box .el-form-item__content) {
    flex: 1;
  }

  .device-config-page :deep(.app-search-box .el-input),
  .device-config-page :deep(.app-search-box .el-select) {
    width: 100% !important;
  }

  .device-config-page :deep(.app-btn-list) {
    align-items: stretch;
    flex-direction: column;
  }

  .device-config-page :deep(.app-btn-list .el-button) {
    margin-left: 0;
    width: 100%;
  }

  .device-config-page :deep(.app-pagination) {
    overflow-x: auto;
  }

  .batch-field {
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
  }
}
</style>
