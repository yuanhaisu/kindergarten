package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

// OS-ALB-5 管理端班级相册：只读列表/详情 + 下架。
// 不提供创建/编辑/删除端点（design D3：园长只能读与下架）。

type listAdminAlbumsQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	ClassID  uint64 `form:"class_id"`
	Status   string `form:"status"`
	Category string `form:"category"`
}

func (h *Handler) AdminListAlbums(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listAdminAlbumsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.adminSvc.ListAdminAlbums(
		c.Request.Context(), kgID, q.ClassID, q.Status, q.Category, pq)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if errors.Is(err, service.ErrAdminAlbumInvalidParam) {
			response.ErrValidation(c, "筛选条件无效")
			return
		}
		response.ErrInternal(c, "获取相册列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) AdminGetAlbum(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的相册 ID")
		return
	}
	data, err := h.adminSvc.GetAdminAlbum(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if errors.Is(err, service.ErrAdminAlbumNotFound) {
			response.ErrNotFound(c, "相册不存在")
			return
		}
		response.ErrInternal(c, "获取相册详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminUnpublishAlbum(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的相册 ID")
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.adminSvc.UnpublishAdminAlbum(
		c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if errors.Is(err, service.ErrAdminAlbumNotFound) {
			response.ErrNotFound(c, "相册不存在")
			return
		}
		if errors.Is(err, service.ErrAdminAlbumNotPublished) {
			response.ErrValidation(c, "只有已发布相册可以下架")
			return
		}
		response.ErrInternal(c, "下架相册失败")
		return
	}
	response.OK(c, data)
}
