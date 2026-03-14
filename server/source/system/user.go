package system

import (
	"context"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderUser = initOrderAuthority + 1

type initUser struct{}

// auto run
func init() {
	system.RegisterInit(initOrderUser, &initUser{})
}

func (i *initUser) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.SysUser{})
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.SysUser{})
}

func (i *initUser) InitializerName() string {
	return sysModel.SysUser{}.TableName()
}

func (i *initUser) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	ap := ctx.Value("adminPassword")
	apStr, ok := ap.(string)
	if !ok {
		apStr = "123456"
	}

	superAdminPassword := utils.BcryptHash(apStr)
	bizAdminPassword := utils.BcryptHash("123456")

	entities := []sysModel.SysUser{
		// 保留超级管理员用于开发维护（密码走系统初始化参数）
		{
			UUID:        uuid.New(),
			Username:    "admin",
			Password:    superAdminPassword,
			NickName:    "超级管理员",
			HeaderImg:   "https://qmplusimg.henrongyi.top/gva_header.jpg",
			AuthorityId: 888,
			Phone:       "17611111111",
			Email:       "333333333@qq.com",
		},
		// 默认生成业务管理员账号 root/123456
		{
			UUID:        uuid.New(),
			Username:    "root",
			Password:    bizAdminPassword,
			NickName:    "管理员",
			HeaderImg:   "https://qmplusimg.henrongyi.top/1572075907logo.png",
			AuthorityId: 100,
			Phone:       "17611111112",
			Email:       "root@qq.com",
		},
	}
	if err = db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.SysUser{}.TableName()+"表数据初始化失败!")
	}
	next = context.WithValue(ctx, i.InitializerName(), entities)
	authorityEntities, ok := ctx.Value(new(initAuthority).InitializerName()).([]sysModel.SysAuthority)
	if !ok {
		return next, errors.Wrap(system.ErrMissingDependentContext, "创建 [用户-权限] 关联失败, 未找到权限表初始化数据")
	}
	if err = db.Model(&entities[0]).Association("Authorities").Replace(authorityEntities); err != nil {
		return next, err
	}
	// 业务管理员仅保留管理员角色，不提供角色切换能力
	if err = db.Model(&entities[1]).Association("Authorities").Replace([]sysModel.SysAuthority{
		{AuthorityId: 100},
	}); err != nil {
		return next, err
	}
	return next, err
}

func (i *initUser) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	var record sysModel.SysUser
	if errors.Is(db.Where("username = ?", "root").
		Preload("Authorities").First(&record).Error, gorm.ErrRecordNotFound) { // 判断是否存在数据
		return false
	}
	if len(record.Authorities) != 1 {
		return false
	}
	return record.Authorities[0].AuthorityId == 100
}
