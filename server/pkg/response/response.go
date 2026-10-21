package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

type PageData struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func JSON(c *gin.Context, httpStatus int, env Envelope) {
	if env.RequestID == "" {
		env.RequestID = c.GetString("request_id")
	}
	c.JSON(httpStatus, env)
}

func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, Envelope{Code: 0, Message: "success", Data: data})
}

func OKList(c *gin.Context, list any, total int64, page, pageSize int) {
	OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, Envelope{Code: 0, Message: "success", Data: data})
}

func Err(c *gin.Context, httpStatus int, bizCode int, message string) {
	JSON(c, httpStatus, Envelope{Code: bizCode, Message: message, Data: nil})
}

func ErrValidation(c *gin.Context, message string) {
	Err(c, http.StatusUnprocessableEntity, 42200, message)
}

func ErrUnauthorized(c *gin.Context, message string) {
	Err(c, http.StatusUnauthorized, 40100, message)
}

func ErrForbidden(c *gin.Context, message string) {
	Err(c, http.StatusForbidden, 40300, message)
}

func ErrNotFound(c *gin.Context, message string) {
	Err(c, http.StatusNotFound, 40400, message)
}

func ErrInternal(c *gin.Context, message string) {
	Err(c, http.StatusInternalServerError, 50000, message)
}

func ErrConflict(c *gin.Context, message string) {
	Err(c, http.StatusConflict, 40900, message)
}

func ErrUnavailable(c *gin.Context, message string) {
	Err(c, http.StatusServiceUnavailable, 50300, message)
}

func ErrRateLimited(c *gin.Context) {
	Err(c, http.StatusTooManyRequests, 42900, "请求过于频繁，请稍后再试")
}

func ErrBadRequest(c *gin.Context, bizCode int, message string) {
	Err(c, http.StatusBadRequest, bizCode, message)
}
