package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	dateFrom, dateTo, err := parseOrderDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		response.ErrValidation(c, "时间范围不合法，格式为 YYYY-MM-DD 且结束日期不早于开始日期")
		return
	}
	outstandingOnly := false
	if raw := c.Query("outstanding_only"); raw != "" {
		outstandingOnly, err = strconv.ParseBool(raw)
		if err != nil {
			response.ErrValidation(c, "outstanding_only 只能为 true 或 false")
			return
		}
	}
	rows, total, err := h.orderSvc.List(c.Request.Context(), repository.OrderFilter{
		KindergartenID: kgID, Status: c.Query("status"), DistributorID: distID,
		OrderNo:  strings.TrimSpace(c.Query("order_no")),
		DateFrom: dateFrom, DateTo: dateTo, OutstandingOnly: outstandingOnly,
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrInternal(c, "获取订单列表失败")
		return
	}
	response.OKList(c, rows, total, page, pageSize)
}

// parseOrderDateRange 把 YYYY-MM-DD 的闭区间转成左闭右开区间。
// end_date 含当日，因此加一天；两个参数都可单独省略。
func parseOrderDateRange(start, end string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if start != "" {
		t, err := time.ParseInLocation("2006-01-02", start, time.Local)
		if err != nil {
			return nil, nil, err
		}
		from = &t
	}
	if end != "" {
		t, err := time.ParseInLocation("2006-01-02", end, time.Local)
		if err != nil {
			return nil, nil, err
		}
		t = t.AddDate(0, 0, 1)
		to = &t
	}
	if from != nil && to != nil && !to.After(*from) {
		return nil, nil, fmt.Errorf("INVALID_DATE_RANGE")
	}
	return from, to, nil
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
		"pay_due_at": order.PayDueAt.Format("2006-01-02"), "plan_snapshot": snap,
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
	case errors.Is(err, service.ErrInvalidPayDueAt):
		response.ErrValidation(c, "付款到期日格式不合法，应为 YYYY-MM-DD")
	case errors.Is(err, service.ErrPayDueAtInPast):
		response.ErrValidation(c, "付款到期日不能早于今天")
	case errors.Is(err, service.ErrOrderAmountNotPositive):
		response.ErrValidation(c, "订单应收金额必须大于 0")
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
	// reason 只在净收款为零时必填：它就是信用开通的审计理由（design D2）。
	// 空 body 仍是合法请求，由 service 按实收金额判断是否缺理由。
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	order, ents, err := h.orderSvc.Confirm(c.Request.Context(), id, req.Reason, h.operatorOf(c), c.ClientIP())
	if err != nil {
		writeConfirmOrderError(c, err)
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

// writeConfirmOrderError 是确认失败的唯一映射出口。
// 零收款确认缺理由必须返回可识别的 422，管理端据此提示补充理由，
// 而不是让运营把 500 当成系统故障。
func writeConfirmOrderError(c *gin.Context, err error) {
	// #203 守卫与 service 内的优先级一致：服务期结束 > 过期 > 部分收款 > 零收款理由。
	// 服务期已结束的订单即使全额已付也不可确认；过期且未收清时理由不能绕过。
	if errors.Is(err, service.ErrOrderServiceEnded) {
		response.ErrConflict(c, "订单服务期已结束，不可确认；请取消后重新创建订单")
		return
	}
	if errors.Is(err, service.ErrOrderExpired) {
		response.ErrValidation(c, "订单已过付款到期日且未收清，不可确认；请取消后重新创建订单")
		return
	}
	// 部分收款一律不可确认：除了提示先收清，还要把尚欠金额结构化返回，
	// 管理端据此直接展示，不必解析文案（#201）。
	var underpaid *service.OrderUnderpaidError
	if errors.As(err, &underpaid) {
		response.JSON(c, http.StatusUnprocessableEntity, response.Envelope{
			Code: 42200,
			Message: fmt.Sprintf("订单未收清，尚欠 %.2f 元；部分收款不可确认，请先收清或退款后取消订单",
				float64(underpaid.RemainingCents)/100),
			Data: gin.H{"remaining_cents": underpaid.RemainingCents},
		})
		return
	}
	if errors.Is(err, service.ErrCreditReasonRequired) {
		response.ErrValidation(c, "零收款确认必须填写信用开通理由")
		return
	}
	if errors.Is(err, service.ErrCreditReasonTooLong) {
		response.ErrValidation(c, "信用开通理由不能超过 255 个字符")
		return
	}
	switch err.Error() {
	case "ORDER_NOT_FOUND":
		response.ErrNotFound(c, "订单不存在")
	case "ORDER_ALREADY_CONFIRMED", "ORDER_CANCELLED":
		response.ErrConflict(c, "订单状态不允许确认")
	default:
		response.ErrInternal(c, "确认订单失败")
	}
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
		writeCancelOrderError(c, err)
		return
	}
	response.OK(c, gin.H{"cancelled_at": order.CancelledAt.Format(time.RFC3339)})
}

// writeCancelOrderError 是取消失败的唯一映射出口。
func writeCancelOrderError(c *gin.Context, err error) {
	// 未退款收款阻断取消：除了提示先退款，还要把待退金额结构化返回，
	// 管理端据此直接展示，不必解析文案。
	var unrefunded *service.UnrefundedPaymentError
	if errors.As(err, &unrefunded) {
		response.JSON(c, http.StatusConflict, response.Envelope{
			Code: 40900,
			Message: fmt.Sprintf("订单仍有未退款收款 %.2f 元，请先登记退款后再取消",
				float64(unrefunded.AmountCents)/100),
			Data: gin.H{"unrefunded_amount_cents": unrefunded.AmountCents},
		})
		return
	}
	switch err.Error() {
	case "ORDER_NOT_FOUND":
		response.ErrNotFound(c, "订单不存在")
	case "ORDER_ALREADY_CANCELLED":
		response.ErrConflict(c, "订单已取消")
	default:
		response.ErrInternal(c, "取消订单失败")
	}
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
	// 幂等键优先取请求头（契约推荐），兼容早期把键放在 JSON body 的调用方。
	if key := strings.TrimSpace(c.GetHeader("Idempotency-Key")); key != "" {
		req.IdempotencyKey = key
	}
	res, err := h.orderSvc.AddPayment(c.Request.Context(), id, req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		writeAddPaymentError(c, err)
		return
	}
	response.Created(c, res)
}

// writeAddPaymentError 是登记收款错误的唯一映射出口。金额类错误附带结构化金额，
// 管理端可直接展示「超出多少 / 最多可退多少」，不必解析文案。
func writeAddPaymentError(c *gin.Context, err error) {
	var exceeds *service.PaymentExceedsAmountDueError
	if errors.As(err, &exceeds) {
		response.JSON(c, http.StatusUnprocessableEntity, response.Envelope{
			Code: 42200,
			Message: fmt.Sprintf("收款金额超过订单应收，超出 %.2f 元",
				float64(exceeds.ExcessCents)/100),
			Data: gin.H{"excess_cents": exceeds.ExcessCents, "remaining_cents": exceeds.RemainingCents},
		})
		return
	}
	var refundExceeds *service.RefundExceedsPaidAmountError
	if errors.As(err, &refundExceeds) {
		response.JSON(c, http.StatusUnprocessableEntity, response.Envelope{
			Code: 42200,
			Message: fmt.Sprintf("退款金额超过订单已收，最多可退 %.2f 元",
				float64(refundExceeds.AvailableCents)/100),
			Data: gin.H{
				"available_cents": refundExceeds.AvailableCents,
				"requested_cents": refundExceeds.RequestedCents,
			},
		})
		return
	}
	switch err.Error() {
	case "ORDER_NOT_FOUND":
		response.ErrNotFound(c, "订单不存在")
	case "ORDER_CANCELLED":
		response.ErrConflict(c, "订单已取消")
	case "INVALID_AMOUNT", "INVALID_RECEIVED_AT":
		response.ErrValidation(c, "收款参数不合法")
	case "RECEIVED_AT_IN_FUTURE":
		response.ErrValidation(c, "到账时间不能晚于当前时间")
	case "INVALID_IDEMPOTENCY_KEY":
		response.ErrValidation(c, "幂等键不合法：长度不能超过 64 个字符")
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
