package repository

import (
	"errors"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

// 建单撞号重试只认 order_no 唯一键：其他唯一约束冲突必须原样抛给调用方，
// 否则重试会掩盖真实的数据冲突（例如同一手机号并发开通）。
// 名字以 TestOrder 开头才能进 CI clean-DB 的商业域过滤（#197）。
func TestOrderIsDuplicateKeyFor_OnlyMatchesTargetKey(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		keyPart string
		want    bool
	}{
		{
			name:    "goose 迁移索引 uk_order_no",
			err:     &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'ORD1' for key 'orders.uk_order_no'"},
			keyPart: "order_no",
			want:    true,
		},
		{
			name:    "AutoMigrate 索引 idx_orders_order_no",
			err:     &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'ORD1' for key 'orders.idx_orders_order_no'"},
			keyPart: "order_no",
			want:    true,
		},
		{
			name:    "其他唯一键不得重试",
			err:     &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry '139...' for key 'users.uk_users_phone_active'"},
			keyPart: "order_no",
			want:    false,
		},
		{
			name:    "非唯一键错误不得重试",
			err:     &mysqldriver.MySQLError{Number: 1452, Message: "Cannot add or update a child row"},
			keyPart: "order_no",
			want:    false,
		},
		{
			name:    "非 MySQL 错误不得重试",
			err:     errors.New("connection reset"),
			keyPart: "order_no",
			want:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsDuplicateKeyFor(tc.err, tc.keyPart))
		})
	}
}
