package system

import (
	"context"

	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderAuthority = initOrderCasbin + 1

type initAuthority struct{}

// auto run
func init() {
	system.RegisterInit(initOrderAuthority, &initAuthority{})
}

func (i *initAuthority) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysAuthority{})
}

func (i *initAuthority) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysAuthority{})
}

func (i *initAuthority) InitializerName() string {
	return sysModel.SysAuthority{}.TableName()
}

func (i *initAuthority) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	entities := []sysModel.SysAuthority{
		// 保留原有开发用超级管理员角色
		{AuthorityId: 888, AuthorityName: "超级管理员", ParentId: utils.Pointer[uint](0), DefaultRouter: "accountManage"},
		// 业务角色
		{AuthorityId: 100, AuthorityName: "管理员", ParentId: utils.Pointer[uint](0), DefaultRouter: "accountManage"},
		{AuthorityId: 200, AuthorityName: "团长", ParentId: utils.Pointer[uint](100), DefaultRouter: "accountManage"},
		{AuthorityId: 300, AuthorityName: "地推", ParentId: utils.Pointer[uint](200), DefaultRouter: "registerTaskCenter"},
	}

	if err := db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysAuthority{}.TableName())
	}
	// data authority
	if err := db.Model(&entities[0]).Association("DataAuthorityId").Replace(
		[]*sysModel.SysAuthority{
			{AuthorityId: 888},
			{AuthorityId: 100},
			{AuthorityId: 200},
			{AuthorityId: 300},
		}); err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!",
			db.Model(&entities[0]).Association("DataAuthorityId").Relationship.JoinTable.Name)
	}

	// 管理员可见管理员/团长/地推数据
	if err := db.Model(&entities[1]).Association("DataAuthorityId").Replace(
		[]*sysModel.SysAuthority{
			{AuthorityId: 100},
			{AuthorityId: 200},
			{AuthorityId: 300},
		}); err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!",
			db.Model(&entities[3]).Association("DataAuthorityId").Relationship.JoinTable.Name)
	}
	// 团长可见团长/地推数据
	if err := db.Model(&entities[2]).Association("DataAuthorityId").Replace(
		[]*sysModel.SysAuthority{
			{AuthorityId: 200},
			{AuthorityId: 300},
		}); err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!",
			db.Model(&entities[4]).Association("DataAuthorityId").Relationship.JoinTable.Name)
	}

	next := context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}

func (i *initAuthority) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("authority_id = ?", "300").
		First(&sysModel.SysAuthority{}).Error, gorm.ErrRecordNotFound) { // 判断是否存在数据
		return false
	}
	return true
}
