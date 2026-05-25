package system

import (
	"errors"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	sysReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/golang-jwt/jwt/v5"

	"time"
)

type ApiTokenService struct{}

const (
	apiTokenRoleSuperAdmin = uint(888)
	apiTokenRoleAdmin      = uint(100)
	apiTokenRoleLeader     = uint(200)
	apiTokenRolePromoter   = uint(300)
	apiTokenRoleAppExtract = uint(400)
	apiTokenRoleAppUpload  = uint(500)
)

func canManageApiTokenTarget(operatorAuthorityID, targetAuthorityID uint) bool {
	switch operatorAuthorityID {
	case apiTokenRoleSuperAdmin:
		return true
	case apiTokenRoleAdmin:
		return targetAuthorityID == apiTokenRoleLeader ||
			targetAuthorityID == apiTokenRolePromoter ||
			targetAuthorityID == apiTokenRoleAppExtract ||
			targetAuthorityID == apiTokenRoleAppUpload
	case apiTokenRoleLeader:
		return targetAuthorityID == apiTokenRolePromoter
	default:
		return false
	}
}

func (apiVersion *ApiTokenService) CreateApiToken(apiToken system.SysApiToken, days int) (string, error) {
	return apiVersion.createApiToken(apiToken, days, 0, false)
}

func (apiVersion *ApiTokenService) CreateApiTokenForOperator(operatorAuthorityID uint, apiToken system.SysApiToken, days int) (string, error) {
	return apiVersion.createApiToken(apiToken, days, operatorAuthorityID, true)
}

func (apiVersion *ApiTokenService) createApiToken(apiToken system.SysApiToken, days int, operatorAuthorityID uint, enforceScope bool) (string, error) {
	var user system.SysUser
	if err := global.GVA_DB.Preload("Authorities").Where("id = ?", apiToken.UserID).First(&user).Error; err != nil {
		return "", errors.New("用户不存在")
	}
	if enforceScope && (!canManageApiTokenTarget(operatorAuthorityID, user.AuthorityId) || !canManageApiTokenTarget(operatorAuthorityID, apiToken.AuthorityID)) {
		return "", errors.New("无权为该用户签发Token")
	}
	if enforceScope && (user.AuthorityId != apiTokenRolePromoter || apiToken.AuthorityID != apiTokenRolePromoter) {
		return "", errors.New("仅支持为地推账号签发OpenAPI Token")
	}
	if enforceScope {
		days = 90
	}

	hasAuth := false
	for _, auth := range user.Authorities {
		if auth.AuthorityId == apiToken.AuthorityID {
			hasAuth = true
			break
		}
	}
	if !hasAuth && user.AuthorityId != apiToken.AuthorityID {
		return "", errors.New("用户不具备该角色权限")
	}

	j := &utils.JWT{SigningKey: []byte(global.GVA_CONFIG.JWT.SigningKey)} // 唯一不同的部分是过期时间

	expireTime := time.Duration(days) * 24 * time.Hour
	if days == -1 {
		expireTime = 100 * 365 * 24 * time.Hour
	}

	bf, _ := utils.ParseDuration(global.GVA_CONFIG.JWT.BufferTime)

	claims := sysReq.CustomClaims{
		BaseClaims: sysReq.BaseClaims{
			UUID:        user.UUID,
			ID:          user.ID,
			Username:    user.Username,
			NickName:    user.NickName,
			AuthorityId: apiToken.AuthorityID,
		},
		BufferTime: int64(bf / time.Second), // 缓冲时间
		TokenType:  sysReq.TokenTypeOpenAPI,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"GVA"},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			Issuer:    global.GVA_CONFIG.JWT.Issuer,
		},
	}

	token, err := j.CreateToken(claims)
	if err != nil {
		return "", err
	}

	apiToken.Token = token
	apiToken.Status = true
	apiToken.ExpiresAt = time.Now().Add(expireTime)
	err = global.GVA_DB.Create(&apiToken).Error
	return token, err
}

func (apiVersion *ApiTokenService) GetApiTokenList(info sysReq.SysApiTokenSearch) (list []system.SysApiToken, total int64, err error) {
	return apiVersion.getApiTokenList(info, 0, false)
}

func (apiVersion *ApiTokenService) GetApiTokenListForOperator(operatorAuthorityID uint, info sysReq.SysApiTokenSearch) (list []system.SysApiToken, total int64, err error) {
	return apiVersion.getApiTokenList(info, operatorAuthorityID, true)
}

func (apiVersion *ApiTokenService) getApiTokenList(info sysReq.SysApiTokenSearch, operatorAuthorityID uint, enforceScope bool) (list []system.SysApiToken, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&system.SysApiToken{})

	db = db.Preload("User")

	if info.UserID != 0 {
		db = db.Where("user_id = ?", info.UserID)
	}
	if info.Status != nil {
		db = db.Where("status = ?", *info.Status)
	}
	if enforceScope {
		roles := manageableApiTokenRoles(operatorAuthorityID)
		if len(roles) == 0 {
			return []system.SysApiToken{}, 0, nil
		}
		db = db.Joins("JOIN sys_users ON sys_users.id = sys_api_tokens.user_id").
			Where("sys_users.authority_id IN ? AND sys_api_tokens.authority_id IN ?", roles, roles)
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("sys_api_tokens.created_at desc").Find(&list).Error
	return list, total, err
}

func (apiVersion *ApiTokenService) DeleteApiToken(id uint) error {
	return apiVersion.deleteApiToken(id, 0, false)
}

func (apiVersion *ApiTokenService) DeleteApiTokenForOperator(operatorAuthorityID uint, id uint) error {
	return apiVersion.deleteApiToken(id, operatorAuthorityID, true)
}

func (apiVersion *ApiTokenService) deleteApiToken(id uint, operatorAuthorityID uint, enforceScope bool) error {
	var apiToken system.SysApiToken
	err := global.GVA_DB.Preload("User").First(&apiToken, id).Error
	if err != nil {
		return err
	}
	if enforceScope && (!canManageApiTokenTarget(operatorAuthorityID, apiToken.User.AuthorityId) || !canManageApiTokenTarget(operatorAuthorityID, apiToken.AuthorityID)) {
		return errors.New("无权作废该Token")
	}

	jwtService := JwtService{}
	err = jwtService.JsonInBlacklist(system.JwtBlacklist{Jwt: apiToken.Token})
	if err != nil {
		return err
	}

	return global.GVA_DB.Model(&apiToken).Update("status", false).Error
}

func manageableApiTokenRoles(operatorAuthorityID uint) []uint {
	switch operatorAuthorityID {
	case apiTokenRoleSuperAdmin:
		return []uint{apiTokenRoleSuperAdmin, apiTokenRoleAdmin, apiTokenRoleLeader, apiTokenRolePromoter, apiTokenRoleAppExtract, apiTokenRoleAppUpload}
	case apiTokenRoleAdmin:
		return []uint{apiTokenRoleLeader, apiTokenRolePromoter, apiTokenRoleAppExtract, apiTokenRoleAppUpload}
	case apiTokenRoleLeader:
		return []uint{apiTokenRolePromoter}
	default:
		return nil
	}
}
