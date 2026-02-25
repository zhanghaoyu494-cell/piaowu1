// Package handler 实现 Gateway 网关 HTTP 请求处理
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example_shop/gateway/config"
	"example_shop/gateway/middleware"
	"example_shop/gateway/rpc"
	"example_shop/service/customer/kitex_gen/customer"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
)

// ============ 处理器结构体定义 ============

// CustomerHandler 网关客服服务处理?// 作为 HTTP 请求?RPC 服务之间的桥?// 负责处理前端 HTTP 请求并透传/适配到客?RPC 服务
type CustomerHandler struct { // 类型/结构体定?
	client *rpc.CustomerClient // 客服 RPC 客户端，用于调用后端服务
}

// NewCustomerHandler 创建客服处理器实?// 参数:
//   - client: 已初始化?RPC 客户?//
//
// 返回:
//   - *CustomerHandler: 处理器实?
func NewCustomerHandler(client *rpc.CustomerClient) *CustomerHandler {
	return &CustomerHandler{client: client}
}

// ============ 客服信息管理 ============

// GetCustomerService 获取单个客服详细信息
// 请求方式: GET
// 请求参数:
//   - cs_id: 客服ID（必填）
//
// 响应: 客服详细信息，包括姓名、部门、状态等
func (h *CustomerHandler) GetCustomerService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	csID := strings.TrimSpace(r.URL.Query().Get("cs_id"))
	if csID == "" {
		respondJSON(w, http.StatusBadRequest, &customer.GetCustomerServiceResp{
			BaseResp: &customer.BaseResp{Code: 400, Msg: "cs_id is required"},
		})
		return
	}

	req := &customer.GetCustomerServiceReq{
		CsId: csID,
	}

	resp, err := h.client.GetCustomerService(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": 500,
			"msg":  "Internal server error: " + err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// ListCustomerService 分页查询客服列表
// 请求方式: GET
// 请求参数:
//   - dept_id: 部门ID（可选，筛选指定部门）
//   - page: 页码（默??//   - page_size: 每页数量（默?0?//
//
// 响应: 客服列表及分页信?
func (h *CustomerHandler) ListCustomerService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	deptID := strings.TrimSpace(r.URL.Query().Get("dept_id"))
	page, pageSize := parsePaginationParams(r, 10)

	req := &customer.ListCustomerServiceReq{
		DeptId:   deptID,
		Page:     page,
		PageSize: pageSize,
	}

	resp, err := h.client.ListCustomerService(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": 500,
			"msg":  "Internal server error: " + err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// ============ 班次配置管理 ============

// CreateShiftConfig 创建班次配置模板
// 请求方式: POST
// 请求体参?
//   - shift_name: 班次名称（必填，?早班"?晚班"?//   - start_time: 开始时间（必填，格?HH:MM:SS?//   - end_time: 结束时间（必填，格式 HH:MM:SS?//   - min_staff: 最少在班人数（必填?=0?//   - is_holiday: 是否节假日班次（0-否，1-是）
//   - create_by: 创建?//
//
// 响应: 创建结果及新班次ID
func (h *CustomerHandler) CreateShiftConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 请求体参数结?
	var body struct {
		ShiftName string `json:"shift_name"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		MinStaff  int32  `json:"min_staff"`
		IsHoliday int8   `json:"is_holiday"`
		CreateBy  string `json:"create_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, &customer.CreateShiftConfigResp{
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"},
		})
		return
	}
	body.ShiftName = strings.TrimSpace(body.ShiftName)
	body.StartTime = strings.TrimSpace(body.StartTime)
	body.EndTime = strings.TrimSpace(body.EndTime)
	body.CreateBy = strings.TrimSpace(body.CreateBy)
	if body.ShiftName == "" || body.StartTime == "" || body.EndTime == "" {
		respondJSON(w, http.StatusBadRequest, &customer.CreateShiftConfigResp{
			BaseResp: &customer.BaseResp{Code: 400, Msg: "shift_name, start_time and end_time are required"},
		})
		return
	}
	if body.MinStaff < 0 {
		respondJSON(w, http.StatusBadRequest, &customer.CreateShiftConfigResp{
			BaseResp: &customer.BaseResp{Code: 400, Msg: "min_staff must be >= 0"},
		})
		return
	}
	if body.IsHoliday != 0 && body.IsHoliday != 1 {
		respondJSON(w, http.StatusBadRequest, &customer.CreateShiftConfigResp{
			BaseResp: &customer.BaseResp{Code: 400, Msg: "is_holiday must be 0 or 1"},
		})
		return
	}

	req := &customer.CreateShiftConfigReq{
		Shift: &customer.ShiftConfig{
			ShiftName: body.ShiftName,
			StartTime: body.StartTime,
			EndTime:   body.EndTime,
			MinStaff:  body.MinStaff,
			IsHoliday: body.IsHoliday,
			CreateBy:  body.CreateBy,
		},
	}
	resp, err := h.client.CreateShiftConfig(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": 500,
			"msg":  "Internal server error: " + err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

// ListShiftConfig 查询班次配置列表
// 接收 GET 请求，根据条件查询班次配置信?
func (h *CustomerHandler) ListShiftConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	isHolidayStr := r.URL.Query().Get("is_holiday")
	shiftName := strings.TrimSpace(r.URL.Query().Get("shift_name"))
	isHoliday := int64(-1)
	if isHolidayStr != "" {
		if v, err := strconv.ParseInt(isHolidayStr, 10, 8); err == nil {
			if v == 0 || v == 1 {
				isHoliday = v
			}
		}
	}

	req := &customer.ListShiftConfigReq{
		IsHoliday: int8(isHoliday),
		ShiftName: shiftName,
	}
	resp, err := h.client.ListShiftConfig(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": 500,
			"msg":  "Internal server error: " + err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

// UpdateShiftConfig 更新班次配置
// 接收 POST 请求，更新现有班次模板的信息
func (h *CustomerHandler) UpdateShiftConfig(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var reqBody struct { // 变量声明
		ShiftID   int64  `json:"shift_id"`   // JSON字段：shift_id
		ShiftName string `json:"shift_name"` // JSON字段：shift_name
		StartTime string `json:"start_time"` // JSON字段：start_time
		EndTime   string `json:"end_time"`   // JSON字段：end_time
		MinStaff  int32  `json:"min_staff"`  // JSON字段：min_staff
		IsHoliday int8   `json:"is_holiday"` // JSON字段：is_holiday
		CreateBy  string `json:"create_by"`  // JSON字段：create_by
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	reqBody.ShiftName = strings.TrimSpace(reqBody.ShiftName)                                                 // 逻辑处理
	reqBody.StartTime = strings.TrimSpace(reqBody.StartTime)                                                 // 逻辑处理
	reqBody.EndTime = strings.TrimSpace(reqBody.EndTime)                                                     // 逻辑处理
	reqBody.CreateBy = strings.TrimSpace(reqBody.CreateBy)                                                   // 逻辑处理
	if reqBody.ShiftID <= 0 || reqBody.ShiftName == "" || reqBody.StartTime == "" || reqBody.EndTime == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "shift_id, shift_name, start_time and end_time are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if reqBody.MinStaff < 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "min_staff must be >= 0"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if reqBody.IsHoliday != 0 && reqBody.IsHoliday != 1 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "is_holiday must be 0 or 1"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.UpdateShiftConfigReq{ // 逻辑处理
		Shift: &customer.ShiftConfig{ // 逻辑处理
			ShiftId:   reqBody.ShiftID,   // 逻辑处理
			ShiftName: reqBody.ShiftName, // 逻辑处理
			StartTime: reqBody.StartTime, // 逻辑处理
			EndTime:   reqBody.EndTime,   // 逻辑处理
			MinStaff:  reqBody.MinStaff,  // 逻辑处理
			IsHoliday: reqBody.IsHoliday, // 逻辑处理
			CreateBy:  reqBody.CreateBy,  // 逻辑处理
		}, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpdateShiftConfig(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DeleteShiftConfig 删除班次配置
// 接收 POST 请求，根?ID 删除特定班次模板
func (h *CustomerHandler) DeleteShiftConfig(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ShiftID int64 `json:"shift_id"` // JSON字段：shift_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.DeleteShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.ShiftID <= 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.DeleteShiftConfigResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "shift_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	resp, err := h.client.DeleteShiftConfig(r.Context(), &customer.DeleteShiftConfigReq{ShiftId: body.ShiftID}) // 调用并接收错?
	if err != nil {                                                                                             // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 排班管理 ============

// AssignSchedule 手动批量分配排班
// 请求方式: POST
// 请求体参?
//   - schedule_date: 排班日期（必填，格式 YYYY-MM-DD?//   - shift_id: 班次ID（必填）
//   - cs_ids: 客服ID列表（必填，批量分配?//   - create_by: 操作?//
//
// 响应: 排班分配结果
func (h *CustomerHandler) AssignSchedule(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 请求体参数结?
	var body struct { // 变量声明
		ScheduleDate string   `json:"schedule_date"` // 排班日期
		ShiftID      int64    `json:"shift_id"`      // 班次ID
		CsIDs        []string `json:"cs_ids"`        // 客服ID列表
		CreateBy     string   `json:"create_by"`     // 创建?
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AssignScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.ScheduleDate = strings.TrimSpace(body.ScheduleDate)                  // 逻辑处理
	body.CreateBy = strings.TrimSpace(body.CreateBy)                          // 逻辑处理
	if body.ScheduleDate == "" || body.ShiftID <= 0 || len(body.CsIDs) == 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AssignScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "schedule_date, shift_id and cs_ids are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if _, err := time.Parse("2006-01-02", body.ScheduleDate); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AssignScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "schedule_date must be YYYY-MM-DD"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.AssignScheduleReq{ // 逻辑处理
		ScheduleDate: body.ScheduleDate, // 逻辑处理
		ShiftId:      body.ShiftID,      // 逻辑处理
		CsIds:        body.CsIDs,        // 逻辑处理
		CreateBy:     body.CreateBy,     // 逻辑处理
	} // 代码块结?
	resp, err := h.client.AssignSchedule(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// AutoSchedule 自动排班
func (h *CustomerHandler) AutoSchedule(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		StartDate  string `json:"start_date"`  // JSON字段：start_date
		EndDate    string `json:"end_date"`    // JSON字段：end_date
		DeptID     string `json:"dept_id"`     // JSON字段：dept_id
		TeamID     string `json:"team_id"`     // JSON字段：team_id
		OperatorID string `json:"operator_id"` // JSON字段：operator_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AutoScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.StartDate = strings.TrimSpace(body.StartDate)   // 逻辑处理
	body.EndDate = strings.TrimSpace(body.EndDate)       // 逻辑处理
	body.OperatorID = strings.TrimSpace(body.OperatorID) // 逻辑处理
	if body.StartDate == "" || body.EndDate == "" {      // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AutoScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "start_date and end_date are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if _, err := time.Parse("2006-01-02", body.StartDate); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AutoScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "start_date must be YYYY-MM-DD"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if _, err := time.Parse("2006-01-02", body.EndDate); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AutoScheduleResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "end_date must be YYYY-MM-DD"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.OperatorID == "" { // 条件判断
		body.OperatorID = "ADMIN" // 逻辑处理
	} // 代码块结?
	req := &customer.AutoScheduleReq{ // 逻辑处理
		StartDate:  body.StartDate,  // 逻辑处理
		EndDate:    body.EndDate,    // 逻辑处理
		DeptId:     body.DeptID,     // 逻辑处理
		TeamId:     body.TeamID,     // 逻辑处理
		OperatorId: body.OperatorID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.AutoSchedule(r.Context(), req) // 调用并接收错?
	if err != nil {                                      // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 请假调班管理 ============

// ApplyLeaveTransfer 申请请假或调?// 请求方式: POST
// 请求体参?
//   - cs_id: 申请人客服ID（必填）
//   - apply_type: 申请类型?-请假?-调班?//   - start_date: 开始日期（必填，格?YYYY-MM-DD?//   - end_date: 结束日期（必填，格式 YYYY-MM-DD?//   - start_period: 开始时段（0-全天?-上午?-下午?//   - end_period: 结束时段
//   - shift_id: 班次ID（调班时必填?//   - target_cs_id: 调班目标客服ID（调班时必填?//   - reason: 申请原因
//
// 响应: 申请提交结果及申请单ID
func (h *CustomerHandler) ApplyLeaveTransfer(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 从token获取操作人信?
	operatorInfo := getOperatorInfoFromContext(r.Context()) // 逻辑处理
	if operatorInfo.ID == "" {                              // 条件判断
		respondJSON(w, http.StatusOK, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 401, Msg: "请先登录"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 请求体参数结?
	var body struct { // 变量声明
		ApplyType   int8   `json:"apply_type"`   // 申请类型?-请假?-调班
		StartDate   string `json:"start_date"`   // 开始日?
		EndDate     string `json:"end_date"`     // 结束日期
		StartPeriod int8   `json:"start_period"` // 开始时?
		EndPeriod   int8   `json:"end_period"`   // 结束时段
		ShiftID     int64  `json:"shift_id"`     // 班次ID
		TargetCsID  string `json:"target_cs_id"` // 调班目标客服ID
		Reason      string `json:"reason"`       // 申请原因
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.StartDate = strings.TrimSpace(body.StartDate)   // 逻辑处理
	body.EndDate = strings.TrimSpace(body.EndDate)       // 逻辑处理
	body.TargetCsID = strings.TrimSpace(body.TargetCsID) // 逻辑处理
	body.Reason = strings.TrimSpace(body.Reason)         // 逻辑处理

	if body.StartDate == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "start_date is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.EndDate == "" { // 条件判断
		body.EndDate = body.StartDate // 逻辑处理
	} // 代码块结?
	if _, err := time.Parse("2006-01-02", body.StartDate); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "start_date must be YYYY-MM-DD"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if _, err := time.Parse("2006-01-02", body.EndDate); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "end_date must be YYYY-MM-DD"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.ApplyType != 0 && body.ApplyType != 1 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "apply_type must be 0 or 1"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.ApplyType == 1 && (body.TargetCsID == "" || body.ShiftID <= 0) { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApplyLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "target_cs_id and shift_id are required for transfer"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 使用token中的操作人信息作为申请人
	req := &customer.ApplyLeaveTransferReq{ // 逻辑处理
		CsId:        operatorInfo.ID,  // 逻辑处理
		ApplyType:   body.ApplyType,   // 逻辑处理
		StartDate:   body.StartDate,   // 逻辑处理
		EndDate:     body.EndDate,     // 逻辑处理
		StartPeriod: body.StartPeriod, // 逻辑处理
		EndPeriod:   body.EndPeriod,   // 逻辑处理
		ShiftId:     body.ShiftID,     // 逻辑处理
		TargetCsId:  body.TargetCsID,  // 逻辑处理
		Reason:      body.Reason,      // 逻辑处理
		TargetDate:  body.StartDate,   // 兼容旧字?
	} // 代码块结?
	resp, err := h.client.ApplyLeaveTransfer(r.Context(), req) // 调用并接收错?
	if err != nil {                                            // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ApproveLeaveTransfer 审批请假调班申请
// 请求方式: POST
// 请求体参?
//   - apply_id: 申请单ID（必填）
//   - approval_status: 审批状态（1-通过?-拒绝?//   - approver_id: 审批人ID（必填）
//   - approver_name: 审批人姓?//   - approval_remark: 审批备注
//
// 响应: 审批结果
func (h *CustomerHandler) ApproveLeaveTransfer(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 从token获取操作人信?
	operatorInfo := getOperatorInfoFromContext(r.Context()) // 逻辑处理
	if operatorInfo.ID == "" {                              // 条件判断
		respondJSON(w, http.StatusOK, &customer.ApproveLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 401, Msg: "请先登录"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 请求体参数结?
	var body struct { // 变量声明
		ApplyID        int64  `json:"apply_id"`        // 申请单ID
		ApprovalStatus int8   `json:"approval_status"` // 审批状?
		ApprovalRemark string `json:"approval_remark"` // 审批备注
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApproveLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.ApplyID <= 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApproveLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "apply_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.ApprovalStatus != 1 && body.ApprovalStatus != 2 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ApproveLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "approval_status must be 1 or 2"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 使用token中的操作人信?
	req := &customer.ApproveLeaveTransferReq{ // 逻辑处理
		ApplyId:        body.ApplyID,        // 逻辑处理
		ApprovalStatus: body.ApprovalStatus, // 逻辑处理
		ApproverId:     operatorInfo.ID,     // 逻辑处理
		ApproverName:   operatorInfo.Name,   // 逻辑处理
		ApprovalRemark: body.ApprovalRemark, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ApproveLeaveTransfer(r.Context(), req) // 调用并接收错?
	if err != nil {                                              // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ApplyChainSwap 提交链式调班申请
func (h *CustomerHandler) ApplyChainSwap(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 从token获取操作人信?
	operatorInfo := getOperatorInfoFromContext(r.Context()) // 逻辑处理
	if operatorInfo.ID == "" {                              // 条件判断
		respondJSON(w, http.StatusOK, map[string]interface{}{"code": 401, "msg": "请先登录"}) // 写回JSON响应
		return                                                                            // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		DeptID string     `json:"dept_id"` // JSON字段：dept_id
		Reason string     `json:"reason"`  // JSON字段：reason
		Items  []struct { // 逻辑处理
			CsID           string `json:"cs_id"`            // JSON字段：cs_id
			FromScheduleID int64  `json:"from_schedule_id"` // JSON字段：from_schedule_id
			ToScheduleID   int64  `json:"to_schedule_id"`   // JSON字段：to_schedule_id
			Step           int32  `json:"step"`             // JSON字段：step
		} `json:"items"` // JSON字段：items
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": "invalid json"}) // 写回JSON响应
		return                                                                                            // 返回结果/结束处理
	} // 代码块结?
	// 使用token中的操作人信息作为申请人
	req := &customer.ApplyChainSwapReq{ // 逻辑处理
		ApplicantId: operatorInfo.ID, // 逻辑处理
		DeptId:      body.DeptID,     // 逻辑处理
		Reason:      body.Reason,     // 逻辑处理
	} // 代码块结?
	for _, it := range body.Items { // 循环处理
		req.Items = append(req.Items, &customer.ChainSwapItem{ // 逻辑处理
			CsId:           it.CsID,           // 逻辑处理
			FromScheduleId: it.FromScheduleID, // 逻辑处理
			ToScheduleId:   it.ToScheduleID,   // 逻辑处理
			Step:           it.Step,           // 逻辑处理
		}) // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ApplyChainSwap(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()}) // 写回JSON响应
		return                                                                                                  // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ApproveChainSwap 审批链式调班申请
func (h *CustomerHandler) ApproveChainSwap(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 从token获取操作人信?
	operatorInfo := getOperatorInfoFromContext(r.Context()) // 逻辑处理
	if operatorInfo.ID == "" {                              // 条件判断
		respondJSON(w, http.StatusOK, map[string]interface{}{"code": 401, "msg": "请先登录"}) // 写回JSON响应
		return                                                                            // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		RequestID      int64  `json:"request_id"`      // JSON字段：request_id
		ApprovalStatus int8   `json:"approval_status"` // JSON字段：approval_status
		ApprovalRemark string `json:"approval_remark"` // JSON字段：approval_remark
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"code": 400, "msg": "invalid json"}) // 写回JSON响应
		return                                                                                            // 返回结果/结束处理
	} // 代码块结?
	// 使用token中的操作人信?
	req := &customer.ApproveChainSwapReq{ // 逻辑处理
		RequestId:      body.RequestID,      // 逻辑处理
		ApprovalStatus: body.ApprovalStatus, // 逻辑处理
		ApproverId:     operatorInfo.ID,     // 逻辑处理
		ApproverName:   operatorInfo.Name,   // 逻辑处理
		ApprovalRemark: body.ApprovalRemark, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ApproveChainSwap(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "msg": err.Error()}) // 写回JSON响应
		return                                                                                                  // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListChainSwap 查询链式调班申请列表
// 请求方式: GET
// 请求参数:
//   - status: 状态筛选（-1=全部, 0=待审? 1=已通过, 2=已拒绝）
//   - keyword: 关键词搜?//   - page: 页码（默??//   - page_size: 每页数量（默?0?
func (h *CustomerHandler) ListChainSwap(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	status := int8(-1)                                                // 逻辑处理
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" { // 条件判断
		n, err := strconv.ParseInt(v, 10, 8) // 调用并接收错?
		if err == nil {                      // 条件判断
			status = int8(n) // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword")) // 逻辑处理

	page, pageSize := parsePaginationParams(r, 20) // 逻辑处理

	req := &customer.ListChainSwapReq{ // 逻辑处理
		Status:   status,   // 逻辑处理
		Keyword:  keyword,  // 逻辑处理
		Page:     page,     // 逻辑处理
		PageSize: pageSize, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ListChainSwap(r.Context(), req) // 调用并接收错?
	if err != nil {                                       // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// GetChainSwap 获取链式调班申请详情
// 请求方式: GET
// 请求参数:
//   - swap_id: 链式调班申请ID（必填）
func (h *CustomerHandler) GetChainSwap(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	swapIDStr := strings.TrimSpace(r.URL.Query().Get("swap_id")) // 逻辑处理
	swapID, err := strconv.ParseInt(swapIDStr, 10, 64)           // 调用并接收错?
	if err != nil || swapID <= 0 {                               // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                   // 依赖导入
			"msg":  "swap_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.GetChainSwapReq{SwapId: swapID}     // 逻辑处理
	resp, err := h.client.GetChainSwap(r.Context(), req) // 调用并接收错?
	if err != nil {                                      // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// GetLeaveTransfer 获取单个请假调班申请详情
// 请求方式: GET
// 请求参数:
//   - apply_id: 申请单ID（必填）
//
// 响应: 申请详情，包括申请人、类型、状态等
func (h *CustomerHandler) GetLeaveTransfer(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	applyIDStr := strings.TrimSpace(r.URL.Query().Get("apply_id")) // 逻辑处理
	applyID, err := strconv.ParseInt(applyIDStr, 10, 64)           // 调用并接收错?
	if err != nil || applyID <= 0 {                                // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.GetLeaveTransferResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "apply_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.GetLeaveTransferReq{ApplyId: applyID}   // 逻辑处理
	resp, err := h.client.GetLeaveTransfer(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListLeaveTransfer 分页查询请假调班申请列表
// 请求方式: GET
// 请求参数:
//   - approval_status: 审批状态筛选（-1-全部?-待审批，1-已通过?-已拒绝）
//   - keyword: 关键词搜索（搜索申请人姓名）
//   - page: 页码（默??//   - page_size: 每页数量（默?0?//
//
// 响应: 申请列表及分页信?
func (h *CustomerHandler) ListLeaveTransfer(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	approvalStatus := int8(-1)                                                 // 逻辑处理
	if v := strings.TrimSpace(r.URL.Query().Get("approval_status")); v != "" { // 条件判断
		n, err := strconv.ParseInt(v, 10, 8) // 调用并接收错?
		if err == nil {                      // 条件判断
			approvalStatus = int8(n) // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword")) // 逻辑处理

	page, pageSize := parsePaginationParams(r, 20) // 逻辑处理

	req := &customer.ListLeaveTransferReq{ // 逻辑处理
		ApprovalStatus: approvalStatus,                       // 逻辑处理
		Keyword:        keyword,                              // 逻辑处理
		Page:           page,                                 // 逻辑处理
		PageSize:       pageSize,                             // 逻辑处理
		OperatorId:     resolveCustomerCsID(r.Context(), ""), // 获取当前操作人ID
	} // 代码块结?
	resp, err := h.client.ListLeaveTransfer(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// GetLeaveAuditLog 获取请假/调班申请的审计日?// 请求方式: GET
// 请求参数:
//   - apply_id: 申请单ID（必填）
//
// 响应: 审计日志列表
func (h *CustomerHandler) GetLeaveAuditLog(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	applyIDStr := strings.TrimSpace(r.URL.Query().Get("apply_id")) // 逻辑处理
	applyID, err := strconv.ParseInt(applyIDStr, 10, 64)           // 调用并接收错?
	if err != nil || applyID <= 0 {                                // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                    // 依赖导入
			"msg":  "apply_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.GetLeaveAuditLogReq{ApplyId: applyID}   // 逻辑处理
	resp, err := h.client.GetLeaveAuditLog(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListScheduleGrid 查询排班表格数据

func (h *CustomerHandler) ListScheduleGrid(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date")) // 逻辑处理
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))     // 逻辑处理
	deptID := strings.TrimSpace(r.URL.Query().Get("dept_id"))       // 逻辑处理
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))       // 逻辑处理
	if startDate == "" || endDate == "" {                           // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ListScheduleGridResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "start_date and end_date are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.ListScheduleGridReq{ // 逻辑处理
		StartDate: startDate, // 逻辑处理
		EndDate:   endDate,   // 逻辑处理
		DeptId:    deptID,    // 逻辑处理
		TeamId:    teamID,    // 逻辑处理
	} // 代码块结?
	// 如果是客服角色，只能查看自己的排?
	operator := getOperatorInfoFromContext(r.Context()) // 逻辑处理
	if operator.Role == "customer_service" {            // 条件判断
		req.CsId = operator.ID // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ListScheduleGrid(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ExportScheduleExcel 导出排班表为Excel文件
// 请求方式: GET
// 请求参数:
//   - start_date: 开始日期（必填，格?YYYY-MM-DD?//   - end_date: 结束日期（必填，格式 YYYY-MM-DD?//   - dept_id: 部门ID（可选）
//   - team_id: 小组ID（可选）
//
// 响应: Excel 文件下载流（Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet?// 文件名格? schedule_开始日期_结束日期.xlsx
func (h *CustomerHandler) ExportScheduleExcel(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 解析查询参数
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date")) // 逻辑处理
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))     // 逻辑处理
	deptID := strings.TrimSpace(r.URL.Query().Get("dept_id"))       // 逻辑处理
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))       // 逻辑处理
	if startDate == "" || endDate == "" {                           // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                                    // 依赖导入
			"msg":  "start_date and end_date are required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	grid, err := h.client.ListScheduleGrid(r.Context(), &customer.ListScheduleGridReq{ // 调用并接收错?
		StartDate: startDate, // 逻辑处理
		EndDate:   endDate,   // 逻辑处理
		DeptId:    deptID,    // 逻辑处理
		TeamId:    teamID,    // 逻辑处理
	}) // 逻辑处理
	if err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if grid == nil || grid.BaseResp == nil || grid.BaseResp.Code != 0 { // 条件判断
		code := 500                              // 逻辑处理
		msg := "export failed"                   // 逻辑处理
		if grid != nil && grid.BaseResp != nil { // 条件判断
			code = int(grid.BaseResp.Code) // 逻辑处理
			msg = grid.BaseResp.Msg        // 逻辑处理
		} // 代码块结?
		respondJSON(w, http.StatusOK, map[string]interface{}{ // 写回JSON响应
			"code": code, // 依赖导入
			"msg":  msg,  // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	shiftNameByID := map[int64]string{} // 逻辑处理
	for _, s := range grid.Shifts {     // 循环处理
		if s == nil { // 条件判断
			continue // 逻辑处理
		} // 代码块结?
		shiftNameByID[s.ShiftId] = s.ShiftName // 逻辑处理
	} // 代码块结?
	cellByKey := map[string]*customer.ScheduleCell{} // 逻辑处理
	for _, c := range grid.Cells {                   // 循环处理
		if c == nil { // 条件判断
			continue // 逻辑处理
		} // 代码块结?
		cellByKey[c.CsId+"|"+c.ScheduleDate] = c // 逻辑处理
	} // 代码块结?
	f := excelize.NewFile()         // 逻辑处理
	sheet := "排班?"                  // 逻辑处理
	f.SetSheetName("Sheet1", sheet) // 逻辑处理

	header := make([]interface{}, 0, 2+len(grid.Dates)) // 逻辑处理
	header = append(header, "客服ID", "客服姓名")             // 逻辑处理
	for _, d := range grid.Dates {                      // 循环处理
		header = append(header, d) // 逻辑处理
	} // 代码块结?
	if err := f.SetSheetRow(sheet, "A1", &header); err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                             // 依赖导入
			"msg":  "export failed: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	for i, cs := range grid.Customers { // 循环处理
		if cs == nil { // 条件判断
			continue // 逻辑处理
		} // 代码块结?
		row := make([]interface{}, 0, 2+len(grid.Dates)) // 逻辑处理
		row = append(row, cs.CsId, cs.CsName)            // 逻辑处理
		for _, d := range grid.Dates {                   // 循环处理
			cell := cellByKey[cs.CsId+"|"+d]      // 逻辑处理
			if cell == nil || cell.ShiftId <= 0 { // 条件判断
				row = append(row, "") // 逻辑处理
				continue              // 逻辑处理
			} // 代码块结?
			name := shiftNameByID[cell.ShiftId] // 逻辑处理
			if name == "" {                     // 条件判断
				name = fmt.Sprintf("班次%d", cell.ShiftId) // 逻辑处理
			} // 代码块结?
			if cell.Status == 1 { // 条件判断
				name += "(请假)" // 逻辑处理
			} else if cell.Status == 2 { // 逻辑处理
				name += "(调班)" // 逻辑处理
			} // 代码块结?
			row = append(row, name) // 逻辑处理
		} // 代码块结?
		addr, _ := excelize.CoordinatesToCellName(1, i+2)        // 逻辑处理
		if err := f.SetSheetRow(sheet, addr, &row); err != nil { // 条件判断
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
				"code": 500,                             // 依赖导入
				"msg":  "export failed: " + err.Error(), // 依赖导入
			}) // 逻辑处理
			return // 返回结果/结束处理
		} // 代码块结?
	} // 代码块结?
	f.SetColWidth(sheet, "A", "A", 14) // 逻辑处理
	f.SetColWidth(sheet, "B", "B", 16) // 逻辑处理
	if len(grid.Dates) > 0 {           // 条件判断
		lastCol, _ := excelize.ColumnNumberToName(2 + len(grid.Dates)) // 逻辑处理
		_ = f.SetColWidth(sheet, "C", lastCol, 14)                     // 逻辑处理
	} // 代码块结?
	var buf bytes.Buffer                  // 变量声明
	if err := f.Write(&buf); err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                             // 依赖导入
			"msg":  "export failed: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	fileName := fmt.Sprintf("schedule_%s_%s.xlsx", startDate, endDate)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK) // 写入HTTP状态码
	_, _ = w.Write(buf.Bytes())  // 逻辑处理
} // 代码块结?
// UpsertScheduleCell 更新/清空排班单元格（shift_id=0 清空?
func (h *CustomerHandler) UpsertScheduleCell(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CsID         string `json:"cs_id"`         // JSON字段：cs_id
		ScheduleDate string `json:"schedule_date"` // JSON字段：schedule_date
		ShiftID      int64  `json:"shift_id"`      // JSON字段：shift_id
		OperatorID   string `json:"operator_id"`   // JSON字段：operator_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpsertScheduleCellResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.CsID = strings.TrimSpace(body.CsID)                 // 逻辑处理
	body.ScheduleDate = strings.TrimSpace(body.ScheduleDate) // 逻辑处理
	body.OperatorID = strings.TrimSpace(body.OperatorID)     // 逻辑处理
	if body.CsID == "" || body.ScheduleDate == "" {          // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpsertScheduleCellResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "cs_id and schedule_date are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.OperatorID == "" { // 条件判断
		body.OperatorID = "ADMIN" // 逻辑处理
	} // 代码块结?
	req := &customer.UpsertScheduleCellReq{ // 逻辑处理
		CsId:         body.CsID,         // 逻辑处理
		ScheduleDate: body.ScheduleDate, // 逻辑处理
		ShiftId:      body.ShiftID,      // 逻辑处理
		OperatorId:   body.OperatorID,   // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpsertScheduleCell(r.Context(), req) // 调用并接收错?
	if err != nil {                                            // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 会话管理 ============

// AssignCustomer 自动分配客服
// 为用户自动分配当前在线且负载最低的客服
// 请求方式: POST
// 请求体参?
//   - user_id: 用户ID
//   - user_nickname: 用户昵称
//   - source: 来源渠道
//
// 响应: 分配的客服信息和新创建的会话ID
func (h *CustomerHandler) AssignCustomer(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		UserID       string `json:"user_id"`       // JSON字段：user_id
		UserNickname string `json:"user_nickname"` // JSON字段：user_nickname
		Source       string `json:"source"`        // JSON字段：source
		CsID         string `json:"cs_id"`         // JSON字段：cs_id (新支?
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.AssignCustomerResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 逻辑对齐：如果指定了 CsID，则调用 CreateConversation 接口以支持手动指定客?
	if body.CsID != "" {
		req := &customer.CreateConversationReq{
			UserId:       body.UserID,
			UserNickname: body.UserNickname,
			Source:       body.Source,
			CsId:         body.CsID,
		}
		resp, err := retryRPC(r.Context(), func(ctx context.Context) (*customer.CreateConversationResp, error) {
			return h.client.CreateConversation(ctx, req)
		})
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"code": 500, "msg": err.Error(),
			})
			return
		}
		// 转换?AssignCustomerResp 格式返回，保证前端兼?
		respondJSON(w, http.StatusOK, &customer.AssignCustomerResp{
			BaseResp: resp.BaseResp,
			CsId:     resp.CsId,
			CsName:   resp.CsName,
			ConvId:   resp.ConvId,
		})
		return
	}

	req := &customer.AssignCustomerReq{ // 逻辑处理
		UserId:       body.UserID,       // 逻辑处理
		UserNickname: body.UserNickname, // 逻辑处理
		Source:       body.Source,       // 逻辑处理
	} // 代码块结?
	resp, err := retryRPC(r.Context(), func(ctx context.Context) (*customer.AssignCustomerResp, error) {
		return h.client.AssignCustomer(ctx, req)
	})
	if err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, &customer.AssignCustomerResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 500, Msg: err.Error()}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListConversation 分页查询当前会话列表
// 查询客服的当前进行中会话（状态为进行中的会话?// 请求方式: GET
// 请求参数:
//   - cs_id: 客服ID（可选，不填则查询全部）
//   - keyword: 关键词搜索（搜索用户昵称/ID?//   - status: 会话状态筛选（-1-全部?-进行中，1-已结束，2-已转接）
//   - page: 页码（默??//   - page_size: 每页数量（默?0?//
//
// 响应: 会话列表及分页信息、未读消息数
func (h *CustomerHandler) ListConversation(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	csID := strings.TrimSpace(r.URL.Query().Get("cs_id"))       // 逻辑处理
	csID = resolveCustomerCsID(r.Context(), csID)               // 逻辑处理
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))  // 逻辑处理
	statusStr := strings.TrimSpace(r.URL.Query().Get("status")) // 逻辑处理

	status := int64(-1)  // 逻辑处理
	if statusStr != "" { // 条件判断
		if v, err := strconv.ParseInt(statusStr, 10, 8); err == nil { // 条件判断
			status = v // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	page, pageSize := parsePaginationParams(r, 20) // 逻辑处理

	req := &customer.ListConversationReq{ // 逻辑处理
		CsId:     csID,         // 逻辑处理
		Keyword:  keyword,      // 逻辑处理
		Status:   int8(status), // 逻辑处理
		Page:     page,         // 逻辑处理
		PageSize: pageSize,     // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ListConversation(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListConversationHistory 查询会话历史记录列表
// 支持分页查询，可按客服ID、关键词、状态筛?// 仅返回已结束或已转接的会话，且必须包含用户发送的消息
func (h *CustomerHandler) ListConversationHistory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	// 仅允?GET 请求
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 解析查询参数
	csID := strings.TrimSpace(r.URL.Query().Get("cs_id"))       // 逻辑处理
	csID = resolveCustomerCsID(r.Context(), csID)               // 逻辑处理
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))  // 逻辑处理
	statusStr := strings.TrimSpace(r.URL.Query().Get("status")) // 逻辑处理

	// 状态参数处理：默认?1（不筛选）
	status := int64(-1)  // 逻辑处理
	if statusStr != "" { // 条件判断
		if v, err := strconv.ParseInt(statusStr, 10, 8); err == nil { // 条件判断
			status = v // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	// 分页参数处理：默认第1页，每页20?
	page, pageSize := parsePaginationParams(r, 20) // 逻辑处理

	// 构建 RPC 请求对象
	req := &customer.ListConversationHistoryReq{ // 逻辑处理
		CsId:     csID,         // 逻辑处理
		Keyword:  keyword,      // 逻辑处理
		Status:   int8(status), // 逻辑处理
		Page:     page,         // 逻辑处理
		PageSize: pageSize,     // 逻辑处理
	} // 代码块结?	// 调用 RPC 服务查询历史会话
	resp, err := h.client.ListConversationHistory(r.Context(), req) // 调用并接收错?
	if err != nil {                                                 // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListConversationMessage 查询会话消息
// 支持分页查询，可指定排序方式（正?倒序?
func (h *CustomerHandler) ListConversationMessage(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	// 仅允?GET 请求
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 校验必填参数 conv_id
	convID := strings.TrimSpace(r.URL.Query().Get("conv_id")) // 逻辑处理
	if convID == "" {                                         // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.ListConversationMessageResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "conv_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 解析分页与排序参?
	orderAscStr := strings.TrimSpace(r.URL.Query().Get("order_asc")) // 逻辑处理

	page, pageSize := parsePaginationParams(r, 50) // 逻辑处理
	orderAsc := int64(0)                           // 默认倒序(0)
	if orderAscStr != "" {                         // 条件判断
		if v, err := strconv.ParseInt(orderAscStr, 10, 8); err == nil { // 条件判断
			orderAsc = v // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	// 构建 RPC 请求对象
	req := &customer.ListConversationMessageReq{ // 逻辑处理
		ConvId:   convID,         // 逻辑处理
		Page:     page,           // 逻辑处理
		PageSize: pageSize,       // 逻辑处理
		OrderAsc: int8(orderAsc), // 逻辑处理
	} // 代码块结?	// 调用 RPC 服务查询消息
	resp, err := h.client.ListConversationMessage(r.Context(), req) // 调用并接收错?
	if err != nil {                                                 // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// SendConversationMessage 发送会话消?// 客服或用户发送会话消息，支持普通消息和快捷回复
// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID（必填）
//   - sender_type: 发送方类型?-用户?-客服?//   - sender_id: 发送方ID
//   - msg_content: 消息内容（必填）
//   - is_quick_reply: 是否快捷回复?-否，1-是）
//   - quick_reply_id: 快捷回复ID（快捷回复时必填?//
//
// 响应: 发送结果及消息ID
func (h *CustomerHandler) SendConversationMessage(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID       string `json:"conv_id"`        // JSON字段：conv_id
		SenderType   int8   `json:"sender_type"`    // JSON字段：sender_type
		SenderID     string `json:"sender_id"`      // JSON字段：sender_id
		MsgContent   string `json:"msg_content"`    // JSON字段：msg_content
		IsQuickReply int8   `json:"is_quick_reply"` // JSON字段：is_quick_reply
		QuickReplyID int64  `json:"quick_reply_id"` // JSON字段：quick_reply_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.SendConversationMessageResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.ConvID = strings.TrimSpace(body.ConvID)         // 逻辑处理
	body.SenderID = strings.TrimSpace(body.SenderID)     // 逻辑处理
	body.MsgContent = strings.TrimSpace(body.MsgContent) // 逻辑处理
	if body.ConvID == "" || body.MsgContent == "" {      // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.SendConversationMessageResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "conv_id and msg_content are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.SenderID == "" { // 条件判断
		body.SenderID = "KF001" // 逻辑处理
	} // 代码块结?
	req := &customer.SendConversationMessageReq{ // 逻辑处理
		ConvId:       body.ConvID,       // 逻辑处理
		SenderType:   body.SenderType,   // 逻辑处理
		SenderId:     body.SenderID,     // 逻辑处理
		MsgContent:   body.MsgContent,   // 逻辑处理
		IsQuickReply: body.IsQuickReply, // 逻辑处理
		QuickReplyId: body.QuickReplyID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.SendConversationMessage(r.Context(), req) // 调用并接收错?
	if err != nil {                                                 // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// CreateConversation 创建会话
// 用户发起新会话，可指定客服或自动分配
// 请求方式: POST
// 请求体参?
//   - user_id: 用户ID（必填）
//   - user_nickname: 用户昵称
//   - source: 来源渠道（APP/Web/H5/WeChat?//   - cs_id: 指定客服ID（可选，为空则自动分配）
//   - first_msg: 首条消息（可选）
//
// 响应: 会话eID、客服信息、是否新创建
func (h *CustomerHandler) CreateConversation(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		UserID       string `json:"user_id"`       // JSON字段：user_id
		UserNickname string `json:"user_nickname"` // JSON字段：user_nickname
		Source       string `json:"source"`        // JSON字段：source
		CsID         string `json:"cs_id"`         // JSON字段：cs_id
		FirstMsg     string `json:"first_msg"`     // JSON字段：first_msg
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.UserID = strings.TrimSpace(body.UserID)             // 逻辑处理
	body.UserNickname = strings.TrimSpace(body.UserNickname) // 逻辑处理
	body.Source = strings.TrimSpace(body.Source)             // 逻辑处理
	body.CsID = strings.TrimSpace(body.CsID)                 // 逻辑处理
	body.FirstMsg = strings.TrimSpace(body.FirstMsg)         // 逻辑处理

	if body.UserID == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "user_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.CreateConversationReq{ // 逻辑处理
		UserId:       body.UserID,       // 逻辑处理
		UserNickname: body.UserNickname, // 逻辑处理
		Source:       body.Source,       // 逻辑处理
		CsId:         body.CsID,         // 逻辑处理
		FirstMsg:     body.FirstMsg,     // 逻辑处理
	} // 代码块结?
	resp, err := retryRPC(r.Context(), func(ctx context.Context) (*customer.CreateConversationResp, error) {
		return h.client.CreateConversation(ctx, req)
	}) // 调用并接收错?
	if err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500, "msg": "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// EndConversation 结束会话
// 客服或系统主动结束会?// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID（必填）
//   - operator_id: 操作人（客服ID或系统）
//   - end_reason: 结束原因（可选）
//
// 响应: 结束结果及会话时?
func (h *CustomerHandler) EndConversation(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID     string `json:"conv_id"`     // JSON字段：conv_id
		OperatorID string `json:"operator_id"` // JSON字段：operator_id
		EndReason  string `json:"end_reason"`  // JSON字段：end_reason
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.ConvID = strings.TrimSpace(body.ConvID)         // 逻辑处理
	body.OperatorID = strings.TrimSpace(body.OperatorID) // 逻辑处理
	body.EndReason = strings.TrimSpace(body.EndReason)   // 逻辑处理

	if body.ConvID == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "conv_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.EndConversationReq{ // 逻辑处理
		ConvId:     body.ConvID,     // 逻辑处理
		OperatorId: body.OperatorID, // 逻辑处理
		EndReason:  body.EndReason,  // 逻辑处理
	} // 代码块结?
	resp, err := h.client.EndConversation(r.Context(), req) // 调用并接收错?
	if err != nil {                                         // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500, "msg": "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// TransferConversation 转接会话
// 将会话从当前客服转接给另一位客?// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID（必填）
//   - from_cs_id: 转出客服ID（必填）
//   - to_cs_id: 转入客服ID（必填）
//   - transfer_reason: 转接原因（可选）
//   - context_remark: 上下文备注（可选，JSON格式?//
//
// 响应: 转接结果及转接记录ID
func (h *CustomerHandler) TransferConversation(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID         string `json:"conv_id"`         // JSON字段：conv_id
		FromCsID       string `json:"from_cs_id"`      // JSON字段：from_cs_id
		ToCsID         string `json:"to_cs_id"`        // JSON字段：to_cs_id
		TransferReason string `json:"transfer_reason"` // JSON字段：transfer_reason
		ContextRemark  string `json:"context_remark"`  // JSON字段：context_remark
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.ConvID = strings.TrimSpace(body.ConvID)                 // 逻辑处理
	body.FromCsID = strings.TrimSpace(body.FromCsID)             // 逻辑处理
	body.ToCsID = strings.TrimSpace(body.ToCsID)                 // 逻辑处理
	body.TransferReason = strings.TrimSpace(body.TransferReason) // 逻辑处理

	if body.ConvID == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "conv_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.FromCsID == "" || body.ToCsID == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400, "msg": "from_cs_id and to_cs_id are required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.TransferConversationReq{ // 逻辑处理
		ConvId:         body.ConvID,         // 逻辑处理
		FromCsId:       body.FromCsID,       // 逻辑处理
		ToCsId:         body.ToCsID,         // 逻辑处理
		TransferReason: body.TransferReason, // 逻辑处理
		ContextRemark:  body.ContextRemark,  // 逻辑处理
	} // 代码块结?
	resp, err := h.client.TransferConversation(r.Context(), req) // 调用并接收错?
	if err != nil {                                              // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500, "msg": "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListQuickReply 分页查询快捷回复列表
// 查询系统预设和自定义的快捷回复语
// 请求方式: GET
// 请求参数:
//   - keyword: 关键词搜索（搜索快捷回复内容?//   - reply_type: 回复类型筛选（-1-全部?//   - is_public: 是否公开?1-全部?-私有?-公开?//   - page: 页码（默??//   - page_size: 每页数量（默?0?//
//
// 响应: 快捷回复列表及分页信?
func (h *CustomerHandler) ListQuickReply(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))         // 逻辑处理
	replyTypeStr := strings.TrimSpace(r.URL.Query().Get("reply_type")) // 逻辑处理
	isPublicStr := strings.TrimSpace(r.URL.Query().Get("is_public"))   // 逻辑处理

	replyType := int64(-1)  // 逻辑处理
	if replyTypeStr != "" { // 条件判断
		if v, err := strconv.ParseInt(replyTypeStr, 10, 8); err == nil { // 条件判断
			replyType = v // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	isPublic := int64(-1)  // 逻辑处理
	if isPublicStr != "" { // 条件判断
		if v, err := strconv.ParseInt(isPublicStr, 10, 8); err == nil { // 条件判断
			isPublic = v // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	page, pageSize := parsePaginationParams(r, 50) // 逻辑处理

	req := &customer.ListQuickReplyReq{ // 逻辑处理
		Keyword:   keyword,         // 逻辑处理
		ReplyType: int8(replyType), // 逻辑处理
		IsPublic:  int8(isPublic),  // 逻辑处理
		Page:      page,            // 逻辑处理
		PageSize:  pageSize,        // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ListQuickReply(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 会话分类管理 ============

// CreateConvCategory 新增会话分类
// 创建一个新的会话分类（?用户咨询"?投诉建议"等）
// 请求方式: POST
// 请求体参?
//   - category_name: 分类名称（必填）
//   - sort_no: 排序号（越小越前?//   - create_by: 创建?//
//
// 响应: 创建结果及新分类ID
func (h *CustomerHandler) CreateConvCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CategoryName string `json:"category_name"` // JSON字段：category_name
		SortNo       int32  `json:"sort_no"`       // JSON字段：sort_no
		CreateBy     string `json:"create_by"`     // JSON字段：create_by
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CreateConvCategoryResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.CategoryName = strings.TrimSpace(body.CategoryName) // 逻辑处理
	body.CreateBy = strings.TrimSpace(body.CreateBy)         // 逻辑处理
	if body.CategoryName == "" {                             // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CreateConvCategoryResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "category_name is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.CreateBy == "" { // 条件判断
		body.CreateBy = "ADMIN" // 逻辑处理
	} // 代码块结?
	req := &customer.CreateConvCategoryReq{ // 逻辑处理
		CategoryName: body.CategoryName, // 逻辑处理
		SortNo:       body.SortNo,       // 逻辑处理
		CreateBy:     body.CreateBy,     // 逻辑处理
	} // 代码块结?
	resp, err := h.client.CreateConvCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                            // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListConvCategory 查询所有会话分?// 获取系统中所有已启用的会话分类，按排序号排序
// 请求方式: GET
// 请求参数: ?//
// 响应: 分类列表
func (h *CustomerHandler) ListConvCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	req := &customer.ListConvCategoryReq{}                   // 逻辑处理
	resp, err := h.client.ListConvCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// UpdateConversationClassify 更新会话分类、标签和核心标记
// 为指定会话设?更新分类、标签和核心会话标记
// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID（必填）
//   - category_id: 分类ID?表示不更新）
//   - tags: 标签（逗号分隔的标签ID，空字符串清除标签）
//   - is_core: 是否核心会话?-否，1-是）
//   - operator_id: 操作人ID
//
// 响应: 更新结果
func (h *CustomerHandler) UpdateConversationClassify(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID     string `json:"conv_id"`     // JSON字段：conv_id
		CategoryID int64  `json:"category_id"` // JSON字段：category_id
		Tags       string `json:"tags"`        // JSON字段：tags
		IsCore     int8   `json:"is_core"`     // JSON字段：is_core
		OperatorID string `json:"operator_id"` // JSON字段：operator_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateConversationClassifyResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.ConvID = strings.TrimSpace(body.ConvID)         // 逻辑处理
	body.Tags = strings.TrimSpace(body.Tags)             // 逻辑处理
	body.OperatorID = strings.TrimSpace(body.OperatorID) // 逻辑处理
	if body.ConvID == "" {                               // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateConversationClassifyResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "conv_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.OperatorID == "" { // 条件判断
		body.OperatorID = "ADMIN" // 逻辑处理
	} // 代码块结?
	req := &customer.UpdateConversationClassifyReq{ // 逻辑处理
		ConvId:     body.ConvID,     // 逻辑处理
		CategoryId: body.CategoryID, // 逻辑处理
		Tags:       body.Tags,       // 逻辑处理
		IsCore:     body.IsCore,     // 逻辑处理
		OperatorId: body.OperatorID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpdateConversationClassify(r.Context(), req) // 调用并接收错?
	if err != nil {                                                    // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ HTTP 响应工具函数 ============

// respondJSON 返回JSON格式的HTTP响应
// 参数:
//   - w: HTTP响应写入?//   - status: HTTP状态码
//   - data: 响应数据（将被序列化为JSON?
func respondJSON(w http.ResponseWriter, status int, data interface{}) { // 函数定义/HTTP处理入口
	w.Header().Set("Content-Type", "application/json") // 设置响应?
	w.WriteHeader(status)                              // 写入HTTP状态码
	json.NewEncoder(w).Encode(data)                    // 编码并写出JSON响应?
} // 代码块结?
// respondError 返回错误响应
// 参数:
//   - w: HTTP响应写入?//   - status: HTTP状态码
//   - message: 错误消息
func respondError(w http.ResponseWriter, status int, message string) { // 函数定义/HTTP处理入口
	respondJSON(w, status, map[string]interface{}{ // 写回JSON响应
		"code": status,  // 依赖导入
		"msg":  message, // 依赖导入
	}) // 逻辑处理
} // 代码块结?
// ============ 会话标签管理 ============

// CreateConvTag 创建会话标签
func (h *CustomerHandler) CreateConvTag(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		TagName  string `json:"tag_name"`  // JSON字段：tag_name
		TagColor string `json:"tag_color"` // JSON字段：tag_color
		SortNo   int32  `json:"sort_no"`   // JSON字段：sort_no
		CreateBy string `json:"create_by"` // JSON字段：create_by
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CreateConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.TagName = strings.TrimSpace(body.TagName)   // 逻辑处理
	body.TagColor = strings.TrimSpace(body.TagColor) // 逻辑处理
	body.CreateBy = strings.TrimSpace(body.CreateBy) // 逻辑处理
	if body.TagName == "" {                          // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CreateConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "tag_name is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.CreateConvTagReq{ // 逻辑处理
		TagName:  body.TagName,  // 逻辑处理
		TagColor: body.TagColor, // 逻辑处理
		SortNo:   body.SortNo,   // 逻辑处理
		CreateBy: body.CreateBy, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.CreateConvTag(r.Context(), req) // 调用并接收错?
	if err != nil {                                       // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListConvTag 查询会话标签列表
func (h *CustomerHandler) ListConvTag(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	resp, err := h.client.ListConvTag(r.Context(), &customer.ListConvTagReq{}) // 调用并接收错?
	if err != nil {                                                            // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// UpdateConvTag 更新会话标签
func (h *CustomerHandler) UpdateConvTag(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		TagID    int64  `json:"tag_id"`    // JSON字段：tag_id
		TagName  string `json:"tag_name"`  // JSON字段：tag_name
		TagColor string `json:"tag_color"` // JSON字段：tag_color
		SortNo   int32  `json:"sort_no"`   // JSON字段：sort_no
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.TagID <= 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.UpdateConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "tag_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.UpdateConvTagReq{ // 逻辑处理
		TagId:    body.TagID,                       // 逻辑处理
		TagName:  strings.TrimSpace(body.TagName),  // 逻辑处理
		TagColor: strings.TrimSpace(body.TagColor), // 逻辑处理
		SortNo:   body.SortNo,                      // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpdateConvTag(r.Context(), req) // 调用并接收错?
	if err != nil {                                       // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DeleteConvTag 删除会话标签
func (h *CustomerHandler) DeleteConvTag(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		TagID int64 `json:"tag_id"` // JSON字段：tag_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.DeleteConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.TagID <= 0 { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.DeleteConvTagResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "tag_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	resp, err := h.client.DeleteConvTag(r.Context(), &customer.DeleteConvTagReq{TagId: body.TagID}) // 调用并接收错?
	if err != nil {                                                                                 // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 会话统计看板 ============

// GetConversationStats 获取会话统计数据
func (h *CustomerHandler) GetConversationStats(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date")) // 逻辑处理
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))     // 逻辑处理
	statType := strings.TrimSpace(r.URL.Query().Get("stat_type"))   // 逻辑处理

	req := &customer.GetConversationStatsReq{ // 逻辑处理
		StartDate: startDate, // 逻辑处理
		EndDate:   endDate,   // 逻辑处理
		StatType:  statType,  // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetConversationStats(r.Context(), req) // 调用并接收错?
	if err != nil {                                              // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 用户认证相关接口 ============

// Login 用户登录
// 调用RPC验证用户名密码，成功后生成JWT Token返回
func (h *CustomerHandler) Login(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		UserName string `json:"user_name"` // JSON字段：user_name
		Password string `json:"password"`  // JSON字段：password
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.UserName = strings.TrimSpace(body.UserName) // 逻辑处理
	body.Password = strings.TrimSpace(body.Password) // 逻辑处理
	if body.UserName == "" || body.Password == "" {  // 条件判断
		respondJSON(w, http.StatusOK, map[string]interface{}{ // 写回JSON响应
			"code": 400,          // 依赖导入
			"msg":  "用户名和密码不能为空", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 调用RPC验证登录
	req := &customer.LoginReq{ // 逻辑处理
		UserName: body.UserName, // 逻辑处理
		Password: body.Password, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.Login(r.Context(), req) // 调用并接收错?
	if err != nil {                               // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 登录验证失败，直接返回RPC响应
	if resp.BaseResp == nil || resp.BaseResp.Code != 0 { // 条件判断
		respondJSON(w, http.StatusOK, resp) // 写回JSON响应
		return                              // 返回结果/结束处理
	} // 代码块结?
	// 登录成功，生成JWT Token
	token, err := generateJWTToken(resp.UserInfo.Id, resp.UserInfo.UserName, resp.UserInfo.RoleCode) // 调用并接收错?
	if err != nil {                                                                                  // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                         // 依赖导入
			"msg":  "生成Token失败: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 返回登录成功响应（包含Token?
	respondJSON(w, http.StatusOK, map[string]interface{}{ // 写回JSON响应
		"code": 0,      // 依赖导入
		"msg":  "登录成功", // 依赖导入
		"data": map[string]interface{}{ // 依赖导入
			"token":     token,         // 依赖导入
			"user_info": resp.UserInfo, // 依赖导入
		}, // 逻辑处理
	}) // 逻辑处理
} // 代码块结?
// GetCurrentUser 获取当前登录用户信息
func (h *CustomerHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	// 从上下文获取用户ID（由认证中间件写入）
	userID := getUserIDFromContext(r.Context()) // 逻辑处理
	if userID <= 0 {                            // 条件判断
		respondJSON(w, http.StatusOK, map[string]interface{}{ // 写回JSON响应
			"code": 401,    // 依赖导入
			"msg":  "请先登录", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 调用RPC获取用户信息
	req := &customer.GetCurrentUserReq{ // 逻辑处理
		UserId: userID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetCurrentUser(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ JWT Token 工具函数 ============

// generateJWTToken 生成JWT Token的包装函?// 参数:
//   - userID: 用户ID
//   - userName: 用户?//   - roleCode: 角色编码
//
// 返回:
//   - string: 生成的Token字符?//   - error: 错误信息
func generateJWTToken(userID int64, userName, roleCode string) (string, error) { // 函数定义/HTTP处理入口
	return generateToken(userID, userName, roleCode) // 返回结果/结束处理
} // 代码块结?
// generateToken 生成JWT Token的实际实?// 使用HS256算法签名，包含用户ID、用户名、角色编码、过期时间等声明
// 参数:
//   - userID: 用户ID
//   - userName: 用户?//   - roleCode: 角色编码
//
// 返回:
//   - string: 生成的Token字符?//   - error: 错误信息
func generateToken(userID int64, userName, roleCode string) (string, error) { // 函数定义/HTTP处理入口
	now := time.Now()        // 逻辑处理
	claims := jwt.MapClaims{ // 逻辑处理
		"user_id":   userID,                                                                // 依赖导入
		"user_name": userName,                                                              // 依赖导入
		"role_code": roleCode,                                                              // 依赖导入
		"exp":       now.Add(time.Duration(config.GetJWTExpireHours()) * time.Hour).Unix(), // 依赖导入
		"iat":       now.Unix(),                                                            // 依赖导入
	} // 代码块结?
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // 逻辑处理
	return token.SignedString([]byte(config.GetJWTSecret()))   // 返回结果/结束处理
} // 代码块结?
// ============ 上下文工具函?============

// getUserIDFromContext 从上下文中获取用户ID
// 由认证中间件将用户ID存入context，此函数用于提取
// 参数:
//   - ctx: 请求上下?//
//
// 返回:
//   - int64: 用户ID，未找到返回0
func getUserIDFromContext(ctx context.Context) int64 { // 函数定义/HTTP处理入口
	if userID, ok := ctx.Value("user_id").(int64); ok { // 条件判断
		return userID // 返回结果/结束处理
	} // 代码块结?
	return 0 // 返回结果/结束处理
} // 代码块结?
// resolveCustomerCsID 解析客服ID
// 根据当前登录用户角色解析客服ID?// - 管理员：返回传入的csID
// - 客服：返回当前登录用户名或根据用户ID生成
// 参数:
//   - ctx: 请求上下文（包含用户信息?//   - csID: 传入的客服ID
//
// 返回:
//   - string: 解析后的客服ID
func resolveCustomerCsID(ctx context.Context, csID string) string { // 函数定义/HTTP处理入口
	roleCode := middleware.GetRoleCodeFromContext(ctx) // 逻辑处理
	if roleCode != middleware.RoleCustomerService {    // 条件判断
		return strings.TrimSpace(csID) // 返回结果/结束处理
	} // 代码块结?
	userName := strings.TrimSpace(middleware.GetUserNameFromContext(ctx)) // 逻辑处理
	if userName != "" {                                                   // 条件判断
		upper := strings.ToUpper(userName)                                    // 逻辑处理
		if strings.HasPrefix(upper, "CS") || strings.HasPrefix(upper, "KF") { // 条件判断
			return userName // 返回结果/结束处理
		} // 代码块结?
	} // 代码块结?
	userID := middleware.GetUserIDFromContext(ctx) // 逻辑处理
	if userID > 0 {                                // 条件判断
		return fmt.Sprintf("CS%d", userID) // 返回结果/结束处理
	} // 代码块结?
	return strings.TrimSpace(csID) // 返回结果/结束处理
} // 代码块结?
// OperatorInfo 操作人信息结构体
type OperatorInfo struct {
	ID   string // 操作人ID（客服ID格式）
	Name string // 操作人姓名
	Role string // 操作人角色（admin/manager/customer_service）
}

// getOperatorInfoFromContext 从token上下文获取操作人信息
// 自动从JWT token中提取当前登录用户的ID、姓名和角色
// 用于审批、创建等操作的身份记?
func getOperatorInfoFromContext(ctx context.Context) OperatorInfo { // 函数定义/HTTP处理入口
	info := OperatorInfo{} // 逻辑处理

	// 获取姓名
	info.Name = strings.TrimSpace(middleware.GetUserNameFromContext(ctx)) // 逻辑处理

	// 获取角色和用户ID
	roleCode := middleware.GetRoleCodeFromContext(ctx) // 逻辑处理
	userID := middleware.GetUserIDFromContext(ctx)     // 逻辑处理

	switch roleCode { // 分支选择
	case middleware.RoleAdmin: // 分支条件
		info.Role = "admin"  // 逻辑处理
		if info.Name != "" { // 条件判断
			info.ID = info.Name // 逻辑处理
		} else { // 逻辑处理
			info.ID = fmt.Sprintf("ADMIN%d", userID) // 逻辑处理
		} // 代码块结?
	case middleware.RoleCustomerService: // 分支条件
		info.Role = "customer_service" // 逻辑处理
		// 优先使用用户名（如果符合客服ID格式?
		if info.Name != "" { // 条件判断
			upper := strings.ToUpper(info.Name)                                   // 逻辑处理
			if strings.HasPrefix(upper, "CS") || strings.HasPrefix(upper, "KF") { // 条件判断
				info.ID = info.Name // 逻辑处理
			} // 代码块结?
		} // 代码块结?		// 如果用户名不符合格式，使用用户ID生成
		if info.ID == "" && userID > 0 { // 条件判断
			info.ID = fmt.Sprintf("CS%d", userID) // 逻辑处理
		} // 代码块结?
	default: // 默认分支
		info.Role = roleCode // 逻辑处理
		if userID > 0 {      // 条件判断
			info.ID = fmt.Sprintf("USER%d", userID) // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	return info // 返回结果/结束处理
} // 代码块结?
// parsePaginationParams 从请求中解析分页参数
// 返回 page, pageSize (int32)
func parsePaginationParams(r *http.Request, defaultPageSize int) (int32, int32) { // 函数定义/HTTP处理入口
	pageStr := r.URL.Query().Get("page")          // 逻辑处理
	pageSizeStr := r.URL.Query().Get("page_size") // 逻辑处理

	page, _ := strconv.Atoi(pageStr) // 逻辑处理
	if page <= 0 {                   // 条件判断
		page = 1 // 逻辑处理
	} // 代码块结?
	pageSize, _ := strconv.Atoi(pageSizeStr) // 逻辑处理
	if pageSize <= 0 {                       // 条件判断
		pageSize = defaultPageSize // 逻辑处理
	} // 代码块结?
	return int32(page), int32(pageSize) // 返回结果/结束处理
} // 代码块结?
// Register 用户注册
// 仅允许注册客服账号，调用RPC完成注册
func (h *CustomerHandler) Register(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		UserName string `json:"user_name"` // JSON字段：user_name
		Password string `json:"password"`  // JSON字段：password
		RealName string `json:"real_name"` // JSON字段：real_name
		Phone    string `json:"phone"`     // JSON字段：phone
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.UserName = strings.TrimSpace(body.UserName) // 逻辑处理
	body.Password = strings.TrimSpace(body.Password) // 逻辑处理
	body.RealName = strings.TrimSpace(body.RealName) // 逻辑处理
	body.Phone = strings.TrimSpace(body.Phone)       // 逻辑处理

	// 调用RPC注册
	req := &customer.RegisterReq{ // 逻辑处理
		UserName: body.UserName, // 逻辑处理
		Password: body.Password, // 逻辑处理
		RealName: body.RealName, // 逻辑处理
		Phone:    body.Phone,    // 逻辑处理
	} // 代码块结?
	resp, err := h.client.Register(r.Context(), req) // 调用并接收错?
	if err != nil {                                  // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 快捷回复管理 ============

// CreateQuickReply 创建快捷回复
// 管理员或客服创建预设的快捷回复话?// 请求方式: POST
// 请求体参?
//   - reply_type: 回复类型?-通用, 1-售前, 2-售后, 3-投诉?//   - reply_content: 回复内容（必填）
//   - create_by: 创建?//   - is_public: 是否公开?-私有, 1-公开?//
//
// 响应: 创建结果及新回复ID
func (h *CustomerHandler) CreateQuickReply(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ReplyType    int8   `json:"reply_type"`    // JSON字段：reply_type
		ReplyContent string `json:"reply_content"` // JSON字段：reply_content
		CreateBy     string `json:"create_by"`     // JSON字段：create_by
		IsPublic     int8   `json:"is_public"`     // JSON字段：is_public
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 如果未指定创建人，使用当前登录用?
	if strings.TrimSpace(body.CreateBy) == "" { // 条件判断
		userName := middleware.GetUserNameFromContext(r.Context()) // 逻辑处理
		if userName != "" {                                        // 条件判断
			body.CreateBy = userName // 逻辑处理
		} else { // 逻辑处理
			body.CreateBy = "ADMIN" // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	req := &customer.CreateQuickReplyReq{ // 逻辑处理
		ReplyType:    body.ReplyType,                       // 逻辑处理
		ReplyContent: strings.TrimSpace(body.ReplyContent), // 逻辑处理
		CreateBy:     strings.TrimSpace(body.CreateBy),     // 逻辑处理
		IsPublic:     body.IsPublic,                        // 逻辑处理
	} // 代码块结?
	resp, err := h.client.CreateQuickReply(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// UpdateQuickReply 更新快捷回复
// 修改已有的快捷回复内容或属?// 请求方式: POST
// 请求体参?
//   - reply_id: 回复ID（必填）
//   - reply_type: 回复类型
//   - reply_content: 回复内容（必填）
//   - is_public: 是否公开
//
// 响应: 更新结果
func (h *CustomerHandler) UpdateQuickReply(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ReplyID      int64  `json:"reply_id"`      // JSON字段：reply_id
		ReplyType    int8   `json:"reply_type"`    // JSON字段：reply_type
		ReplyContent string `json:"reply_content"` // JSON字段：reply_content
		IsPublic     int8   `json:"is_public"`     // JSON字段：is_public
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.UpdateQuickReplyReq{ // 逻辑处理
		ReplyId:      body.ReplyID,                         // 逻辑处理
		ReplyType:    body.ReplyType,                       // 逻辑处理
		ReplyContent: strings.TrimSpace(body.ReplyContent), // 逻辑处理
		IsPublic:     body.IsPublic,                        // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpdateQuickReply(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DeleteQuickReply 删除快捷回复
// 删除指定的快捷回复记?// 请求方式: POST
// 请求体参?
//   - reply_id: 回复ID（必填）
//
// 响应: 删除结果
func (h *CustomerHandler) DeleteQuickReply(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ReplyID int64 `json:"reply_id"` // JSON字段：reply_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.DeleteQuickReplyReq{ // 逻辑处理
		ReplyId: body.ReplyID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.DeleteQuickReply(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 会话监控与导?============

// GetConversationMonitor 获取会话监控数据
// 实时查看会话状态、客服在线状态、等待中会话数等
// 请求方式: GET
// 请求参数:
//   - dept_id: 部门ID（可选，筛选指定部门）
//   - status_filter: 状态筛?-1-全部 0-等待 1-进行?//
//
// 响应: 客服状态列表、会话列表、统计数?
func (h *CustomerHandler) GetConversationMonitor(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	deptID := strings.TrimSpace(r.URL.Query().Get("dept_id")) // 逻辑处理
	statusFilterStr := r.URL.Query().Get("status_filter")     // 逻辑处理
	statusFilter := int8(-1)                                  // 默认全部
	if statusFilterStr != "" {                                // 条件判断
		if v, err := strconv.Atoi(statusFilterStr); err == nil { // 条件判断
			statusFilter = int8(v) // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	req := &customer.GetConversationMonitorReq{ // 逻辑处理
		DeptId:       deptID,       // 逻辑处理
		StatusFilter: statusFilter, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetConversationMonitor(r.Context(), req) // 调用并接收错?
	if err != nil {                                                // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ExportConversations 导出会话记录
// 支持按条件筛选导出会话记录为Excel/CSV格式
// 请求方式: GET
// 请求参数:
//   - cs_id: 客服ID筛选（可选）
//   - user_id: 用户ID筛选（可选）
//   - start_date: 开始日?YYYY-MM-DD
//   - end_date: 结束日期 YYYY-MM-DD
//   - status: 状态筛?-1-全部
//   - keyword: 关键词搜?//   - export_format: 导出格式 excel/csv（默认excel?//
//
// 响应: 文件流下?
func (h *CustomerHandler) ExportConversations(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	csID := strings.TrimSpace(r.URL.Query().Get("cs_id"))                 // 逻辑处理
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))             // 逻辑处理
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))       // 逻辑处理
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))           // 逻辑处理
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))            // 逻辑处理
	exportFormat := strings.TrimSpace(r.URL.Query().Get("export_format")) // 逻辑处理
	statusStr := r.URL.Query().Get("status")                              // 逻辑处理

	status := int8(-1)   // 逻辑处理
	if statusStr != "" { // 条件判断
		if v, err := strconv.Atoi(statusStr); err == nil { // 条件判断
			status = int8(v) // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	if exportFormat == "" { // 条件判断
		exportFormat = "excel" // 逻辑处理
	} // 代码块结?
	req := &customer.ExportConversationsReq{ // 逻辑处理
		CsId:         csID,         // 逻辑处理
		UserId:       userID,       // 逻辑处理
		StartDate:    startDate,    // 逻辑处理
		EndDate:      endDate,      // 逻辑处理
		Status:       status,       // 逻辑处理
		Keyword:      keyword,      // 逻辑处理
		ExportFormat: exportFormat, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ExportConversations(r.Context(), req) // 调用并接收错?
	if err != nil {                                             // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if resp.BaseResp != nil && resp.BaseResp.Code != 0 { // 条件判断
		respondJSON(w, http.StatusOK, resp) // 写回JSON响应
		return                              // 返回结果/结束处理
	} // 代码块结?
	// 返回文件?
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" // 逻辑处理
	if exportFormat == "csv" {                                                         // 条件判断
		contentType = "text/csv" // 逻辑处理
	} // 代码块结?
	w.Header().Set("Content-Type", contentType) // 设置响应?	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", resp.FileName)) // 设置响应?
	w.Write(resp.FileData)                      // 逻辑处理
} // 代码块结?
// ============ 消息分类管理 ============

// MsgAutoClassify 消息自动分类
// 基于关键词匹配对会话消息进行自动分类
// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID
//   - msg_contents: 消息内容列表
//
// 响应: 分类ID、分类名称、置信度、匹配的关键?
func (h *CustomerHandler) MsgAutoClassify(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID      string   `json:"conv_id"`      // JSON字段：conv_id
		MsgContents []string `json:"msg_contents"` // JSON字段：msg_contents
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.MsgAutoClassifyReq{ // 逻辑处理
		ConvId:      strings.TrimSpace(body.ConvID), // 逻辑处理
		MsgContents: body.MsgContents,               // 逻辑处理
	} // 代码块结?
	resp, err := h.client.MsgAutoClassify(r.Context(), req) // 调用并接收错?
	if err != nil {                                         // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// AdjustMsgClassify 人工调整消息分类
// 客服手动修正自动分类结果
// 请求方式: POST
// 请求体参?
//   - conv_id: 会话ID
//   - original_category_id: 原分类ID
//   - new_category_id: 新分类ID
//   - operator_id: 操作人id
//   - adjust_reason: 调整原因
//
// 响应: 调整记录ID
func (h *CustomerHandler) AdjustMsgClassify(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		ConvID             string `json:"conv_id"`              // JSON字段：conv_id
		OriginalCategoryID int64  `json:"original_category_id"` // JSON字段：original_category_id
		NewCategoryID      int64  `json:"new_category_id"`      // JSON字段：new_category_id
		OperatorID         string `json:"operator_id"`          // JSON字段：operator_id
		AdjustReason       string `json:"adjust_reason"`        // JSON字段：adjust_reason
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 如果未指定操作人，使用当前登录用?
	if strings.TrimSpace(body.OperatorID) == "" { // 条件判断
		userName := middleware.GetUserNameFromContext(r.Context()) // 逻辑处理
		if userName != "" {                                        // 条件判断
			body.OperatorID = userName // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	req := &customer.AdjustMsgClassifyReq{ // 逻辑处理
		ConvId:             strings.TrimSpace(body.ConvID),       // 逻辑处理
		OriginalCategoryId: body.OriginalCategoryID,              // 逻辑处理
		NewCategoryId_:     body.NewCategoryID,                   // 逻辑处理
		OperatorId:         strings.TrimSpace(body.OperatorID),   // 逻辑处理
		AdjustReason:       strings.TrimSpace(body.AdjustReason), // 逻辑处理
	} // 代码块结?
	resp, err := h.client.AdjustMsgClassify(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// GetClassifyStats 获取分类统计数据
// 查询消息分类的统计信?// 请求方式: GET
// 请求参数:
//   - start_date: 开始日?YYYY-MM-DD
//   - end_date: 结束日期 YYYY-MM-DD
//   - stat_type: 统计类型 day/week/month
//
// 响应: 每日统计、分类汇总、准确率?
func (h *CustomerHandler) GetClassifyStats(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date")) // 逻辑处理
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))     // 逻辑处理
	statType := strings.TrimSpace(r.URL.Query().Get("stat_type"))   // 逻辑处理

	req := &customer.GetClassifyStatsReq{ // 逻辑处理
		StartDate: startDate, // 逻辑处理
		EndDate:   endDate,   // 逻辑处理
		StatType:  statType,  // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetClassifyStats(r.Context(), req) // 调用并接收错?
	if err != nil {                                          // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 消息分类维度CRUD ============

// CreateMsgCategory 创建消息分类维度
// 创建新的消息分类类型（如咨询类、投诉类、建议类等）
// 请求方式: POST
// 请求体参?
//   - category_name: 分类名称（必填）
//   - keywords: 关键词列?JSON)
//   - sort_no: 排序?//   - create_by: 创建?//
//
// 响应: 创建结果及新分类ID
func (h *CustomerHandler) CreateMsgCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CategoryName string `json:"category_name"` // JSON字段：category_name
		Keywords     string `json:"keywords"`      // JSON字段：keywords
		SortNo       int32  `json:"sort_no"`       // JSON字段：sort_no
		CreateBy     string `json:"create_by"`     // JSON字段：create_by
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 如果未指定创建人，使用当前登录用?
	if strings.TrimSpace(body.CreateBy) == "" { // 条件判断
		userName := middleware.GetUserNameFromContext(r.Context()) // 逻辑处理
		if userName != "" {                                        // 条件判断
			body.CreateBy = userName // 逻辑处理
		} else { // 逻辑处理
			body.CreateBy = "ADMIN" // 逻辑处理
		} // 代码块结?
	} // 代码块结?
	req := &customer.CreateMsgCategoryReq{ // 逻辑处理
		CategoryName: strings.TrimSpace(body.CategoryName), // 逻辑处理
		Keywords:     body.Keywords,                        // 逻辑处理
		SortNo:       body.SortNo,                          // 逻辑处理
		CreateBy:     strings.TrimSpace(body.CreateBy),     // 逻辑处理
	} // 代码块结?
	resp, err := h.client.CreateMsgCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListMsgCategory 查询消息分类维度列表
// 获取所有消息分类类?// 请求方式: GET
// 响应: 分类列表
func (h *CustomerHandler) ListMsgCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	req := &customer.ListMsgCategoryReq{} // 逻辑处理

	resp, err := h.client.ListMsgCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                         // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// UpdateMsgCategory 更新消息分类维度
// 修改消息分类的名称、关键词等属?// 请求方式: POST
// 请求体参?
//   - category_id: 分类ID（必填）
//   - category_name: 分类名称
//   - keywords: 关键词列?JSON)
//   - sort_no: 排序?//
//
// 响应: 更新结果
func (h *CustomerHandler) UpdateMsgCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CategoryID   int64  `json:"category_id"`   // JSON字段：category_id
		CategoryName string `json:"category_name"` // JSON字段：category_name
		Keywords     string `json:"keywords"`      // JSON字段：keywords
		SortNo       int32  `json:"sort_no"`       // JSON字段：sort_no
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.UpdateMsgCategoryReq{ // 逻辑处理
		CategoryId:   body.CategoryID,                      // 逻辑处理
		CategoryName: strings.TrimSpace(body.CategoryName), // 逻辑处理
		Keywords:     body.Keywords,                        // 逻辑处理
		SortNo:       body.SortNo,                          // 逻辑处理
	} // 代码块结?
	resp, err := h.client.UpdateMsgCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DeleteMsgCategory 删除消息分类维度
// 删除指定的消息分类类?// 请求方式: POST
// 请求体参?
//   - category_id: 分类ID（必填）
//
// 响应: 删除结果
func (h *CustomerHandler) DeleteMsgCategory(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CategoryID int64 `json:"category_id"` // JSON字段：category_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.DeleteMsgCategoryReq{ // 逻辑处理
		CategoryId: body.CategoryID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.DeleteMsgCategory(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 消息加密与脱?============

// EncryptMessage 加密消息内容
// 使用AES-256-GCM算法加密敏感消息
// 请求方式: POST
// 请求体参?
//   - msg_content: 待加密的消息内容（必填）
//
// 响应: 加密后的内容
func (h *CustomerHandler) EncryptMessage(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		MsgContent string `json:"msg_content"` // JSON字段：msg_content
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.MsgContent == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                       // 依赖导入
			"msg":  "msg_content is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.EncryptMessageReq{ // 逻辑处理
		MsgContent: body.MsgContent, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.EncryptMessage(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DecryptMessage 解密消息内容
// 解密已加密的消息内容
// 请求方式: POST
// 请求体参?
//   - encrypted_content: 加密后的消息内容（必填）
//
// 响应: 解密后的原始内容
func (h *CustomerHandler) DecryptMessage(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		EncryptedContent string `json:"encrypted_content"` // JSON字段：encrypted_content
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.EncryptedContent == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                             // 依赖导入
			"msg":  "encrypted_content is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.DecryptMessageReq{ // 逻辑处理
		EncryptedContent: body.EncryptedContent, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.DecryptMessage(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// DesensitizeMessage 消息脱敏处理
// 对消息中的敏感信息（手机号、身份证、银行卡、邮箱）进行脱敏
// 请求方式: POST
// 请求体参?
//   - msg_content: 待脱敏的消息内容（必填）
//
// 响应: 脱敏后的内容和检测到的敏感信息类?
func (h *CustomerHandler) DesensitizeMessage(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		MsgContent string `json:"msg_content"` // JSON字段：msg_content
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.MsgContent == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                       // 依赖导入
			"msg":  "msg_content is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.DesensitizeMessageReq{ // 逻辑处理
		MsgContent: body.MsgContent, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.DesensitizeMessage(r.Context(), req) // 调用并接收错?
	if err != nil {                                            // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 数据归档管理 ============

// ArchiveConversations 归档历史会话
// 将指定日期之前的会话数据归档到归档表
// 请求方式: POST
// 请求体参?
//   - end_date: 截止日期，归档此日期之前的数据（格式?006-01-02?//   - retention_days: 归档数据保留天数（默?65天）
//   - operator_id: 操作人ID
//
// 响应: 归档任务ID和预计归档数?
func (h *CustomerHandler) ArchiveConversations(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		EndDate       string `json:"end_date"`       // JSON字段：end_date
		RetentionDays int32  `json:"retention_days"` // JSON字段：retention_days
		OperatorID    string `json:"operator_id"`    // JSON字段：operator_id
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	if body.EndDate == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                    // 依赖导入
			"msg":  "end_date is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	// 获取操作人信?
	operatorInfo := getOperatorInfoFromContext(r.Context()) // 逻辑处理

	req := &customer.ArchiveConversationsReq{ // 逻辑处理
		EndDate:       body.EndDate,       // 逻辑处理
		RetentionDays: body.RetentionDays, // 逻辑处理
		OperatorId:    operatorInfo.ID,    // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ArchiveConversations(r.Context(), req) // 调用并接收错?
	if err != nil {                                              // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// GetArchiveTask 获取归档任务状?// 查询归档任务的执行进度和状?// 请求方式: GET
// 查询参数:
//   - task_id: 归档任务ID（必填）
//
// 响应: 任务状态、进度、已归档数量等信?
func (h *CustomerHandler) GetArchiveTask(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	taskIDStr := r.URL.Query().Get("task_id") // 逻辑处理
	if taskIDStr == "" {                      // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                   // 依赖导入
			"msg":  "task_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64) // 调用并接收错?
	if err != nil {                                    // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,               // 依赖导入
			"msg":  "invalid task_id", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.GetArchiveTaskReq{ // 逻辑处理
		TaskId: taskID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetArchiveTask(r.Context(), req) // 调用并接收错?
	if err != nil {                                        // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// QueryArchivedConversation 查询归档会话
// 从归档表中查询历史会话数?// 请求方式: GET
// 查询参数:
//   - user_id: 用户ID（可选，按用户查询）
//   - cs_id: 客服ID（可选，按客服查询）
//   - start_date: 开始日期（格式?006-01-02?//   - end_date: 结束日期（格式：2006-01-02?//   - page: 页码（默??//   - page_size: 每页条数（默?0?//
//
// 响应: 归档会话列表和总数
func (h *CustomerHandler) QueryArchivedConversation(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	query := r.URL.Query() // 逻辑处理

	page, _ := strconv.Atoi(query.Get("page")) // 逻辑处理
	if page <= 0 {                             // 条件判断
		page = 1 // 逻辑处理
	} // 代码块结?
	pageSize, _ := strconv.Atoi(query.Get("page_size")) // 逻辑处理
	if pageSize <= 0 {                                  // 条件判断
		pageSize = 20 // 逻辑处理
	} // 代码块结?
	req := &customer.QueryArchivedConversationReq{ // 逻辑处理
		UserId:    query.Get("user_id"),    // 逻辑处理
		CsId:      query.Get("cs_id"),      // 逻辑处理
		StartDate: query.Get("start_date"), // 逻辑处理
		EndDate:   query.Get("end_date"),   // 逻辑处理
		Page:      int32(page),             // 逻辑处理
		PageSize:  int32(pageSize),         // 逻辑处理
	} // 代码块结?
	resp, err := h.client.QueryArchivedConversation(r.Context(), req) // 调用并接收错?
	if err != nil {                                                   // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 心跳与在线状态管?============

// Heartbeat 客服心跳上报
// 客服端定期调用此接口保持在线状?// 请求方式: POST
// 请求体参?
//   - cs_id: 客服ID（必填）
//
// 响应: 在线状态确?
func (h *CustomerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CsID string `json:"cs_id"` // 客服ID
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.HeartbeatResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.CsID = strings.TrimSpace(body.CsID) // 逻辑处理
	if body.CsID == "" {                     // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.HeartbeatResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "cs_id is required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.HeartbeatReq{ // 逻辑处理
		CsId: body.CsID, // 逻辑处理
	} // 代码块结?
	resp, err := retryRPC(r.Context(), func(ctx context.Context) (*customer.HeartbeatResp, error) {
		return h.client.Heartbeat(ctx, req)
	}) // 调用并接收错?
	if err != nil { // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ListOnlineCustomers 获取当前在线客服列表
// 请求方式: GET
// 查询参数:
//   - dept_id: 部门ID（可选，按部门筛选）
//
// 响应: 在线客服列表
func (h *CustomerHandler) ListOnlineCustomers(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	deptID := strings.TrimSpace(r.URL.Query().Get("dept_id")) // 逻辑处理

	req := &customer.ListOnlineCustomersReq{ // 逻辑处理
		DeptId: deptID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.ListOnlineCustomers(r.Context(), req) // 调用并接收错?
	if err != nil {                                             // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 调班辅助接口 ============

// GetSwapCandidates 获取调班候选人
// 返回指定日期有排班的其他客服及班次信?// 请求方式: GET
// 查询参数:
//   - cs_id: 发起人客服ID（必填）
//   - target_date: 调班日期（必填）
//
// 响应: 可调班候选人列表
func (h *CustomerHandler) GetSwapCandidates(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodGet { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	csID := strings.TrimSpace(r.URL.Query().Get("cs_id"))             // 逻辑处理
	targetDate := strings.TrimSpace(r.URL.Query().Get("target_date")) // 逻辑处理

	if csID == "" || targetDate == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.GetSwapCandidatesResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "cs_id and target_date are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.GetSwapCandidatesReq{ // 逻辑处理
		CsId:       csID,       // 逻辑处理
		TargetDate: targetDate, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.GetSwapCandidates(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// CheckSwapConflict 检测调班冲?// 检测发起人与目标客服之间是否存在调班冲?// 请求方式: POST
// 请求体参?
//   - initiator_cs_id: 发起人客服ID（必填）
//   - target_cs_id: 目标客服ID（必填）
//   - target_date: 调班日期（必填）
//
// 响应: 冲突检测结?
func (h *CustomerHandler) CheckSwapConflict(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		InitiatorCsID string `json:"initiator_cs_id"` // 发起人客服ID
		TargetCsID    string `json:"target_cs_id"`    // 目标客服ID
		TargetDate    string `json:"target_date"`     // 调班日期
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CheckSwapConflictResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "invalid json body"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.InitiatorCsID = strings.TrimSpace(body.InitiatorCsID) // 逻辑处理
	body.TargetCsID = strings.TrimSpace(body.TargetCsID)       // 逻辑处理
	body.TargetDate = strings.TrimSpace(body.TargetDate)       // 逻辑处理

	if body.InitiatorCsID == "" || body.TargetCsID == "" || body.TargetDate == "" { // 条件判断
		respondJSON(w, http.StatusBadRequest, &customer.CheckSwapConflictResp{ // 写回JSON响应
			BaseResp: &customer.BaseResp{Code: 400, Msg: "initiator_cs_id, target_cs_id and target_date are required"}, // 逻辑处理
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.CheckSwapConflictReq{ // 逻辑处理
		InitiatorCsId: body.InitiatorCsID, // 逻辑处理
		TargetCsId:    body.TargetCsID,    // 逻辑处理
		TargetDate:    body.TargetDate,    // 逻辑处理
	} // 代码块结?
	resp, err := h.client.CheckSwapConflict(r.Context(), req) // 调用并接收错?
	if err != nil {                                           // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ============ 退出登?============

// Logout 客服退出登?// 将客服置为离线状?// 请求方式: POST
// 请求体参?
//   - cs_id: 客服ID（必填）
//
// 响应: 退出结?
func (h *CustomerHandler) Logout(w http.ResponseWriter, r *http.Request) { // 函数定义/HTTP处理入口
	if r.Method != http.MethodPost { // 条件判断
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed") // 写回错误响应
		return                                                             // 返回结果/结束处理
	} // 代码块结?
	var body struct { // 变量声明
		CsID string `json:"cs_id"` // 客服ID
	} // 代码块结?
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "invalid json body", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	body.CsID = strings.TrimSpace(body.CsID) // 逻辑处理
	if body.CsID == "" {                     // 条件判断
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{ // 写回JSON响应
			"code": 400,                 // 依赖导入
			"msg":  "cs_id is required", // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	req := &customer.LogoutReq{ // 逻辑处理
		CsId: body.CsID, // 逻辑处理
	} // 代码块结?
	resp, err := h.client.Logout(r.Context(), req) // 调用并接收错?
	if err != nil {                                // 条件判断
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{ // 写回JSON响应
			"code": 500,                                     // 依赖导入
			"msg":  "Internal server error: " + err.Error(), // 依赖导入
		}) // 逻辑处理
		return // 返回结果/结束处理
	} // 代码块结?
	respondJSON(w, http.StatusOK, resp) // 写回JSON响应
} // 代码块结?
// ProxyAIProcess 转发请求?chatModel 服务
func (h *CustomerHandler) ProxyAIProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从配置中获取 chatModel 服务地址
	chatModelService := config.GlobalConfig.Services["chatModel"]
	baseURL := chatModelService.Address
	if baseURL == "" {
		baseURL = "http://chat-model:8082"
	}
	targetURL := baseURL + "/api/ai/process"

	// 创建转发请求
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create proxy request")
		return
	}

	// 复制 Header
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "AI service unavailable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// 复制响应 Header
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ProxyAIJobStatus 转发任务状态查?
func (h *CustomerHandler) ProxyAIJobStatus(w http.ResponseWriter, r *http.Request) {
	// 从配置中获取 chatModel 服务地址
	chatModelService := config.GlobalConfig.Services["chatModel"]
	baseURL := chatModelService.Address
	if baseURL == "" {
		baseURL = "http://chat-model:8082"
	}
	jobID := r.URL.Query().Get("job_id")
	targetURL := fmt.Sprintf("%s/api/ai/job/status?job_id=%s", baseURL, jobID)

	resp, err := http.Get(targetURL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "AI service unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// retryRPC 内部 RPC 调用重试助手
// 当遇到连接拒绝或网络类瞬态错误时，进行指数退避重试，以平滑度过服务启动期?
func retryRPC[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := fn(ctx)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		errMsg := strings.ToLower(err.Error())
		// 判定是否为建议重试的网络错误：连接拒绝、连接超时、网络不可达?
		if strings.Contains(errMsg, "connection refused") ||
			strings.Contains(errMsg, "connectex") ||
			strings.Contains(errMsg, "network error") ||
			strings.Contains(errMsg, "timeout") {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		// 其他业务或代码错误直接返回，不触发重?
		return resp, err
	}
	var zero T
	return zero, lastErr
}
