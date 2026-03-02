// service/ticket.go
// 搶票核心業務邏輯層（系統最關鍵的併發控制點）
// 使用 sync.Mutex 確保搶票操作的原子性：讀取庫存 → 判斷 → 扣減 → 建立訂單
// 這四個步驟必須在同一把鎖內完成，避免超賣問題。
// 統計計數器使用 store 層的 atomic 操作，不需額外加鎖。

package service

import (
	"fmt"
	"go_api_01/model"
	"go_api_01/store"
	"sync"
	"time"
)

// TicketService 搶票服務
// 持有獨立的 Mutex 鎖，與 store 層的 RWMutex 分離，
// 實現兩層鎖分離設計，減少鎖競爭。
type TicketService struct {
	// mu 搶票業務互斥鎖
	// 保護「讀庫存→判斷→扣減→下單」的原子性
	mu sync.Mutex
	// store 記憶體存儲實例
	store *store.Store
}

// NewTicketService 建立新的搶票服務實例
func NewTicketService(s *store.Store) *TicketService {
	return &TicketService{store: s}
}

// GrabTicket 搶票核心邏輯
// 此方法是整個系統的核心，透過 Mutex 確保搶票的原子性。
// 參數 eventID 為目標活動 ID，userID 為搶票用戶 ID，quantity 為購買數量。
// 回傳搶票結果（成功/失敗）與錯誤。
func (ts *TicketService) GrabTicket(eventID, userID string, quantity int) (*model.GrabResult, error) {
	startTime := time.Now()

	// 記錄搶票嘗試（atomic，不需鎖）
	ts.store.IncrementAttempts()

	// 加鎖：確保搶票操作的原子性
	ts.mu.Lock()

	// 讀取活動資料
	event := ts.store.GetEvent(eventID)
	if event == nil {
		ts.mu.Unlock()
		ts.store.IncrementFail()
		elapsed := time.Since(startTime)
		ts.store.AddResponseTime(elapsed)
		return &model.GrabResult{
			UserID:       userID,
			Success:      false,
			Message:      "活動不存在",
			ResponseTime: float64(elapsed.Nanoseconds()) / float64(time.Millisecond),
		}, fmt.Errorf("活動 %s 不存在", eventID)
	}

	// 判斷庫存是否足夠
	if event.Remaining < quantity {
		ts.mu.Unlock()
		ts.store.IncrementFail()
		elapsed := time.Since(startTime)
		ts.store.AddResponseTime(elapsed)
		return &model.GrabResult{
			UserID:       userID,
			Success:      false,
			Message:      fmt.Sprintf("票數不足，剩餘 %d 張", event.Remaining),
			ResponseTime: float64(elapsed.Nanoseconds()) / float64(time.Millisecond),
		}, nil
	}

	// 扣減庫存
	event.Remaining -= quantity

	// 更新活動資料
	ts.store.UpdateEvent(event)

	// 建立訂單
	order := &model.Order{
		EventID:    eventID,
		UserID:     userID,
		Quantity:   quantity,
		TotalPrice: float64(quantity) * event.Price,
	}
	ts.store.CreateOrder(order)

	// 解鎖：搶票操作完成
	ts.mu.Unlock()

	// 記錄成功統計（atomic，不需鎖）
	ts.store.IncrementSuccess()
	elapsed := time.Since(startTime)
	ts.store.AddResponseTime(elapsed)

	return &model.GrabResult{
		UserID:       userID,
		Success:      true,
		Message:      fmt.Sprintf("搶票成功！訂單編號: %s", order.ID),
		ResponseTime: float64(elapsed.Nanoseconds()) / float64(time.Millisecond),
	}, nil
}

// GetOrdersByEvent 查詢指定活動的所有訂單
func (ts *TicketService) GetOrdersByEvent(eventID string) []*model.Order {
	return ts.store.GetOrdersByEvent(eventID)
}
