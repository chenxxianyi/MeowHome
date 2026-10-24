package handler

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/app"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// maxPurchaseImages 单次采购导入允许的最大截图数。
const maxPurchaseImages = 5

// purchaseImportJSONInput 以媒体键创建导入的请求体（小程序逐张上传后使用）。
type purchaseImportJSONInput struct {
	MediaKeys []string `json:"media_keys"`
}

// UploadPurchaseMedia 上传单张订单截图，返回私有媒体键。
// 微信小程序 wx.uploadFile 只支持单文件，故客户端逐张上传后以 media_keys 建单。
func (h *Handler) UploadPurchaseMedia(c *gin.Context) {
	if h.storageDir == "" {
		response.Err(c, apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "截图存储暂不可用"))
		return
	}
	familyID := c.Param("familyId")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes+(1<<20))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Err(c, apperr.InvalidRequest(apperr.CodeInvalidRequest, "请选择一张订单截图"))
		return
	}
	key, err := h.savePurchaseMedia(familyID, fileHeader)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, gin.H{"media_key": key})
}

// CreatePurchaseImport 上传订单截图并创建导入草稿。
// 支持 multipart（files 字段，1–5 张）或 JSON（media_keys，逐张上传后的键）。
// 截图只写入私有媒体存储；不写库存或支出。
func (h *Handler) CreatePurchaseImport(c *gin.Context) {
	if h.storageDir == "" {
		response.Err(c, apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "截图存储暂不可用"))
		return
	}
	familyID := c.Param("familyId")
	userID := mustUserID(c)

	var keys []string
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes*maxPurchaseImages+(1<<20))
		form, err := c.MultipartForm()
		if err != nil {
			response.Err(c, apperr.InvalidRequest(apperr.CodeInvalidRequest, "请以 multipart 形式上传订单截图"))
			return
		}
		files := form.File["files"]
		if len(files) == 0 {
			response.Err(c, apperr.InvalidRequest(apperr.CodeValidationFailed, "请至少选择一张订单截图"))
			return
		}
		if len(files) > maxPurchaseImages {
			response.Err(c, apperr.InvalidRequest(apperr.CodeValidationFailed, "单次最多上传 5 张订单截图"))
			return
		}
		cleanup := func() {
			for _, key := range keys {
				_ = os.Remove(filepath.Join(h.storageDir, filepath.FromSlash(key)))
			}
		}
		for _, fileHeader := range files {
			key, err := h.savePurchaseMedia(familyID, fileHeader)
			if err != nil {
				cleanup()
				response.Err(c, err)
				return
			}
			keys = append(keys, key)
		}
	} else {
		var in purchaseImportJSONInput
		if err := c.ShouldBindJSON(&in); err != nil {
			response.Err(c, apperr.InvalidRequest(apperr.CodeInvalidJSON, "请提供 media_keys 或上传截图"))
			return
		}
		keys = in.MediaKeys
	}

	result, err := h.purchase.CreateImport(c.Request.Context(), familyID, userID, keys)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, result)
}

// savePurchaseMedia 校验并保存一张订单截图到私有存储，返回对象键。
func (h *Handler) savePurchaseMedia(familyID string, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader.Size > h.maxUploadBytes {
		return "", apperr.InvalidRequest(apperr.CodeUploadTooLarge, "单张订单截图不能超过上传大小限制")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", apperr.InvalidRequest(apperr.CodeInvalidRequest, "无法读取所选截图")
	}
	defer file.Close()
	imageBytes, err := io.ReadAll(io.LimitReader(file, h.maxUploadBytes+1))
	if err != nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "读取截图失败")
	}
	if int64(len(imageBytes)) > h.maxUploadBytes {
		return "", apperr.InvalidRequest(apperr.CodeUploadTooLarge, "单张订单截图不能超过上传大小限制")
	}
	ext, ok := avatarExtension(imageBytes)
	if !ok {
		return "", apperr.InvalidRequest(apperr.CodeInvalidRequest, "订单截图仅支持 JPG、PNG、GIF 或 WebP 图片")
	}

	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "生成截图文件名失败")
	}
	fileName := hex.EncodeToString(randomName) + ext
	dirSegment := opaqueSegment(familyID)
	key := path.Join("purchase-imports", dirSegment, fileName)
	directory := filepath.Join(h.storageDir, "purchase-imports", dirSegment)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "创建截图存储目录失败")
	}
	filePath := filepath.Join(directory, fileName)
	if err := os.WriteFile(filePath, imageBytes, 0o644); err != nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeStorageUnavailable, "保存截图失败")
	}
	return key, nil
}

// GetPurchaseImport 查询导入状态、草稿与商品行。
func (h *Handler) GetPurchaseImport(c *gin.Context) {
	imp, err := h.purchase.GetImport(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, imp)
}

// ListPurchaseImports 列出采购导入历史。
func (h *Handler) ListPurchaseImports(c *gin.Context) {
	limit := 30
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	list, err := h.purchase.ListImports(c.Request.Context(), c.Param("familyId"), mustUserID(c), limit)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// RecognizePurchaseImport 启动或重试识别；失败时导入记录保留为 failed。
func (h *Handler) RecognizePurchaseImport(c *gin.Context) {
	result, err := h.purchase.Recognize(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// UpdatePurchaseDraft 保存用户修订后的订单草稿。
func (h *Handler) UpdatePurchaseDraft(c *gin.Context) {
	var in app.PurchaseDraftInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	imp, err := h.purchase.UpdateDraft(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, imp)
}

// ConfirmPurchaseImport 幂等确认采购并落账。
func (h *Handler) ConfirmPurchaseImport(c *gin.Context) {
	var in app.PurchaseConfirmInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	summary, err := h.purchase.Confirm(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, summary)
}

// PurchaseImportMedia 读取导入关联的原始截图（按索引）。
func (h *Handler) PurchaseImportMedia(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		response.Err(c, apperr.NotFound(apperr.CodeNotFound, "screenshot not found"))
		return
	}
	data, mime, err := h.purchase.ReadMedia(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"), index)
	if err != nil {
		response.Err(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, mime, data)
}

// DeletePurchaseMedia 删除导入的一张原始截图（隐私：允许用户立即删除原图）。
func (h *Handler) DeletePurchaseMedia(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		response.Err(c, apperr.NotFound(apperr.CodeNotFound, "screenshot not found"))
		return
	}
	imp, err := h.purchase.DeleteMedia(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"), index)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, imp)
}

// CancelPurchaseImport 取消未确认的导入草稿。
func (h *Handler) CancelPurchaseImport(c *gin.Context) {
	imp, err := h.purchase.Cancel(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("importId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, imp)
}
