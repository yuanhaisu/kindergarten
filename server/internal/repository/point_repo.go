package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ErrInsufficientPointBalance 扣减时可用积分不足。
var ErrInsufficientPointBalance = errors.New("INSUFFICIENT_POINT_BALANCE")

type PointRepo struct {
	db *gorm.DB
}

func NewPointRepo(db *gorm.DB) *PointRepo {
	return &PointRepo{db: db}
}

// SumAvailablePoints 园所内学生可用积分之和 Σ(total_points - used_points)。
func (r *PointRepo) SumAvailablePoints(ctx context.Context, kindergartenID uint64) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(SUM(pa.total_points - pa.used_points), 0)
FROM point_accounts pa
INNER JOIN students s ON s.id = pa.student_id AND s.deleted_at IS NULL
WHERE s.kindergarten_id = ?`, kindergartenID).Scan(&sum).Error
	return sum, err
}

func (r *PointRepo) SumPositiveTxInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(SUM(pt.points), 0)
FROM point_transactions pt
INNER JOIN students s ON s.id = pt.student_id AND s.deleted_at IS NULL
WHERE s.kindergarten_id = ?
  AND pt.points > 0
  AND pt.created_at >= ? AND pt.created_at < ?`,
		kindergartenID, from, to).Scan(&sum).Error
	return sum, err
}

func (r *PointRepo) SumNegativeTxAbsInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(SUM(-pt.points), 0)
FROM point_transactions pt
INNER JOIN students s ON s.id = pt.student_id AND s.deleted_at IS NULL
WHERE s.kindergarten_id = ?
  AND pt.points < 0
  AND pt.created_at >= ? AND pt.created_at < ?`,
		kindergartenID, from, to).Scan(&sum).Error
	return sum, err
}

type topEarnerRow struct {
	StudentID   uint64 `gorm:"column:student_id"`
	StudentName string `gorm:"column:student_name"`
	ClassName   string `gorm:"column:class_name"`
	Balance     int64  `gorm:"column:balance"`
}

func (r *PointRepo) TopEarnersByBalance(ctx context.Context, kindergartenID uint64, limit int) ([]topEarnerRow, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []topEarnerRow
	err := r.db.WithContext(ctx).Raw(`
SELECT s.id AS student_id, s.name AS student_name, c.name AS class_name,
       (pa.total_points - pa.used_points) AS balance
FROM point_accounts pa
INNER JOIN students s ON s.id = pa.student_id AND s.deleted_at IS NULL
INNER JOIN classes c ON c.id = s.class_id AND c.deleted_at IS NULL
WHERE s.kindergarten_id = ?
ORDER BY balance DESC, s.id ASC
LIMIT ?`, kindergartenID, limit).Scan(&rows).Error
	return rows, err
}

// PointTxListRow 流水列表行。
type PointTxListRow struct {
	ID          uint64    `gorm:"column:id"`
	StudentID   uint64    `gorm:"column:student_id"`
	StudentName string    `gorm:"column:student_name"`
	ClassName   string    `gorm:"column:class_name"`
	Points      int       `gorm:"column:points"`
	Source      string    `gorm:"column:source"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (r *PointRepo) baseTxListQuery(kindergartenID uint64, studentID *uint64, classID *uint64, typeEarn *bool) *gorm.DB {
	q := r.db.Table("point_transactions AS pt").
		Joins("INNER JOIN students AS s ON s.id = pt.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = s.class_id AND c.deleted_at IS NULL").
		Where("s.kindergarten_id = ?", kindergartenID)
	if studentID != nil && *studentID > 0 {
		q = q.Where("pt.student_id = ?", *studentID)
	}
	if classID != nil && *classID > 0 {
		q = q.Where("s.class_id = ?", *classID)
	}
	if typeEarn != nil {
		if *typeEarn {
			q = q.Where("pt.points > 0")
		} else {
			q = q.Where("pt.points < 0")
		}
	}
	return q
}

func (r *PointRepo) ListTransactions(
	ctx context.Context,
	kindergartenID uint64,
	studentID *uint64,
	classID *uint64,
	typeEarn *bool,
	offset, limit int,
) ([]PointTxListRow, int64, error) {
	base := r.baseTxListQuery(kindergartenID, studentID, classID, typeEarn)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`pt.id, pt.student_id, s.name AS student_name, c.name AS class_name,
			pt.points, pt.source, pt.description, pt.created_at`).
		Order("pt.created_at DESC, pt.id DESC").
		Offset(offset).Limit(limit)
	var rows []PointTxListRow
	err := sel.Scan(&rows).Error
	return rows, total, err
}

func (r *PointRepo) StudentExistsInKindergarten(ctx context.Context, kindergartenID, studentID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", studentID, kindergartenID).
		Count(&n).Error
	return n > 0, err
}

// ApplyAdjustTx 在事务内调整积分并写入流水；earn 增加 total_points，spend 增加 used_points。
func (r *PointRepo) ApplyAdjustTx(
	ctx context.Context,
	studentID uint64,
	earn bool,
	amount int,
	reason string,
) (txnID uint64, balance int, err error) {
	if amount <= 0 {
		return 0, 0, errors.New("BAD_POINT_AMOUNT")
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var acc domain.PointAccount
		e := tx.Where("student_id = ?", studentID).First(&acc).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			acc = domain.PointAccount{StudentID: studentID, TotalPoints: 0, UsedPoints: 0}
			if e2 := tx.Create(&acc).Error; e2 != nil {
				return e2
			}
		} else if e != nil {
			return e
		}
		newTotal := acc.TotalPoints
		newUsed := acc.UsedPoints
		var pts int
		if earn {
			newTotal = acc.TotalPoints + amount
			pts = amount
		} else {
			avail := acc.TotalPoints - acc.UsedPoints
			if avail < amount {
				return ErrInsufficientPointBalance
			}
			newUsed = acc.UsedPoints + amount
			pts = -amount
		}
		bal := newTotal - newUsed
		if err := tx.Model(&domain.PointAccount{}).Where("id = ?", acc.ID).
			Updates(map[string]interface{}{
				"total_points": newTotal,
				"used_points":  newUsed,
			}).Error; err != nil {
			return err
		}
		txn := domain.PointTransaction{
			StudentID:   studentID,
			Points:      pts,
			Balance:     bal,
			Source:      "admin_adjust",
			ReferenceID: 0,
			Description: reason,
		}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		txnID = txn.ID
		balance = bal
		return nil
	})
	return txnID, balance, err
}
