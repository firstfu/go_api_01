// model/model.go
// 資料模型定義層
// 定義搶票模擬系統中所有的資料結構，包含活動、訂單、搶票請求、模擬請求、
// 搶票結果、模擬結果、統計資訊與統一 API 回應格式。
// 所有結構體皆附有 JSON tag 以支援 HTTP JSON 序列化。

package model

import "time"

// Event 活動資料結構
// 表示一場售票活動的完整資訊
type Event struct {
	// ID 活動唯一識別碼（格式: evt-1, evt-2, ...）
	ID string `json:"id"`
	// Name 活動名稱
	Name string `json:"name" binding:"required"`
	// Description 活動描述
	Description string `json:"description"`
	// TotalTickets 總票數
	TotalTickets int `json:"total_tickets" binding:"required,gt=0"`
	// Remaining 剩餘票數
	Remaining int `json:"remaining"`
	// Price 票價（單位：元）
	Price float64 `json:"price" binding:"required,gt=0"`
	// CreatedAt 建立時間
	CreatedAt time.Time `json:"created_at"`
}

// Order 訂單資料結構
// 表示一筆搶票成功後產生的訂單
type Order struct {
	// ID 訂單唯一識別碼（格式: ord-1, ord-2, ...）
	ID string `json:"id"`
	// EventID 所屬活動的 ID
	EventID string `json:"event_id"`
	// UserID 搶票用戶的 ID
	UserID string `json:"user_id"`
	// Quantity 購買數量
	Quantity int `json:"quantity"`
	// TotalPrice 訂單總金額
	TotalPrice float64 `json:"total_price"`
	// CreatedAt 訂單建立時間
	CreatedAt time.Time `json:"created_at"`
}

// GrabRequest 搶票請求
// 用戶發送搶票時的請求參數
type GrabRequest struct {
	// UserID 用戶 ID
	UserID string `json:"user_id" binding:"required"`
	// Quantity 購買數量
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

// SimulateRequest 模擬請求
// 啟動搶票模擬時的請求參數
type SimulateRequest struct {
	// EventID 目標活動 ID
	EventID string `json:"event_id" binding:"required"`
	// Concurrency 併發用戶數量
	Concurrency int `json:"concurrency" binding:"required,gt=0"`
	// PerUser 每位用戶搶購的票數
	PerUser int `json:"per_user" binding:"required,gt=0"`
}

// GrabResult 單次搶票結果
// 記錄每一次搶票嘗試的結果
type GrabResult struct {
	// UserID 用戶 ID
	UserID string `json:"user_id"`
	// Success 是否搶票成功
	Success bool `json:"success"`
	// Message 結果訊息
	Message string `json:"message"`
	// ResponseTime 回應時間（毫秒）
	ResponseTime float64 `json:"response_time_ms"`
}

// SimulationResult 模擬結果
// 一次完整模擬的統計摘要
type SimulationResult struct {
	// EventID 活動 ID
	EventID string `json:"event_id"`
	// TotalRequests 總請求數
	TotalRequests int `json:"total_requests"`
	// SuccessCount 成功數
	SuccessCount int `json:"success_count"`
	// FailCount 失敗數
	FailCount int `json:"fail_count"`
	// SuccessRate 成功率（百分比）
	SuccessRate float64 `json:"success_rate"`
	// AvgResponseTime 平均回應時間（毫秒）
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	// MinResponseTime 最小回應時間（毫秒）
	MinResponseTime float64 `json:"min_response_time_ms"`
	// MaxResponseTime 最大回應時間（毫秒）
	MaxResponseTime float64 `json:"max_response_time_ms"`
	// Results 所有搶票結果明細
	Results []GrabResult `json:"results"`
}

// Stats 全域統計資訊
// 記錄系統運行以來的累計統計資料
type Stats struct {
	// TotalAttempts 總搶票嘗試次數
	TotalAttempts int64 `json:"total_attempts"`
	// TotalSuccess 總成功次數
	TotalSuccess int64 `json:"total_success"`
	// TotalFail 總失敗次數
	TotalFail int64 `json:"total_fail"`
	// SuccessRate 整體成功率（百分比）
	SuccessRate float64 `json:"success_rate"`
	// AvgResponseTime 整體平均回應時間（毫秒）
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	// TotalResponseTime 總回應時間（用於計算平均值）
	TotalResponseTime float64 `json:"total_response_time_ms"`
}

// APIResponse 統一 API 回應格式
// 所有 API 端點皆使用此結構回傳結果
type APIResponse struct {
	// Code 狀態碼，0 表示成功，非 0 表示錯誤
	Code int `json:"code"`
	// Message 回應訊息
	Message string `json:"message"`
	// Data 回應資料（可為任意類型）
	Data interface{} `json:"data,omitempty"`
}
