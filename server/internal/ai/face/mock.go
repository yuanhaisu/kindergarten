package face

import (
	"context"
	"strconv"
	"strings"
)

// MockProvider 确定性假实现：让无云凭证的环境（开发机/CI）完整跑通
// 「底库注册 → 识别 → 建议 → 确认 → 家长筛选」全链路。
//
// 约定（确定性便于断言）：
//   - EnsurePerson：personRef = kg{id}_stu{sid}（与 tencent 同构），至少一张图才成功
//   - Search：解析图片内容开头的 mock 指令行（可多行叠加）
//       "MOCK_STUDENTS:1,5"   → 返回 [{1, 90}, {5, 72}]（高分在前）
//       "MOCK_NONE"           → 返回空（模拟图中无人脸）
//       "MOCK_FACES:8"        → FaceNum=8（模拟大合影分型；缺省 1 = 单人照）
//       其他内容               → 空（模拟无命中）
type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (p *MockProvider) Name() string { return "mock" }

func (p *MockProvider) EnsurePerson(_ context.Context, kgID, studentID uint64, images [][]byte) (string, error) {
	if len(images) == 0 {
		return "", errNoImage
	}
	return personID(kgID, studentID), nil
}

func (p *MockProvider) RemovePerson(_ context.Context, kgID, studentID uint64) error {
	_ = personID(kgID, studentID)
	return nil
}

func (p *MockProvider) Search(_ context.Context, kgID uint64, image []byte) (SearchResult, error) {
	result := SearchResult{FaceNum: 1} // 缺省单人照
	// 解析开头的指令行（可多行叠加，如 "MOCK_FACES:8" + "MOCK_STUDENTS:21,22"）；
	// 首个非 MOCK_ 行视为指令块结束（后续是二进制内容）
	for _, line := range strings.Split(string(image), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "MOCK_") {
			break
		}
		switch {
		case line == "MOCK_NONE":
			return SearchResult{}, nil // 图中无人脸
		case strings.HasPrefix(line, "MOCK_FACES:"):
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "MOCK_FACES:"))); err == nil && n >= 0 {
				result.FaceNum = n
			}
		case strings.HasPrefix(line, "MOCK_STUDENTS:"):
			score := float64(90)
			for _, tok := range strings.Split(strings.TrimPrefix(line, "MOCK_STUDENTS:"), ",") {
				sid, err := strconv.ParseUint(strings.TrimSpace(tok), 10, 64)
				if err != nil {
					continue
				}
				result.Matches = append(result.Matches, Match{StudentID: sid, Score: score})
				score -= 18 // 递减：便于断言排序与阈值分档
				if score < 1 {
					score = 1
				}
			}
		}
	}
	return result, nil
}

var errNoImage = &faceError{"mock: 需要至少一张人脸照片"}

type faceError struct{ msg string }

func (e *faceError) Error() string { return e.msg }
