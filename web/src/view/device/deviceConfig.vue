<template>
  <div>
    <div class="gva-search-box">
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
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
          <el-button type="primary" icon="plus" @click="openDialog()">新增</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="ID">
        <el-table-column label="ID" prop="ID" width="90" />
        <el-table-column label="设备ID" prop="deviceId" min-width="180" show-overflow-tooltip />
        <el-table-column label="账号类型" width="140">
          <template #default="{ row }">
            {{ accountTypeLabel(row.accountType) }}
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="180" show-overflow-tooltip />
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">
            {{ row.updatedAt ? formatDate(row.updatedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
            <el-button type="danger" link @click="onDelete(row)">删除</el-button>
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
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="3" maxlength="255" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { getQQCacheAccountTypes } from '@/api/qqCache'
import { deleteDeviceConfig, getDeviceConfigList, saveDeviceConfig } from '@/api/deviceConfig'

defineOptions({
  name: 'DeviceConfig'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const accountTypes = ref([])
const dialogVisible = ref(false)
const form = ref({
  ID: 0,
  deviceId: '',
  accountType: 'default',
  remark: ''
})
const searchInfo = ref({
  deviceId: '',
  accountType: ''
})

const accountTypeLabel = (value) => {
  return accountTypes.value.find((item) => item.value === value)?.label || '默认账号'
}

const fetchAccountTypes = async () => {
  const { data } = await getQQCacheAccountTypes()
  accountTypes.value = data || []
  if (!form.value.accountType && accountTypes.value.length) {
    form.value.accountType = accountTypes.value[0].value
  }
}

const fetchList = async () => {
  try {
    const { data } = await getDeviceConfigList({
      page: page.value,
      pageSize: pageSize.value,
      deviceId: searchInfo.value.deviceId || undefined,
      accountType: searchInfo.value.accountType || undefined
    })
    tableData.value = data?.list || []
    total.value = data?.total || 0
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const openDialog = (row) => {
  form.value = {
    ID: row?.ID || 0,
    deviceId: row?.deviceId || '',
    accountType: row?.accountType || 'default',
    remark: row?.remark || ''
  }
  dialogVisible.value = true
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

const onSearch = async () => {
  page.value = 1
  await fetchList()
}

const resetSearch = () => {
  searchInfo.value = {
    deviceId: '',
    accountType: ''
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
  await fetchList()
})
</script>
