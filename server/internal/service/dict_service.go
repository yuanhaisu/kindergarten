package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

var (
	ErrDictTypeNotFound  = errors.New("DICT_TYPE_NOT_FOUND")
	ErrDictItemNotFound  = errors.New("DICT_ITEM_NOT_FOUND")
	ErrDictTypeKeyExists = errors.New("DICT_TYPE_KEY_EXISTS")
	ErrDictSystemType    = errors.New("DICT_SYSTEM_TYPE_CANNOT_DELETE")
	ErrDictTypeHasItems  = errors.New("DICT_TYPE_HAS_ITEMS")
)

func IsDictClientErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDictTypeKeyExists) || errors.Is(err, ErrDictSystemType) ||
		errors.Is(err, ErrDictTypeHasItems) {
		return true
	}
	switch err.Error() {
	case "KEY_EMPTY", "LABEL_EMPTY":
		return true
	}
	return false
}

// ==================== DTOs ====================

type DictTypeDetail struct {
	ID          uint64 `json:"id"`
	TypeKey     string `json:"type_key"`
	TypeName    string `json:"type_name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	SortOrder   int    `json:"sort_order"`
	ItemCount   int64  `json:"item_count"`
	CreatedAt   string `json:"created_at"`
}

type ListDictTypesResp struct {
	List []DictTypeDetail `json:"list"`
}

type CreateDictTypeReq struct {
	TypeKey     string `json:"type_key" binding:"required"`
	TypeName    string `json:"type_name" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateDictTypeReq struct {
	TypeName    *string `json:"type_name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type DictItemDetail struct {
	ID        uint64          `json:"id"`
	TypeID    uint64          `json:"type_id"`
	ItemKey   string          `json:"item_key"`
	ItemLabel string          `json:"item_label"`
	Extra     json.RawMessage `json:"extra"`
	SortOrder int             `json:"sort_order"`
	IsEnabled bool            `json:"is_enabled"`
}

type ListDictItemsResp struct {
	List []DictItemDetail `json:"list"`
}

type DictItemPublic struct {
	ItemKey   string          `json:"item_key"`
	ItemLabel string          `json:"item_label"`
	Extra     json.RawMessage `json:"extra"`
	SortOrder int             `json:"sort_order"`
}

type CreateDictItemReq struct {
	ItemKey   string          `json:"item_key" binding:"required"`
	ItemLabel string          `json:"item_label" binding:"required"`
	Extra     json.RawMessage `json:"extra"`
	SortOrder int             `json:"sort_order"`
	IsEnabled *bool           `json:"is_enabled"`
}

type UpdateDictItemReq struct {
	ItemKey   *string          `json:"item_key"`
	ItemLabel *string          `json:"item_label"`
	Extra     *json.RawMessage `json:"extra"`
	SortOrder *int             `json:"sort_order"`
	IsEnabled *bool            `json:"is_enabled"`
}

type BatchSortReq struct {
	Items []BatchSortItem `json:"items" binding:"required"`
}

type BatchSortItem struct {
	ID        uint64 `json:"id"`
	SortOrder int    `json:"sort_order"`
}

type GetDictItemsResp struct {
	TypeKey string           `json:"type_key"`
	Items   []DictItemPublic `json:"items"`
}

type GetDictItemsBatchResp struct {
	Dicts map[string][]DictItemPublic `json:"dicts"`
}

// ==================== Service ====================

type DictService struct {
	db   *gorm.DB
	dict *repository.DictRepo
}

func NewDictService(db *gorm.DB, dict *repository.DictRepo) *DictService {
	return &DictService{db: db, dict: dict}
}

// ==================== Admin: Dict Types ====================

func (s *DictService) ListTypes(ctx context.Context, kgID uint64) (*ListDictTypesResp, error) {
	types, err := s.dict.ListTypes(ctx, kgID)
	if err != nil {
		return nil, err
	}
	list := make([]DictTypeDetail, 0, len(types))
	for _, t := range types {
		count, _ := s.dict.CountItemsByTypeID(ctx, t.ID)
		list = append(list, DictTypeDetail{
			ID:          t.ID,
			TypeKey:     t.TypeKey,
			TypeName:    t.TypeName,
			Description: t.Description,
			IsSystem:    t.IsSystem,
			SortOrder:   t.SortOrder,
			ItemCount:   count,
			CreatedAt:   t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &ListDictTypesResp{List: list}, nil
}

func (s *DictService) CreateType(ctx context.Context, kgID uint64, req CreateDictTypeReq) (*DictTypeDetail, error) {
	// Check uniqueness
	existing, err := s.dict.FindTypeByKey(ctx, kgID, req.TypeKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: 字典类型 %s 已存在", ErrDictTypeKeyExists, req.TypeKey)
	}
	t := &domain.SysDictType{
		KindergartenID: kgID,
		TypeKey:        req.TypeKey,
		TypeName:       req.TypeName,
		Description:    req.Description,
		SortOrder:      req.SortOrder,
	}
	if err := s.dict.CreateType(ctx, t); err != nil {
		return nil, err
	}
	return &DictTypeDetail{
		ID:          t.ID,
		TypeKey:     t.TypeKey,
		TypeName:    t.TypeName,
		Description: t.Description,
		SortOrder:   t.SortOrder,
		ItemCount:   0,
		CreatedAt:   t.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DictService) UpdateType(ctx context.Context, kgID, typeID uint64, req UpdateDictTypeReq) (*DictTypeDetail, error) {
	t, err := s.dict.FindTypeByID(ctx, kgID, typeID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrDictTypeNotFound
	}
	if req.TypeName != nil {
		t.TypeName = *req.TypeName
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.SortOrder != nil {
		t.SortOrder = *req.SortOrder
	}
	if err := s.dict.UpdateType(ctx, t); err != nil {
		return nil, err
	}
	count, _ := s.dict.CountItemsByTypeID(ctx, t.ID)
	return &DictTypeDetail{
		ID:          t.ID,
		TypeKey:     t.TypeKey,
		TypeName:    t.TypeName,
		Description: t.Description,
		IsSystem:    t.IsSystem,
		SortOrder:   t.SortOrder,
		ItemCount:   count,
		CreatedAt:   t.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DictService) DeleteType(ctx context.Context, kgID, typeID uint64) error {
	t, err := s.dict.FindTypeByID(ctx, kgID, typeID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrDictTypeNotFound
	}
	if t.IsSystem {
		return fmt.Errorf("%w: %s 是系统内置类型，不可删除", ErrDictSystemType, t.TypeName)
	}
	count, err := s.dict.CountItemsByTypeID(ctx, typeID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: 请先删除该类型下的 %d 个字典项", ErrDictTypeHasItems, count)
	}
	rows, err := s.dict.DeleteType(ctx, kgID, typeID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDictTypeNotFound
	}
	return nil
}

// ==================== Admin: Dict Items ====================

func (s *DictService) ListItems(ctx context.Context, typeID uint64) (*ListDictItemsResp, error) {
	items, err := s.dict.ListItemsByTypeID(ctx, typeID)
	if err != nil {
		return nil, err
	}
	list := make([]DictItemDetail, 0, len(items))
	for _, item := range items {
		extra := json.RawMessage(item.Extra)
		if item.Extra == "" {
			extra = nil
		}
		list = append(list, DictItemDetail{
			ID:        item.ID,
			TypeID:    item.TypeID,
			ItemKey:   item.ItemKey,
			ItemLabel: item.ItemLabel,
			Extra:     extra,
			SortOrder: item.SortOrder,
			IsEnabled: item.IsEnabled,
		})
	}
	return &ListDictItemsResp{List: list}, nil
}

func (s *DictService) CreateItem(ctx context.Context, typeID uint64, req CreateDictItemReq) (*DictItemDetail, error) {
	extra := string(req.Extra)
	if extra == "" {
		extra = "{}"
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	item := &domain.SysDictItem{
		TypeID:    typeID,
		ItemKey:   req.ItemKey,
		ItemLabel: req.ItemLabel,
		Extra:     extra,
		SortOrder: req.SortOrder,
		IsEnabled: isEnabled,
	}
	if err := s.dict.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return &DictItemDetail{
		ID:        item.ID,
		TypeID:    item.TypeID,
		ItemKey:   item.ItemKey,
		ItemLabel: item.ItemLabel,
		Extra:     json.RawMessage(extra),
		SortOrder: item.SortOrder,
		IsEnabled: item.IsEnabled,
	}, nil
}

func (s *DictService) UpdateItem(ctx context.Context, itemID uint64, req UpdateDictItemReq) (*DictItemDetail, error) {
	item, err := s.dict.FindItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrDictItemNotFound
	}
	if req.ItemKey != nil {
		item.ItemKey = *req.ItemKey
	}
	if req.ItemLabel != nil {
		item.ItemLabel = *req.ItemLabel
	}
	if req.Extra != nil {
		item.Extra = string(*req.Extra)
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}
	if req.IsEnabled != nil {
		item.IsEnabled = *req.IsEnabled
	}
	if err := s.dict.UpdateItem(ctx, item); err != nil {
		return nil, err
	}
	extra := json.RawMessage(item.Extra)
	if item.Extra == "" {
		extra = nil
	}
	return &DictItemDetail{
		ID:        item.ID,
		TypeID:    item.TypeID,
		ItemKey:   item.ItemKey,
		ItemLabel: item.ItemLabel,
		Extra:     extra,
		SortOrder: item.SortOrder,
		IsEnabled: item.IsEnabled,
	}, nil
}

func (s *DictService) DeleteItem(ctx context.Context, itemID uint64) error {
	rows, err := s.dict.DeleteItem(ctx, itemID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDictItemNotFound
	}
	return nil
}

func (s *DictService) BatchSortItems(ctx context.Context, req BatchSortReq) error {
	items := make([]domain.SysDictItem, 0, len(req.Items))
	for _, si := range req.Items {
		item := domain.SysDictItem{SortOrder: si.SortOrder}
		item.ID = si.ID
		items = append(items, item)
	}
	return s.dict.BatchUpdateSortOrder(ctx, items)
}

// ==================== Teacher: Public Read ====================

func (s *DictService) GetDictItems(ctx context.Context, kgID uint64, typeKey string) (*GetDictItemsResp, error) {
	items, err := s.dict.ListEnabledItemsByTypeKey(ctx, kgID, typeKey)
	if err != nil {
		return nil, err
	}
	list := make([]DictItemPublic, 0, len(items))
	for _, item := range items {
		extra := json.RawMessage(item.Extra)
		if item.Extra == "" || item.Extra == "{}" {
			extra = nil
		}
		list = append(list, DictItemPublic{
			ItemKey:   item.ItemKey,
			ItemLabel: item.ItemLabel,
			Extra:     extra,
			SortOrder: item.SortOrder,
		})
	}
	return &GetDictItemsResp{TypeKey: typeKey, Items: list}, nil
}

func (s *DictService) GetDictItemsBatch(ctx context.Context, kgID uint64, typeKeys []string) (*GetDictItemsBatchResp, error) {
	dicts := make(map[string][]DictItemPublic)
	for _, tk := range typeKeys {
		resp, err := s.GetDictItems(ctx, kgID, tk)
		if err != nil {
			return nil, err
		}
		dicts[tk] = resp.Items
	}
	return &GetDictItemsBatchResp{Dicts: dicts}, nil
}
