package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"time"
)

// ServeWithGracefulShutdown 在收到终止信号后停止接收新连接，并在 timeout 内
// 等待在途请求完成。返回 nil 表示正常退出；超时返回 context 错误。
//
// 抽成独立函数是为了让「在途请求不被掐断」可以被测试覆盖，而不是只能在
// 部署时靠观察日志验证。
func ServeWithGracefulShutdown(srv *http.Server, ln net.Listener, timeout time.Duration, signals <-chan os.Signal) error {
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return srv.Shutdown(ctx)
	case err := <-serveErr:
		return err
	}
}
