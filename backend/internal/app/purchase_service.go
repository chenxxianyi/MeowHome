package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/platform/id"
)

// OCRProvider 订单截图识别提供方；返回模型输出原文，结构化解析在本包完成。
type OCRProvider interface {
	RecognizeOrder(ctx context.Context, images [][]byte) (string, error)
}

// MediaReader 读取私有媒体内容（存储对象键 → 字节）。
type MediaReader func(ctx context.Context, key string) ([]byte, error)

// MediaRemover 删除私有媒体内容；文件已不存在不算失败。
type MediaRemover func(ctx context.Context, key string) error

// PurchaseService 覆盖订单截图导入、识别、草稿修订与确认落账。
type PurchaseService struct {
	imports     repository.PurchaseImportRepo
	inventory   repository.InventoryRepo
	shopping    repository.ShoppingListRepo
	members     repository.MemberRepo
	ocr         OCRProvider
	readMedia   MediaReader
	removeMedia MediaRemover
}

// NewPurchaseService 创建采购导入服务。
func NewPurchaseService(
	imports repository.PurchaseImportRepo,
	inventory repository.InventoryRepo,
	shopping repository.ShoppingListRepo,
	members repository.MemberRepo,
) *PurchaseService {
	return &PurchaseService{imports: imports, inventory: inventory, shopping: shopping, members: members}
}

// SetOCRProvider 安装订单截图识别提供方。
func (s *PurchaseService) SetOCRProvider(provider OCRProvider) { s.ocr = provider }

// SetMediaReader 安装私有媒体读取器（识别与回看原图都经过它）。
func (s *PurchaseService) SetMediaReader(reader MediaReader) { s.readMedia = reader }

// SetMediaRemover 安装私有媒体删除器（原图删除与保留期限）。
func (s *PurchaseService) SetMediaRemover(remover MediaRemover) { s.removeMedia = remover }

// PurchaseLineEnvelope 采购商品行响应。
type PurchaseLineEnvelope struct {
	ID                     string   `json:"id"`
	RawTitle               string   `json:"raw_title,omitempty"`
	Name                   string   `json:"name"`
	Specification          string   `json:"specification,omitempty"`
	Quantity               int      `json:"quantity"`
	Unit                   string   `json:"unit,omitempty"`
	LineAmount             *float64 `json:"line_amount,omitempty"`
	Confidence             string   `json:"confidence"`
	Evidence               string   `json:"evidence,omitempty"`
	IncludeInStock         bool     `json:"include_in_stock"`
	MatchedInventoryItemID string   `json:"matched_inventory_item_id,omitempty"`
	MatchedShoppingItemID  string   `json:"matched_shopping_item_id,omitempty"`
}

// PurchaseImportEnvelope 采购导入响应。
type PurchaseImportEnvelope struct {
	ID                 string                  `json:"id"`
	Status             string                  `json:"status"`
	Platform           string                  `json:"platform,omitempty"`
	OrderDate          string                  `json:"order_date,omitempty"`
	OrderNo            string                  `json:"order_no,omitempty"`
	TotalAmount        float64                 `json:"total_amount"`
	Currency           string                  `json:"currency"`
	MediaKeys          []string                `json:"media_keys"`
	RecognitionVersion int                     `json:"recognition_version"`
	ErrorSummary       string                  `json:"error_summary,omitempty"`
	ConfirmedAt        string                  `json:"confirmed_at,omitempty"`
	ExpenseID          string                  `json:"expense_id,omitempty"`
	Lines              []*PurchaseLineEnvelope `json:"lines"`
	CreatedAt          string                  `json:"created_at,omitempty"`
}

func purchaseLineEnvelope(line *model.PurchaseImportLine) *PurchaseLineEnvelope {
	env := &PurchaseLineEnvelope{
		ID:             line.ID,
		RawTitle:       line.RawTitle,
		Name:           line.Name,
		Specification:  line.Specification,
		Quantity:       line.Quantity,
		Unit:           line.Unit,
		Confidence:     line.Confidence,
		IncludeInStock: line.IncludeInStock,
	}
	if line.LineAmount != nil {
		yuan := float64(*line.LineAmount) / 100
		env.LineAmount = &yuan
	}
	if line.MatchedInventoryItemID != nil {
		env.MatchedInventoryItemID = *line.MatchedInventoryItemID
	}
	if line.MatchedShoppingItemID != nil {
		env.MatchedShoppingItemID = *line.MatchedShoppingItemID
	}
	return env
}

func purchaseImportEnvelope(imp *model.PurchaseImport, lines []*model.PurchaseImportLine) *PurchaseImportEnvelope {
	env := &PurchaseImportEnvelope{
		ID:                 imp.ID,
		Status:             imp.Status,
		Platform:           imp.Platform,
		OrderDate:          imp.OrderDate,
		OrderNo:            imp.OrderNo,
		TotalAmount:        float64(imp.TotalAmount) / 100,
		Currency:           imp.Currency,
		MediaKeys:          imp.MediaKeys,
		RecognitionVersion: imp.RecognitionVersion,
		ErrorSummary:       imp.ErrorSummary,
		ExpenseID:          imp.ExpenseID,
		Lines:              make([]*PurchaseLineEnvelope, 0, len(lines)),
		CreatedAt:          imp.CreatedAt.Format(time.RFC3339),
	}
	for _, line := range lines {
		env.Lines = append(env.Lines, purchaseLineEnvelope(line))
	}
	if imp.ConfirmedAt != nil {
		env.ConfirmedAt = imp.ConfirmedAt.Format(time.RFC3339)
	}
	return env
}

// maxImportImages 单次导入最多截图数。
const maxImportImages = 5

// CreateImport 创建采购导入（仅保存原始截图引用，不写库存或支出）。
func (s *PurchaseService) CreateImport(ctx context.Context, familyID, userID string, mediaKeys []string) (*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if len(mediaKeys) == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "at least one screenshot is required")
	}
	if len(mediaKeys) > maxImportImages {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed,
			fmt.Sprintf("at most %d screenshots per import", maxImportImages))
	}
	now := time.Now().UTC()
	imp := &model.PurchaseImport{
		Base:        model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:    familyID,
		Status:      model.PurchaseStatusUploaded,
		MediaKeys:   append([]string(nil), mediaKeys...),
		TotalAmount: 0,
		Currency:    "CNY",
	}
	if err := s.imports.Create(ctx, imp); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create purchase import", err)
	}
	return purchaseImportEnvelope(imp, nil), nil
}

// GetImport 查询导入状态、草稿与商品行。
func (s *PurchaseService) GetImport(ctx context.Context, familyID, userID, importID string) (*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, lines, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	return purchaseImportEnvelope(imp, lines), nil
}

// ListImports 列出家庭采购导入历史。
func (s *PurchaseService) ListImports(ctx context.Context, familyID, userID string, limit int) ([]*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.imports.ListByFamily(ctx, familyID, limit)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list purchase imports", err)
	}
	out := make([]*PurchaseImportEnvelope, 0, len(rows))
	for _, row := range rows {
		out = append(out, purchaseImportEnvelope(row, nil))
	}
	return out, nil
}

// ReadMedia 读取导入关联的原始截图（按索引），供预览使用。
func (s *PurchaseService) ReadMedia(ctx context.Context, familyID, userID, importID string, index int) ([]byte, string, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, "", err
	}
	imp, err := s.imports.FindByID(ctx, familyID, importID)
	if err != nil {
		return nil, "", mapImportRepoError(err)
	}
	if index < 0 || index >= len(imp.MediaKeys) {
		return nil, "", errors.NotFound(errors.CodeNotFound, "screenshot not found")
	}
	if s.readMedia == nil {
		return nil, "", errors.New(errors.TypeExternal, errors.CodeStorageUnavailable,
			"截图存储暂不可用")
	}
	data, err := s.readMedia(ctx, imp.MediaKeys[index])
	if err != nil {
		return nil, "", errors.New(errors.TypeExternal, errors.CodeStorageUnavailable,
			"读取截图失败")
	}
	return data, detectImageMIME(data), nil
}

// DeleteMedia 删除导入的一张原始截图并更新媒体引用。
// 订单截图可能含地址/电话，方案要求允许用户立即删除原图。
func (s *PurchaseService) DeleteMedia(ctx context.Context, familyID, userID, importID string, index int) (*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, lines, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(imp.MediaKeys) {
		return nil, errors.NotFound(errors.CodeNotFound, "screenshot not found")
	}
	key := imp.MediaKeys[index]
	keys := make([]string, 0, len(imp.MediaKeys)-1)
	keys = append(keys, imp.MediaKeys[:index]...)
	keys = append(keys, imp.MediaKeys[index+1:]...)
	imp.MediaKeys = keys
	if err := s.imports.UpdateImport(ctx, imp); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update screenshots", err)
	}
	if s.removeMedia != nil {
		// 文件删除失败不回滚引用：记录已删，存储残留由清理任务处理。
		_ = s.removeMedia(ctx, key)
	}
	return purchaseImportEnvelope(imp, lines), nil
}

// Cancel 取消未确认的导入草稿；已确认的采购不可取消（通过冲正流程处理）。
func (s *PurchaseService) Cancel(ctx context.Context, familyID, userID, importID string) (*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, lines, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	switch imp.Status {
	case model.PurchaseStatusConfirmed:
		return nil, errors.Conflict(errors.CodeConflict, "已确认的采购不能取消，请使用冲正")
	case model.PurchaseStatusProcessing:
		return nil, errors.Conflict(errors.CodeConflict, "识别进行中，请稍后再取消")
	case model.PurchaseStatusCancelled:
		return purchaseImportEnvelope(imp, lines), nil
	}
	imp.Status = model.PurchaseStatusCancelled
	imp.UpdatedAt = time.Now().UTC()
	if err := s.imports.UpdateImport(ctx, imp); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to cancel import", err)
	}
	return purchaseImportEnvelope(imp, lines), nil
}

// RecognizeResult 识别结果。
type RecognizeResult struct {
	Import  *PurchaseImportEnvelope `json:"import"`
	Issues  []string                `json:"issues,omitempty"`
	Success bool                    `json:"success"`
}

// Recognize 启动或重试识别。识别失败时保留导入记录并返回 failed 状态与错误摘要。
func (s *PurchaseService) Recognize(ctx context.Context, familyID, userID, importID string) (*RecognizeResult, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, _, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	switch imp.Status {
	case model.PurchaseStatusUploaded, model.PurchaseStatusFailed,
		model.PurchaseStatusParsed, model.PurchaseStatusPartial:
		// 可识别或重试。
	case model.PurchaseStatusProcessing:
		return nil, errors.Conflict(errors.CodeConflict, "recognition is already running")
	default:
		return nil, errors.Conflict(errors.CodeConflict, "import is finalized and cannot be recognized again")
	}

	// 标记处理中：请求期间再次发起识别会被上面的状态守卫拒绝。
	imp.Status = model.PurchaseStatusProcessing
	imp.ErrorSummary = ""
	if err := s.imports.UpdateImport(ctx, imp); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update import status", err)
	}

	images, err := s.readMediaKeys(ctx, imp.MediaKeys)
	if err == nil && s.ocr == nil {
		err = errors.New(errors.TypeExternal, errors.CodeOCRUnavailable, "OCR 未配置，请手动填写订单信息")
	}
	if err == nil {
		var raw string
		raw, err = s.ocr.RecognizeOrder(ctx, images)
		if err == nil {
			result := s.applyRecognition(ctx, familyID, userID, imp, raw)
			return result, nil
		}
	}

	// 失败路径：保留记录、记录错误摘要、状态置为 failed，允许重试。
	failed := asAppError(err, errors.CodeOCRFailed, "识别失败")
	imp.Status = model.PurchaseStatusFailed
	imp.ErrorSummary = truncateRunes(failed.Message, 500)
	imp.UpdatedAt = time.Now().UTC()
	_ = s.imports.UpdateImport(ctx, imp)
	lines, _ := s.imports.ListLines(ctx, familyID, imp.ID)
	return &RecognizeResult{Import: purchaseImportEnvelope(imp, lines), Issues: []string{failed.Message}, Success: false}, nil
}

// applyRecognition 解析模型输出并写入草稿。
func (s *PurchaseService) applyRecognition(ctx context.Context, familyID, userID string, imp *model.PurchaseImport, raw string) *RecognizeResult {
	draft, err := parseOCRJSON(raw)
	if err != nil {
		failed := errors.New(errors.TypeExternal, errors.CodeOCRFailed, "识别结果无法解析："+err.Error())
		imp.Status = model.PurchaseStatusFailed
		imp.ErrorSummary = truncateRunes(failed.Message, 500)
		imp.UpdatedAt = time.Now().UTC()
		_ = s.imports.UpdateImport(ctx, imp)
		lines, _ := s.imports.ListLines(ctx, familyID, imp.ID)
		return &RecognizeResult{Import: purchaseImportEnvelope(imp, lines), Issues: []string{failed.Message}, Success: false}
	}
	issues := validateDraft(draft)
	now := time.Now().UTC()
	imp.Platform = draft.Platform
	imp.OrderDate = draft.OrderDate
	imp.OrderNo = draft.OrderNo
	imp.Currency = draft.Currency
	if draft.TotalAmount != nil {
		imp.TotalAmount = *draft.TotalAmount
	}
	imp.Status = draftStatus(draft)
	imp.RecognitionVersion++
	imp.ErrorSummary = truncateRunes(strings.Join(issues, "；"), 500)
	imp.UpdatedAt = now
	if err := s.imports.ReplaceLines(ctx, familyID, imp.ID, draftToLines(draft, familyID, userID, imp.ID, now)); err != nil {
		imp.Status = model.PurchaseStatusFailed
		imp.ErrorSummary = "保存识别结果失败"
		_ = s.imports.UpdateImport(ctx, imp)
		return &RecognizeResult{Import: purchaseImportEnvelope(imp, nil), Issues: []string{"保存识别结果失败"}, Success: false}
	}
	if err := s.imports.UpdateImport(ctx, imp); err != nil {
		return &RecognizeResult{Import: purchaseImportEnvelope(imp, nil), Issues: []string{"保存识别结果失败"}, Success: false}
	}
	lines, _ := s.imports.ListLines(ctx, familyID, imp.ID)
	return &RecognizeResult{Import: purchaseImportEnvelope(imp, lines), Issues: issues, Success: true}
}

// DraftLineInput 用户修订后的商品行。
type DraftLineInput struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Specification          string   `json:"specification"`
	Quantity               int      `json:"quantity"`
	Unit                   string   `json:"unit"`
	LineAmount             *float64 `json:"line_amount"`
	Confidence             string   `json:"confidence"`
	Evidence               string   `json:"evidence"`
	IncludeInStock         bool     `json:"include_in_stock"`
	MatchedInventoryItemID string   `json:"matched_inventory_item_id"`
	MatchedShoppingItemID  string   `json:"matched_shopping_item_id"`
}

// PurchaseDraftInput 保存草稿的输入。
type PurchaseDraftInput struct {
	ExpectedVersion *int             `json:"expected_version"`
	Platform        string           `json:"platform"`
	OrderDate       string           `json:"order_date"`
	TotalAmount     *float64         `json:"total_amount"`
	Currency        string           `json:"currency"`
	Lines           []DraftLineInput `json:"lines"`
}

// UpdateDraft 保存用户修订后的订单草稿。
// expected_version 提供时必须与当前版本一致，否则返回 409（草稿版本冲突）。
func (s *PurchaseService) UpdateDraft(ctx context.Context, familyID, userID, importID string, in PurchaseDraftInput) (*PurchaseImportEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, existing, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	switch imp.Status {
	case model.PurchaseStatusProcessing:
		return nil, errors.Conflict(errors.CodeConflict, "recognition is running; wait for it to finish")
	case model.PurchaseStatusConfirmed, model.PurchaseStatusCancelled:
		return nil, errors.Conflict(errors.CodeConflict, "import is finalized and cannot be edited")
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != imp.RecognitionVersion {
		return nil, errors.Conflict(errors.CodeConflict, "draft version conflict; reload before saving")
	}
	if len(in.Lines) > 100 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "too many product lines")
	}

	existingByID := make(map[string]*model.PurchaseImportLine, len(existing))
	for _, line := range existing {
		existingByID[line.ID] = line
	}
	now := time.Now().UTC()
	lines := make([]*model.PurchaseImportLine, 0, len(in.Lines))
	for i, input := range in.Lines {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		quantity := input.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		if quantity > 10000 {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed,
				fmt.Sprintf("line %d quantity is out of range", i+1))
		}
		line := &model.PurchaseImportLine{
			Base:                   model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
			FamilyID:               familyID,
			ImportID:               imp.ID,
			LineNo:                 i + 1,
			Name:                   name,
			Specification:          strings.TrimSpace(input.Specification),
			Quantity:               quantity,
			Unit:                   strings.TrimSpace(input.Unit),
			Confidence:             normalizeConfidence(input.Confidence),
			Evidence:               strings.TrimSpace(input.Evidence),
			IncludeInStock:         input.IncludeInStock,
			MatchedInventoryItemID: optionalString(input.MatchedInventoryItemID),
			MatchedShoppingItemID:  optionalString(input.MatchedShoppingItemID),
		}
		if previous, ok := existingByID[input.ID]; ok && input.ID != "" {
			// 沿用行 ID：允许客户端在修订间保持行标识稳定。
			line.Base = previous.Base
			line.Base.UpdatedAt = now
		}
		if input.LineAmount != nil {
			cents := int64(*input.LineAmount*100 + 0.5)
			if cents < 0 {
				cents = 0
			}
			line.LineAmount = &cents
		}
		lines = append(lines, line)
	}

	if in.Platform != "" {
		imp.Platform = normalizePlatform(in.Platform)
	}
	if in.OrderDate != "" {
		if normalized := normalizeOrderDate(in.OrderDate); normalized != "" {
			imp.OrderDate = normalized
		}
	}
	if in.TotalAmount != nil {
		cents := int64(*in.TotalAmount*100 + 0.5)
		if cents < 0 {
			cents = 0
		}
		imp.TotalAmount = cents
	}
	if in.Currency != "" {
		imp.Currency = normalizeCurrency(in.Currency)
	}
	imp.RecognitionVersion++
	if len(lines) > 0 {
		imp.Status = model.PurchaseStatusParsed
	} else {
		imp.Status = model.PurchaseStatusPartial
	}
	imp.ErrorSummary = ""
	imp.UpdatedAt = now
	if err := s.imports.ReplaceLines(ctx, familyID, imp.ID, lines); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to save draft lines", err)
	}
	if err := s.imports.UpdateImport(ctx, imp); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to save draft", err)
	}
	saved, _ := s.imports.ListLines(ctx, familyID, imp.ID)
	return purchaseImportEnvelope(imp, saved), nil
}

// PurchaseConfirmInput 确认采购的输入。
type PurchaseConfirmInput struct {
	ExpectedVersion *int  `json:"expected_version"`
	IncludeExpense  *bool `json:"include_expense"`
}

// ConfirmSummary 确认采购后的落账摘要。
type ConfirmSummary struct {
	Import            *PurchaseImportEnvelope `json:"import"`
	Created           bool                    `json:"created"`
	ExpenseID         string                  `json:"expense_id,omitempty"`
	ExpenseAmount     float64                 `json:"expense_amount,omitempty"`
	Transactions      []*TxEnvelope           `json:"transactions"`
	ShoppingCompleted []string                `json:"shopping_completed"`
}

// Confirm 幂等确认采购：支出、库存流水、库存数量与待购完成在同一事务内落账。
// 同一导入重复确认不重复写入，返回 created=false 的相同摘要。
func (s *PurchaseService) Confirm(ctx context.Context, familyID, userID, importID string, in PurchaseConfirmInput) (*ConfirmSummary, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	imp, lines, err := s.loadImport(ctx, familyID, importID)
	if err != nil {
		return nil, err
	}
	if imp.Status == model.PurchaseStatusProcessing {
		return nil, errors.Conflict(errors.CodeConflict, "recognition is running; wait for it to finish")
	}
	if imp.Status == model.PurchaseStatusCancelled {
		return nil, errors.Conflict(errors.CodeConflict, "import was cancelled")
	}
	if imp.Status == model.PurchaseStatusConfirmed {
		// 幂等重放：不重复落账，返回现有摘要。
		return &ConfirmSummary{
			Import:    purchaseImportEnvelope(imp, lines),
			Created:   false,
			ExpenseID: imp.ExpenseID,
		}, nil
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != imp.RecognitionVersion {
		return nil, errors.Conflict(errors.CodeConflict, "draft version conflict; reload before confirming")
	}
	if len(lines) == 0 && imp.TotalAmount == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "nothing to record: add product lines or a paid amount")
	}

	includeExpense := true
	if in.IncludeExpense != nil {
		includeExpense = *in.IncludeExpense
	}

	now := time.Now().UTC()
	plan := repository.PurchaseConfirmPlan{}
	summary := &ConfirmSummary{}

	// 逐行：入库（既有关联库存项则复用，否则新建），并完成匹配的待购项。
	var expenseAmount int64
	shoppingByLine := make(map[string]*model.ShoppingListItem)
	targetItemByLine := make(map[string]string)
	for _, line := range lines {
		if line.Quantity <= 0 {
			continue
		}
		var matchedShopping *model.ShoppingListItem
		if line.MatchedShoppingItemID != nil {
			shoppingItem, err := s.shopping.FindByID(ctx, familyID, *line.MatchedShoppingItemID)
			if err != nil || shoppingItem.FamilyID != familyID {
				return nil, errors.NotFound(errors.CodeNotFound, "matched shopping item not found")
			}
			matchedShopping = shoppingItem
		}

		var targetItemID string
		var newItem *model.InventoryItem
		if line.IncludeInStock {
			if line.MatchedInventoryItemID != nil {
				item, err := s.inventory.FindByID(ctx, *line.MatchedInventoryItemID)
				if err != nil || item.FamilyID != familyID || item.DeletedAt != nil {
					return nil, errors.NotFound(errors.CodeNotFound, "matched inventory item not found")
				}
				targetItemID = item.ID
			} else {
				newItem = &model.InventoryItem{
					Base:          model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
					FamilyID:      familyID,
					Name:          line.Name,
					Specification: line.Specification,
					Unit:          line.Unit,
				}
				if matchedShopping != nil {
					newItem.Category = matchedShopping.Category
				}
				if newItem.Category == "" {
					newItem.Category = "supplies"
				}
				targetItemID = newItem.ID
			}
			// 同一计划行同时携带（可选的新库存项）与入库流水；流水 ItemID 已预置。
			tx := &model.InventoryTransaction{
				Base:       model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
				FamilyID:   familyID,
				ItemID:     targetItemID,
				Change:     line.Quantity,
				Reason:     model.TxReasonPurchaseImport,
				SourceType: model.TxReasonPurchaseImport,
				SourceID:   line.ID,
			}
			plan.StockLines = append(plan.StockLines, repository.PurchaseStockLine{NewItem: newItem, Tx: tx})
			summary.Transactions = append(summary.Transactions, txEnvelope(tx))
		}
		if matchedShopping != nil && matchedShopping.Status != model.ShoppingStatusPurchased {
			shoppingByLine[matchedShopping.ID] = matchedShopping
			if targetItemID != "" {
				targetItemByLine[matchedShopping.ID] = targetItemID
			}
		}
		if line.LineAmount != nil {
			expenseAmount += *line.LineAmount
		}
	}

	// 待购完成：一行只能完成一次（按待购项去重）。
	for id, shoppingItem := range shoppingByLine {
		if targetID, ok := targetItemByLine[id]; ok {
			shoppingItem.MatchedInventoryItemID = &targetID
		}
		completedAt := now
		shoppingItem.CompletedAt = &completedAt
		shoppingItem.Status = model.ShoppingStatusPurchased
		plan.ShoppingCompletions = append(plan.ShoppingCompletions, shoppingItem)
		summary.ShoppingCompleted = append(summary.ShoppingCompleted, id)
	}

	// 订单支出：以实付总额为准（识别出的行金额合计仅作校验，不作为记账来源）。
	if includeExpense && imp.TotalAmount > 0 {
		label := "采购订单"
		if imp.Platform != "" {
			label += " · " + platformLabel(imp.Platform)
		}
		if imp.OrderDate != "" {
			label += " · " + imp.OrderDate
		}
		occurredOn := imp.OrderDate
		if occurredOn == "" {
			occurredOn = now.Format("2006-01-02")
		}
		plan.Expense = &model.Expense{
			Base:       model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
			FamilyID:   familyID,
			OccurredOn: occurredOn,
			Amount:     imp.TotalAmount,
			Category:   "pet_supplies",
			Label:      label,
			SourceType: model.TxReasonPurchaseImport,
			SourceID:   imp.ID,
		}
		expenseAmount = imp.TotalAmount
	}

	if len(plan.StockLines) == 0 && len(plan.ShoppingCompletions) == 0 && plan.Expense == nil {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "nothing to record: select at least one product line to include or a paid amount")
	}

	saved, created, err := s.imports.Confirm(ctx, familyID, userID, imp.ID, plan)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			// 并发确认已在另一事务落账：读取结果并按幂等重放返回。
			current, loadErr := s.imports.FindByID(ctx, familyID, imp.ID)
			if loadErr != nil {
				return nil, mapImportRepoError(loadErr)
			}
			savedLines, _ := s.imports.ListLines(ctx, familyID, imp.ID)
			return &ConfirmSummary{
				Import:    purchaseImportEnvelope(current, savedLines),
				Created:   false,
				ExpenseID: current.ExpenseID,
			}, nil
		}
		switch err {
		case repository.ErrNotFound:
			return nil, errors.NotFound(errors.CodeNotFound, "purchase import not found")
		case repository.ErrConflict:
			return nil, errors.Conflict(errors.CodeConflict, "import cannot be confirmed in its current state")
		default:
			return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to confirm purchase", err)
		}
	}
	savedLines, _ := s.imports.ListLines(ctx, familyID, imp.ID)
	summary.Import = purchaseImportEnvelope(saved, savedLines)
	summary.Created = created
	if saved.ExpenseID != "" {
		summary.ExpenseID = saved.ExpenseID
	}
	if plan.Expense != nil && created {
		summary.ExpenseAmount = float64(expenseAmount) / 100
	}
	return summary, nil
}

// loadImport 读取导入与其商品行。
func (s *PurchaseService) loadImport(ctx context.Context, familyID, importID string) (*model.PurchaseImport, []*model.PurchaseImportLine, error) {
	imp, err := s.imports.FindByID(ctx, familyID, importID)
	if err != nil {
		return nil, nil, mapImportRepoError(err)
	}
	lines, err := s.imports.ListLines(ctx, familyID, importID)
	if err != nil {
		return nil, nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load import lines", err)
	}
	return imp, lines, nil
}

// readMediaKeys 读取导入关联的全部截图；任一读取失败即整体失败。
func (s *PurchaseService) readMediaKeys(ctx context.Context, keys []string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "import has no screenshots")
	}
	if s.readMedia == nil {
		return nil, errors.New(errors.TypeExternal, errors.CodeStorageUnavailable, "截图存储暂不可用")
	}
	images := make([][]byte, 0, len(keys))
	for _, key := range keys {
		data, err := s.readMedia(ctx, key)
		if err != nil {
			return nil, errors.New(errors.TypeExternal, errors.CodeStorageUnavailable, "读取截图失败")
		}
		images = append(images, data)
	}
	return images, nil
}

// mapImportRepoError 把仓储错误映射为业务错误。
func mapImportRepoError(err error) error {
	if err == repository.ErrNotFound {
		return errors.NotFound(errors.CodeNotFound, "purchase import not found")
	}
	return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "purchase import unavailable", err)
}

// asAppError 把任意错误规整为 AppError。
func asAppError(err error, fallbackCode, fallbackMessage string) *errors.AppError {
	if appErr, ok := err.(*errors.AppError); ok && appErr != nil {
		return appErr
	}
	return errors.New(errors.TypeExternal, fallbackCode, fallbackMessage)
}

// truncateRunes 按字符截断字符串，避免破坏多字节字符。
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// platformLabel 返回平台展示名。
func platformLabel(platform string) string {
	switch platform {
	case "taobao":
		return "淘宝"
	case "jd":
		return "京东"
	case "pdd":
		return "拼多多"
	default:
		return platform
	}
}

// detectImageMIME 由文件头推断图片 MIME。
func detectImageMIME(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
