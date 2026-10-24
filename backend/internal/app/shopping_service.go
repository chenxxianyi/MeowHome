package app

import (
	"context"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/platform/id"
)

// ShoppingService 覆盖待购买清单、库存手动调整与库存流水。
type ShoppingService struct {
	shopping  repository.ShoppingListRepo
	inventory repository.InventoryRepo
	tx        repository.InventoryTransactionRepo
	members   repository.MemberRepo
}

// NewShoppingService 创建待购与库存调整服务。
func NewShoppingService(
	shopping repository.ShoppingListRepo,
	inventory repository.InventoryRepo,
	tx repository.InventoryTransactionRepo,
	members repository.MemberRepo,
) *ShoppingService {
	return &ShoppingService{shopping: shopping, inventory: inventory, tx: tx, members: members}
}

// ShoppingItemEnvelope 待购买项响应。
type ShoppingItemEnvelope struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Brand                  string `json:"brand,omitempty"`
	Specification          string `json:"specification,omitempty"`
	Category               string `json:"category,omitempty"`
	Quantity               int    `json:"quantity"`
	Unit                   string `json:"unit,omitempty"`
	Note                   string `json:"note,omitempty"`
	Priority               string `json:"priority"`
	Status                 string `json:"status"`
	MatchedInventoryItemID string `json:"matched_inventory_item_id,omitempty"`
	CompletedAt            string `json:"completed_at,omitempty"`
	CreatedBy              string `json:"created_by,omitempty"`
	CreatedAt              string `json:"created_at,omitempty"`
}

// shoppingEnvelope 把模型转成响应。
func shoppingEnvelope(item *model.ShoppingListItem) *ShoppingItemEnvelope {
	env := &ShoppingItemEnvelope{
		ID:            item.ID,
		Name:          item.Name,
		Brand:         item.Brand,
		Specification: item.Specification,
		Category:      item.Category,
		Quantity:      item.Quantity,
		Unit:          item.Unit,
		Note:          item.Note,
		Priority:      item.Priority,
		Status:        item.Status,
		CreatedBy:     item.CreatedBy,
		CreatedAt:     item.CreatedAt.Format(time.RFC3339),
	}
	if item.MatchedInventoryItemID != nil {
		env.MatchedInventoryItemID = *item.MatchedInventoryItemID
	}
	if item.CompletedAt != nil {
		env.CompletedAt = item.CompletedAt.Format(time.RFC3339)
	}
	return env
}

// ShoppingInput 创建待购买项的输入。
type ShoppingInput struct {
	Name          string `json:"name"`
	Brand         string `json:"brand"`
	Specification string `json:"specification"`
	Category      string `json:"category"`
	Quantity      int    `json:"quantity"`
	Unit          string `json:"unit"`
	Note          string `json:"note"`
	Priority      string `json:"priority"`
}

// ShoppingList 列出家庭待购买项；status 为空表示全部，todo/purchased/dismissed。
func (s *ShoppingService) ShoppingList(ctx context.Context, familyID, userID, status string) ([]*ShoppingItemEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if status != "" && !validShoppingStatus(status) {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "status must be todo, purchased or dismissed")
	}
	rows, err := s.shopping.ListByFamily(ctx, familyID, status)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list shopping items", err)
	}
	out := make([]*ShoppingItemEnvelope, 0, len(rows))
	for _, row := range rows {
		out = append(out, shoppingEnvelope(row))
	}
	return out, nil
}

// CreateShopping 创建待购买项。加入待购不改变库存数量。
func (s *ShoppingService) CreateShopping(ctx context.Context, familyID, userID string, in ShoppingInput) (*ShoppingItemEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "name is required")
	}
	quantity := in.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	priority := in.Priority
	if priority == "" {
		priority = model.ShoppingPriorityNormal
	}
	if priority != model.ShoppingPriorityLow && priority != model.ShoppingPriorityNormal && priority != model.ShoppingPriorityHigh {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "priority must be low, normal or high")
	}
	now := time.Now().UTC()
	item := &model.ShoppingListItem{
		Base:          model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:      familyID,
		Name:          name,
		Brand:         strings.TrimSpace(in.Brand),
		Specification: strings.TrimSpace(in.Specification),
		Category:      strings.TrimSpace(in.Category),
		Quantity:      quantity,
		Unit:          strings.TrimSpace(in.Unit),
		Note:          strings.TrimSpace(in.Note),
		Priority:      priority,
		Status:        model.ShoppingStatusTodo,
	}
	if err := s.shopping.Create(ctx, item); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create shopping item", err)
	}
	return shoppingEnvelope(item), nil
}

// ShoppingUpdateInput 修改待购买项的补丁；未设置的字段保持原值。
type ShoppingUpdateInput struct {
	Name                   *string `json:"name"`
	Brand                  *string `json:"brand"`
	Specification          *string `json:"specification"`
	Category               *string `json:"category"`
	Quantity               *int    `json:"quantity"`
	Unit                   *string `json:"unit"`
	Note                   *string `json:"note"`
	Priority               *string `json:"priority"`
	Status                 *string `json:"status"`
	MatchedInventoryItemID *string `json:"matched_inventory_item_id"`
}

// UpdateShopping 修改待购买项；status 置为 purchased 即完成（也可改为 dismissed）。
func (s *ShoppingService) UpdateShopping(ctx context.Context, familyID, userID, itemID string, in ShoppingUpdateInput) (*ShoppingItemEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	item, err := s.shopping.FindByID(ctx, familyID, itemID)
	if err != nil {
		return nil, mapShoppingRepoError(err)
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "name must not be empty")
		}
		item.Name = name
	}
	if in.Brand != nil {
		item.Brand = strings.TrimSpace(*in.Brand)
	}
	if in.Specification != nil {
		item.Specification = strings.TrimSpace(*in.Specification)
	}
	if in.Category != nil {
		item.Category = strings.TrimSpace(*in.Category)
	}
	if in.Quantity != nil {
		if *in.Quantity <= 0 {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "quantity must be positive")
		}
		item.Quantity = *in.Quantity
	}
	if in.Unit != nil {
		item.Unit = strings.TrimSpace(*in.Unit)
	}
	if in.Note != nil {
		item.Note = strings.TrimSpace(*in.Note)
	}
	if in.Priority != nil {
		if *in.Priority != model.ShoppingPriorityLow && *in.Priority != model.ShoppingPriorityNormal && *in.Priority != model.ShoppingPriorityHigh {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "priority must be low, normal or high")
		}
		item.Priority = *in.Priority
	}
	if in.Status != nil {
		status := *in.Status
		if !validShoppingStatus(status) {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "status must be todo, purchased or dismissed")
		}
		if status != item.Status {
			item.Status = status
			if status == model.ShoppingStatusTodo {
				item.CompletedAt = nil
			} else {
				now := time.Now().UTC()
				item.CompletedAt = &now
			}
		}
	}
	if in.MatchedInventoryItemID != nil {
		if *in.MatchedInventoryItemID == "" {
			item.MatchedInventoryItemID = nil
		} else {
			item.MatchedInventoryItemID = in.MatchedInventoryItemID
		}
	}
	if err := s.shopping.Update(ctx, item); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update shopping item", err)
	}
	return shoppingEnvelope(item), nil
}

// DeleteShopping 移除待购买项。
func (s *ShoppingService) DeleteShopping(ctx context.Context, familyID, userID, itemID string) error {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return err
	}
	if _, err := s.shopping.FindByID(ctx, familyID, itemID); err != nil {
		return mapShoppingRepoError(err)
	}
	if err := s.shopping.Delete(ctx, familyID, itemID); err != nil {
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to delete shopping item", err)
	}
	return nil
}

// InventoryAdjustInput 手动库存调整的输入；delta 为正入库、负出库。
type InventoryAdjustInput struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
}

// InventoryAdjustmentEnvelope 调整结果：更新后的库存项与对应流水。
type InventoryAdjustmentEnvelope struct {
	Item        *InventoryEnvelope `json:"item"`
	Transaction *TxEnvelope        `json:"transaction"`
}

// TxEnvelope 库存流水响应。
type TxEnvelope struct {
	ID         string `json:"id"`
	ItemID     string `json:"item_id"`
	Change     int    `json:"change"`
	Reason     string `json:"reason"`
	SourceType string `json:"source_type,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	CreatedBy  string `json:"created_by,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

func txEnvelope(tx *model.InventoryTransaction) *TxEnvelope {
	return &TxEnvelope{
		ID:         tx.ID,
		ItemID:     tx.ItemID,
		Change:     tx.Change,
		Reason:     tx.Reason,
		SourceType: tx.SourceType,
		SourceID:   tx.SourceID,
		CreatedBy:  tx.CreatedBy,
		CreatedAt:  tx.CreatedAt.Format(time.RFC3339),
	}
}

// AdjustInventory 手动调整库存数量并记录流水（数量与流水同事务）。
func (s *ShoppingService) AdjustInventory(ctx context.Context, familyID, userID, itemID string, in InventoryAdjustInput) (*InventoryAdjustmentEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if in.Delta == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "delta must not be zero")
	}
	if in.Delta > 0 && in.Delta > 1000000 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "delta is too large")
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = model.TxReasonManualAdjustment
	}
	if len(reason) > 120 {
		reason = reason[:120]
	}
	now := time.Now().UTC()
	tx := &model.InventoryTransaction{
		Base:       model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:   familyID,
		Change:     in.Delta,
		Reason:     reason,
		SourceType: model.TxReasonManualAdjustment,
	}
	// 流水 ID 作为来源标识：每次手动调整独立一条，唯一键不会误判重复。
	tx.SourceID = tx.ID
	item, err := s.inventory.Adjust(ctx, familyID, itemID, in.Delta, tx)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return nil, errors.NotFound(errors.CodeNotFound, "inventory item not found")
		case repository.ErrConflict:
			return nil, errors.Conflict(errors.CodeConflict, "inventory quantity cannot become negative")
		default:
			return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to adjust inventory", err)
		}
	}
	return &InventoryAdjustmentEnvelope{
		Item:        inventoryEnvelope(item),
		Transaction: txEnvelope(tx),
	}, nil
}

// InventoryTransactions 列出库存流水，itemID 为空表示全部。
func (s *ShoppingService) InventoryTransactions(ctx context.Context, familyID, userID, itemID string, limit int) ([]*TxEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.tx.ListByFamily(ctx, familyID, itemID, limit)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list inventory transactions", err)
	}
	out := make([]*TxEnvelope, 0, len(rows))
	for _, row := range rows {
		out = append(out, txEnvelope(row))
	}
	return out, nil
}

func validShoppingStatus(status string) bool {
	return status == model.ShoppingStatusTodo || status == model.ShoppingStatusPurchased || status == model.ShoppingStatusDismissed
}

// mapShoppingRepoError 把仓储错误映射为业务错误。
func mapShoppingRepoError(err error) error {
	if err == repository.ErrNotFound {
		return errors.NotFound(errors.CodeNotFound, "shopping item not found")
	}
	return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "shopping item unavailable", err)
}
