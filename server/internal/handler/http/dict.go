package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// ==================== Admin Dict Handlers ====================

func (h *Handler) ListDictTypes(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.dictSvc.ListTypes(c.Request.Context(), kgID)
	if err != nil {
		response.ErrInternal(c, "获取字典类型列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateDictType(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateDictTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	data, err := h.dictSvc.CreateType(c.Request.Context(), kgID, req)
	if err != nil {
		if service.IsDictClientErr(err) {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "创建字典类型失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateDictType(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的类型 ID")
		return
	}
	var req service.UpdateDictTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	data, err := h.dictSvc.UpdateType(c.Request.Context(), kgID, typeID, req)
	if err != nil {
		if errors.Is(err, service.ErrDictTypeNotFound) {
			response.ErrNotFound(c, "字典类型不存在")
			return
		}
		response.ErrInternal(c, "更新字典类型失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteDictType(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的类型 ID")
		return
	}
	if err := h.dictSvc.DeleteType(c.Request.Context(), kgID, typeID); err != nil {
		if errors.Is(err, service.ErrDictTypeNotFound) {
			response.ErrNotFound(c, "字典类型不存在")
			return
		}
		if errors.Is(err, service.ErrDictSystemType) || errors.Is(err, service.ErrDictTypeHasItems) {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "删除字典类型失败")
		return
	}
	response.OK(c, nil)
}

// --- Dict Items ---

func (h *Handler) ListDictItems(c *gin.Context) {
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的类型 ID")
		return
	}
	data, err := h.dictSvc.ListItems(c.Request.Context(), typeID)
	if err != nil {
		response.ErrInternal(c, "获取字典项列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateDictItem(c *gin.Context) {
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的类型 ID")
		return
	}
	var req service.CreateDictItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	data, err := h.dictSvc.CreateItem(c.Request.Context(), typeID, req)
	if err != nil {
		response.ErrInternal(c, "创建字典项失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateDictItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的字典项 ID")
		return
	}
	var req service.UpdateDictItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	data, err := h.dictSvc.UpdateItem(c.Request.Context(), itemID, req)
	if err != nil {
		if errors.Is(err, service.ErrDictItemNotFound) {
			response.ErrNotFound(c, "字典项不存在")
			return
		}
		response.ErrInternal(c, "更新字典项失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteDictItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的字典项 ID")
		return
	}
	if err := h.dictSvc.DeleteItem(c.Request.Context(), itemID); err != nil {
		if errors.Is(err, service.ErrDictItemNotFound) {
			response.ErrNotFound(c, "字典项不存在")
			return
		}
		response.ErrInternal(c, "删除字典项失败")
		return
	}
	response.OK(c, nil)
}

func (h *Handler) BatchSortDictItems(c *gin.Context) {
	var req service.BatchSortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.dictSvc.BatchSortItems(c.Request.Context(), req); err != nil {
		response.ErrInternal(c, "批量排序失败")
		return
	}
	response.OK(c, nil)
}

// ==================== Teacher Dict Handlers ====================

func (h *Handler) GetDictItems(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	typeKey := c.Param("type_key")
	if typeKey == "" {
		response.ErrValidation(c, "缺少字典类型标识")
		return
	}
	data, err := h.dictSvc.GetDictItems(c.Request.Context(), kgID, typeKey)
	if err != nil {
		response.ErrInternal(c, "获取字典数据失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetDictItemsBatch(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	typesStr := c.Query("types")
	if typesStr == "" {
		response.ErrValidation(c, "缺少 types 参数")
		return
	}
	parts := strings.Split(typesStr, ",")
	typeKeys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			typeKeys = append(typeKeys, p)
		}
	}
	if len(typeKeys) == 0 {
		response.ErrValidation(c, "types 参数为空")
		return
	}
	data, err := h.dictSvc.GetDictItemsBatch(c.Request.Context(), kgID, typeKeys)
	if err != nil {
		response.ErrInternal(c, "获取字典数据失败")
		return
	}
	response.OK(c, data)
}
