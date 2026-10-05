package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"kindergarten-server/internal/domain"
)

// NOTE: Mock structs (mockAuthUserRepo, mockAuthRoleRepo) are defined but not used
// in this phase — they will be used in Phase 2 when Redis interface is extracted
// and login flow tests are enabled.

// --- tests ---
//
// NOTE: PasswordLogin/SMSLogin/WeChatLogin tests require Redis mock (the auth.SaveRefresh
// call needs *redis.Client). Those will be added in Phase 2 when Redis interface is extracted.
// For now, we test all pure business logic functions exhaustively.

func TestMaskPhone(t *testing.T) {
	assert.Equal(t, "138****8000", maskPhone("13800138000"))
	assert.Equal(t, "13", maskPhone("13"))           // short phone
	assert.Equal(t, "130****0001", maskPhone("13000000001"))
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("test123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// verify hash can be compared
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("test123"))
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong"))
	assert.Error(t, err)
}

func TestValidateGrade(t *testing.T) {
	assert.NoError(t, validateGrade("小班"))
	assert.NoError(t, validateGrade("中班"))
	assert.NoError(t, validateGrade("大班"))
	assert.Error(t, validateGrade("托班"))
	assert.Error(t, validateGrade("学前班"))
	assert.Error(t, validateGrade(""))
}

func TestTitleLabel(t *testing.T) {
	assert.Equal(t, "主班老师", titleLabel(domain.TeacherTitleHead))
	assert.Equal(t, "副班老师", titleLabel(domain.TeacherTitleAssistant))
	assert.Equal(t, "生活老师", titleLabel(domain.TeacherTitleNursery))
}

func TestParseTeacherTitle(t *testing.T) {
	tt, err := parseTeacherTitle("head")
	assert.NoError(t, err)
	assert.Equal(t, domain.TeacherTitleHead, tt)

	tt, err = parseTeacherTitle("assistant")
	assert.NoError(t, err)
	assert.Equal(t, domain.TeacherTitleAssistant, tt)

	_, err = parseTeacherTitle("invalid")
	assert.Error(t, err)
}

func TestRoleNamesFromUserRoles(t *testing.T) {
	roles := []domain.UserRole{
		{Role: "teacher"}, {Role: "teacher"}, {Role: "admin"},
	}
	names := roleNamesFromUserRoles(roles)
	assert.Len(t, names, 2) // deduplicated
	assert.Contains(t, names, "teacher")
	assert.Contains(t, names, "admin")
}

func TestRoleNamesFromUserRoles_EmptyDefaultsToParent(t *testing.T) {
	names := roleNamesFromUserRoles([]domain.UserRole{})
	assert.Len(t, names, 1)
	assert.Equal(t, "parent", names[0])
}

func TestKindergartenIDFromUserRoles_AdminPreferred(t *testing.T) {
	kg1 := uint64(1)
	kg2 := uint64(2)
	roles := []domain.UserRole{
		{Role: "teacher", KindergartenID: &kg1},
		{Role: "admin", KindergartenID: &kg2},
	}
	assert.Equal(t, uint64(2), kindergartenIDFromUserRoles(roles))
}

func TestKindergartenIDFromUserRoles_NoAdmin(t *testing.T) {
	kg := uint64(3)
	roles := []domain.UserRole{
		{Role: "teacher", KindergartenID: &kg},
	}
	assert.Equal(t, uint64(3), kindergartenIDFromUserRoles(roles))
}

func TestKindergartenIDFromUserRoles_Empty(t *testing.T) {
	assert.Equal(t, uint64(0), kindergartenIDFromUserRoles([]domain.UserRole{}))
}
