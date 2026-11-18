// Package requestid 在 context 中传递 HTTP 请求链路标识。
//
// 中间件 RequestID 在进入路由时写入，审计等下游从 context 读取，
// 避免每个 service 方法都多传一个参数、也避免某个调用点漏传导致链路断掉。
package requestid

import "context"

type ctxKey struct{}

// With 返回携带链路标识的 context；空标识原样返回，避免写入无意义的值。
func With(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// From 取出链路标识；没有时返回空串。
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
