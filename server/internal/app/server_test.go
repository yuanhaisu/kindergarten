package app

import (
	"context"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeWithGracefulShutdownWaitsForInflight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{Handler: handler}
	signals := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() { done <- ServeWithGracefulShutdown(srv, ln, 2*time.Second, signals) }()

	respCh := make(chan *http.Response, 1)
	go func() {
		resp, reqErr := http.Get("http://" + ln.Addr().String())
		if reqErr != nil {
			respCh <- nil
			return
		}
		respCh <- resp
	}()

	<-started
	signals <- syscall.SIGTERM
	close(release)

	resp := <-respCh
	require.NotNil(t, resp, "在途请求应完成而不是被中断")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
	require.NoError(t, <-done)
}

func TestServeWithGracefulShutdownTimesOut(t *testing.T) {
	started := make(chan struct{})
	block := make(chan struct{})
	defer close(block)

	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-block
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{Handler: handler}
	signals := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() { done <- ServeWithGracefulShutdown(srv, ln, 50*time.Millisecond, signals) }()
	go func() { _, _ = http.Get("http://" + ln.Addr().String()) }()

	<-started
	signals <- syscall.SIGTERM

	err = <-done
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
