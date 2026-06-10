package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listTeachersQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	ClassID  uint64 `form:"class_id"`
	Status   *int8  `form:"status"`
}

func (h *Handler) ListTeachers(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listTeachersQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword}
	list, total, page, pageSize, err := h.teacherSvc.ListTeachers(c.Request.Context(), kgID, pq, classID, q.Status)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取老师列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetTeacher(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	data, err := h.teacherSvc.GetTeacher(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		response.ErrInternal(c, "获取老师详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateTeacher(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateTeacherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.teacherSvc.CreateTeacher(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsTeacherClientErr(err) {
			response.ErrBadRequest(c, 40040, err.Error())
			return
		}
		response.ErrInternal(c, "添加老师失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateTeacher(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	var req service.UpdateTeacherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.teacherSvc.UpdateTeacher(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		if service.IsTeacherClientErr(err) {
			response.ErrBadRequest(c, 40040, err.Error())
			return
		}
		response.ErrInternal(c, "更新老师失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteTeacher(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.teacherSvc.DeleteTeacher(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		response.ErrInternal(c, "删除老师失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) UpdateTeacherStatus(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	var req service.UpdateTeacherStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.teacherSvc.UpdateTeacherStatus(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		if service.IsTeacherClientErr(err) {
			response.ErrBadRequest(c, 40040, err.Error())
			return
		}
		response.ErrInternal(c, "更新老师状态失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetTeacherInvitation(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	data, err := h.teacherSvc.GetTeacherInvitation(c.Request.Context(), kgID, id)
	if err != nil {
		response.ErrInternal(c, "获取邀请码失败")
		return
	}
	if data == nil {
		response.OK(c, gin.H{"invitation": nil})
		return
	}
	response.OK(c, data)
}

func (h *Handler) RegenerateTeacherInvitation(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.teacherSvc.RegenerateTeacherInvitation(c.Request.Context(), kgID, uid, id)
	if err != nil {
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		response.ErrInternal(c, "重新生成邀请码失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) GetTeacherInviteQRCode(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的老师 ID")
		return
	}
	png, err := h.teacherSvc.GenerateInviteQRCode(c.Request.Context(), kgID, id)
	if err != nil {
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "老师不存在")
			return
		}
		response.ErrInternal(c, "生成小程序码失败")
		return
	}
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Writer.Write(png)
}
