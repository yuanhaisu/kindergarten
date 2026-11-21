package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 客户端身份支持操作（#172）：园所管理员或平台支持给家长「换微信」与
// 「更正录入手机号」。
//
// 两条铁律：
//   - 解绑只释放 user_identities 里的一行（物理删除，释放唯一键），
//     绝不删账号、角色、幼儿关联、积分或个人档案；
//   - 更正手机号必须先解绑错误身份，再改手机号，由正确家长重新绑定；
//     不静默删除或合并账号。

var (
	// ErrSupportUserNotFound 目标账号不存在，或不在调用者的可见范围（跨园）。
	ErrSupportUserNotFound = errors.New("SUPPORT_USER_NOT_FOUND")
	// ErrSupportIdentityNotFound 身份不存在，或不属于目标账号。
	ErrSupportIdentityNotFound = errors.New("SUPPORT_IDENTITY_NOT_FOUND")
	// ErrSupportKindergartenRequired 园所管理员操作必须带园所上下文。
	ErrSupportKindergartenRequired = errors.New("SUPPORT_KINDERGARTEN_REQUIRED")
	// ErrSupportInvalidPhone 更正的手机号格式不合法。
	ErrSupportInvalidPhone = errors.New("SUPPORT_INVALID_PHONE")
	// ErrSupportIdentityRemains 更正手机号前必须先解绑账号上的客户端身份，
	// 否则录错号对应的微信会随着新号继续拥有访问权。
	ErrSupportIdentityRemains = errors.New("SUPPORT_IDENTITY_REMAINS")
	// ErrSupportPhoneConflict 新手机号已被另一个活跃账号占用。
	ErrSupportPhoneConflict = errors.New("SUPPORT_PHONE_CONFLICT")
)

var supportPhoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

// SupportActorKind 区分两个独立的信任域：园所管理端与平台控制台。
// 审计落两张表，token 也永不混用。
type SupportActorKind string

const (
	SupportActorAdmin    SupportActorKind = "admin"
	SupportActorPlatform SupportActorKind = "platform"
)

// SupportActor 是审计所需的操作者上下文。
type SupportActor struct {
	Kind             SupportActorKind
	AdminUserID      uint64
	PlatformOperator string
	KindergartenID   uint64
	IP               string
	UserAgent        string
}

// SupportIdentity 支持视图里的客户端身份。openid 是微信侧标识，
// 界面只需要能区分「哪个微信」，因此只回掩码。
type SupportIdentity struct {
	ID           uint64 `json:"id"`
	AppID        string `json:"app_id"`
	OpenIDMasked string `json:"open_id_masked"`
	CreatedAt    string `json:"created_at"`
}

// UnbindIdentityResult 解绑结果：目标账号与剩余身份，便于界面原地刷新。
type UnbindIdentityResult struct {
	UserID            uint64            `json:"user_id"`
	UnboundIdentityID uint64            `json:"unbound_identity_id"`
	Identities        []SupportIdentity `json:"identities"`
}

// SupportMembership 目标账号在某园的成员关系。
type SupportMembership struct {
	KindergartenID   uint64 `json:"kindergarten_id"`
	KindergartenName string `json:"kindergarten_name"`
	Role             string `json:"role"`
	ClassID          uint64 `json:"class_id,omitempty"`
	ClassName        string `json:"class_name,omitempty"`
	Title            string `json:"title,omitempty"`
	TitleLabel       string `json:"title_label,omitempty"`
}

// SupportUserView 是支持操作的读取模型：账号基本信息 + 客户端身份（掩码）+
// 园所成员关系。解绑与更正手机号都返回它，界面无需二次请求。
type SupportUserView struct {
	UserID      uint64              `json:"user_id"`
	Nickname    string              `json:"nickname"`
	Phone       string              `json:"phone"`
	Identities  []SupportIdentity   `json:"identities"`
	Memberships []SupportMembership `json:"memberships"`
}

// IdentitySupportService 身份支持操作。
type IdentitySupportService struct {
	db           *gorm.DB
	users        *repository.UserRepo
	identities   *repository.UserIdentityRepo
	roles        *repository.UserRoleRepo
	kgs          *repository.KindergartenRepo
	oplogs       *repository.OperationLogRepo
	platformLogs *repository.PlatformLogRepo
}

func NewIdentitySupportService(
	db *gorm.DB,
	users *repository.UserRepo,
	identities *repository.UserIdentityRepo,
	roles *repository.UserRoleRepo,
	kgs *repository.KindergartenRepo,
	oplogs *repository.OperationLogRepo,
	platformLogs *repository.PlatformLogRepo,
) *IdentitySupportService {
	return &IdentitySupportService{
		db:           db,
		users:        users,
		identities:   identities,
		roles:        roles,
		kgs:          kgs,
		oplogs:       oplogs,
		platformLogs: platformLogs,
	}
}

// UnbindIdentity 把指定身份从账号上解绑。
//
// 只删 user_identities 行本身；删除与审计在同一事务里，审计写失败则解绑回滚。
func (s *IdentitySupportService) UnbindIdentity(
	ctx context.Context,
	actor SupportActor,
	userID, identityID uint64,
) (*UnbindIdentityResult, error) {
	if userID == 0 || identityID == 0 {
		return nil, ErrSupportIdentityNotFound
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrSupportUserNotFound
	}
	if err := s.ensureUserVisible(ctx, actor, userID); err != nil {
		return nil, err
	}

	var unbound domain.UserIdentity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 行锁 + 归属复核：并发下另一个请求可能刚把身份挪走或删掉。
		var row domain.UserIdentity
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, identityID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSupportIdentityNotFound
		}
		if err != nil {
			return err
		}
		if row.UserID != userID {
			return ErrSupportIdentityNotFound
		}
		if err := tx.Delete(&domain.UserIdentity{}, row.ID).Error; err != nil {
			return err
		}
		unbound = row
		return s.writeAuditTx(ctx, tx, actor, "unbind_identity", "user_identity", row.ID,
			fmt.Sprintf("app_id=%s open_id=%s", row.AppID, maskOpenID(row.OpenID)))
	})
	if err != nil {
		return nil, err
	}

	remaining, err := s.identities.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UnbindIdentityResult{
		UserID:            userID,
		UnboundIdentityID: unbound.ID,
		Identities:        supportIdentities(remaining),
	}, nil
}

// ViewUser 读取目标账号的支持视图。园所管理员只能看到本园的成员关系，
// 平台支持能看到全部园所。
func (s *IdentitySupportService) ViewUser(
	ctx context.Context,
	actor SupportActor,
	userID uint64,
) (*SupportUserView, error) {
	if userID == 0 {
		return nil, ErrSupportUserNotFound
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrSupportUserNotFound
	}
	if err := s.ensureUserVisible(ctx, actor, userID); err != nil {
		return nil, err
	}

	identities, err := s.identities.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles, err := s.roles.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	memberships, err := s.supportMemberships(ctx, roles, actor)
	if err != nil {
		return nil, err
	}
	return &SupportUserView{
		UserID:      user.ID,
		Nickname:    user.Nickname,
		Phone:       user.Phone,
		Identities:  supportIdentities(identities),
		Memberships: memberships,
	}, nil
}

// CorrectPhone 更正预创建账号的手机号。
//
// 顺序是硬性的：账号上还有客户端身份时拒绝更正（先解绑错误身份），
// 新号被其他活跃账号占用时拒绝。手机号更新与审计同事务，任一步失败整体回滚，
// 不会出现「号改了但审计没写」或半截状态。
func (s *IdentitySupportService) CorrectPhone(
	ctx context.Context,
	actor SupportActor,
	userID uint64,
	newPhone string,
) (*SupportUserView, error) {
	newPhone = strings.TrimSpace(newPhone)
	if userID == 0 {
		return nil, ErrSupportUserNotFound
	}
	if !supportPhoneRe.MatchString(newPhone) {
		return nil, ErrSupportInvalidPhone
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrSupportUserNotFound
	}
	if err := s.ensureUserVisible(ctx, actor, userID); err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.User
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, userID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSupportUserNotFound
		}
		if err != nil {
			return err
		}

		var identityCount int64
		if err := tx.Model(&domain.UserIdentity{}).
			Where("user_id = ?", userID).Count(&identityCount).Error; err != nil {
			return err
		}
		if identityCount > 0 {
			return ErrSupportIdentityRemains
		}

		var occupied domain.User
		err = tx.Where("phone_active = ? AND id <> ?", newPhone, userID).First(&occupied).Error
		if err == nil {
			return ErrSupportPhoneConflict
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&domain.User{}).Where("id = ?", userID).
			Update("phone", newPhone).Error; err != nil {
			// 并发下预检查可能同时通过，唯一索引是最终裁决。
			if repository.IsDuplicateKey(err) {
				return ErrSupportPhoneConflict
			}
			return err
		}
		return s.writeAuditTx(ctx, tx, actor, "correct_phone", "user", userID,
			fmt.Sprintf("phone %s -> %s", row.Phone, newPhone))
	})
	if err != nil {
		return nil, err
	}
	return s.ViewUser(ctx, actor, userID)
}

// supportMemberships 把角色行映射成成员关系。园所管理员只看本园；
// 园所名尽量取角色行上的快照，缺失时回查园所。
func (s *IdentitySupportService) supportMemberships(
	ctx context.Context,
	roles []domain.UserRole,
	actor SupportActor,
) ([]SupportMembership, error) {
	out := make([]SupportMembership, 0, len(roles))
	names := make(map[uint64]string, len(roles))
	for _, ur := range roles {
		if ur.KindergartenID == nil {
			// 园所级成员关系才有「在哪个园」可言；平台级角色不在此列出。
			continue
		}
		kgID := *ur.KindergartenID
		if actor.Kind == SupportActorAdmin && kgID != actor.KindergartenID {
			continue
		}
		name := ur.KindergartenName
		if name == "" {
			name = names[kgID]
		}
		if name == "" {
			kg, err := s.kgs.FindByID(ctx, kgID)
			if err != nil {
				return nil, err
			}
			if kg != nil {
				name = kg.Name
				names[kgID] = name
			}
		}
		m := SupportMembership{
			KindergartenID:   kgID,
			KindergartenName: name,
			Role:             ur.Role,
			ClassName:        ur.ClassName,
			Title:            ur.Title,
			TitleLabel:       ur.TitleLabel,
		}
		if ur.ClassID != nil {
			m.ClassID = *ur.ClassID
		}
		out = append(out, m)
	}
	return out, nil
}

// ensureUserVisible 园所管理员只能操作本园的家长；跨园与不存在同样返回
// 「账号不存在」，不泄露其他园所的用户是否存在。平台支持不受园所限制。
func (s *IdentitySupportService) ensureUserVisible(ctx context.Context, actor SupportActor, userID uint64) error {
	if actor.Kind == SupportActorPlatform {
		return nil
	}
	if actor.Kind != SupportActorAdmin || actor.KindergartenID == 0 {
		return ErrSupportKindergartenRequired
	}
	role, err := s.roles.FindParentInKindergarten(ctx, userID, actor.KindergartenID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrSupportUserNotFound
	}
	return nil
}

// writeAuditTx 按信任域写各自的审计表，并与业务写入同事务。
func (s *IdentitySupportService) writeAuditTx(
	ctx context.Context,
	tx *gorm.DB,
	actor SupportActor,
	action, targetType string,
	targetID uint64,
	detail string,
) error {
	if actor.Kind == SupportActorPlatform {
		if s.platformLogs == nil {
			return errors.New("platform operation log repo not configured")
		}
		return s.platformLogs.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator:   actor.PlatformOperator,
			Module:     "support",
			Action:     action,
			TargetType: targetType,
			TargetID:   targetID,
			Detail:     detail,
			IP:         actor.IP,
		})
	}
	if s.oplogs == nil {
		return errors.New("operation log repo not configured")
	}
	return s.oplogs.CreateTx(ctx, tx, &domain.OperationLog{
		KindergartenID: actor.KindergartenID,
		UserID:         actor.AdminUserID,
		Module:         "parents",
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             actor.IP,
		UserAgent:      truncateUAParent(actor.UserAgent),
	})
}

func supportIdentities(rows []domain.UserIdentity) []SupportIdentity {
	out := make([]SupportIdentity, 0, len(rows))
	for _, row := range rows {
		out = append(out, SupportIdentity{
			ID:           row.ID,
			AppID:        row.AppID,
			OpenIDMasked: maskOpenID(row.OpenID),
			CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// maskOpenID 保留前 4 后 4。openid 通常 28 位，足以区分不同微信。
func maskOpenID(openID string) string {
	if len(openID) <= 8 {
		if openID == "" {
			return ""
		}
		return "****"
	}
	return openID[:4] + "****" + openID[len(openID)-4:]
}
