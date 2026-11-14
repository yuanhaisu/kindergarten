package log

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactAttrMasksPhone(t *testing.T) {
	got := redactAttr(nil, slog.String("phone", "13800138000"))
	assert.Equal(t, "138****8000", got.Value.String())
}

func TestRedactAttrMasksOpenID(t *testing.T) {
	got := redactAttr(nil, slog.String("openid", "oABCDEFGHIJK"))
	assert.Equal(t, "oABC****HIJK", got.Value.String())
}

func TestRedactAttrMasksSecrets(t *testing.T) {
	for _, key := range []string{"password", "secret", "token", "access_key_secret", "jwt"} {
		got := redactAttr(nil, slog.String(key, "plaintext-value"))
		assert.Equal(t, "***", got.Value.String(), key)
	}
}

func TestRedactAttrKeepsSafeFields(t *testing.T) {
	got := redactAttr(nil, slog.String("request_id", "req-123"))
	assert.Equal(t, "req-123", got.Value.String())
}
