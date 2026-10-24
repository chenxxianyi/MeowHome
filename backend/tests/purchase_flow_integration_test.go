package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
)

// purchaseFixture 预置家庭/成员/库存与待购项，返回服务与仓储。
func purchaseFixture(t *testing.T) (
	*gorm.DB,
	*app.PurchaseService,
	*app.ShoppingService,
	*model.InventoryItem,
	*model.ShoppingListItem,
	string,
	string,
) {
	t.Helper()
	db := SetupTestDB(t)
	now := time.Now().UTC()
	familyID, userID := ulid.Make().String(), ulid.Make().String()
	require.NoError(t, db.Create(&model.Family{
		Base:     model.Base{ID: familyID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		Name:     "采购测试家庭",
		Timezone: "Asia/Shanghai",
		Currency: "CNY",
	}).Error)
	require.NoError(t, db.Create(&model.Member{
		Base:     model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID: familyID,
		UserID:   userID,
		Role:     "owner",
		Timezone: "Asia/Shanghai",
	}).Error)

	inventoryRepo := mysql.NewInventoryRepo(db)
	memberRepo := mysql.NewMemberRepo(db)
	shoppingRepo := mysql.NewShoppingListRepo(db)
	inventoryTxRepo := mysql.NewInventoryTransactionRepo(db)
	purchaseRepo := mysql.NewPurchaseImportRepo(db)

	item := &model.InventoryItem{
		Base:          model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:      familyID,
		Name:          "皇家成猫粮",
		Specification: "2kg",
		Category:      "food",
		Unit:          "袋",
		Quantity:      3,
	}
	require.NoError(t, inventoryRepo.Create(context.Background(), item))

	shopItem := &model.ShoppingListItem{
		Base:     model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID: familyID,
		Name:     "猫砂",
		Category: "litter",
		Quantity: 1,
		Status:   model.ShoppingStatusTodo,
	}
	require.NoError(t, shoppingRepo.Create(context.Background(), shopItem))

	purchaseSvc := app.NewPurchaseService(purchaseRepo, inventoryRepo, shoppingRepo, memberRepo)
	shoppingSvc := app.NewShoppingService(shoppingRepo, inventoryRepo, inventoryTxRepo, memberRepo)
	return db, purchaseSvc, shoppingSvc, item, shopItem, familyID, userID
}

// createImportWithLines 创建导入并通过 PATCH 保存草稿，返回导入 ID。
func createImportWithLines(t *testing.T, purchaseSvc *app.PurchaseService, familyID, userID string, lines []app.DraftLineInput) string {
	t.Helper()
	created, err := purchaseSvc.CreateImport(context.Background(), familyID, userID, []string{"purchase-imports/x/1.png"})
	require.NoError(t, err)
	version := 0
	total := 0.0
	for _, line := range lines {
		if line.LineAmount != nil {
			total += *line.LineAmount
		}
	}
	_, err = purchaseSvc.UpdateDraft(context.Background(), familyID, userID, created.ID, app.PurchaseDraftInput{
		ExpectedVersion: &version,
		Platform:        "taobao",
		OrderDate:       "2026-10-21",
		TotalAmount:     &total,
		Lines:           lines,
	})
	require.NoError(t, err)
	return created.ID
}

func f64(v float64) *float64 { return &v }

// TestPurchaseConfirmIdempotentAndAtomicity 覆盖确认落账的完整性与幂等性。
func TestPurchaseConfirmIdempotentAndAtomicity(t *testing.T) {
	db, purchaseSvc, _, item, shopItem, familyID, userID := purchaseFixture(t)
	ctx := context.Background()

	importID := createImportWithLines(t, purchaseSvc, familyID, userID, []app.DraftLineInput{
		{Name: "皇家成猫粮", Specification: "2kg", Quantity: 2, Unit: "袋", LineAmount: f64(99), IncludeInStock: true, MatchedInventoryItemID: item.ID},
		{Name: "猫砂", Quantity: 1, Unit: "袋", LineAmount: f64(50), IncludeInStock: true, MatchedShoppingItemID: shopItem.ID},
	})

	summary, err := purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
	require.NoError(t, err)
	require.True(t, summary.Created)
	require.Equal(t, model.PurchaseStatusConfirmed, summary.Import.Status)

	// 支出只生成一笔，金额取订单实付总额。
	var expenses []model.Expense
	require.NoError(t, db.Find(&expenses).Error)
	require.Len(t, expenses, 1)
	require.EqualValues(t, 14900, expenses[0].Amount)
	require.Equal(t, model.TxReasonPurchaseImport, expenses[0].SourceType)
	require.Equal(t, importID, expenses[0].SourceID)

	// 库存流水：两行入库；数量与流水一致（3 + 2 = 5；新项 0 + 1 = 1）。
	var rows []model.InventoryTransaction
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 2)
	var refreshed model.InventoryItem
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 5, refreshed.Quantity)

	// 待购项完成。
	var shop model.ShoppingListItem
	require.NoError(t, db.First(&shop, "id = ?", shopItem.ID).Error)
	require.Equal(t, model.ShoppingStatusPurchased, shop.Status)
	require.NotNil(t, shop.CompletedAt)

	// 幂等重放：不新增支出与流水，数量不变。
	again, err := purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
	require.NoError(t, err)
	require.False(t, again.Created)
	var expensesAfter []model.Expense
	require.NoError(t, db.Find(&expensesAfter).Error)
	require.Len(t, expensesAfter, 1)
	var rowsAfter []model.InventoryTransaction
	require.NoError(t, db.Find(&rowsAfter).Error)
	require.Len(t, rowsAfter, 2)
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 5, refreshed.Quantity)
}

// TestPurchaseConfirmConcurrent 覆盖并发确认只落账一次。
func TestPurchaseConfirmConcurrent(t *testing.T) {
	db, purchaseSvc, _, item, _, familyID, userID := purchaseFixture(t)
	ctx := context.Background()
	importID := createImportWithLines(t, purchaseSvc, familyID, userID, []app.DraftLineInput{
		{Name: "皇家成猫粮", Specification: "2kg", Quantity: 1, Unit: "袋", LineAmount: f64(99), IncludeInStock: true, MatchedInventoryItemID: item.ID},
	})

	results := make([]*app.ConfirmSummary, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errs[index] = purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
		}(i)
	}
	wg.Wait()

	createdCount := 0
	for i := range results {
		require.NoError(t, errs[i])
		require.NotNil(t, results[i])
		if results[i].Created {
			createdCount++
		}
	}
	require.Equal(t, 1, createdCount, "并发确认只有一次真正落账")

	var expenses []model.Expense
	require.NoError(t, db.Find(&expenses).Error)
	require.Len(t, expenses, 1)
	var rows []model.InventoryTransaction
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	var refreshed model.InventoryItem
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 4, refreshed.Quantity)
}

// TestAdjustInventoryAtomic 覆盖手动调整：数量与流水同事务、负数拒绝。
func TestAdjustInventoryAtomic(t *testing.T) {
	db, _, shoppingSvc, item, _, familyID, userID := purchaseFixture(t)
	ctx := context.Background()

	_, err := shoppingSvc.AdjustInventory(ctx, familyID, userID, item.ID, app.InventoryAdjustInput{Delta: -1})
	require.NoError(t, err)
	var refreshed model.InventoryItem
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 2, refreshed.Quantity)
	var rows []model.InventoryTransaction
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, -1, rows[0].Change)
	require.Equal(t, model.TxReasonManualAdjustment, rows[0].SourceType)

	// 超量出库被拒绝，且不产生流水、数量不变。
	_, err = shoppingSvc.AdjustInventory(ctx, familyID, userID, item.ID, app.InventoryAdjustInput{Delta: -10})
	require.Error(t, err)
	var rowsAfter []model.InventoryTransaction
	require.NoError(t, db.Find(&rowsAfter).Error)
	require.Len(t, rowsAfter, 1)
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 2, refreshed.Quantity)

	// 跨家庭资源不可见。
	_, err = shoppingSvc.AdjustInventory(ctx, ulid.Make().String(), userID, item.ID, app.InventoryAdjustInput{Delta: 1})
	require.Error(t, err)
}

// TestOpeningBalanceMigrationBackfill 覆盖迁移为已有库存补录期初流水且幂等。
func TestOpeningBalanceMigrationBackfill(t *testing.T) {
	db := SetupTestDB(t)
	now := time.Now().UTC()
	familyID, userID := ulid.Make().String(), ulid.Make().String()
	itemID := ulid.Make().String()
	require.NoError(t, db.Create(&model.InventoryItem{
		Base:     model.Base{ID: itemID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID: familyID,
		Name:     "期初库存",
		Quantity: 7,
	}).Error)

	// 再次执行全部迁移（幂等）：为已有库存补一条期初流水，数量不变。
	require.NoError(t, applyMigrations(db, t))
	var rows []model.InventoryTransaction
	require.NoError(t, db.Where("item_id = ?", itemID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, model.TxReasonOpeningBalance, rows[0].SourceType)
	require.Equal(t, 7, rows[0].Change)

	// 重复迁移不会补第二条。
	require.NoError(t, applyMigrations(db, t))
	var rowsAgain []model.InventoryTransaction
	require.NoError(t, db.Where("item_id = ?", itemID).Find(&rowsAgain).Error)
	require.Len(t, rowsAgain, 1)

	var item model.InventoryItem
	require.NoError(t, db.First(&item, "id = ?", itemID).Error)
	require.Equal(t, 7, item.Quantity)
}

// TestPurchaseConfirmRejectsForeignInventoryMatch 覆盖匹配到其他家庭库存项被拒绝。
func TestPurchaseConfirmRejectsForeignInventoryMatch(t *testing.T) {
	db, purchaseSvc, _, _, _, familyID, userID := purchaseFixture(t)
	ctx := context.Background()

	foreign := &model.InventoryItem{
		Base:     model.Base{ID: ulid.Make().String(), CreatedBy: ulid.Make().String(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		FamilyID: ulid.Make().String(),
		Name:     "别人的猫粮",
		Quantity: 1,
	}
	require.NoError(t, db.Create(foreign).Error)

	importID := createImportWithLines(t, purchaseSvc, familyID, userID, []app.DraftLineInput{
		{Name: "别人的猫粮", Quantity: 1, LineAmount: f64(10), IncludeInStock: true, MatchedInventoryItemID: foreign.ID},
	})
	_, err := purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
	require.Error(t, err)
}

// TestConfirmCancelledImportRejected 覆盖已取消导入不可确认，且不残留落账。
func TestConfirmCancelledImportRejected(t *testing.T) {
	db, purchaseSvc, _, item, _, familyID, userID := purchaseFixture(t)
	ctx := context.Background()
	importID := createImportWithLines(t, purchaseSvc, familyID, userID, []app.DraftLineInput{
		{Name: "皇家成猫粮", Quantity: 1, LineAmount: f64(9), IncludeInStock: true, MatchedInventoryItemID: item.ID},
	})
	first, err := purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
	require.NoError(t, err)
	require.True(t, first.Created)
	var refreshed model.InventoryItem
	require.NoError(t, db.First(&refreshed, "id = ?", item.ID).Error)
	require.Equal(t, 4, refreshed.Quantity)

	// 取消后再次确认：冲突，不重复落账。
	var imp model.PurchaseImport
	require.NoError(t, db.First(&imp, "id = ?", importID).Error)
	imp.Status = model.PurchaseStatusCancelled
	require.NoError(t, db.Save(&imp).Error)
	_, err = purchaseSvc.Confirm(ctx, familyID, userID, importID, app.PurchaseConfirmInput{})
	require.Error(t, err)
	var expenses []model.Expense
	require.NoError(t, db.Find(&expenses).Error)
	require.Len(t, expenses, 1)
}
