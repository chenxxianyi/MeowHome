package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// --- 记录 ---

// ListRecords 查询家庭记录。查询参数：cat_id / type（可重复或逗号分隔）/ from / to / limit。
func (h *Handler) ListRecords(c *gin.Context) {
	q := repository.DailyRecordQuery{
		CatID: c.Query("cat_id"),
		From:  c.Query("from"),
		To:    c.Query("to"),
	}
	if raw := c.Query("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			response.Err(c, apperr.InvalidRequest(apperr.CodeValidationFailed, "offset must be a non-negative integer"))
			return
		}
		q.Offset = offset
	}
	if raw := c.Query("type"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				q.Types = append(q.Types, t)
			}
		}
	}
	if n, err := strconv.Atoi(c.Query("limit")); err == nil {
		q.Limit = n
	}

	list, err := h.record.List(c.Request.Context(), c.Param("familyId"), mustUserID(c), q)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// GetRecord 按 ID 获取家庭内未删除的记录。
func (h *Handler) GetRecord(c *gin.Context) {
	record, err := h.record.Get(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("recordId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, record)
}

// UpdateRecord 完整更正记录字段，创建者与创建时间由服务端保留。
func (h *Handler) UpdateRecord(c *gin.Context) {
	var in app.RecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	rec, err := h.record.Update(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("recordId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, rec)
}

// DeleteRecord soft-deletes one record from active queries.
func (h *Handler) DeleteRecord(c *gin.Context) {
	if err := h.record.Delete(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("recordId")); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// CreateRecord 创建单条记录。
func (h *Handler) CreateRecord(c *gin.Context) {
	var in app.RecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	rec, err := h.record.Create(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, rec)
}

// SuggestNextVaccine returns a rule-backed, non-binding follow-up date suggestion.
func (h *Handler) SuggestNextVaccine(c *gin.Context) {
	if h.vaccine == nil {
		response.Err(c, apperr.New(apperr.TypeInternal, apperr.CodeInvalidRequest, "vaccine service is unavailable"))
		return
	}
	suggestion, err := h.vaccine.SuggestNext(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("recordId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, suggestion)
}

type CreateVaccineReminderRequest struct {
	ScheduledDate string `json:"scheduled_date" binding:"required"`
	DateSource    string `json:"date_source" binding:"required"`
	Timezone      string `json:"timezone" binding:"required"`
}

// CreateVaccineReminder creates one user-confirmed, source-linked reminder.
func (h *Handler) CreateVaccineReminder(c *gin.Context) {
	if h.vaccine == nil {
		response.Err(c, apperr.New(apperr.TypeInternal, apperr.CodeInvalidRequest, "vaccine service is unavailable"))
		return
	}
	var in CreateVaccineReminderRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	reminder, err := h.vaccine.CreateReminder(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("recordId"), app.VaccineReminderInput{
		ScheduledDate: in.ScheduledDate, DateSource: in.DateSource, Timezone: in.Timezone,
	})
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, reminder)
}

type CompleteVaccineReminderRequest struct {
	RecordID string `json:"record_id" binding:"required"`
}

// CompleteVaccineReminder links the newly recorded vaccination to the due reminder.
func (h *Handler) CompleteVaccineReminder(c *gin.Context) {
	if h.vaccine == nil {
		response.Err(c, apperr.New(apperr.TypeInternal, apperr.CodeInvalidRequest, "vaccine service is unavailable"))
		return
	}
	var in CompleteVaccineReminderRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	reminder, err := h.vaccine.CompleteReminderWithRecord(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("reminderId"), in.RecordID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, reminder)
}

type UpdateVaccineReminderRequest struct {
	ScheduledDate string `json:"scheduled_date" binding:"required"`
	Timezone      string `json:"timezone" binding:"required"`
}

func (h *Handler) UpdateVaccineReminder(c *gin.Context) {
	if h.vaccine == nil {
		response.Err(c, apperr.New(apperr.TypeInternal, apperr.CodeInvalidRequest, "vaccine service is unavailable"))
		return
	}
	var in UpdateVaccineReminderRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	reminder, err := h.vaccine.UpdateReminderDate(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("reminderId"), in.ScheduledDate, in.Timezone)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, reminder)
}

func (h *Handler) CancelVaccineReminder(c *gin.Context) {
	if h.vaccine == nil {
		response.Err(c, apperr.New(apperr.TypeInternal, apperr.CodeInvalidRequest, "vaccine service is unavailable"))
		return
	}
	if err := h.vaccine.CancelReminder(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("reminderId")); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// CreateRecordBatchRequest 批量创建请求体。
type CreateRecordBatchRequest struct {
	Records []app.RecordInput `json:"records"`
}

// CreateRecordBatch 批量创建记录（供 AI 确认入库使用）。
func (h *Handler) CreateRecordBatch(c *gin.Context) {
	var req CreateRecordBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	list, err := h.record.CreateBatch(c.Request.Context(), c.Param("familyId"), mustUserID(c), req.Records)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, list)
}

// --- 今日状态 / 焦点 / 趋势 ---

// TodayStatus 聚合指定猫咪的今日状态。
func (h *Handler) TodayStatus(c *gin.Context) {
	data, err := h.care.TodayStatus(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("cat_id"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

// FocusItems 需要关注的条目。
func (h *Handler) FocusItems(c *gin.Context) {
	list, err := h.care.FocusItems(c.Request.Context(), c.Param("familyId"), mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// Trends 趋势数据点。
func (h *Handler) Trends(c *gin.Context) {
	days := 30
	if n, err := strconv.Atoi(c.Query("days")); err == nil && n > 0 {
		days = n
	}
	list, err := h.care.Trends(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("cat_id"), days)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// --- 提醒 ---

// ListReminders 列出提醒。查询参数 state=todo|done。
func (h *Handler) ListReminders(c *gin.Context) {
	list, err := h.reminder.List(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("state"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// CreateReminder 创建提醒。
func (h *Handler) CreateReminder(c *gin.Context) {
	var in app.ReminderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	rem, err := h.reminder.Create(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, rem)
}

// CompleteReminder 标记提醒完成。
func (h *Handler) CompleteReminder(c *gin.Context) {
	rem, err := h.reminder.Complete(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("reminderId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, rem)
}
