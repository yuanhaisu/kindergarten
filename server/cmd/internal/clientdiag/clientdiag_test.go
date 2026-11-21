package clientdiag_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/cmd/internal/clientdiag"
	"kindergarten-server/internal/domain"
)

// 平台客户端只读诊断（#179）的行为测试。
//
// fakeLister 只实现 List：诊断拿不到任何写方法，因此「不改 status」不是靠
// 约定，而是类型层面就拿不到。测试断言外部输出，不碰内部实现。

type fakeLister struct {
	rows []domain.Client
	err  error
}

func (f fakeLister) List(context.Context) ([]domain.Client, error) {
	return f.rows, f.err
}

func enabledTeacher() domain.Client {
	return domain.Client{
		AppID:          "wx_teacher_diag",
		Role:           domain.ClientRoleTeacher,
		ScopeKind:      domain.ClientScopeShared,
		ScopeID:        0,
		Status:         domain.ClientStatusEnabled,
		ReleaseVersion: "1.4.2",
	}
}

func disabledParent() domain.Client {
	return domain.Client{
		AppID:          "wx_parent_diag_old",
		Role:           domain.ClientRoleParent,
		ScopeKind:      domain.ClientScopeShared,
		ScopeID:        0,
		Status:         domain.ClientStatusDisabled,
		ReleaseVersion: "0.9.0",
	}
}

// 平台支持要判断「问题出在哪一端」：app_id / 角色 / 作用域 / 状态 / 版本缺一不可。
func TestCollectReportsClientRegistryFields(t *testing.T) {
	got, err := clientdiag.Collect(context.Background(), fakeLister{
		rows: []domain.Client{enabledTeacher(), disabledParent()},
	})
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "wx_teacher_diag", got[0].AppID)
	assert.Equal(t, domain.ClientRoleTeacher, got[0].Role)
	assert.Equal(t, domain.ClientScopeShared, got[0].ScopeKind)
	assert.Equal(t, uint64(0), got[0].ScopeID)
	assert.Equal(t, domain.ClientStatusEnabled, got[0].Status)
	assert.True(t, got[0].Enabled, "enabled 由 status 派生，供脚本直接判断")
	assert.Equal(t, "1.4.2", got[0].ReleaseVersion)

	assert.Equal(t, "wx_parent_diag_old", got[1].AppID)
	assert.Equal(t, domain.ClientStatusDisabled, got[1].Status)
	assert.False(t, got[1].Enabled)
}

// 读库失败必须冒泡为错误，不能打印一张空表让人误以为「注册表本来就是空的」。
func TestCollectPropagatesRepositoryError(t *testing.T) {
	_, err := clientdiag.Collect(context.Background(), fakeLister{
		err: assert.AnError,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError, "仓储错误要原样冒泡，便于定位连库/权限问题")
}

// 人类可读输出必须一行一条客户端，字段齐全；停用的客户端要一眼能看出来。
func TestWriteTableRendersEachClient(t *testing.T) {
	rows := []clientdiag.ClientDiagnostic{
		{AppID: "wx_teacher_diag", Role: "teacher", ScopeKind: "shared", Status: "enabled", Enabled: true, ReleaseVersion: "1.4.2"},
		{AppID: "wx_parent_diag_old", Role: "parent", ScopeKind: "shared", Status: "disabled", Enabled: false, ReleaseVersion: "0.9.0"},
	}

	var buf bytes.Buffer
	require.NoError(t, clientdiag.WriteTable(&buf, rows))
	out := buf.String()

	assert.Contains(t, out, "APP_ID")
	assert.Contains(t, out, "RELEASE_VERSION")

	fields := rowFields(out, "wx_teacher_diag")
	assert.Equal(t, []string{"wx_teacher_diag", "teacher", "shared", "0", "enabled", "yes", "1.4.2"}, fields,
		"字段顺序要与表头一致，运维才能按列读")

	fields = rowFields(out, "wx_parent_diag_old")
	assert.Equal(t, []string{"wx_parent_diag_old", "parent", "shared", "0", "disabled", "no", "0.9.0"}, fields)
}

// 空注册表必须给出明确提示，而不是一张只有表头的表——否则看不出是「没同步」还是「命令坏了」。
func TestWriteTableEmptyRegistryPrintsHint(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, clientdiag.WriteTable(&buf, nil))

	out := buf.String()
	assert.Contains(t, out, "0 条")
	assert.Contains(t, out, "为空")
}

// rowFields 取包含 appID 的那一行，按空白切分成列。
func rowFields(out, appID string) []string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, appID) {
			return strings.Fields(line)
		}
	}
	return nil
}

// JSON 输出供脚本/工单粘贴：字段名固定，enabled 是布尔值而不是字符串。
func TestWriteJSONEmitsStableShape(t *testing.T) {
	rows := []clientdiag.ClientDiagnostic{
		{AppID: "wx_teacher_diag", Role: "teacher", ScopeKind: "shared", Status: "enabled", Enabled: true, ReleaseVersion: "1.4.2"},
	}

	var buf bytes.Buffer
	require.NoError(t, clientdiag.WriteJSON(&buf, rows))

	var got struct {
		Count   int `json:"count"`
		Clients []struct {
			AppID          string `json:"app_id"`
			Role           string `json:"role"`
			ScopeKind      string `json:"scope_kind"`
			ScopeID        uint64 `json:"scope_id"`
			Status         string `json:"status"`
			Enabled        bool   `json:"enabled"`
			ReleaseVersion string `json:"release_version"`
		} `json:"clients"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "JSON 必须可解析；实际输出：%s", buf.String())
	assert.Equal(t, 1, got.Count)
	require.Len(t, got.Clients, 1)
	assert.Equal(t, "wx_teacher_diag", got.Clients[0].AppID)
	assert.Equal(t, "teacher", got.Clients[0].Role)
	assert.Equal(t, "shared", got.Clients[0].ScopeKind)
	assert.Equal(t, uint64(0), got.Clients[0].ScopeID)
	assert.Equal(t, "enabled", got.Clients[0].Status)
	assert.True(t, got.Clients[0].Enabled)
	assert.Equal(t, "1.4.2", got.Clients[0].ReleaseVersion)
}

// 空注册表输出 []（不是 null），脚本才能统一按数组处理。
func TestWriteJSONEmptyRegistryUsesEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, clientdiag.WriteJSON(&buf, nil))
	assert.JSONEq(t, `{"count":0,"clients":[]}`, buf.String())
}
