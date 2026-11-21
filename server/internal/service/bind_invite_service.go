package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 老师邀请绑定（#168）。
//
// 方向与家长绑定一致（#165）：登录产生的「待绑定账号」并入园所预创建的教师
// 账号，保留预创建账号的角色、带班关系、档案与幼儿关联，待绑定账号硬删除。
// 与旧实现的关键差别：
//   - 身份只写 user_identities（users.open_id/union_id 已在 #173 删除）；
//   - 邀请所属园所的通道必须与当前小程序客户端匹配（#177）；
//   - 绑定成功后立刻换发预创建账号的老师端凭证。
var (
	// ErrBindInviteInvalid 邀请码不存在、已过期，或邀请指向的园所/教师已不可用。
	ErrBindInviteInvalid = errors.New("INVITE_INVALID")
	// ErrBindInviteUsed 邀请码已被使用。
	ErrBindInviteUsed = errors.New("INVITE_USED")
	// ErrBindInviteTeacherOnly 只有老师端客户端可以使用邀请码绑定。
	ErrBindInviteTeacherOnly = errors.New("INVITE_CLIENT_ROLE")
	// ErrBindInviteChannelMismatch 邀请所属园所的通道与当前客户端作用域不匹配。
	ErrBindInviteChannelMismatch = errors.New("INVITE_CLIENT_SCOPE")
	// ErrBindInviteAlreadyBound 预创建账号在同一客户端已绑定其他微信。
	ErrBindInviteAlreadyBound = errors.New("INVITE_PHONE_BOUND")
)

// BindInviteRequest 由 HTTP 层从待绑定 token 与请求体组装。
type BindInviteRequest struct {
	// UserID/AppID 来自待绑定 token；不接受请求体自报，避免越权绑定。
	UserID     uint64
	AppID      string
	InviteCode string
}

type BindInviteService struct {
	invitations   *repository.TeacherInvitationRepo
	kindergartens *repository.KindergartenRepo
	identities    *repository.UserIdentityRepo
	users         *repository.UserRepo
	clients       *ClientService
	merge         *IdentityMergeService
	auth          *AuthService
}

func NewBindInviteService(
	invitations *repository.TeacherInvitationRepo,
	kindergartens *repository.KindergartenRepo,
	identities *repository.UserIdentityRepo,
	users *repository.UserRepo,
	clients *ClientService,
	merge *IdentityMergeService,
	auth *AuthService,
) *BindInviteService {
	return &BindInviteService{
		invitations:   invitations,
		kindergartens: kindergartens,
		identities:    identities,
		users:         users,
		clients:       clients,
		merge:         merge,
		auth:          auth,
	}
}

// Bind 把当前待绑定身份并入邀请指向的预创建教师账号，并返回该账号的凭证。
func (s *BindInviteService) Bind(ctx context.Context, req BindInviteRequest) (*LoginResult, error) {
	if req.UserID == 0 || req.AppID == "" {
		return nil, fmt.Errorf("%w: missing client claims", ErrBindInviteInvalid)
	}
	inviteCode := strings.TrimSpace(req.InviteCode)
	if inviteCode == "" {
		return nil, fmt.Errorf("%w: empty invite code", ErrBindInviteInvalid)
	}

	client, err := s.clients.ResolveEnabledByAppID(ctx, req.AppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientUnavailable, err)
	}
	if client.Role != domain.ClientRoleTeacher {
		return nil, fmt.Errorf("%w: role=%s", ErrBindInviteTeacherOnly, client.Role)
	}

	invitation, err := s.invitations.FindByToken(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("load teacher invitation: %w", err)
	}
	if invitation == nil {
		return nil, ErrBindInviteInvalid
	}
	if invitation.Status == domain.TeacherInvitationUsed {
		return nil, ErrBindInviteUsed
	}
	if invitation.Status != domain.TeacherInvitationPending {
		return nil, ErrBindInviteInvalid
	}
	if !invitation.ExpiresAt.After(time.Now()) {
		return nil, ErrBindInviteInvalid
	}

	kg, err := s.kindergartens.FindByID(ctx, invitation.KindergartenID)
	if err != nil {
		return nil, fmt.Errorf("load invitation kindergarten: %w", err)
	}
	if kg == nil {
		return nil, ErrBindInviteInvalid
	}
	// 通道不匹配时一律拒绝：shared 老师端不能把身份绑进专属园所（#177）。
	if !client.ServesKindergarten(kg) {
		return nil, fmt.Errorf("%w: app=%s scope=%s kg=%d channel=%s",
			ErrBindInviteChannelMismatch, client.AppID, client.ScopeKind, kg.ID, kg.AppChannel)
	}

	target, err := s.users.FindByID(ctx, invitation.UserID)
	if err != nil {
		return nil, fmt.Errorf("load pre-created teacher: %w", err)
	}
	if target == nil {
		return nil, ErrBindInviteInvalid
	}

	identity, err := s.identities.FindByUserAndApp(ctx, req.UserID, req.AppID)
	if err != nil {
		return nil, fmt.Errorf("load pending identity: %w", err)
	}
	if identity == nil {
		return nil, fmt.Errorf("%w: no client identity for user", ErrBindInviteInvalid)
	}

	if identity.UserID != target.ID {
		_, err = s.merge.Merge(ctx, IdentityMergeRequest{
			AppID:        req.AppID,
			OpenID:       identity.OpenID,
			TargetUserID: target.ID,
		})
		switch {
		case errors.Is(err, ErrIdentityMergePhoneAlreadyBound):
			return nil, ErrBindInviteAlreadyBound
		case errors.Is(err, ErrIdentityMergeIdentityNotFound), errors.Is(err, ErrIdentityMergeTargetNotFound):
			return nil, fmt.Errorf("%w: %v", ErrBindInviteInvalid, err)
		case err != nil:
			return nil, fmt.Errorf("merge pending identity: %w", err)
		}
	}

	if err := s.invitations.MarkUsed(ctx, invitation.ID, target.ID, req.AppID, identity.OpenID); err != nil {
		if errors.Is(err, repository.ErrTeacherInvitationNotPending) {
			return nil, ErrBindInviteUsed
		}
		return nil, fmt.Errorf("mark invitation used: %w", err)
	}

	result, err := s.auth.IssueClientLogin(ctx, target.ID, client)
	if err != nil {
		return nil, fmt.Errorf("issue bound teacher login: %w", err)
	}
	return result, nil
}
