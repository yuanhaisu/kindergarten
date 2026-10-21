package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// faceRegistryErr 底库端点错误映射（与 student.go 的映射风格一致）。
func faceRegistryErr(c *gin.Context, err error, fallback string) {
	switch {
	case err.Error() == "NO_KINDERGARTEN":
		response.ErrForbidden(c, "未绑定园所，请重新登录")
	case errors.Is(err, service.ErrFaceNotEnabled):
		response.ErrBadRequest(c, 40030, "本园所未开启人脸识别，暂不能录入底库照片")
	case service.IsNotFound(err):
		response.ErrNotFound(c, "学生或照片不存在")
	case err.Error() == "INVALID_FILE":
		response.ErrBadRequest(c, 40030, "照片无效：必须是已上传的图片素材")
	case err.Error() == "FACE_LIMIT_EXCEEDED":
		response.ErrBadRequest(c, 40030, "底库照片最多 5 张")
	default:
		response.ErrInternal(c, fallback)
	}
}

func faceStudentID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return 0, false
	}
	return id, true
}

func (h *Handler) ListStudentFaces(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	studentID, ok := faceStudentID(c)
	if !ok {
		return
	}
	data, err := h.faceSvc.ListStudentFaces(c.Request.Context(), kgID, studentID)
	if err != nil {
		faceRegistryErr(c, err, "获取底库照片失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) AddStudentFaces(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	studentID, ok := faceStudentID(c)
	if !ok {
		return
	}
	var req struct {
		FileIDs []uint64 `json:"file_ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.faceSvc.AddStudentFaces(c.Request.Context(), kgID, studentID, req.FileIDs)
	if err != nil {
		faceRegistryErr(c, err, "录入底库照片失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) RetryStudentFaces(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	studentID, ok := faceStudentID(c)
	if !ok {
		return
	}
	data, err := h.faceSvc.RetryStudentFaces(c.Request.Context(), kgID, studentID)
	if err != nil {
		faceRegistryErr(c, err, "重试注册失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteStudentFace(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	studentID, ok := faceStudentID(c)
	if !ok {
		return
	}
	faceID, err := strconv.ParseUint(c.Param("faceId"), 10, 64)
	if err != nil || faceID == 0 {
		response.ErrValidation(c, "无效的照片 ID")
		return
	}
	if err := h.faceSvc.DeleteStudentFace(c.Request.Context(), kgID, studentID, faceID); err != nil {
		faceRegistryErr(c, err, "删除底库照片失败")
		return
	}
	response.OK(c, gin.H{})
}
