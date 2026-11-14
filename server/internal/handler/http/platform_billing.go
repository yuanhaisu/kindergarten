package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// ==================== 订单与收款 ====================

func (h *Handler) PlatformListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	kgID, _ := strconv.ParseUint(c.Query("kindergarten_id"), 10, 64)
	distID, _ := strconv.ParseUint(c.Query("distributor_id"), 10, 64)
	rows, total, err := h.orderSvc.List(c.Request.Context(), repository.OrderFilter{
		KindergartenID: kgID, Status: c.Query("status"), DistributorID: distID, Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrInternal(c, "获取订单列表失败")
		return
	}
	response.OKList(c, rows, total, page, pageSize)
}

func (h *Handler) PlatformCreateOrder(c *gin.Context) {
	var req service.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	order, err := h.orderSvc.Create(c.Request.Context(), req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		writeCreateOrderError(c, err)
		return
	}
	var snap service.PlanSnapshot
	_ = json.Unmarshal([]byte(order.PlanSnapshot), &snap)
	response.Created(c, gin.H{
		"id": order.ID, "order_no": order.OrderNo, "status": order.Status,
		"amount_due_cents": order.AmountDueCents, "student_tier": order.StudentTier,
		"starts_at": order.StartsAt.Format("2006-01-02"), "expires_at": order.ExpiresAt.Format("2006-01-02"),
		"plan_snapshot": snap,
	})
}

// writeCreateOrderError 是建单错误的唯一映射出口。历史档位配置无法落档
// 必须作为可识别的校验错误返回，不能退化成 500 或静默按最低价成交。
func writeCreateOrderError(c *gin.Context, err error) {
	switch {
	case err.Error() == "KINDERGARTEN_NOT_FOUND":
		response.ErrNotFound(c, "园所不存在")
	case err.Error() == "PLAN_NOT_FOUND":
		response.ErrNotFound(c, "套餐不存在")
	case err.Error() == "PLAN_ARCHIVED":
		response.ErrValidation(c, "套餐已下架")
	case err.Error() == "DISTRIBUTOR_NOT_FOUND":
		response.ErrNotFound(c, "分销商不存在")
	case errors.Is(err, service.ErrTierCoverageUnresolved):
		response.ErrValidation(c, "按在园幼儿数无法落入该套餐的任何档位：档位配置不完整，请先修正套餐档位后重试")
	default:
		response.ErrInternal(c, "创建订单失败")
	}
}

func (h *Handler) PlatformGetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "订单 ID 不合法")
		return
	}
	detail, err := h.orderSvc.Detail(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "ORDER_NOT_FOUND" {
			response.ErrNotFound(c, "订单不存在")
			return
		}
		response.ErrInternal(c, "获取订单失败")
		return
	}
	response.OK(c, detail)
}

func (h *Handler) PlatformConfirmOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "订单 ID 不合法")
		return
	}
	order, ents, err := h.orderSvc.Confirm(c.Request.Context(), id, h.operatorOf(c), c.ClientIP())
	if err != nil {
		switch err.Error() {
		case "ORDER_NOT_FOUND":
			response.ErrNotFound(c, "订单不存在")
		case "ORDER_ALREADY_CONFIRMED", "ORDER_CANCELLED":
			response.ErrConflict(c, "订单状态不允许确认")
		default:
			response.ErrInternal(c, "确认订单失败")
		}
		return
	}
	items := make([]gin.H, 0, len(ents))
	for _, e := range ents {
		items = append(items, gin.H{
			"module_code": e.ModuleCode, "source": e.Source,
			"expires_at": e.ExpiresAt.Format("2006-01-02"),
		})
	}
	response.OK(c, gin.H{"id": order.ID, "status": order.Status, "entitlements": items})
}

func (h *Handler) PlatformCancelOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "订单 ID 不合法")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	order, err := h.orderSvc.Cancel(c.Request.Context(), id, req.Reason, h.operatorOf(c), c.ClientIP())
	if err != nil {
		switch err.Error() {
		case "ORDER_NOT_FOUND":
			response.ErrNotFound(c, "订单不存在")
		case "ORDER_ALREADY_CANCELLED":
			response.ErrConflict(c, "订单已取消")
		default:
			response.ErrInternal(c, "取消订单失败")
		}
		return
	}
	response.OK(c, gin.H{"cancelled_at": order.CancelledAt.Format(time.RFC3339)})
}

func (h *Handler) PlatformAddPayment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "订单 ID 不合法")
		return
	}
	var req service.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	res, err := h.orderSvc.AddPayment(c.Request.Context(), id, req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		switch err.Error() {
		case "ORDER_NOT_FOUND":
			response.ErrNotFound(c, "订单不存在")
		case "ORDER_CANCELLED":
			response.ErrConflict(c, "订单已取消")
		case "INVALID_AMOUNT", "INVALID_RECEIVED_AT":
			response.ErrValidation(c, "收款参数不合法")
		case "REFUND_PAYMENT_REQUIRED":
			response.ErrValidation(c, "退款必须指定被退的原收款")
		case "REFUND_PAYMENT_INVALID":
			response.ErrValidation(c, "被退收款不合法")
		case "REFUND_PAYMENT_NOT_ALLOWED":
			response.ErrValidation(c, "正数收款不能指定被退收款")
		case "REFUND_PAYMENT_NOT_FOUND":
			response.ErrNotFound(c, "被退收款不存在")
		case "REFUND_EXCEEDS_PAYMENT":
			response.ErrValidation(c, "退款金额超过该笔收款的可退金额")
		default:
			response.ErrInternal(c, "登记收款失败")
		}
		return
	}
	response.Created(c, res)
}

func (h *Handler) PlatformListPayments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "订单 ID 不合法")
		return
	}
	rows, err := h.orderSvc.ListPayments(c.Request.Context(), id)
	if err != nil {
		response.ErrInternal(c, "获取收款流水失败")
		return
	}
	response.OK(c, gin.H{"list": rows})
}

// ==================== 分销商与佣金 ====================

func (h *Handler) PlatformListDistributors(c *gin.Context) {
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))
	rows, total, err := h.distSvc.ListDistributors(c.Request.Context(), int8(status))
	if err != nil {
		response.ErrInternal(c, "获取分销商列表失败")
		return
	}
	response.OKList(c, rows, total, 1, int(total))
}

func (h *Handler) PlatformCreateDistributor(c *gin.Context) {
	var req service.DistributorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	d, err := h.distSvc.CreateDistributor(c.Request.Context(), req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		if err.Error() == "INVALID_RATE" {
			response.ErrValidation(c, "佣金比例必须在 0-100 之间")
			return
		}
		response.ErrInternal(c, "创建分销商失败")
		return
	}
	response.Created(c, gin.H{"id": d.ID})
}

func (h *Handler) PlatformUpdateDistributor(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "分销商 ID 不合法")
		return
	}
	var req service.DistributorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.distSvc.UpdateDistributor(c.Request.Context(), id, req, h.operatorOf(c), c.ClientIP()); err != nil {
		switch err.Error() {
		case "DISTRIBUTOR_NOT_FOUND":
			response.ErrNotFound(c, "分销商不存在")
		case "INVALID_RATE":
			response.ErrValidation(c, "佣金比例必须在 0-100 之间")
		default:
			response.ErrInternal(c, "更新分销商失败")
		}
		return
	}
	response.OK(c, gin.H{"updated_at": time.Now().Format(time.RFC3339)})
}

func (h *Handler) PlatformListCommissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	distID, _ := strconv.ParseUint(c.Query("distributor_id"), 10, 64)
	rows, total, summary, err := h.distSvc.ListCommissions(c.Request.Context(), repository.CommissionFilter{
		Month: c.Query("month"), DistributorID: distID, Status: c.Query("status"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrInternal(c, "获取佣金台账失败")
		return
	}
	response.OK(c, gin.H{
		"list": rows, "total": total, "page": page, "page_size": pageSize,
		"summary": gin.H{
			"pending_cents":    summary[domain.CommissionStatusPending],
			"settleable_cents": summary[domain.CommissionStatusSettleable],
			"settled_cents":    summary[domain.CommissionStatusSettled],
			"reversed_cents":   summary[domain.CommissionStatusReversed],
		},
	})
}

func (h *Handler) PlatformGetCommission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "佣金 ID 不合法")
		return
	}
	row, err := h.distSvc.GetCommission(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "COMMISSION_NOT_FOUND" {
			response.ErrNotFound(c, "佣金记录不存在")
			return
		}
		response.ErrInternal(c, "获取佣金明细失败")
		return
	}
	response.OK(c, row)
}
