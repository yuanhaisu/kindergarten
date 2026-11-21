package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// #160 L205：会话建立审计写入 module=auth。管理端日志列表按 module/action
// 映射中文标签，未登记的键会原样透出（界面上出现英文 action）。
func TestOperationLogLabelsCoverAuthActions(t *testing.T) {
	assert.Equal(t, "登录与绑定", moduleLabelForAPI("auth"))

	cases := map[string]string{
		"login":               "登录",
		"bind_parent_phone":   "家长绑定手机号",
		"bind_invite":         "老师扫码绑定",
		"switch_kindergarten": "切换园所",
	}
	for action, want := range cases {
		assert.Equal(t, want, actionLabelFor(action), action)
	}
}
