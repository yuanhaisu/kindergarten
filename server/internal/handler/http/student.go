package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listStudentsQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	ClassID  uint64 `form:"class_id"`
	Status   *int8  `form:"status"`
}

func (h *Handler) ListStudents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listStudentsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword}
	list, total, page, pageSize, err := h.studentSvc.ListStudents(c.Request.Context(), kgID, pq, classID, q.Status)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取学生列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetStudent(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	data, err := h.studentSvc.GetStudent(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "学生不存在")
			return
		}
		response.ErrInternal(c, "获取学生详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateStudent(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateStudentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.studentSvc.CreateStudent(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsStudentClientErr(err) {
			response.ErrBadRequest(c, 40030, err.Error())
			return
		}
		response.ErrInternal(c, "添加学生失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateStudent(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	var req service.UpdateStudentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.studentSvc.UpdateStudent(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "学生不存在")
			return
		}
		if service.IsStudentClientErr(err) {
			response.ErrBadRequest(c, 40030, err.Error())
			return
		}
		response.ErrInternal(c, "更新学生失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteStudent(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.studentSvc.DeleteStudent(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "学生不存在")
			return
		}
		response.ErrInternal(c, "删除学生失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) UpdateStudentStatus(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	var req service.UpdateStudentStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.studentSvc.UpdateStudentStatus(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "学生不存在")
			return
		}
		if service.IsStudentClientErr(err) {
			response.ErrBadRequest(c, 40030, err.Error())
			return
		}
		response.ErrInternal(c, "更新学生状态失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) ImportStudents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.ErrValidation(c, "请上传 Excel 文件（字段名 file）")
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.ErrValidation(c, "无法读取上传文件")
		return
	}
	defer func() { _ = src.Close() }()
	body, err := io.ReadAll(src)
	if err != nil {
		response.ErrInternal(c, "读取文件失败")
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.studentSvc.ImportStudentsExcel(c.Request.Context(), kgID, uid, body, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsStudentClientErr(err) {
			response.ErrBadRequest(c, 40030, err.Error())
			return
		}
		response.ErrInternal(c, "导入失败")
		return
	}
	response.OK(c, data)
}

type exportStudentsQuery struct {
	ClassID uint64 `form:"class_id"`
	Status  *int8  `form:"status"`
	Format  string `form:"format"`
}

func (h *Handler) ExportStudents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q exportStudentsQuery
	_ = c.ShouldBindQuery(&q)
	if q.Format != "" && q.Format != "xlsx" {
		response.ErrBadRequest(c, 40030, "仅支持 format=xlsx")
		return
	}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	data, fn, err := h.studentSvc.ExportStudentsXLSX(c.Request.Context(), kgID, classID, q.Status)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "导出失败")
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
