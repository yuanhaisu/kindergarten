package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

// PlatformEpochStore 暴露平台账号的凭证失效存储给路由装配。
// 与 TokenRevoker 同理，走 Handler 以避免改 NewRouter 签名。
func (h *Handler) PlatformEpochStore() auth.PlatformEpochStore {
	if h == nil || h.platformSvc == nil {
		return nil
	}
	return h.platformSvc.EpochStore()
}

// operatorOf 取当前平台账号的用户名，用于操作留痕。
func (h *Handler) operatorOf(c *gin.Context) string {
	idStr := middleware.GetPlatformAdminID(c)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return idStr
	}
	return h.platformSvc.AdminUsername(c.Request.Context(), id)
}

// PlatformLogin 平台方登录。
func (h *Handler) PlatformLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	res, err := h.platformSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err.Error() == "INVALID_CREDENTIALS" {
			response.ErrUnauthorized(c, "用户名或密码错误")
			return
		}
		response.ErrInternal(c, "登录失败")
		return
	}
	response.OK(c, res)
}

// PlatformLogout 注销平台会话。
func (h *Handler) PlatformLogout(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
		_ = h.authSvc.Logout(c.Request.Context(), token, "")
	}
	response.OK(c, gin.H{})
}

// PlatformListKindergartens 园所列表（跨园）。
func (h *Handler) PlatformListKindergartens(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")
	status := c.Query("status")
	list, total, err := h.platformSvc.ListKindergartens(c.Request.Context(), page, pageSize, keyword, status)
	if err != nil {
		response.ErrInternal(c, "获取园所列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// PlatformGetKindergarten 园所详情。
func (h *Handler) PlatformGetKindergarten(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "园所 ID 不合法")
		return
	}
	detail, err := h.platformSvc.GetKindergarten(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "KINDERGARTEN_NOT_FOUND" {
			response.ErrNotFound(c, "园所不存在")
			return
		}
		response.ErrInternal(c, "获取园所详情失败")
		return
	}
	response.OK(c, detail)
}

// PlatformOnboardKindergarten 开通园所并创建园长账号。
func (h *Handler) PlatformOnboardKindergarten(c *gin.Context) {
	var req service.OnboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	operator := h.operatorOf(c)
	res, err := h.platformSvc.Onboard(c.Request.Context(), req, operator, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPhoneExists):
			response.ErrConflict(c, "该手机号已有账号")
		case strings.HasPrefix(err.Error(), "INVALID_PHONE:"):
			response.ErrValidation(c, "手机号格式不正确")
		case strings.HasPrefix(err.Error(), "WEAK_PASSWORD:"):
			response.ErrValidation(c, "初始密码强度不足")
		default:
			response.ErrInternal(c, "开通园所失败")
		}
		return
	}
	response.Created(c, res)
}

// PlatformSetKindergartenStatus 启用或停用园所。
func (h *Handler) PlatformSetKindergartenStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "园所 ID 不合法")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.platformSvc.SetKindergartenStatus(c.Request.Context(), id, req.Status, h.operatorOf(c), c.ClientIP()); err != nil {
		switch err.Error() {
		case "KINDERGARTEN_NOT_FOUND":
			response.ErrNotFound(c, "园所不存在")
		case "INVALID_STATUS":
			response.ErrValidation(c, "状态取值不合法")
		default:
			response.ErrInternal(c, "更新园所状态失败")
		}
		return
	}
	response.OK(c, gin.H{"updated_at": time.Now().Format(time.RFC3339)})
}

// PlatformListLogs 平台操作留痕。支持按时间范围、操作人、模块与目标园所检索。
func (h *Handler) PlatformListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	f := repository.PlatformLogFilter{
		Operator: c.Query("operator"),
		Module:   c.Query("module"),
		Page:     page,
		PageSize: pageSize,
	}
	if v := c.Query("kindergarten_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			response.ErrValidation(c, "园所 ID 不合法")
			return
		}
		f.KindergartenID = id
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.ErrValidation(c, "from 时间格式不合法：需要 RFC 3339")
			return
		}
		f.From = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.ErrValidation(c, "to 时间格式不合法：需要 RFC 3339")
			return
		}
		f.To = &t
	}
	list, total, err := h.platformSvc.ListLogs(c.Request.Context(), f)
	if err != nil {
		response.ErrInternal(c, "获取操作记录失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// ==================== 套餐目录 ====================

// planCatalogService 是套餐目录 handler 依赖的行为集合。抽成接口让分页契约
// 可以用测试替身验证，不必为了跑 handler 测试而连开发库。
type planCatalogService interface {
	ListModules() []service.ModuleDef
	ListPlans(ctx context.Context, status string, pq pagination.Query) ([]service.PlanRow, int64, int, int, error)
	Create(ctx context.Context, req service.PlanRequest, operator, ip string) (*domain.Plan, error)
	Update(ctx context.Context, id uint64, req service.PlanRequest, operator, ip string) error
	ListTiers(ctx context.Context, planID uint64) ([]domain.PlanTier, error)
	ReplaceTiers(ctx context.Context, planID uint64, tiers []domain.PlanTier, operator, ip string) error
}

func (h *Handler) PlatformListModules(c *gin.Context) {
	response.OK(c, gin.H{"list": h.planSvc.ListModules()})
}

func (h *Handler) PlatformListPlans(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	q.Normalize()
	list, total, page, pageSize, err := h.planSvc.ListPlans(c.Request.Context(), c.Query("status"), q)
	if err != nil {
		response.ErrInternal(c, "获取套餐列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) PlatformCreatePlan(c *gin.Context) {
	var req service.PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	plan, err := h.planSvc.Create(c.Request.Context(), req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		if strings.HasPrefix(err.Error(), "UNKNOWN_MODULE") || strings.HasPrefix(err.Error(), "PLAN_") ||
			strings.HasPrefix(err.Error(), "INVALID_") {
			if err.Error() == "PLAN_CODE_EXISTS" {
				response.ErrConflict(c, "套餐编码已存在")
				return
			}
			response.ErrValidation(c, "套餐参数不合法")
			return
		}
		response.ErrInternal(c, "创建套餐失败")
		return
	}
	response.Created(c, gin.H{"id": plan.ID})
}

func (h *Handler) PlatformUpdatePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "套餐 ID 不合法")
		return
	}
	var req service.PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.planSvc.Update(c.Request.Context(), id, req, h.operatorOf(c), c.ClientIP()); err != nil {
		switch {
		case err.Error() == "PLAN_NOT_FOUND":
			response.ErrNotFound(c, "套餐不存在")
		case strings.HasPrefix(err.Error(), "UNKNOWN_MODULE"), strings.HasPrefix(err.Error(), "INVALID_"):
			response.ErrValidation(c, "套餐参数不合法")
		default:
			response.ErrInternal(c, "更新套餐失败")
		}
		return
	}
	response.OK(c, gin.H{"updated_at": time.Now().Format(time.RFC3339)})
}

func (h *Handler) PlatformListPlanTiers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "套餐 ID 不合法")
		return
	}
	tiers, err := h.planSvc.ListTiers(c.Request.Context(), id)
	if err != nil {
		response.ErrInternal(c, "获取档位失败")
		return
	}
	response.OK(c, gin.H{"list": tiers})
}

func (h *Handler) PlatformReplacePlanTiers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "套餐 ID 不合法")
		return
	}
	var req struct {
		List []domain.PlanTier `json:"list" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.planSvc.ReplaceTiers(c.Request.Context(), id, req.List, h.operatorOf(c), c.ClientIP()); err != nil {
		writePlanTierError(c, err)
		return
	}
	response.OK(c, gin.H{"updated_at": time.Now().Format(time.RFC3339)})
}

// writePlanTierError 是档位保存错误的唯一映射出口：覆盖校验失败必须
// 返回具体原因，管理端据此展示可操作的修复提示。
func writePlanTierError(c *gin.Context, err error) {
	switch {
	case err.Error() == "PLAN_NOT_FOUND":
		response.ErrNotFound(c, "套餐不存在")
	case err.Error() == "INVALID_TIER":
		response.ErrValidation(c, "档位序号必须从 1 开始")
	case err.Error() == "DUPLICATE_TIER":
		response.ErrValidation(c, "档位序号不能重复")
	case err.Error() == "INVALID_TIER_RANGE":
		response.ErrValidation(c, "档位人数区间不合法：上限不能小于下限")
	case errors.Is(err, service.ErrTierCoverageEmpty):
		response.ErrValidation(c, "至少需要配置一个档位，且第一档必须从 0 人开始")
	case errors.Is(err, service.ErrTierCoverageNotFromZero):
		response.ErrValidation(c, "档位必须从 0 人开始：第一档的人数下限应为 0")
	case errors.Is(err, service.ErrTierCoverageGap):
		response.ErrValidation(c, "档位区间不连续：后一档的人数下限必须等于前一档上限 + 1")
	case errors.Is(err, service.ErrTierCoverageOverlap):
		response.ErrValidation(c, "档位区间重叠：后一档的人数下限必须等于前一档上限 + 1")
	case errors.Is(err, service.ErrTierCoverageBoundedTop):
		response.ErrValidation(c, "最高档必须无上限：请把最高档的人数上限填为 0")
	case errors.Is(err, service.ErrTierCoverageUnboundedNotLast):
		response.ErrValidation(c, "只有最高档可以无上限：无上限的档位必须排在最后")
	default:
		response.ErrInternal(c, "保存档位失败")
	}
}
