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

	opts := &slog.HandlerOptions{Level: lv}
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
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
