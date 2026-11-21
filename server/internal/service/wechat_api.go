package service

import (
	"context"

	"kindergarten-server/internal/auth"
)

// WeChatAPI 是微信小程序开放能力的消费方接口。
//
// 生产实现是 *auth.WeChatClient；测试通过 HTTP 单缝注入进程内替身，
// 认证/绑定/小程序码测试因此不需要访问真实微信，也不会因缺凭据而跳过。
// 只声明 AuthService 与 TeacherService 实际用到的方法，不把 access_token、
// 凭据等实现细节泄漏给调用方。
type WeChatAPI interface {
	Code2Session(ctx context.Context, code, role string) (*auth.Code2SessionResp, error)
	GetPhoneNumber(ctx context.Context, code string) (*auth.PhoneNumberResp, error)
	GetWXACodeUnlimit(ctx context.Context, scene, page string) ([]byte, error)
	GetParentWXACodeUnlimit(ctx context.Context, scene, page, envVersion string) ([]byte, error)
	CodeEnvVersion() string
}
