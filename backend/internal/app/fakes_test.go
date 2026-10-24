package app

import (
	"context"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// fakeMembers 只实现 FindByUser，供 requireFamilyAccess 使用。
type fakeMembers struct {
	familyIDs []string
}

func (f *fakeMembers) Create(context.Context, *model.Member) error { return nil }
func (f *fakeMembers) FindByFamily(context.Context, string) ([]*model.Member, error) {
	return nil, nil
}
func (f *fakeMembers) FindByUser(_ context.Context, userID string) ([]*model.Member, error) {
	rows := make([]*model.Member, 0, len(f.familyIDs))
	for _, familyID := range f.familyIDs {
		rows = append(rows, &model.Member{FamilyID: familyID, UserID: userID, Role: "owner"})
	}
	return rows, nil
}
func (f *fakeMembers) UpdateRole(context.Context, string, string) error { return nil }
func (f *fakeMembers) Delete(context.Context, string) error             { return nil }

// fakeShopping 内存待购仓储。
type fakeShopping struct {
	items map[string]*model.ShoppingListItem
}

func newFakeShopping() *fakeShopping {
	return &fakeShopping{items: make(map[string]*model.ShoppingListItem)}
}

func (f *fakeShopping) Create(_ context.Context, item *model.ShoppingListItem) error {
	copied := *item
	f.items[item.ID] = &copied
	return nil
}

func (f *fakeShopping) FindByID(_ context.Context, familyID, id string) (*model.ShoppingListItem, error) {
	item, ok := f.items[id]
	if !ok || item.FamilyID != familyID || item.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	copied := *item
	return &copied, nil
}

func (f *fakeShopping) ListByFamily(_ context.Context, familyID, status string) ([]*model.ShoppingListItem, error) {
	out := make([]*model.ShoppingListItem, 0)
	for _, item := range f.items {
		if item.FamilyID != familyID || item.DeletedAt != nil {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		copied := *item
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeShopping) Update(_ context.Context, item *model.ShoppingListItem) error {
	copied := *item
	f.items[item.ID] = &copied
	return nil
}

func (f *fakeShopping) Delete(_ context.Context, familyID, id string) error {
	item, ok := f.items[id]
	if !ok || item.FamilyID != familyID {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	return nil
}

// fakeInventory 内存库存仓储；Adjust 模拟事务语义（余额不足返回冲突）。
type fakeInventory struct {
	items  map[string]*model.InventoryItem
	txSink *fakeTx
}

func newFakeInventory() *fakeInventory {
	return &fakeInventory{items: make(map[string]*model.InventoryItem)}
}

func (f *fakeInventory) Create(_ context.Context, item *model.InventoryItem) error {
	copied := *item
	f.items[item.ID] = &copied
	return nil
}

func (f *fakeInventory) FindByID(_ context.Context, id string) (*model.InventoryItem, error) {
	item, ok := f.items[id]
	if !ok || item.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	copied := *item
	return &copied, nil
}

func (f *fakeInventory) ListByFamily(_ context.Context, familyID string) ([]*model.InventoryItem, error) {
	out := make([]*model.InventoryItem, 0)
	for _, item := range f.items {
		if item.FamilyID == familyID && item.DeletedAt == nil {
			copied := *item
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeInventory) Update(_ context.Context, item *model.InventoryItem) error {
	copied := *item
	f.items[item.ID] = &copied
	return nil
}

func (f *fakeInventory) Delete(_ context.Context, id string) error {
	item, ok := f.items[id]
	if !ok {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	return nil
}

func (f *fakeInventory) Adjust(_ context.Context, familyID, itemID string, delta int, tx *model.InventoryTransaction) (*model.InventoryItem, error) {
	item, ok := f.items[itemID]
	if !ok || item.FamilyID != familyID || item.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	next := item.Quantity + delta
	if next < 0 {
		return nil, repository.ErrConflict
	}
	item.Quantity = next
	tx.ItemID = itemID
	if f.txSink != nil {
		copied := *tx
		f.txSink.rows = append(f.txSink.rows, &copied)
	}
	copied := *item
	return &copied, nil
}

// fakeTx 内存流水仓储。
type fakeTx struct {
	rows []*model.InventoryTransaction
}

func (f *fakeTx) Create(_ context.Context, tx *model.InventoryTransaction) error {
	copied := *tx
	f.rows = append(f.rows, &copied)
	return nil
}

func (f *fakeTx) ListByFamily(_ context.Context, familyID, itemID string, limit int) ([]*model.InventoryTransaction, error) {
	out := make([]*model.InventoryTransaction, 0)
	for i := len(f.rows) - 1; i >= 0; i-- {
		row := f.rows[i]
		if row.FamilyID != familyID {
			continue
		}
		if itemID != "" && row.ItemID != itemID {
			continue
		}
		copied := *row
		out = append(out, &copied)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// fakePurchase 内存采购导入仓储；Confirm 复刻真实仓储的幂等语义。
type fakePurchase struct {
	imports     map[string]*model.PurchaseImport
	lines       map[string][]*model.PurchaseImportLine
	expenses    []*model.Expense
	txRows      []*model.InventoryTransaction
	confirmCall int
	// failConfirm 非空时 Confirm 直接返回该错误。
	failConfirm error
}

func newFakePurchase() *fakePurchase {
	return &fakePurchase{
		imports: make(map[string]*model.PurchaseImport),
		lines:   make(map[string][]*model.PurchaseImportLine),
	}
}

func (f *fakePurchase) Create(_ context.Context, imp *model.PurchaseImport) error {
	copied := *imp
	copied.MediaKeys = append([]string(nil), imp.MediaKeys...)
	f.imports[imp.ID] = &copied
	return nil
}

func (f *fakePurchase) FindByID(_ context.Context, familyID, id string) (*model.PurchaseImport, error) {
	imp, ok := f.imports[id]
	if !ok || imp.FamilyID != familyID || imp.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	copied := *imp
	copied.MediaKeys = append([]string(nil), imp.MediaKeys...)
	return &copied, nil
}

func (f *fakePurchase) ListByFamily(_ context.Context, familyID string, limit int) ([]*model.PurchaseImport, error) {
	out := make([]*model.PurchaseImport, 0)
	for _, imp := range f.imports {
		if imp.FamilyID == familyID && imp.DeletedAt == nil {
			copied := *imp
			copied.MediaKeys = append([]string(nil), imp.MediaKeys...)
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakePurchase) ListLines(_ context.Context, familyID, importID string) ([]*model.PurchaseImportLine, error) {
	out := make([]*model.PurchaseImportLine, 0)
	for _, line := range f.lines[importID] {
		if line.FamilyID != familyID {
			continue
		}
		copied := *line
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakePurchase) UpdateImport(_ context.Context, imp *model.PurchaseImport) error {
	copied := *imp
	copied.MediaKeys = append([]string(nil), imp.MediaKeys...)
	f.imports[imp.ID] = &copied
	return nil
}

func (f *fakePurchase) ReplaceLines(_ context.Context, familyID, importID string, lines []*model.PurchaseImportLine) error {
	f.lines[importID] = nil
	for _, line := range lines {
		copied := *line
		f.lines[importID] = append(f.lines[importID], &copied)
	}
	return nil
}

func (f *fakePurchase) Confirm(_ context.Context, familyID, userID, importID string, plan repository.PurchaseConfirmPlan) (*model.PurchaseImport, bool, error) {
	f.confirmCall++
	if f.failConfirm != nil {
		return nil, false, f.failConfirm
	}
	imp, ok := f.imports[importID]
	if !ok || imp.FamilyID != familyID {
		return nil, false, repository.ErrNotFound
	}
	if imp.Status == model.PurchaseStatusConfirmed || imp.ConfirmedAt != nil {
		copied := *imp
		return &copied, false, nil
	}
	if imp.Status == model.PurchaseStatusCancelled {
		return nil, false, repository.ErrConflict
	}
	for _, line := range plan.StockLines {
		if line.Tx != nil {
			f.txRows = append(f.txRows, line.Tx)
		}
	}
	if plan.Expense != nil {
		f.expenses = append(f.expenses, plan.Expense)
		imp.ExpenseID = plan.Expense.ID
	}
	now := time.Now().UTC()
	imp.Status = model.PurchaseStatusConfirmed
	imp.ConfirmedAt = &now
	copied := *imp
	return &copied, true, nil
}

// fakeOCR 可编程识别提供方。
type fakeOCR struct {
	raw    string
	err    error
	called int
}

func (f *fakeOCR) RecognizeOrder(context.Context, [][]byte) (string, error) {
	f.called++
	if f.err != nil {
		return "", f.err
	}
	return f.raw, nil
}
