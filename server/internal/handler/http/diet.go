package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// dietConfigRequest 是 PUT /admin/diet/config 的请求体（契约 23-diet.md）。
// name/sort/coefficient 只读，客户端传了也不采纳。
type dietConfigRequest struct {
	CareType      domain.DietCareType          `json:"care_type"`
	FoodInputType domain.DietFoodInputType     `json:"food_input_type"`
	RatioMode     domain.DietRatioMode         `json:"ratio_mode"`
	Meals         []dietConfigRequestMealInput `json:"meals"`
}

type dietConfigRequestMealInput struct {
	MealKey       domain.DietMealKey `json:"meal_key"`
	Time          string             `json:"time"`
	EnergyPercent float64            `json:"energy_percent"`
	Enabled       bool               `json:"enabled"`
}

func (r dietConfigRequest) toInput() service.DietMealConfigInput {
	meals := make([]service.DietMealConfigMealInput, 0, len(r.Meals))
	for _, meal := range r.Meals {
		meals = append(meals, service.DietMealConfigMealInput{
			MealKey:       meal.MealKey,
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Enabled:       meal.Enabled,
		})
	}
	return service.DietMealConfigInput{
		CareType:      r.CareType,
		FoodInputType: r.FoodInputType,
		RatioMode:     r.RatioMode,
		Meals:         meals,
	}
}

// NewDietConfigHandler 组装只依赖供餐配置服务的 Handler。
// 生产路径仍由 NewHandler 全量注入；路由级测试用它避免构造数十个无关 service。
func NewDietConfigHandler(dietConfigSvc *service.DietConfigService) *Handler {
	return &Handler{dietConfigSvc: dietConfigSvc}
}

// GetDietConfig 返回园所供餐配置；未配置时返回默认值。
func (h *Handler) GetDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.dietConfigSvc.GetConfig(c.Request.Context(), kgID)
	if err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, data)
}

// PutDietConfig 保存园所供餐配置（整组替换餐次），响应与 GET 同构。
func (h *Handler) PutDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietConfigSvc.SaveConfig(c.Request.Context(), kgID, req.toInput())
	if err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, data)
}

// respondDietConfigError 把 service 错误映射到项目统一错误包络。
// 业务校验失败返回 422 且带可读 message；仓储等内部错误不向外泄露细节。
func respondDietConfigError(c *gin.Context, err error) {
	var invalid *service.DietConfigValidationError
	if errors.As(err, &invalid) {
		response.ErrValidation(c, invalid.Message)
		return
	}
	response.ErrInternal(c, "膳食配置服务异常")
}
