package app

import (
	"context"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
)

type purchaseTestEnv struct {
	service   *PurchaseService
	purchases *fakePurchase
	inventory *fakeInventory
	shopping  *fakeShopping
	ocr       *fakeOCR
}

func newPurchaseTestEnv() *purchaseTestEnv {
	members := &fakeMembers{familyIDs: []string{"fam-1"}}
	purchases := newFakePurchase()
	inventory := newFakeInventory()
	shopping := newFakeShopping()
	svc := NewPurchaseService(purchases, inventory, shopping, members)
	ocr := &fakeOCR{}
	svc.SetOCRProvider(ocr)
	svc.SetMediaReader(func(_ context.Context, _ string) ([]byte, error) {
		return []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, nil
	})
	return &purchaseTestEnv{service: svc, purchases: purchases, inventory: inventory, shopping: shopping, ocr: ocr}
}

func (e *purchaseTestEnv) seedImport(t *testing.T, status string, total int64) *model.PurchaseImport {
	t.Helper()
	now := time.Now().UTC()
	imp := &model.PurchaseImport{
		Base:        model.Base{ID: "imp-1", CreatedBy: "u1", CreatedAt: now, UpdatedAt: now},
		FamilyID:    "fam-1",
		Status:      status,
		MediaKeys:   []string{"purchase-imports/x/1.png"},
		TotalAmount: total,
		Currency:    "CNY",
	}
	if err := e.purchases.Create(context.Background(), imp); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	return imp
}

func (e *purchaseTestEnv) seedLines(t *testing.T, lines []*model.PurchaseImportLine) {
	t.Helper()
	if err := e.purchases.ReplaceLines(context.Background(), "fam-1", "imp-1", lines); err != nil {
		t.Fatalf("seed lines: %v", err)
	}
}

func makeLine(name string, qty int, includeInStock bool, amount int64, matchedInventory, matchedShopping string) *model.PurchaseImportLine {
	line := &model.PurchaseImportLine{
		Base:           model.Base{ID: "line-" + name, CreatedBy: "u1", CreatedAt: time.Now().UTC()},
		FamilyID:       "fam-1",
		ImportID:       "imp-1",
		Name:           name,
		Quantity:       qty,
		IncludeInStock: includeInStock,
		Confidence:     model.ConfidenceHigh,
	}
	if amount > 0 {
		c := amount
		line.LineAmount = &c
	}
	if matchedInventory != "" {
		line.MatchedInventoryItemID = &matchedInventory
	}
	if matchedShopping != "" {
		line.MatchedShoppingItemID = &matchedShopping
	}
	return line
}

// TestCreateImportValidation 覆盖截图数量校验。
func TestCreateImportValidation(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	if _, err := env.service.CreateImport(ctx, "fam-1", "u1", nil); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("empty images should be rejected, got %v", err)
	}
	tooMany := []string{"a", "b", "c", "d", "e", "f"}
	if _, err := env.service.CreateImport(ctx, "fam-1", "u1", tooMany); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("6 images should be rejected, got %v", err)
	}
	if _, err := env.service.CreateImport(ctx, "fam-2", "u1", []string{"a"}); appErrorCode(t, err) != apperr.CodeFamilyForbidden {
		t.Fatalf("cross-family create should be forbidden, got %v", err)
	}
}

// TestRecognizeSuccess 覆盖识别成功写入草稿。
func TestRecognizeSuccess(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusUploaded, 0)
	env.ocr.raw = `{"platform":"taobao","order_date":"2026-10-21","total_amount":199.00,"lines":[{"name":"皇家成猫粮","specification":"2kg","quantity":1,"unit":"袋","line_amount":199.00,"confidence":"high"}]}`

	result, err := env.service.Recognize(ctx, "fam-1", "u1", "imp-1")
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if !result.Success || len(result.Issues) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if result.Import.Status != model.PurchaseStatusParsed {
		t.Fatalf("status = %q", result.Import.Status)
	}
	if result.Import.RecognitionVersion != 1 || len(result.Import.Lines) != 1 {
		t.Fatalf("import = %+v", result.Import)
	}
	if result.Import.Lines[0].Name != "皇家成猫粮" || result.Import.Lines[0].Quantity != 1 {
		t.Fatalf("line = %+v", result.Import.Lines[0])
	}
}

// TestRecognizeFailureKeepsImportForRetry 覆盖识别失败时保留记录并可重试。
func TestRecognizeFailureKeepsImportForRetry(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusUploaded, 0)
	env.ocr.err = apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 服务超时")

	result, err := env.service.Recognize(ctx, "fam-1", "u1", "imp-1")
	if err != nil {
		t.Fatalf("recognize should return result, not error: %v", err)
	}
	if result.Success || result.Import.Status != model.PurchaseStatusFailed || result.Import.ErrorSummary == "" {
		t.Fatalf("failed result = %+v", result.Import)
	}
	// 记录保留，仍可读取。
	imp, err := env.service.GetImport(ctx, "fam-1", "u1", "imp-1")
	if err != nil || imp.Status != model.PurchaseStatusFailed {
		t.Fatalf("import lost after failure: %v", err)
	}
	// 重试成功。
	env.ocr.err = nil
	env.ocr.raw = `{"platform":"jd","total_amount":10,"lines":[{"name":"猫砂","quantity":1}]}`
	retry, err := env.service.Recognize(ctx, "fam-1", "u1", "imp-1")
	if err != nil || !retry.Success || retry.Import.Status != model.PurchaseStatusParsed {
		t.Fatalf("retry = %+v, err = %v", retry, err)
	}
}

// TestRecognizeUnparsableOutput 覆盖模型输出无法解析时进入 failed。
func TestRecognizeUnparsableOutput(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusUploaded, 0)
	env.ocr.raw = "抱歉，我识别不出来"

	result, err := env.service.Recognize(ctx, "fam-1", "u1", "imp-1")
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if result.Success || result.Import.Status != model.PurchaseStatusFailed {
		t.Fatalf("result = %+v", result)
	}
}

// TestRecognizeStateGuards 覆盖已确认导入不可再识别、处理中冲突。
func TestRecognizeStateGuards(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusConfirmed, 0)
	if _, err := env.service.Recognize(ctx, "fam-1", "u1", "imp-1"); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("confirmed import must be CONFLICT, got %v", err)
	}
	env2 := newPurchaseTestEnv()
	env2.seedImport(t, model.PurchaseStatusProcessing, 0)
	if _, err := env2.service.Recognize(ctx, "fam-1", "u1", "imp-1"); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("processing import must be CONFLICT, got %v", err)
	}
}

// TestUpdateDraftVersionConflict 覆盖草稿版本冲突。
func TestUpdateDraftVersionConflict(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusPartial, 0)
	stale := 0
	total := 99.5
	amount := 99.5
	imp, err := env.service.UpdateDraft(ctx, "fam-1", "u1", "imp-1", PurchaseDraftInput{
		ExpectedVersion: &stale,
		Platform:        "淘宝",
		OrderDate:       "2026-10-21",
		TotalAmount:     &total,
		Lines: []DraftLineInput{
			{Name: "冻干", Quantity: 1, LineAmount: &amount, IncludeInStock: true, Confidence: "high"},
		},
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if imp.Status != model.PurchaseStatusParsed || imp.RecognitionVersion != 1 || len(imp.Lines) != 1 {
		t.Fatalf("import = %+v", imp)
	}
	if imp.Platform != "taobao" || imp.TotalAmount != 99.5 {
		t.Fatalf("platform/total = %q/%.2f", imp.Platform, imp.TotalAmount)
	}
	// 用旧版本再次保存：冲突。
	if _, err := env.service.UpdateDraft(ctx, "fam-1", "u1", "imp-1", PurchaseDraftInput{ExpectedVersion: &stale}); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("stale version must be CONFLICT, got %v", err)
	}
	// 已确认的导入不可编辑。
	env2 := newPurchaseTestEnv()
	env2.seedImport(t, model.PurchaseStatusConfirmed, 0)
	if _, err := env2.service.UpdateDraft(ctx, "fam-1", "u1", "imp-1", PurchaseDraftInput{Platform: "taobao"}); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("confirmed draft edit must be CONFLICT, got %v", err)
	}
}

// TestConfirmRecordsEverythingOnce 覆盖确认落账：支出、流水、待购完成只执行一次。
func TestConfirmRecordsEverythingOnce(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 10000)
	// 待购项：猫砂（关联行）。
	shoppingItem := &model.ShoppingListItem{
		Base:     model.Base{ID: "shop-1", CreatedBy: "u1"},
		FamilyID: "fam-1",
		Name:     "猫砂",
		Category: "litter",
		Quantity: 1,
		Status:   model.ShoppingStatusTodo,
	}
	_ = env.shopping.Create(ctx, shoppingItem)
	env.seedLines(t, []*model.PurchaseImportLine{
		makeLine("猫砂", 2, true, 5000, "", "shop-1"),
		makeLine("冻干", 1, true, 5000, "", ""),
		makeLine("玩具", 1, false, 0, "", ""),
	})

	summary, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if !summary.Created || summary.Import.Status != model.PurchaseStatusConfirmed {
		t.Fatalf("summary = %+v", summary)
	}
	if len(env.purchases.expenses) != 1 {
		t.Fatalf("expenses = %d, want 1", len(env.purchases.expenses))
	}
	expense := env.purchases.expenses[0]
	if expense.Amount != 10000 || expense.SourceType != model.TxReasonPurchaseImport || expense.SourceID != "imp-1" {
		t.Fatalf("expense = %+v", expense)
	}
	if len(env.purchases.txRows) != 2 {
		t.Fatalf("stock transactions = %d, want 2（只入库勾选行）", len(env.purchases.txRows))
	}
	if len(summary.ShoppingCompleted) != 1 || summary.ShoppingCompleted[0] != "shop-1" {
		t.Fatalf("shopping completed = %v", summary.ShoppingCompleted)
	}
	if summary.ExpenseAmount != 100 {
		t.Fatalf("expense amount = %v", summary.ExpenseAmount)
	}

	// 幂等重放：再次确认不新增支出与流水。
	again, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{})
	if err != nil {
		t.Fatalf("re-confirm: %v", err)
	}
	if again.Created || len(env.purchases.expenses) != 1 || len(env.purchases.txRows) != 2 {
		t.Fatalf("idempotent replay wrote again: expenses=%d txs=%d created=%v", len(env.purchases.expenses), len(env.purchases.txRows), again.Created)
	}
}

// TestConfirmWithoutExpense 覆盖关闭记账时只更新库存与待购。
func TestConfirmWithoutExpense(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 10000)
	env.seedLines(t, []*model.PurchaseImportLine{makeLine("猫粮", 1, true, 10000, "", "")})

	includeExpense := false
	summary, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{IncludeExpense: &includeExpense})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(env.purchases.expenses) != 0 {
		t.Fatalf("expenses = %d, want 0", len(env.purchases.expenses))
	}
	if summary.ExpenseID != "" || summary.ExpenseAmount != 0 {
		t.Fatalf("expense fields = %q/%v", summary.ExpenseID, summary.ExpenseAmount)
	}
	if len(env.purchases.txRows) != 1 {
		t.Fatalf("tx rows = %d, want 1", len(env.purchases.txRows))
	}
}

// TestConfirmCrossFamilyMatchRejected 覆盖匹配到其他家庭资源时不可见。
func TestConfirmCrossFamilyMatchRejected(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 5000)
	foreignItem := &model.InventoryItem{Base: model.Base{ID: "inv-2"}, FamilyID: "fam-2", Name: "别人家的猫粮", Quantity: 1}
	_ = env.inventory.Create(ctx, foreignItem)
	env.seedLines(t, []*model.PurchaseImportLine{makeLine("猫粮", 1, true, 5000, "inv-2", "")})

	if _, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{}); appErrorCode(t, err) != apperr.CodeNotFound {
		t.Fatalf("cross-family inventory match must be NOT_FOUND, got %v", err)
	}
}

// TestConfirmNothingToRecord 覆盖没有可落账内容时拒绝。
func TestConfirmNothingToRecord(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 0)
	if _, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{}); appErrorCode(t, err) != apperr.CodeValidationFailed {
		t.Fatalf("empty confirm must be VALIDATION_FAILED, got %v", err)
	}
}

// TestConfirmCancelledImport 覆盖已取消导入不可确认。
func TestConfirmCancelledImport(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusCancelled, 100)
	if _, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{}); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("cancelled import must be CONFLICT, got %v", err)
	}
}

// TestDeleteMediaAndCancel 覆盖原图删除与草稿取消。
func TestDeleteMediaAndCancel(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 100)
	// 两张截图：删除第一张后引用只剩一张，删除器收到原键。
	imp := env.purchases.imports["imp-1"]
	imp.MediaKeys = []string{"purchase-imports/x/1.png", "purchase-imports/x/2.png"}
	var removed []string
	env.service.SetMediaRemover(func(_ context.Context, key string) error {
		removed = append(removed, key)
		return nil
	})
	saved, err := env.service.DeleteMedia(ctx, "fam-1", "u1", "imp-1", 0)
	if err != nil {
		t.Fatalf("delete media: %v", err)
	}
	if len(saved.MediaKeys) != 1 || saved.MediaKeys[0] != "purchase-imports/x/2.png" {
		t.Fatalf("media keys = %v", saved.MediaKeys)
	}
	if len(removed) != 1 || removed[0] != "purchase-imports/x/1.png" {
		t.Fatalf("removed = %v", removed)
	}
	if _, err := env.service.DeleteMedia(ctx, "fam-1", "u1", "imp-1", 5); appErrorCode(t, err) != apperr.CodeNotFound {
		t.Fatalf("out-of-range media should be NOT_FOUND, got %v", err)
	}

	// 未确认草稿可取消；已确认不可取消。
	cancelled, err := env.service.Cancel(ctx, "fam-1", "u1", "imp-1")
	if err != nil || cancelled.Status != model.PurchaseStatusCancelled {
		t.Fatalf("cancel = %+v, err = %v", cancelled, err)
	}
	// 幂等：重复取消返回相同状态。
	again, err := env.service.Cancel(ctx, "fam-1", "u1", "imp-1")
	if err != nil || again.Status != model.PurchaseStatusCancelled {
		t.Fatalf("second cancel = %+v, err = %v", again, err)
	}

	env2 := newPurchaseTestEnv()
	env2.seedImport(t, model.PurchaseStatusConfirmed, 100)
	if _, err := env2.service.Cancel(ctx, "fam-1", "u1", "imp-1"); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("confirmed cancel should be CONFLICT, got %v", err)
	}
}

// TestConfirmRepoConflictMapsToConflict 覆盖仓储层冲突映射。
func TestConfirmRepoConflictMapsToConflict(t *testing.T) {
	env := newPurchaseTestEnv()
	ctx := context.Background()
	env.seedImport(t, model.PurchaseStatusParsed, 100)
	env.purchases.failConfirm = repository.ErrConflict
	env.seedLines(t, []*model.PurchaseImportLine{makeLine("猫粮", 1, true, 100, "", "")})
	if _, err := env.service.Confirm(ctx, "fam-1", "u1", "imp-1", PurchaseConfirmInput{}); appErrorCode(t, err) != apperr.CodeConflict {
		t.Fatalf("repo conflict must map to CONFLICT, got %v", err)
	}
}
