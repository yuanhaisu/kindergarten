package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

func (h *Handler) Upload(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	bizType := c.PostForm("biz_type")
	if bizType == "" {
		response.ErrValidation(c, "biz_type 为必填项")
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		response.ErrValidation(c, "file 为必填项")
		return
	}

	result, err := h.fileSvc.Upload(c.Request.Context(), uid, fh, bizType)
	if err != nil {
		switch err.Error() {
		case "UPLOAD_TOO_LARGE":
			response.ErrBadRequest(c, 42201, "文件大小超过限制")
		case "UPLOAD_INVALID_TYPE":
			response.ErrBadRequest(c, 42202, "不支持的文件类型")
		default:
			response.ErrInternal(c, "文件上传失败")
		}
		return
	}

	response.OK(c, result)
}

func (h *Handler) BatchUpload(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	bizType := c.PostForm("biz_type")
	if bizType == "" {
		response.ErrValidation(c, "biz_type 为必填项")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		response.ErrValidation(c, "请上传文件")
		return
	}

	files := form.File["files[]"]
	if len(files) == 0 {
		response.ErrValidation(c, "files[] 为必填项")
		return
	}

	result, err := h.fileSvc.BatchUpload(c.Request.Context(), uid, files, bizType)
	if err != nil {
		if err.Error() == "UPLOAD_BATCH_LIMIT" {
			response.ErrBadRequest(c, 42203, "批量上传最多20个文件")
			return
		}
		response.ErrInternal(c, "批量上传失败")
		return
	}

	response.OK(c, result)
}
