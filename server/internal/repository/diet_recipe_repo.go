package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DietRecipeListRow 菜谱列表行：列表只返回名称、拼音码与配料数（契约 §四）。
type DietRecipeListRow struct {
	ID        uint64 `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	SpellCode string `gorm:"column:spell_code"`
	ItemCount int64  `gorm:"column:item_count"`
}

// DietRecipeRepo 菜谱库。菜谱名称唯一性由服务层保证，仓储不做唯一性判定
// （软删除行会占唯一索引，设计上不建唯一键）。
type DietRecipeRepo struct {
	db *gorm.DB
}

func NewDietRecipeRepo(db *gorm.DB) *DietRecipeRepo {
	return &DietRecipeRepo{db: db}
}

const dietRecipeListSelect = `r.id, r.name, r.spell_code,
	(SELECT COUNT(*) FROM diet_recipe_items i
		WHERE i.recipe_id = r.id AND i.deleted_at IS NULL) AS item_count`

func (r *DietRecipeRepo) listQuery(ctx context.Context, kindergartenID uint64, keyword string) *gorm.DB {
	q := r.db.WithContext(ctx).Table("diet_recipes AS r").
		Where("r.kindergarten_id = ? AND r.deleted_at IS NULL", kindergartenID)
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where("r.name LIKE ?", "%"+kw+"%")
	}
	return q
}

func (r *DietRecipeRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	offset, limit int,
) ([]DietRecipeListRow, int64, error) {
	q := r.listQuery(ctx, kindergartenID, keyword)

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []DietRecipeListRow
	err := q.Session(&gorm.Session{}).
		Select(dietRecipeListSelect).
		Order("r.name ASC, r.id ASC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

// Suggest 编辑器菜名联想：名称包含关键字，或拼音码前缀匹配（契约 §四 suggest）。
func (r *DietRecipeRepo) Suggest(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	limit int,
) ([]DietRecipeListRow, error) {
	q := r.db.WithContext(ctx).Table("diet_recipes AS r").
		Where("r.kindergarten_id = ? AND r.deleted_at IS NULL", kindergartenID)
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where(
			"(r.name LIKE ? OR r.spell_code LIKE ?)",
			"%"+kw+"%",
			strings.ToUpper(kw)+"%",
		)
	}

	var rows []DietRecipeListRow
	err := q.Select(dietRecipeListSelect).
		Order("r.name ASC, r.id ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *DietRecipeRepo) FindByIDWithItems(
	ctx context.Context,
	kindergartenID, id uint64,
) (*domain.Recipe, error) {
	var recipe domain.Recipe
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		First(&recipe).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []domain.RecipeItem
	if err := r.db.WithContext(ctx).
		Where("recipe_id = ? AND kindergarten_id = ?", recipe.ID, kindergartenID).
		Order("sort ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	recipe.Items = items
	return &recipe, nil
}

func (r *DietRecipeRepo) FindByName(
	ctx context.Context,
	kindergartenID uint64,
	name string,
) (*domain.Recipe, error) {
	var recipe domain.Recipe
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND name = ?", kindergartenID, name).
		First(&recipe).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &recipe, nil
}

// Create 在同一事务内写入菜谱与配料。
func (r *DietRecipeRepo) Create(ctx context.Context, recipe *domain.Recipe) error {
	if recipe == nil {
		return errors.New("diet recipe: nil recipe")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		items := recipe.Items
		if err := tx.Omit("Items").Create(recipe).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			recipe.Items = nil
			return nil
		}
		for i := range items {
			items[i].BaseModel = domain.BaseModel{}
			items[i].RecipeID = recipe.ID
			items[i].KindergartenID = recipe.KindergartenID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		recipe.Items = items
		return nil
	})
}

// Update 更新菜谱名称/拼音码并整组替换配料。
// 配料硬删后重插：RecipeItem 带软删除列，普通 Delete 会留下软删行。
func (r *DietRecipeRepo) Update(ctx context.Context, recipe *domain.Recipe) error {
	if recipe == nil {
		return errors.New("diet recipe: nil recipe")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Recipe{}).
			Where("id = ? AND kindergarten_id = ?", recipe.ID, recipe.KindergartenID).
			Updates(map[string]any{
				"name":       recipe.Name,
				"spell_code": recipe.SpellCode,
			}).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().
			Where("recipe_id = ? AND kindergarten_id = ?", recipe.ID, recipe.KindergartenID).
			Delete(&domain.RecipeItem{}).Error; err != nil {
			return err
		}
		if len(recipe.Items) == 0 {
			recipe.Items = nil
			return nil
		}

		items := make([]domain.RecipeItem, len(recipe.Items))
		copy(items, recipe.Items)
		for i := range items {
			items[i].BaseModel = domain.BaseModel{}
			items[i].RecipeID = recipe.ID
			items[i].KindergartenID = recipe.KindergartenID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		recipe.Items = items
		return nil
	})
}

// Delete 软删除菜谱并硬删其配料；历史周食谱不受影响（明细存自己的快照）。
func (r *DietRecipeRepo) Delete(ctx context.Context, kindergartenID, id uint64) (int64, error) {
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("recipe_id = ? AND kindergarten_id = ?", id, kindergartenID).
			Delete(&domain.RecipeItem{}).Error; err != nil {
			return err
		}
		res := tx.Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
			Delete(&domain.Recipe{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}
