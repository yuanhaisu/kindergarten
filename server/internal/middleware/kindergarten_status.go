package middleware

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// RequireActiveKindergarten 拒绝已停用园所的后续请求。
// 挂在 AuthJWT 之后：凭证本身有效，但园所被平台方停用后所有园所端接口都应不可用。
func RequireActiveKindergarten(checker service.KindergartenStatusChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			c.Next()
			return
		}
		kgID := GetKindergartenID(c)
		if kgID == 0 {
			c.Next()
			return
		}
		suspended, err := checker.Suspended(c.Request.Context(), kgID)
		if err != nil {
			log.L(c.Request.Context()).Error("kindergarten status check failed", "kindergarten_id", kgID, "err", err)
			response.ErrUnavailable(c, "服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if suspended {
			response.ErrForbidden(c, "园所已停用，请联系平台方")
			c.Abort()
			return
		}
		c.Next()
	}
}
