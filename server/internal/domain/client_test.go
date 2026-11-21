package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 客户端作用域 ↔ 园所通道匹配矩阵（#177）。
//
// 矩阵是 #169（切换/会话只落到作用域内的园）与 #176（二维码按客户端作用域生成）
// 共用的判定，必须在这里把每个分支固定下来：
//
//	shared       → app_channel=shared 的园
//	kindergarten → scope_id 对应的园，且 app_channel=dedicated
//	group        → 需要集团实体与成员表；v1 没有，永不匹配
func TestClientServesKindergartenMatrix(t *testing.T) {
	cases := []struct {
		name      string
		client    Client
		kg        *Kindergarten
		wantMatch bool
	}{
		{
			name:      "shared 客户端服务 shared 园",
			client:    Client{ScopeKind: ClientScopeShared},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelShared},
			wantMatch: true,
		},
		{
			name:      "shared 客户端不能服务 dedicated 园",
			client:    Client{ScopeKind: ClientScopeShared},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelDedicated},
			wantMatch: false,
		},
		{
			name:      "shared 客户端不能服务 group 园",
			client:    Client{ScopeKind: ClientScopeShared},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelGroup},
			wantMatch: false,
		},
		{
			name:      "专属客户端服务 scope_id 对应且通道为 dedicated 的园",
			client:    Client{ScopeKind: ClientScopeKindergarten, ScopeID: 7},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelDedicated},
			wantMatch: true,
		},
		{
			name:      "专属客户端不能服务别的园（scope 锁定）",
			client:    Client{ScopeKind: ClientScopeKindergarten, ScopeID: 7},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 8}, AppChannel: AppChannelDedicated},
			wantMatch: false,
		},
		{
			name:      "专属客户端不能服务 shared 园",
			client:    Client{ScopeKind: ClientScopeKindergarten, ScopeID: 7},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelShared},
			wantMatch: false,
		},
		{
			name:      "scope_id=0 的专属客户端不匹配任何园",
			client:    Client{ScopeKind: ClientScopeKindergarten, ScopeID: 0},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 0}, AppChannel: AppChannelDedicated},
			wantMatch: false,
		},
		{
			name:      "group 客户端 v1 不可用：即使园所是 group 通道也不匹配",
			client:    Client{ScopeKind: ClientScopeGroup, ScopeID: 9},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelGroup},
			wantMatch: false,
		},
		{
			name:      "未知作用域不匹配",
			client:    Client{ScopeKind: "admin_web"},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}, AppChannel: AppChannelShared},
			wantMatch: false,
		},
		{
			name:      "空通道不能被 shared 客户端匹配",
			client:    Client{ScopeKind: ClientScopeShared},
			kg:        &Kindergarten{BaseModel: BaseModel{ID: 7}},
			wantMatch: false,
		},
		{
			name:      "nil 园所不匹配",
			client:    Client{ScopeKind: ClientScopeShared},
			kg:        nil,
			wantMatch: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantMatch, tc.client.ServesKindergarten(tc.kg))
		})
	}
}

func TestIsValidAppChannel(t *testing.T) {
	for _, valid := range []string{AppChannelShared, AppChannelDedicated, AppChannelGroup} {
		assert.True(t, IsValidAppChannel(valid), valid)
	}
	for _, invalid := range []string{"", "Shared", "SHARED", "dedicate", "grouped"} {
		assert.False(t, IsValidAppChannel(invalid), invalid)
	}
}
