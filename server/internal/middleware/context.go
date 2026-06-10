package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetUserID 从 JWT claims 解析用户 ID（解析失败返回 0）
func GetUserID(c *gin.Context) uint64 {
	claims := GetClaims(c)
	if claims == nil || claims.UserID == "" {
		return 0
	}
	id, err := strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// GetKindergartenID 当前登录用户所属园所 ID（claims 未携带时为 0）
func GetKindergartenID(c *gin.Context) uint64 {
	claims := GetClaims(c)
	if claims == nil {
		return 0
	}
	return claims.KindergartenID
}

// GetRoles JWT 中的角色名列表
func GetRoles(c *gin.Context) []string {
	claims := GetClaims(c)
	if claims == nil {
		return nil
	}
	return claims.Roles
}
