package face

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/config"
)

// 覆盖 mock provider 的确定性输出与 tencent provider 的纯函数（person/group 标识与解析）。
// 真实云端行为由 cmd/facecheck 实测承担（TC-NEW-FRG 系列的集成部分）。

func TestMockProvider_DeterministicMatches(t *testing.T) {
	p := NewMockProvider()
	ctx := context.Background()

	// 指令行解析：高分在前、分数递减；缺省单人照（FaceNum=1）
	res, err := p.Search(ctx, 4, []byte("MOCK_STUDENTS:11,25\n<binary>"))
	assert.NoError(t, err)
	assert.Len(t, res.Matches, 2)
	assert.Equal(t, uint64(11), res.Matches[0].StudentID)
	assert.Greater(t, res.Matches[0].Score, res.Matches[1].Score, "分数应降序")
	assert.Equal(t, 1, res.FaceNum, "缺省单人照（D11 分型）")

	// 无人脸指令 → 空且 FaceNum=0
	res, err = p.Search(ctx, 4, []byte("MOCK_NONE\nx"))
	assert.NoError(t, err)
	assert.Empty(t, res.Matches)
	assert.Zero(t, res.FaceNum)

	// 无指令 → 空（模拟无命中）
	res, err = p.Search(ctx, 4, []byte("\x89PNG random"))
	assert.NoError(t, err)
	assert.Empty(t, res.Matches)

	// 多行叠加：大合影分型 + 命中（D11：队列据 FaceNum 丢弃候选）
	res, err = p.Search(ctx, 4, []byte("MOCK_FACES:8\nMOCK_STUDENTS:11\n<bin>"))
	assert.NoError(t, err)
	assert.Equal(t, 8, res.FaceNum)
	assert.Len(t, res.Matches, 1, "分型与命中可同时模拟")
}

func TestMockProvider_EnsurePersonNeedsImage(t *testing.T) {
	p := NewMockProvider()
	_, err := p.EnsurePerson(context.Background(), 4, 11, nil)
	assert.Error(t, err, "无照片必须失败")

	ref, err := p.EnsurePerson(context.Background(), 4, 11, [][]byte{[]byte("img")})
	assert.NoError(t, err)
	assert.Equal(t, "kg4_stu11", ref)
}

func TestIDs_RoundTrip(t *testing.T) {
	assert.Equal(t, "kg_4", groupID(4), "园所分组标识：kg_{id}（设计约定）")
	pid := personID(4, 11)
	assert.Equal(t, "kg4_stu11", pid)
	sid, ok := parseStudentID(pid)
	assert.True(t, ok)
	assert.Equal(t, uint64(11), sid)

	if _, ok := parseStudentID("foreign_person"); ok {
		t.Fatal("非本系统的 PersonId 必须解析失败（防御性跳过）")
	}
}

func TestNew_SelectsProvider(t *testing.T) {
	p, err := New(config.FaceConfig{Provider: "mock"})
	assert.NoError(t, err)
	assert.Equal(t, "mock", p.Name())

	p, err = New(config.FaceConfig{Provider: ""})
	assert.NoError(t, err)
	assert.Equal(t, "mock", p.Name(), "空配置回退 mock（无凭证环境可用）")

	_, err = New(config.FaceConfig{Provider: "tencent"})
	assert.Error(t, err, "tencent 缺密钥必须报错而非降级")

	_, err = New(config.FaceConfig{Provider: "unknown"})
	assert.Error(t, err)
}
