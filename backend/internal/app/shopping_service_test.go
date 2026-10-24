package app

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/meowhome/backend/internal/domain/model"
	apperr "github.com/meowhome/backend/internal/platform/errors"
)

func newShoppingServiceForTest() (*ShoppingService, *fakeShopping, *fakeInventory, *fakeTx) {
	members := &fakeMembers{familyIDs: []string{"fam-1"}}
	shopping := newFakeShopping()
	inventory := newFakeInventory()
	tx := &fakeTx{}
	inventory.txSink = tx
	return NewShoppingService(shopping, inventory, tx, members), shopping, inventory, tx
}

func appErrorCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperr.AppError
	if !stderrors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}
	return appErr.Code
}

// TestShoppingCreateListAndComplete 覆盖待购新增、完成与恢复。
func TestShoppingCreateListAndComplete(t *testing.T) {
	svc, _, _, _ := newShoppingServiceForTest()
	ctx := context.Background()

	created, err := svc.CreateShopping(ctx, "fam-1", "u1", ShoppingInput{Name: "猫粮", Quantity: 2, Unit: "袋", Category: "food"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != model.ShoppingStatusTodo || created.Quantity != 2 || created.Priority != model.ShoppingPriorityNormal {
		t.Fatalf("created = %+v", created)
	}

	list, err := svc.ShoppingList(ctx, "fam-1", "u1", model.ShoppingStatusTodo)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v", list)
	}

	status := model.ShoppingStatusPurchased
	done, err := svc.UpdateShopping(ctx, "fam-1", "u1", created.ID, ShoppingUpdateInput{Status: &status})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if done.Status != model.ShoppingStatusPurchased || done.CompletedAt == "" {
		t.Fatalf("completed = %+v", done)
	}

	todo := model.ShoppingStatusTodo
	restored, err := svc.UpdateShopping(ctx, "fam-1", "u1", created.ID, ShoppingUpdateInput{Status: &todo})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Status != model.ShoppingStatusTodo || restored.CompletedAt != "" {
		t.Fatalf("restored = %+v", restored)
	}
}

// TestShoppingValidation 覆盖名称必填与非法状态。
func TestShoppingValidation(t *testing.T) {
	svc, _, _, _ := newShoppingServiceForTest()
	ctx := context.Background()
	if _, err := svc.CreateShopping(ctx, "fam-1", "u1", ShoppingInput{Name: "  "}); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("blank name should be rejected, got %v", err)
	}
	bad := model.ShoppingStatusDismissed + "x"
	if _, err := svc.UpdateShopping(ctx, "fam-1", "u1", "missing", ShoppingUpdateInput{Status: &bad}); err == nil {
		t.Fatal("expected error for missing item")
	}
	if _, err := svc.ShoppingList(ctx, "fam-1", "u1", "weird"); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("invalid status should be rejected, got %v", err)
	}
}

// TestShoppingCrossFamilyIsolation 覆盖跨家庭资源不可见。
func TestShoppingCrossFamilyIsolation(t *testing.T) {
	svc, shopping, _, _ := newShoppingServiceForTest()
	ctx := context.Background()
	// 另一家庭的待购项：当前用户不可访问。
	foreign := &model.ShoppingListItem{Base: model.Base{ID: "foreign-1"}, FamilyID: "fam-2", Name: "别人的猫粮", Status: model.ShoppingStatusTodo, Quantity: 1}
	_ = shopping.Create(ctx, foreign)
	if _, err := svc.UpdateShopping(ctx, "fam-1", "u1", "foreign-1", ShoppingUpdateInput{}); appErrorCode(t, err) != apperr.CodeNotFound {
		t.Fatalf("cross-family update should be NOT_FOUND, got %v", err)
	}
	if err := svc.DeleteShopping(ctx, "fam-1", "u1", "foreign-1"); appErrorCode(t, err) != apperr.CodeNotFound {
		t.Fatalf("cross-family delete should be NOT_FOUND, got %v", err)
	}
	if err := svc.DeleteShopping(ctx, "fam-1", "u1", "foreign-1"); err == nil {
		t.Fatal("expected error")
	}
}

// TestAdjustInventoryWritesTransaction 覆盖手动调整写流水与数量。
func TestAdjustInventoryWritesTransaction(t *testing.T) {
	svc, _, inventory, txLog := newShoppingServiceForTest()
	ctx := context.Background()
	item := &model.InventoryItem{Base: model.Base{ID: "item-1", CreatedBy: "u1"}, FamilyID: "fam-1", Name: "猫砂", Unit: "袋", Quantity: 5}
	if err := inventory.Create(ctx, item); err != nil {
		t.Fatalf("seed item: %v", err)
	}

	result, err := svc.AdjustInventory(ctx, "fam-1", "u1", "item-1", InventoryAdjustInput{Delta: -3, Reason: "日常消耗"})
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if result.Item.Quantity != 2 {
		t.Fatalf("quantity = %d, want 2", result.Item.Quantity)
	}
	if result.Transaction.Change != -3 || result.Transaction.Reason != "日常消耗" || result.Transaction.ItemID != "item-1" {
		t.Fatalf("transaction = %+v", result.Transaction)
	}

	// 流水可通过接口追溯。
	rows, err := svc.InventoryTransactions(ctx, "fam-1", "u1", "item-1", 10)
	if err != nil {
		t.Fatalf("transactions: %v", err)
	}
	if len(rows) != 1 || rows[0].Change != -3 {
		t.Fatalf("rows = %+v", rows)
	}
	if len(txLog.rows) != 1 {
		t.Fatalf("tx repo rows = %d", len(txLog.rows))
	}
}

// TestAdjustInventoryRejectsNegativeAndZero 覆盖数量为负与 delta 为零。
func TestAdjustInventoryRejectsNegativeAndZero(t *testing.T) {
	svc, _, inventory, _ := newShoppingServiceForTest()
	ctx := context.Background()
	item := &model.InventoryItem{Base: model.Base{ID: "item-1"}, FamilyID: "fam-1", Name: "猫粮", Quantity: 2}
	_ = inventory.Create(ctx, item)

	if _, err := svc.AdjustInventory(ctx, "fam-1", "u1", "item-1", InventoryAdjustInput{Delta: 0}); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("zero delta should be VALIDATION_FAILED, got %v", err)
	}
	if _, err := svc.AdjustInventory(ctx, "fam-1", "u1", "item-1", InventoryAdjustInput{Delta: -5}); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("negative quantity should be CONFLICT, got %v", err)
	}
	// 失败不得写入流水。
	if len(inventory.items) != 1 || inventory.items["item-1"].Quantity != 2 {
		t.Fatalf("quantity mutated after failure")
	}
}
