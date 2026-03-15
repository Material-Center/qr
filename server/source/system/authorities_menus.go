package system

import (
	"context"

	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderMenuAuthority = initOrderMenu + initOrderAuthority

type initMenuAuthority struct{}

// auto run
func init() {
	system.RegisterInit(initOrderMenuAuthority, &initMenuAuthority{})
}

func (i *initMenuAuthority) MigrateTable(ctx context.Context) (context.Context, error) {
	return ctx, nil // do nothing
}

func (i *initMenuAuthority) TableCreated(ctx context.Context) bool {
	return false // always replace
}

func (i *initMenuAuthority) InitializerName() string {
	return "sys_menu_authorities"
}

func (i *initMenuAuthority) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	initAuth := &initAuthority{}
	authorities, ok := ctx.Value(initAuth.InitializerName()).([]sysModel.SysAuthority)
	if !ok {
		return ctx, errors.Wrap(system.ErrMissingDependentContext, "创建 [菜单-权限] 关联失败, 未找到权限表初始化数据")
	}

	allMenus, ok := ctx.Value(new(initMenu).InitializerName()).([]sysModel.SysBaseMenu)
	if !ok {
		return next, errors.Wrap(errors.New(""), "创建 [菜单-权限] 关联失败, 未找到菜单表初始化数据")
	}
	next = ctx

	// 构建菜单ID映射，方便快速查找
	menuMap := make(map[uint]sysModel.SysBaseMenu)
	menuNameMap := make(map[string]sysModel.SysBaseMenu)
	for _, menu := range allMenus {
		menuMap[menu.ID] = menu
		menuNameMap[menu.Name] = menu
	}

	// 构建角色映射，避免依赖切片顺序
	authorityMap := make(map[uint]sysModel.SysAuthority)
	for _, auth := range authorities {
		authorityMap[auth.AuthorityId] = auth
	}

	// 基础菜单集合（无账号管理）
	var basicMenus []sysModel.SysBaseMenu

	for _, menu := range allMenus {
		if menu.ParentId == 0 {
			basicMenus = append(basicMenus, menu)
		}
	}

	// 账号管理菜单集合（管理员 / 团长）
	var accountMenus []sysModel.SysBaseMenu
	accountParent, hasAccountParent := menuNameMap["account"]
	accountChild, hasAccountChild := menuNameMap["accountManage"]
	if hasAccountParent {
		accountMenus = append(accountMenus, accountParent)
	}
	if hasAccountChild {
		accountMenus = append(accountMenus, accountChild)
	}

	// 注册任务菜单
	var registerManageMenus []sysModel.SysBaseMenu
	registerParent, hasRegisterParent := menuNameMap["register"]
	registerManageChild, hasRegisterManageChild := menuNameMap["registerTaskManage"]
	registerCenterChild, hasRegisterCenterChild := menuNameMap["registerTaskCenter"]
	registerConfigChild, hasRegisterConfigChild := menuNameMap["registerConfig"]
	if hasRegisterParent {
		registerManageMenus = append(registerManageMenus, registerParent)
	}
	if hasRegisterManageChild {
		registerManageMenus = append(registerManageMenus, registerManageChild)
	}
	if hasRegisterConfigChild {
		registerManageMenus = append(registerManageMenus, registerConfigChild)
	}

	var registerPromoterMenus []sysModel.SysBaseMenu
	if hasRegisterCenterChild {
		registerPromoterMenus = append(registerPromoterMenus, registerCenterChild)
	}

	// 角色分配函数
	assignMenus := func(authorityID uint, menus []sysModel.SysBaseMenu, errMsg string) error {
		auth, ok := authorityMap[authorityID]
		if !ok {
			return errors.Wrap(system.ErrMissingDependentContext, errMsg+": 未找到角色")
		}
		return db.Model(&auth).Association("SysBaseMenus").Replace(menus)
	}

	// 888 超级管理员保留全量菜单（包含代码生成、系统配置等能力）
	if err = assignMenus(888, allMenus, "为超级管理员分配菜单失败"); err != nil {
		return next, errors.Wrap(err, "为超级管理员分配菜单失败")
	}

	// 100 管理员：基础菜单 + 账号管理 + 注册任务统计
	adminMenus := append([]sysModel.SysBaseMenu{}, basicMenus...)
	adminMenus = append(adminMenus, accountMenus...)
	adminMenus = append(adminMenus, registerManageMenus...)
	if err = assignMenus(100, adminMenus, "为管理员分配菜单失败"); err != nil {
		return next, errors.Wrap(err, "为管理员分配菜单失败")
	}

	// 200 团长：基础菜单 + 账号管理 + 注册任务统计
	leaderMenus := append([]sysModel.SysBaseMenu{}, basicMenus...)
	leaderMenus = append(leaderMenus, accountMenus...)
	leaderMenus = append(leaderMenus, registerManageMenus...)
	if err = assignMenus(200, leaderMenus, "为团长分配菜单失败"); err != nil {
		return next, errors.Wrap(err, "为团长分配菜单失败")
	}

	// 300 地推：基础菜单 + 创建任务页面
	promoterMenus := append([]sysModel.SysBaseMenu{}, basicMenus...)
	promoterMenus = append(promoterMenus, registerPromoterMenus...)
	if err = assignMenus(300, promoterMenus, "为地推分配菜单失败"); err != nil {
		return next, errors.Wrap(err, "为地推分配菜单失败")
	}

	return next, nil
}

func (i *initMenuAuthority) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	required := map[string]struct{}{
		"dashboard": {},
		"about":     {},
		"state":     {},
	}
	checkRole := func(authorityID uint) bool {
		auth := &sysModel.SysAuthority{}
		if err := db.Model(auth).
			Where("authority_id = ?", authorityID).
			Preload("SysBaseMenus").
			First(auth).Error; err != nil {
			return false
		}
		menuNames := map[string]struct{}{}
		for _, menu := range auth.SysBaseMenus {
			menuNames[menu.Name] = struct{}{}
		}
		for name := range required {
			if _, ok := menuNames[name]; !ok {
				return false
			}
		}
		return true
	}
	return checkRole(100) && checkRole(200) && checkRole(300)
}
