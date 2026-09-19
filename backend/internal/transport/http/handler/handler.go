// Package handler 实现 HTTP 处理层（认证、家庭、猫咪）。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// Handler 聚合所有业务 handler。
type Handler struct {
	auth     *app.AuthService
	family   *app.FamilyService
	cat      *app.CatService
	member   *app.MemberService
	record   *app.RecordService
	care     *app.CareService
	reminder *app.ReminderService
	asset    *app.AssetService
	ai       *app.AIService
}

// New 创建 handler 集合。
func New(
	auth *app.AuthService,
	family *app.FamilyService,
	cat *app.CatService,
	member *app.MemberService,
	record *app.RecordService,
	care *app.CareService,
	reminder *app.ReminderService,
	asset *app.AssetService,
	ai *app.AIService,
) *Handler {
	return &Handler{
		auth:     auth,
		family:   family,
		cat:      cat,
		member:   member,
		record:   record,
		care:     care,
		reminder: reminder,
		asset:    asset,
		ai:       ai,
	}
}

// --- Auth Handlers ---

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	UserName   string `json:"user_name" binding:"required,min=1,max=64"`
	FamilyName string `json:"family_name,omitempty"`
}

// Register 用户注册。
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	result, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.UserName)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, result)
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录。
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// RefreshRequest 刷新请求体。
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh 刷新令牌。
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// Logout 登出。
func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// Me 当前用户（含家庭归属与角色，供前端解析 familyId）。
func (h *Handler) Me(c *gin.Context) {
	res, err := h.member.Me(c.Request.Context(), mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res)
}

// ListMyFamilies 列出当前用户所属家庭。
func (h *Handler) ListMyFamilies(c *gin.Context) {
	list, err := h.member.ListMyFamilies(c.Request.Context(), mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// mustUserID 取出认证中间件注入的 user_id。
func mustUserID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	s, _ := v.(string)
	return s
}

// --- Family Handlers ---

// FamilyCreateRequest 创建家庭请求体。
type FamilyCreateRequest struct {
	Name     string `json:"name" binding:"required,min=1"`
	Timezone string `json:"timezone,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// CreateFamily 创建家庭。
func (h *Handler) CreateFamily(c *gin.Context) {
	var req FamilyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	userID := mustUserID(c)
	f, err := h.family.CreateFamily(c.Request.Context(), req.Name, req.Timezone, req.Currency, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, gin.H{
		"id":       f.ID,
		"name":     f.Name,
		"timezone": f.Timezone,
		"currency": f.Currency,
	})
}

// GetFamily 获取家庭详情。
func (h *Handler) GetFamily(c *gin.Context) {
	familyID := c.Param("familyId")
	userID := mustUserID(c)
	f, err := h.family.GetFamily(c.Request.Context(), familyID, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, gin.H{
		"id":       f.ID,
		"name":     f.Name,
		"timezone": f.Timezone,
		"currency": f.Currency,
	})
}

// FamilyUpdateRequest 更新家庭请求体。
type FamilyUpdateRequest struct {
	Name     string `json:"name,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// UpdateFamily 更新家庭。
func (h *Handler) UpdateFamily(c *gin.Context) {
	var req FamilyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	familyID := c.Param("familyId")
	userID := mustUserID(c)
	f, err := h.family.UpdateFamily(c.Request.Context(), familyID, userID, req.Name, req.Timezone, req.Currency)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, gin.H{
		"id":       f.ID,
		"name":     f.Name,
		"timezone": f.Timezone,
		"currency": f.Currency,
	})
}

// ListMembers 成员列表。
func (h *Handler) ListMembers(c *gin.Context) {
	familyID := c.Param("familyId")
	userID := mustUserID(c)
	members, err := h.family.ListMembers(c.Request.Context(), familyID, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	result := make([]gin.H, len(members))
	for i, m := range members {
		result[i] = gin.H{
			"id":        m.ID,
			"family_id": m.FamilyID,
			"user_id":   m.UserID,
			"user_name": m.UserName,
			"role":      m.Role,
		}
	}
	response.OK(c, result)
}

// --- Cat Handlers ---

// CatCreateRequest 创建猫咪请求体。
type CatCreateRequest struct {
	Name      string   `json:"name" binding:"required,min=1,max=64"`
	Breed     string   `json:"breed,omitempty"`
	Gender    string   `json:"gender,omitempty"`
	BirthDate string   `json:"birth_date,omitempty"`
	Neutered  bool     `json:"neutered"`
	Diseases  []string `json:"diseases,omitempty"`
	Allergies []string `json:"allergies,omitempty"`
}

// CreateCat 创建猫咪。
func (h *Handler) CreateCat(c *gin.Context) {
	var req CatCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	familyID := c.Param("familyId")
	userID := mustUserID(c)
	cat, err := h.cat.CreateCat(c.Request.Context(), familyID, req.Name, req.Breed, req.Gender, req.BirthDate, req.Neutered, req.Diseases, req.Allergies, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, catToEnvelope(cat))
}

// GetCat 获取猫咪详情。
func (h *Handler) GetCat(c *gin.Context) {
	catID := c.Param("catId")
	userID := mustUserID(c)
	cat, err := h.cat.GetCat(c.Request.Context(), catID, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, catToEnvelope(cat))
}

// UpdateCat 更新猫咪。
func (h *Handler) UpdateCat(c *gin.Context) {
	var req CatUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	catID := c.Param("catId")
	userID := mustUserID(c)
	cat, err := h.cat.UpdateCat(c.Request.Context(), catID, userID, req.Name, req.Breed, req.Gender, req.BirthDate, req.Neutered)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, catToEnvelope(cat))
}

// CatUpdateRequest 更新猫咪请求体。
type CatUpdateRequest struct {
	Name      string `json:"name,omitempty"`
	Breed     string `json:"breed,omitempty"`
	Gender    string `json:"gender,omitempty"`
	BirthDate string `json:"birth_date,omitempty"`
	Neutered  *bool  `json:"neutered,omitempty"`
}

// DeleteCat 删除猫咪。
func (h *Handler) DeleteCat(c *gin.Context) {
	catID := c.Param("catId")
	userID := mustUserID(c)
	if err := h.cat.DeleteCat(c.Request.Context(), catID, userID); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// ListCats 猫咪列表。
func (h *Handler) ListCats(c *gin.Context) {
	familyID := c.Param("familyId")
	userID := mustUserID(c)
	cats, err := h.cat.ListCats(c.Request.Context(), familyID, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	result := make([]map[string]any, len(cats))
	for i, cat := range cats {
		result[i] = catToEnvelope(cat)
	}
	response.OK(c, result)
}

// catToEnvelope 转换为 JSON 响应格式。
func catToEnvelope(c *model.Cat) map[string]any {
	envelope := map[string]any{
		"id":         c.ID,
		"family_id":  c.FamilyID,
		"name":       c.Name,
		"breed":      c.Breed,
		"gender":     c.Gender,
		"birthday":   c.Birthday,
		"birth_date": c.Birthday,
		"neutered":   c.Neutered,
		"diseases":   c.Diseases,
		"allergies":  c.Allergies,
	}
	if c.AvatarKey != "" {
		envelope["avatar_key"] = c.AvatarKey
	}
	return envelope
}
