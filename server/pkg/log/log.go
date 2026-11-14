package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

func Init(level string, json bool) {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lv, ReplaceAttr: redactAttr}
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

// sensitiveKeys 是日志中绝不允许出现明文的字段名。按字段名统一脱敏，
// 比逐个调用点处理更可靠：新增日志时即使忘了脱敏，也不会把原值写出去。
var sensitiveKeys = map[string]struct{}{
	"phone": {}, "mobile": {},
	"openid": {}, "open_id": {}, "unionid": {}, "union_id": {},
	"password": {}, "passwd": {}, "pwd": {}, "secret": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "authorization": {},
	"api_key": {}, "apikey": {}, "access_key_id": {}, "access_key_secret": {},
	"secret_id": {}, "secret_key": {}, "jwt": {},
}

func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if _, ok := sensitiveKeys[strings.ToLower(a.Key)]; !ok {
		return a
	}
	if a.Value.Kind() == slog.KindString {
		return slog.String(a.Key, maskSensitive(a.Key, a.Value.String()))
	}
	return slog.String(a.Key, "***")
}

func maskSensitive(key, value string) string {
	switch strings.ToLower(key) {
	case "phone", "mobile":
		if len(value) >= 7 {
			return value[:3] + "****" + value[len(value)-4:]
		}
	case "openid", "open_id", "unionid", "union_id":
		if len(value) > 8 {
			return value[:4] + "****" + value[len(value)-4:]
		}
	}
	return "***"
}

// With 向 context 中追加日志字段，返回新 context
func With(ctx context.Context, args ...any) context.Context {
	parent := L(ctx)
	return context.WithValue(ctx, ctxKey{}, parent.With(args...))
}

// L 从 context 中取出带字段的 Logger；无则返回默认
func L(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		return v.(*slog.Logger)
	}
	return slog.Default()
}
