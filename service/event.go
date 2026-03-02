// service/event.go
// 活動管理業務邏輯層
// 封裝活動的建立與查詢邏輯，作為 handler 與 store 之間的橋樑。
// 此層負責業務驗證，store 層負責資料存取。

package service

import (
	"go_api_01/model"
	"go_api_01/store"
)

// EventService 活動管理服務
// 封裝活動相關的業務邏輯
type EventService struct {
	// store 記憶體存儲實例
	store *store.Store
}

// NewEventService 建立新的活動服務實例
// 參數 s 為記憶體存儲實例
func NewEventService(s *store.Store) *EventService {
	return &EventService{store: s}
}

// CreateEvent 建立新活動
// 將活動資料傳遞給 store 層處理 ID 產生與時間戳記
func (es *EventService) CreateEvent(event *model.Event) *model.Event {
	return es.store.CreateEvent(event)
}

// GetEvent 根據 ID 查詢活動
// 若活動不存在則回傳 nil
func (es *EventService) GetEvent(id string) *model.Event {
	return es.store.GetEvent(id)
}
