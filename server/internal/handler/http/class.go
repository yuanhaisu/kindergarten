package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listClassesQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Grade    string `form:"grade"`
}

func (h *Handler) ListClasses(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listClassesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword}
	pq.Normalize()
	list, total, page, pageSize, err := h.classSvc.ListClasses(c.Request.Context(), kgID, pq, q.Grade)
	if err != nil {
		response.ErrInternal(c, "获取班级列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetClassDetail(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	data, err := h.classSvc.GetClassDetail(c.Request.Context(), kgID, classID)
	if err != nil {
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "班级不存在")
			return
		}
		response.ErrInternal(c, "获取班级详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateClass(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.classSvc.CreateClass(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsClientClassErr(err) {
			response.ErrBadRequest(c, 40020, err.Error())
			return
		}
		response.ErrInternal(c, "创建班级失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateClass(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	var req service.UpdateClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.classSvc.UpdateClass(c.Request.Context(), kgID, uid, classID, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "班级不存在")
			return
		}
		if service.IsClientClassErr(err) {
			response.ErrBadRequest(c, 40020, err.Error())
			return
		}
		response.ErrInternal(c, "更新班级失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteClass(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	uid := middleware.GetUserID(c)
	err = h.classSvc.DeleteClass(c.Request.Context(), kgID, uid, classID, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		case "NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
		case "CLASS_NOT_EMPTY":
			response.ErrBadRequest(c, 40021, "班级内仍有学生，无法删除")
		default:
			response.ErrInternal(c, "删除班级失败")
		}
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) AdjustClassStudents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	var req service.AdjustClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.classSvc.AdjustStudents(c.Request.Context(), kgID, uid, classID, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		case "NOT_FOUND", "FROM_CLASS_NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
		case "FROM_CLASS_REQUIRED", "PARTIAL_MOVE", "INVALID_ACTION":
			response.ErrBadRequest(c, 40022, err.Error())
		default:
			response.ErrInternal(c, "调班失败")
		}
		return
	}
	response.OK(c, data)
}
