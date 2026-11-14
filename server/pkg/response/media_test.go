package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mediaDTO struct {
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url,omitempty"`
	Avatar   string `json:"avatar"`
	Other    string `json:"other"`
}

func TestSignMediaMap(t *testing.T) {
	seen := map[string]bool{}
	SetMediaSigner(func(ref string, thumb bool) string {
		seen[ref] = thumb
		return "signed:" + ref
	})
	defer SetMediaSigner(nil)

	data := map[string]any{
		"file_url":  "file.jpg",
		"thumb_url": "thumb.jpg",
		"nested":    []any{map[string]any{"avatar": "avatar.jpg"}},
		"other":     "other.jpg",
	}
	signed := signMedia(data).(map[string]any)

	assert.Equal(t, "signed:file.jpg", signed["file_url"])
	assert.Equal(t, "signed:thumb.jpg", signed["thumb_url"])
	assert.Equal(t, "other.jpg", signed["other"])
	nested := signed["nested"].([]any)[0].(map[string]any)
	assert.Equal(t, "signed:avatar.jpg", nested["avatar"])
	assert.False(t, seen["file.jpg"])
	assert.True(t, seen["thumb.jpg"])
	assert.False(t, seen["avatar.jpg"])
}

func TestSignMediaStruct(t *testing.T) {
	SetMediaSigner(func(ref string, thumb bool) string { return "signed:" + ref })
	defer SetMediaSigner(nil)

	in := mediaDTO{FileURL: "file.jpg", ThumbURL: "thumb.jpg", Avatar: "avatar.jpg", Other: "other.jpg"}
	out := signMedia(in).(mediaDTO)

	assert.Equal(t, "signed:file.jpg", out.FileURL)
	assert.Equal(t, "signed:thumb.jpg", out.ThumbURL)
	assert.Equal(t, "signed:avatar.jpg", out.Avatar)
	assert.Equal(t, "other.jpg", out.Other)
	assert.Equal(t, "file.jpg", in.FileURL, "原始数据不应被修改")
}

func TestSignMediaSlice(t *testing.T) {
	SetMediaSigner(func(ref string, thumb bool) string { return "signed:" + ref })
	defer SetMediaSigner(nil)

	in := []mediaDTO{{FileURL: "a.jpg"}, {FileURL: "b.jpg"}}
	out := signMedia(in).([]mediaDTO)
	assert.Equal(t, "signed:a.jpg", out[0].FileURL)
	assert.Equal(t, "signed:b.jpg", out[1].FileURL)
}

func TestNoSignerLeavesDataUnchanged(t *testing.T) {
	SetMediaSigner(nil)
	data := map[string]any{"file_url": "a.jpg"}
	assert.Equal(t, data, signMedia(data))
}

func TestOKSignsMedia(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetMediaSigner(func(ref string, thumb bool) string { return "signed:" + ref })
	defer SetMediaSigner(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, map[string]any{"file_url": "a.jpg"})

	assert.Contains(t, w.Body.String(), `"file_url":"signed:a.jpg"`)
}

func TestOKListSignsNestedMedia(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetMediaSigner(func(ref string, thumb bool) string { return "signed:" + ref })
	defer SetMediaSigner(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OKList(c, []mediaDTO{{FileURL: "a.jpg", ThumbURL: "a_thumb.jpg"}}, 1, 1, 10)

	assert.Contains(t, w.Body.String(), `"file_url":"signed:a.jpg"`)
	assert.Contains(t, w.Body.String(), `"thumb_url":"signed:a_thumb.jpg"`)
}
