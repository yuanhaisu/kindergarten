package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// openspec class-album-publish-notice（Phase 2-B）：
// TC-NEW-ALBMSG-0001..0006。复用 teacher_album_test.go 的 9960xx 夹具；
// 通知写入 messages 表，清理时一并硬删。

func talPublishNotifyAlbum(t *testing.T, f *talFixture, svc *TeacherPortalService) uint64 {
	t.Helper()
	photos := []TeacherAlbumPhotoInput{
		{FileID: testTALFileA, StudentIDs: []uint64{testTALStudentA}},
		{FileID: testTALFileB, StudentIDs: []uint64{testTALStudentOpt}},
	}
	data, err := svc.CreateAlbum(context.Background(), f.tcA, TeacherAlbumInput{
		Title: "通知测试相册", Category: "daily", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA, testTALStudentOpt},
		Photos:       photos,
	})
	require.NoError(t, err)
	id, ok := data["id"].(uint64)
	require.True(t, ok)
	_, err = svc.PublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	return id
}

func talAlbumMessages(t *testing.T, f *talFixture, albumID uint64) []domain.Message {
	t.Helper()
	var msgs []domain.Message
	require.NoError(t, f.db.Where("type = ? AND reference_id = ?", string(domain.MessageTypeAlbum), albumID).
		Order("user_id ASC").Find(&msgs).Error)
	return msgs
}

// TC-NEW-ALBMSG-0001 / 0003 / 0004 / 0005：
// 只发给参与者监护人；本人 opt-out 不收；同一孩子另一位监护人照收；
// 非参与者家庭不收；一个家长关联多个参与孩子只收一条。
func TestAlbumPublishNotifiesEligibleGuardians(t *testing.T) {
	f := newTALFixture(t)
	albumID := talPublishNotifyAlbum(t, f, f.svc)

	msgs := talAlbumMessages(t, f, albumID)
	require.Len(t, msgs, 2, "只有 parentA 与 parentOpt2 应收")

	byUser := map[uint64]domain.Message{}
	for _, m := range msgs {
		byUser[m.UserID] = m
	}
	msgA, ok := byUser[testTALParentA]
	require.True(t, ok, "参与者 A 的家长必须收到")
	assert.Equal(t, string(domain.MessageTypeAlbum), msgA.Type)
	assert.Equal(t, albumNotifyTitle, msgA.Title, "固定标题是幂等键的一部分")
	assert.Contains(t, msgA.Content, "通知测试相册")
	assert.False(t, msgA.IsRead)
	assert.Equal(t, fmt.Sprintf("/pages/album/detail?id=%d&student_id=%d", albumID, testTALStudentA), msgA.ActionPath)

	msgOpt2, ok := byUser[testTALParentOpt2]
	require.True(t, ok, "同一孩子的另一位监护人（默认允许）必须收到")
	assert.Equal(t, fmt.Sprintf("/pages/album/detail?id=%d&student_id=%d", albumID, testTALStudentOpt), msgOpt2.ActionPath)

	_, optGot := byUser[testTALParentOpt]
	assert.False(t, optGot, "本人关闭班内共享不应收到")
	_, outGot := byUser[testTALParentOut]
	assert.False(t, outGot, "非参与者家庭不应收到")
}

// TC-NEW-ALBMSG-0006：下架后重新发布不重复发。
func TestAlbumRepublishDoesNotDuplicateMessage(t *testing.T) {
	f := newTALFixture(t)
	albumID := talPublishNotifyAlbum(t, f, f.svc)
	require.Len(t, talAlbumMessages(t, f, albumID), 2)

	_, err := f.svc.UnpublishAlbum(context.Background(), f.tcA, albumID)
	require.NoError(t, err)
	_, err = f.svc.PublishAlbum(context.Background(), f.tcA, albumID)
	require.NoError(t, err)

	msgs := talAlbumMessages(t, f, albumID)
	require.Len(t, msgs, 2, "重新发布不能新增消息")
}

// TC-NEW-ALBMSG-0002：消息仓库未接线时发布仍成功。
func TestAlbumPublishWithoutMessageRepoStillSucceeds(t *testing.T) {
	f := newTALFixture(t)
	svc := NewTeacherPortalService(f.db, nil, nil, nil, nil, nil, config.FaceConfig{})
	albumID := talPublishNotifyAlbum(t, f, svc)

	var album domain.Album
	require.NoError(t, f.db.Where("id = ?", albumID).First(&album).Error)
	assert.Equal(t, domain.AlbumStatusPublished, album.Status)
	assert.Empty(t, talAlbumMessages(t, f, albumID))
}
