package service

import (
	"context"
	"fmt"

	"kindergarten-server/internal/domain"
)

// 跨端同人判定。
//
// 同一自然人在家长端与老师端是**两条独立的 User**（openid 是「用户×小程序」二元的，
// 两端 AppID 不同必然得到不同 openid）。判定二者是不是同一个人靠两个键：
//
//	unionid —— 首选。两侧都非空就以它为准。生产上两端绑同一微信开放平台后可用。
//	手机号  —— 兜底。unionid 缺失时（如某端是个人主体小程序，微信不下发 unionid）才比。
//
// 判定只回答「是不是同一个人」，绝不合并账号、不迁移角色或孩子绑定关系
//（见 specs/cross-end-identity「关联判定不修改账号归属」）。

// SamePerson 判定两条 User 是否同一自然人。
//
// 顺序：unionid 优先（两侧都非空才比），否则手机号兜底（两侧都非空才比），
// 两侧键都缺失时返回 false——「无法判定」按不同人处理，不猜测。
func SamePerson(a, b *domain.User) bool {
	if a == nil || b == nil {
		return false
	}

	// unionid 只要两边都非空就以它为准，不因手机号也一致而推翻
	if a.UnionID != "" && b.UnionID != "" {
		return a.UnionID == b.UnionID
	}

	// 至少一侧无 unionid（例如个人主体小程序拿不到），降级到手机号
	if a.Phone != "" && b.Phone != "" {
		return a.Phone == b.Phone
	}

	return false
}

// FindLinkedUser 查出同一自然人在另一端的账号，没有则返回 (nil, nil)。
//
// 先按 unionid 反查；unionid 缺失时按手机号反查。查找一律排除自己——
// 同一键下有两条 User，不排除就会命中自己那条、查不到另一端。
func (s *AuthService) FindLinkedUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user == nil {
		return nil, fmt.Errorf("FIND_LINKED_USER: user is nil")
	}

	// unionid 非空就只按它查，不退化到手机号——两端 unionid 不同就是不同人
	if user.UnionID != "" {
		return s.userRepo.FindOtherByUnionID(ctx, user.UnionID, user.ID)
	}

	if user.Phone == "" {
		return nil, nil
	}
	return s.userRepo.FindOtherByPhone(ctx, user.Phone, user.ID)
}
