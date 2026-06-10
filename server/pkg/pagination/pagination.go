package pagination

// Query 列表接口通用分页与关键词查询（绑定 query string）
type Query struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Normalize 约束 page / page_size 在合理范围内
func (q *Query) Normalize() {
	if q.Page <= 0 {
		q.Page = defaultPage
	}
	if q.PageSize <= 0 {
		q.PageSize = defaultPageSize
	}
	if q.PageSize > maxPageSize {
		q.PageSize = maxPageSize
	}
}

// Offset 用于 GORM / SQL OFFSET
func (q *Query) Offset() int {
	return (q.Page - 1) * q.PageSize
}
