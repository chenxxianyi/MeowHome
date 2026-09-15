package app

import (
	"context"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// AssetService 覆盖「时光 / 库存 / 账目」三个轻量业务域。
type AssetService struct {
	timeline  repository.TimelineRepo
	inventory repository.InventoryRepo
	expenses  repository.ExpenseRepo
	members   repository.MemberRepo
}

// NewAssetService 创建资产服务。
func NewAssetService(
	timeline repository.TimelineRepo,
	inventory repository.InventoryRepo,
	expenses repository.ExpenseRepo,
	members repository.MemberRepo,
) *AssetService {
	return &AssetService{timeline: timeline, inventory: inventory, expenses: expenses, members: members}
}

// MomentEnvelope 时光事件响应（与前端 TimelineEvent 对齐）。
type MomentEnvelope struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	CatID  string `json:"catId,omitempty"`
	Images int    `json:"images,omitempty"`
}

// InventoryEnvelope 库存项响应（与前端 InventoryItem 对齐）。
type InventoryEnvelope struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	Quantity      int    `json:"quantity"`
	Unit          string `json:"unit"`
	EstimatedDays int    `json:"estimatedDays,omitempty"`
	Status        string `json:"status"`
	Expiry        string `json:"expiry"`
}

// ExpenseEnvelope 支出响应（与前端 Expense 对齐）。
type ExpenseEnvelope struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Label    string  `json:"label"`
	CatID    string  `json:"catId,omitempty"`
}

// --- 时光 ---

// Moments 列出家庭时光事件。
func (s *AssetService) Moments(ctx context.Context, familyID, userID string, limit int) ([]*MomentEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.timeline.ListByFamily(ctx, familyID, limit)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list moments", err)
	}
	out := make([]*MomentEnvelope, 0, len(rows))
	for _, e := range rows {
		env := &MomentEnvelope{
			ID:     e.ID,
			Date:   e.OccurredOn,
			Type:   e.EventType,
			Title:  e.Title,
			Body:   e.Body,
			Images: e.ImageCount,
		}
		if len(e.CatIDs) > 0 {
			env.CatID = e.CatIDs[0]
		}
		out = append(out, env)
	}
	return out, nil
}

// MomentInput 创建时光事件的输入。
type MomentInput struct {
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Date   string   `json:"date"`
	CatIDs []string `json:"cat_ids"`
	Images int      `json:"images"`
}

// CreateMoment 创建时光事件。
func (s *AssetService) CreateMoment(ctx context.Context, familyID, userID string, in MomentInput) (*MomentEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if in.Title == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "title is required")
	}
	if in.Type == "" {
		in.Type = "photo"
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}

	now := time.Now().UTC()
	e := &model.TimelineEvent{
		Base:       model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:   familyID,
		CatIDs:     in.CatIDs,
		Title:      in.Title,
		Body:       in.Body,
		EventType:  in.Type,
		OccurredOn: in.Date,
		ImageCount: in.Images,
	}
	if err := s.timeline.Create(ctx, e); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create moment", err)
	}
	env := &MomentEnvelope{ID: e.ID, Date: e.OccurredOn, Type: e.EventType, Title: e.Title, Body: e.Body, Images: e.ImageCount}
	if len(e.CatIDs) > 0 {
		env.CatID = e.CatIDs[0]
	}
	return env, nil
}

// --- 库存 ---

// Inventory 列出家庭库存，并即时推导 status / estimatedDays。
func (s *AssetService) Inventory(ctx context.Context, familyID, userID string) ([]*InventoryEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.inventory.ListByFamily(ctx, familyID)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list inventory", err)
	}

	today := time.Now().Format("2006-01-02")
	out := make([]*InventoryEnvelope, 0, len(rows))
	for _, it := range rows {
		env := &InventoryEnvelope{
			ID:       it.ID,
			Name:     it.Name,
			Category: it.Category,
			Quantity: it.Quantity,
			Unit:     it.Unit,
			Expiry:   it.Expiry,
			Status:   inventoryStatus(it, today),
		}
		out = append(out, env)
	}
	return out, nil
}

// inventoryStatus 推导库存状态：过期 > 不足 > 正常。
func inventoryStatus(it *model.InventoryItem, today string) string {
	if it.Expiry != "" && len(it.Expiry) >= 10 && it.Expiry[:10] < today {
		return "expired"
	}
	if it.Quantity <= 0 {
		return "expired"
	}
	if it.LowStockThreshold > 0 && it.Quantity <= it.LowStockThreshold {
		return "low"
	}
	return "ok"
}

// InventoryInput 创建库存项的输入。
type InventoryInput struct {
	Name              string `json:"name"`
	Category          string `json:"category"`
	Quantity          int    `json:"quantity"`
	Unit              string `json:"unit"`
	LowStockThreshold int    `json:"low_stock_threshold"`
	Expiry            string `json:"expiry"`
}

// CreateInventory 创建库存项。
func (s *AssetService) CreateInventory(ctx context.Context, familyID, userID string, in InventoryInput) (*InventoryEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "name is required")
	}
	now := time.Now().UTC()
	it := &model.InventoryItem{
		Base:              model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:          familyID,
		Name:              in.Name,
		Category:          in.Category,
		Unit:              in.Unit,
		Quantity:          in.Quantity,
		LowStockThreshold: in.LowStockThreshold,
		Expiry:            in.Expiry,
	}
	if err := s.inventory.Create(ctx, it); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create inventory item", err)
	}
	return &InventoryEnvelope{
		ID: it.ID, Name: it.Name, Category: it.Category, Quantity: it.Quantity,
		Unit: it.Unit, Expiry: it.Expiry, Status: inventoryStatus(it, time.Now().Format("2006-01-02")),
	}, nil
}

// --- 账目 ---

// Expenses 列出支出；month 形如 2026-08，为空表示不限。
func (s *AssetService) Expenses(ctx context.Context, familyID, userID, month string) ([]*ExpenseEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.expenses.List(ctx, repository.ExpenseQuery{FamilyID: familyID, Month: month})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list expenses", err)
	}
	out := make([]*ExpenseEnvelope, 0, len(rows))
	for _, e := range rows {
		env := &ExpenseEnvelope{
			ID:       e.ID,
			Date:     e.OccurredOn,
			Amount:   float64(e.Amount) / 100, // 库存的是最小货币单位（分）
			Category: e.Category,
			Label:    e.Label,
		}
		if len(e.CatIDs) > 0 {
			env.CatID = e.CatIDs[0]
		}
		out = append(out, env)
	}
	return out, nil
}

// ExpenseInput 创建支出的输入。
type ExpenseInput struct {
	Date     string   `json:"date"`
	Amount   float64  `json:"amount"`
	Category string   `json:"category"`
	Label    string   `json:"label"`
	CatIDs   []string `json:"cat_ids"`
}

// CreateExpense 创建支出。
func (s *AssetService) CreateExpense(ctx context.Context, familyID, userID string, in ExpenseInput) (*ExpenseEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if in.Amount <= 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "amount must be positive")
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}

	now := time.Now().UTC()
	e := &model.Expense{
		Base:       model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:   familyID,
		OccurredOn: in.Date,
		Amount:     int64(in.Amount*100 + 0.5),
		Category:   in.Category,
		Label:      in.Label,
		CatIDs:     in.CatIDs,
	}
	if err := s.expenses.Create(ctx, e); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create expense", err)
	}
	env := &ExpenseEnvelope{ID: e.ID, Date: e.OccurredOn, Amount: in.Amount, Category: e.Category, Label: e.Label}
	if len(e.CatIDs) > 0 {
		env.CatID = e.CatIDs[0]
	}
	return env, nil
}
