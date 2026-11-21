package repository

import (
	"context"
	"errors"
	"fmt"

	mysqldrv "github.com/go-sql-driver/mysql"
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
		if isDuplicateKeyError(err) {
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

// Delete 物理删除一条身份。没有软删除，删除后唯一键立即释放。
func (r *UserIdentityRepo) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.New("userIdentity id is required")
	}
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.UserIdentity{}).Error
}

// isDuplicateKeyError MySQL 1062（ER_DUP_ENTRY）。唯一索引是并发安全性的最终依据：
// 解析器靠它判断「另一个人刚抢先建好了」。
func isDuplicateKeyError(err error) bool {
	var me *mysqldrv.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
