package testutil

import (
	"time"

	"kindergarten-server/internal/domain"
)

// NewTestClass creates a class entity for tests with sensible defaults.
func NewTestClass(kgID uint64, overrides ...func(*domain.Class)) *domain.Class {
	c := &domain.Class{
		KindergartenID: kgID,
		Name:           "测试班级",
		Grade:          "小班",
		AcademicYear:   "2025-2026",
		Capacity:       30,
		Status:         1,
	}
	c.ID = 1
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	for _, o := range overrides {
		o(c)
	}
	return c
}

// NewTestUser creates a user entity for tests.
func NewTestUser(overrides ...func(*domain.User)) *domain.User {
	u := &domain.User{
		Phone:        "13800000001",
		Nickname:     "测试用户",
		PasswordHash: "$2a$10$dummyhashedpassword",
	}
	u.ID = 1
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	for _, o := range overrides {
		o(u)
	}
	return u
}

// NewTestStudent creates a student entity for tests.
func NewTestStudent(kgID, classID uint64, overrides ...func(*domain.Student)) *domain.Student {
	s := &domain.Student{
		KindergartenID: kgID,
		ClassID:        classID,
		Name:           "测试学生",
		Status:         1,
	}
	s.ID = 1
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	for _, o := range overrides {
		o(s)
	}
	return s
}

// NewTestUserRole creates a user_role entity for tests.
func NewTestUserRole(userID uint64, role string, kgID uint64, overrides ...func(*domain.UserRole)) *domain.UserRole {
	r := &domain.UserRole{
		UserID:         userID,
		Role:           role,
		KindergartenID: &kgID,
	}
	for _, o := range overrides {
		o(r)
	}
	return r
}

// NewTestOperationLog creates an operation log for tests.
func NewTestOperationLog(kgID, userID, targetID uint64, module, action string) *domain.OperationLog {
	return &domain.OperationLog{
		KindergartenID: kgID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     module,
		TargetID:       targetID,
		Detail:         `{"test": true}`,
	}
}
