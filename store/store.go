// store/store.go
// 記憶體存儲層
// 提供線程安全的 in-memory 資料存儲，使用 sync.RWMutex 保護 map 結構，
// 使用 sync/atomic 操作統計計數器以達到高效能的併發安全。
// 此層僅負責資料的讀寫，不包含業務邏輯。

package store

import (
	"fmt"
	"go_api_01/model"
	"sync"
	"sync/atomic"
	"time"
)

// Store 記憶體存儲結構
// 包含活動、訂單的 map 存儲，以及全域統計計數器
type Store struct {
	// mu 讀寫鎖，保護 events 和 orders map 的併發安全
	mu sync.RWMutex
	// events 活動存儲（key: 活動 ID）
	events map[string]*model.Event
	// orders 訂單存儲（key: 活動 ID，value: 該活動的所有訂單）
	orders map[string][]*model.Order
	// eventCounter 活動 ID 自增計數器
	eventCounter int64
	// orderCounter 訂單 ID 自增計數器
	orderCounter int64

	// totalAttempts 搶票總嘗試次數（atomic 操作）
	totalAttempts int64
	// totalSuccess 搶票總成功次數（atomic 操作）
	totalSuccess int64
	// totalFail 搶票總失敗次數（atomic 操作）
	totalFail int64
	// totalResponseTimeNs 總回應時間（奈秒，atomic 操作）
	totalResponseTimeNs int64
}

// NewStore 建立新的記憶體存儲實例
// 初始化所有 map 並回傳 Store 指標
func NewStore() *Store {
	return &Store{
		events: make(map[string]*model.Event),
		orders: make(map[string][]*model.Order),
	}
}

// CreateEvent 建立新活動
// 自動產生唯一 ID，設定剩餘票數等於總票數，記錄建立時間
func (s *Store) CreateEvent(event *model.Event) *model.Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 產生唯一活動 ID
	s.eventCounter++
	event.ID = fmt.Sprintf("evt-%d", s.eventCounter)
	event.Remaining = event.TotalTickets
	event.CreatedAt = time.Now()

	// 存入 map
	s.events[event.ID] = event
	return event
}

// GetEvent 根據 ID 查詢活動
// 回傳活動指標；若不存在則回傳 nil
func (s *Store) GetEvent(id string) *model.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.events[id]
}

// UpdateEvent 更新活動資料
// 直接將傳入的活動寫入 map（呼叫端需確保資料正確性）
func (s *Store) UpdateEvent(event *model.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[event.ID] = event
}

// CreateOrder 建立新訂單
// 自動產生唯一訂單 ID 並記錄建立時間，將訂單歸入對應活動
func (s *Store) CreateOrder(order *model.Order) *model.Order {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 產生唯一訂單 ID
	s.orderCounter++
	order.ID = fmt.Sprintf("ord-%d", s.orderCounter)
	order.CreatedAt = time.Now()

	// 歸入對應活動的訂單列表
	s.orders[order.EventID] = append(s.orders[order.EventID], order)
	return order
}

// GetOrdersByEvent 查詢指定活動的所有訂單
// 回傳訂單切片；若無訂單則回傳空切片
func (s *Store) GetOrdersByEvent(eventID string) []*model.Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := s.orders[eventID]
	if orders == nil {
		return []*model.Order{}
	}
	return orders
}

// IncrementAttempts 增加搶票嘗試次數（atomic 操作，無需加鎖）
func (s *Store) IncrementAttempts() {
	atomic.AddInt64(&s.totalAttempts, 1)
}

// IncrementSuccess 增加搶票成功次數（atomic 操作，無需加鎖）
func (s *Store) IncrementSuccess() {
	atomic.AddInt64(&s.totalSuccess, 1)
}

// IncrementFail 增加搶票失敗次數（atomic 操作，無需加鎖）
func (s *Store) IncrementFail() {
	atomic.AddInt64(&s.totalFail, 1)
}

// AddResponseTime 累加回應時間（atomic 操作，無需加鎖）
// 參數 d 為該次搶票的回應時間（time.Duration）
func (s *Store) AddResponseTime(d time.Duration) {
	atomic.AddInt64(&s.totalResponseTimeNs, int64(d))
}

// GetEventSnapshot 取得活動的快照副本（線程安全）
// 回傳活動的深拷貝，避免外部直接修改 store 內部資料
// 適用於模擬結束後讀取活動最終狀態的場景
func (s *Store) GetEventSnapshot(id string) *model.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event := s.events[id]
	if event == nil {
		return nil
	}
	// 回傳副本，避免外部修改影響 store 內部資料
	snapshot := *event
	return &snapshot
}

// GetStats 取得全域統計資訊
// 計算成功率和平均回應時間後回傳 Stats 結構
func (s *Store) GetStats() model.Stats {
	attempts := atomic.LoadInt64(&s.totalAttempts)
	success := atomic.LoadInt64(&s.totalSuccess)
	fail := atomic.LoadInt64(&s.totalFail)
	totalNs := atomic.LoadInt64(&s.totalResponseTimeNs)

	var successRate float64
	var avgResponseTime float64

	if attempts > 0 {
		// 成功率 = 成功次數 / 總嘗試次數 * 100
		successRate = float64(success) / float64(attempts) * 100
		// 平均回應時間 = 總回應時間(ms) / 總嘗試次數
		avgResponseTime = float64(totalNs) / float64(time.Millisecond) / float64(attempts)
	}

	return model.Stats{
		TotalAttempts:     attempts,
		TotalSuccess:      success,
		TotalFail:         fail,
		SuccessRate:       successRate,
		AvgResponseTime:   avgResponseTime,
		TotalResponseTime: float64(totalNs) / float64(time.Millisecond),
	}
}
