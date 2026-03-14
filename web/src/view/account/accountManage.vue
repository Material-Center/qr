<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="用户名">
          <el-input v-model="searchInfo.username" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="searchInfo.nickname" placeholder="昵称" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="searchInfo.phone" placeholder="手机号" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="searchInfo.email" placeholder="邮箱" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button v-if="canManage" type="primary" icon="plus" @click="openAdd">
          新增账号
        </el-button>
      </div>

      <el-table :data="tableData" row-key="ID">
        <el-table-column align="left" label="头像" min-width="75">
          <template #default="scope">
            <CustomPic style="margin-top: 8px" :pic-src="scope.row.headerImg" />
          </template>
        </el-table-column>
        <el-table-column align="left" label="ID" min-width="60" prop="ID" />
        <el-table-column align="left" label="用户名" min-width="140" prop="userName" />
        <el-table-column align="left" label="昵称" min-width="140" prop="nickName" />
        <el-table-column align="left" label="手机号" min-width="160" prop="phone" />
        <el-table-column align="left" label="邮箱" min-width="180" prop="email" />
        <el-table-column align="left" label="最近登录IP" min-width="150" prop="lastLoginIp" />
        <el-table-column align="left" label="最近登录时间" min-width="190">
          <template #default="scope">
            <span>{{ formatDateText(scope.row.lastLoginAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="角色" min-width="120">
          <template #default="scope">
            <el-tag>{{ roleText(scope.row.authorityId) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="启用" min-width="110">
          <template #default="scope">
            <el-switch
              v-model="scope.row.enable"
              inline-prompt
              :active-value="1"
              :inactive-value="2"
              @change="() => switchEnable(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="220" fixed="right">
          <template #default="scope">
            <el-button type="primary" link icon="edit" @click="openEdit(scope.row)">
              编辑
            </el-button>
            <el-button type="primary" link icon="delete" @click="deleteUserFunc(scope.row)">
              删除
            </el-button>
            <el-button type="primary" link icon="magic-stick" @click="openResetPwd(scope.row)">
              重置密码
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
    </div>

    <el-dialog
      v-model="resetPwdDialog"
      title="重置密码"
      width="500px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <el-form :model="resetPwdInfo" label-width="90px">
        <el-form-item label="用户账号">
          <el-input v-model="resetPwdInfo.userName" disabled />
        </el-form-item>
        <el-form-item label="用户昵称">
          <el-input v-model="resetPwdInfo.nickName" disabled />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="resetPwdInfo.password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmResetPassword">确定</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="showDrawer"
      :size="appStore.drawerSize"
      destroy-on-close
      :show-close="false"
    >
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-lg">{{ drawerMode === 'add' ? '新增账号' : '编辑账号' }}</span>
          <div>
            <el-button @click="showDrawer = false">取消</el-button>
            <el-button type="primary" @click="submitDrawer">确定</el-button>
          </div>
        </div>
      </template>

      <el-form ref="userFormRef" :model="userForm" :rules="rules" label-width="80px">
        <el-form-item v-if="drawerMode === 'add'" label="用户名" prop="userName">
          <el-input v-model="userForm.userName" />
        </el-form-item>
        <el-form-item v-if="drawerMode === 'add'" label="密码" prop="password">
          <el-input v-model="userForm.password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickName">
          <el-input v-model="userForm.nickName" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="userForm.phone" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" />
        </el-form-item>
        <el-form-item label="角色" prop="authorityId">
          <el-select v-model="userForm.authorityId" :disabled="drawerMode === 'edit'" style="width: 100%">
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="[ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId) && userForm.authorityId === ROLE_PROMOTER && drawerMode === 'add'"
          label="所属团长"
        >
          <el-select v-model="userForm.leaderId" style="width: 100%" placeholder="请选择团长">
            <el-option
              v-for="leader in leaderOptions"
              :key="leader.ID"
              :label="`${leader.nickName}(${leader.userName})`"
              :value="leader.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="userForm.enable" inline-prompt :active-value="1" :inactive-value="2" />
        </el-form-item>
        <el-form-item label="头像">
          <SelectImage v-model="userForm.headerImg" />
        </el-form-item>
      </el-form>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAppStore } from '@/pinia'
import { useUserStore } from '@/pinia/modules/user'
import { deleteUser, getUserList, register, resetPassword, setUserInfo } from '@/api/user'
import CustomPic from '@/components/customPic/index.vue'
import SelectImage from '@/components/selectImage/selectImage.vue'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'AccountManage'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300

const appStore = useAppStore()
const userStore = useUserStore()

const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const canManage = computed(() => [ROLE_SUPER, ROLE_ADMIN, ROLE_LEADER].includes(currentRoleId.value))

const roleOptions = computed(() => {
  if (currentRoleId.value === ROLE_SUPER) {
    return [
      { label: '超级管理员', value: ROLE_SUPER },
      { label: '管理员', value: ROLE_ADMIN },
      { label: '团长', value: ROLE_LEADER },
      { label: '地推', value: ROLE_PROMOTER }
    ]
  }
  if (currentRoleId.value === ROLE_ADMIN) {
    return [
      { label: '团长', value: ROLE_LEADER },
      { label: '地推', value: ROLE_PROMOTER }
    ]
  }
  if (currentRoleId.value === ROLE_LEADER) {
    return [{ label: '地推', value: ROLE_PROMOTER }]
  }
  return []
})

const roleText = (authorityId) => {
  if (authorityId === ROLE_SUPER) return '超级管理员'
  if (authorityId === ROLE_ADMIN) return '管理员'
  if (authorityId === ROLE_LEADER) return '团长'
  if (authorityId === ROLE_PROMOTER) return '地推'
  return '未知'
}

const formatDateText = (val) => {
  if (!val) return '-'
  return formatDate(val)
}

const searchInfo = ref({
  username: '',
  nickname: '',
  phone: '',
  email: ''
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const leaderOptions = ref([])
const currentUserId = computed(() => userStore.userInfo?.ID)

const filterByRole = (list) => {
  if (currentRoleId.value === ROLE_SUPER) {
    return list
  }
  if (currentRoleId.value === ROLE_ADMIN) {
    return list.filter((item) => [ROLE_LEADER, ROLE_PROMOTER].includes(item.authorityId))
  }
  if (currentRoleId.value === ROLE_LEADER) {
    return list.filter((item) => item.authorityId === ROLE_PROMOTER && item.leaderId === currentUserId.value)
  }
  return []
}

const fetchUsers = async () => {
  const query = {
    page: page.value,
    pageSize: pageSize.value,
    ...searchInfo.value
  }
  if (currentRoleId.value === ROLE_LEADER) {
    query.authorityId = ROLE_PROMOTER
    query.leaderId = currentUserId.value
  }
  const res = await getUserList(query)
  if (res.code === 0) {
    const list = res.data.list || []
    leaderOptions.value = list.filter((item) => item.authorityId === ROLE_LEADER)
    const filtered = filterByRole(list)
    tableData.value = filtered
    total.value = filtered.length
  }
}

const onSubmit = async () => {
  page.value = 1
  await fetchUsers()
}

const onReset = async () => {
  searchInfo.value = { username: '', nickname: '', phone: '', email: '' }
  page.value = 1
  await fetchUsers()
}

const handleCurrentChange = async (val) => {
  page.value = val
  await fetchUsers()
}

const handleSizeChange = async (val) => {
  pageSize.value = val
  page.value = 1
  await fetchUsers()
}

const showDrawer = ref(false)
const drawerMode = ref('add')
const userFormRef = ref()
const userForm = ref({
  ID: 0,
  userName: '',
  password: '',
  nickName: '',
  phone: '',
  email: '',
  authorityId: undefined,
  leaderId: undefined,
  enable: 1,
  headerImg: ''
})

const rules = {
  userName: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 5, message: '最低5位字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '最低6位字符', trigger: 'blur' }
  ],
  nickName: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  authorityId: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const openAdd = () => {
  drawerMode.value = 'add'
  userForm.value = {
    ID: 0,
    userName: '',
    password: '',
    nickName: '',
    phone: '',
    email: '',
    authorityId: roleOptions.value[0]?.value,
    leaderId: currentRoleId.value === ROLE_LEADER ? currentUserId.value : undefined,
    enable: 1,
    headerImg: ''
  }
  showDrawer.value = true
}

const openEdit = (row) => {
  drawerMode.value = 'edit'
  userForm.value = JSON.parse(JSON.stringify(row))
  showDrawer.value = true
}

const submitDrawer = async () => {
  await userFormRef.value.validate(async (valid) => {
    if (!valid) return

    if (drawerMode.value === 'add') {
      if (
        [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value) &&
        userForm.value.authorityId === ROLE_PROMOTER &&
        !userForm.value.leaderId
      ) {
        ElMessage.warning('请选择所属团长')
        return
      }
      const payload = {
        ...userForm.value,
        authorityIds: [userForm.value.authorityId]
      }
      if (currentRoleId.value === ROLE_LEADER && payload.authorityId === ROLE_PROMOTER) {
        payload.leaderId = currentUserId.value
      }
      const res = await register(payload)
      if (res.code === 0) {
        ElMessage.success('创建成功')
        showDrawer.value = false
        await fetchUsers()
      }
      return
    }

    const payload = {
      ...userForm.value,
      authorityIds: [userForm.value.authorityId]
    }
    const res = await setUserInfo(payload)
    if (res.code === 0) {
      ElMessage.success('编辑成功')
      showDrawer.value = false
      await fetchUsers()
    }
  })
}

const deleteUserFunc = async (row) => {
  await ElMessageBox.confirm('确定删除该账号吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  const res = await deleteUser({ id: row.ID })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    await fetchUsers()
  }
}

const switchEnable = async (row) => {
  const res = await setUserInfo({
    ...row,
    authorityIds: [row.authorityId]
  })
  if (res.code === 0) {
    ElMessage.success(`${row.enable === 1 ? '启用' : '禁用'}成功`)
    await nextTick()
    await fetchUsers()
  }
}

const resetPwdDialog = ref(false)
const resetPwdInfo = ref({
  ID: 0,
  userName: '',
  nickName: '',
  password: ''
})

const openResetPwd = (row) => {
  resetPwdInfo.value = {
    ID: row.ID,
    userName: row.userName,
    nickName: row.nickName,
    password: ''
  }
  resetPwdDialog.value = true
}

const confirmResetPassword = async () => {
  if (!resetPwdInfo.value.password) {
    ElMessage.warning('请输入新密码')
    return
  }
  const res = await resetPassword({
    ID: resetPwdInfo.value.ID,
    password: resetPwdInfo.value.password
  })
  if (res.code === 0) {
    ElMessage.success('重置成功')
    resetPwdDialog.value = false
  }
}

fetchUsers()
</script>
