package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ErrUserIdentityExists 客户端身份与已有唯一键冲突。
//
// 调用方必须把它当作业务冲突处理（换绑/合并流程 #165、#167），
// 不能像 upsert 那样覆盖原行：覆盖会把身份从原账号悄悄挪走。
var ErrUserIdentityExists = errors.New("USER_IDENTITY_EXISTS")

// UserIdentityRepo 客户端身份仓储（#164）。
//
// 本表没有软删除，Delete 即物理删除——见 domain.UserIdentity 的说明。
type UserIdentityRepo struct {
	db *gorm.DB
}

func NewUserIdentityRepo(db *gorm.DB) *UserIdentityRepo {
	return &UserIdentityRepo{db: db}
}

// WithDB 返回绑定到指定事务的仓储副本，供合并/换绑在调用方事务内执行。
func (r *UserIdentityRepo) WithDB(db *gorm.DB) *UserIdentityRepo {
	return &UserIdentityRepo{db: db}
}

func (r *UserIdentityRepo) Create(ctx context.Context, identity *domain.UserIdentity) error {
	if err := r.db.WithContext(ctx).Create(identity).Error; err != nil {
		if IsDuplicateKey(err) {
			return fmt.Errorf("%w: app_id=%s", ErrUserIdentityExists, identity.AppID)
		}
		return err
	}
	return nil
}

// FindByAppAndOpenID 未命中返回 (nil, nil)，由调用方决定是否新建身份。
func (r *UserIdentityRepo) FindByAppAndOpenID(ctx context.Context, appID, openID string) (*domain.UserIdentity, error) {
	if appID == "" || openID == "" {
		return nil, nil
	}
	var row domain.UserIdentity
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND open_id = ?", appID, openID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindByUserAndApp 查该账号在该客户端的身份；未命中返回 (nil, nil)。
func (r *UserIdentityRepo) FindByUserAndApp(ctx context.Context, userID uint64, appID string) (*domain.UserIdentity, error) {
	if userID == 0 || appID == "" {
		return nil, nil
	}
	var row domain.UserIdentity
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND app_id = ?", userID, appID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListByUserID 列出账号的全部客户端身份，按 id 升序保证输出稳定。
func (r *UserIdentityRepo) ListByUserID(ctx context.Context, userID uint64) ([]domain.UserIdentity, error) {
	var rows []domain.UserIdentity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindOrCreatePendingUser 按 (app_id, open_id) 返回账号；没有身份时在同一个事务里
// 创建「待绑定账号 + 身份」（#175）。待绑定账号手机号为空、无角色、不属于任何园所。
//
// 并发首登由 uk_user_identities_app_open 兜底：输掉竞争的事务整体回滚（连同它建的
// 空账号），再按唯一键读回赢家的账号，因此不会留下两条账号或两条身份。
func (r *UserIdentityRepo) FindOrCreatePendingUser(ctx context.Context, appID, openID, unionID string) (*domain.User, error) {
	if appID == "" || openID == "" {
		return nil, errors.New("userIdentity app_id/open_id is required")
	}

	if existing, err := r.FindByAppAndOpenID(ctx, appID, openID); err != nil {
		return nil, err
	} else if existing != nil {
		return r.loadUser(ctx, existing.UserID)
	}

	var user domain.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var identity domain.UserIdentity
		lookup := tx.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error
		switch {
		case lookup == nil:
			return tx.First(&user, identity.UserID).Error
		case !errors.Is(lookup, gorm.ErrRecordNotFound):
			return lookup
		}

		user = domain.User{}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return tx.Create(&domain.UserIdentity{
			UserID:  user.ID,
			AppID:   appID,
			OpenID:  openID,
			UnionID: unionID,
		}).Error
	})
	if err == nil {
		return &user, nil
	}

	// 唯一键冲突说明另一个并发首登已建好身份；读回它的账号，不覆盖、不重试建号。
	existing, findErr := r.FindByAppAndOpenID(ctx, appID, openID)
	if findErr == nil && existing != nil {
		return r.loadUser(ctx, existing.UserID)
	}
	return nil, err
}

func (r *UserIdentityRepo) loadUser(ctx context.Context, userID uint64) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("身份指向的账号 %d 不存在或已删除", userID)
		}
		return nil, err
	}
	return &user, nil
}

// Delete 物理删除一条身份。没有软删除，删除后唯一键立即释放。
func (r *UserIdentityRepo) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.New("userIdentity id is required")
	}
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.UserIdentity{}).Error
}
