// service/simulator.go
// 併發模擬器業務邏輯層
// 使用 goroutine + WaitGroup + buffered channel 實現 fan-out/fan-in 模式。
// fan-out：啟動 N 個 goroutine 同時搶票
// fan-in：透過 buffered channel 收集所有結果，彙整統計
// 此模組用於模擬大量用戶同時搶票的場景，測試系統的併發處理能力。

package service

import (
	"fmt"
	"go_api_01/model"
	"go_api_01/store"
	"math"
	"sync"
)

// SimulatorService 模擬器服務
// 封裝搶票模擬邏輯，依賴 TicketService 執行實際搶票
type SimulatorService struct {
	// ticketService 搶票服務實例
	ticketService *TicketService
	// store 記憶體存儲實例（用於查詢活動是否存在）
	store *store.Store
}

// NewSimulatorService 建立新的模擬器服務實例
func NewSimulatorService(ts *TicketService, s *store.Store) *SimulatorService {
	return &SimulatorService{
		ticketService: ts,
		store:         s,
	}
}

// RunSimulation 執行搶票模擬
// 使用 fan-out/fan-in 模式：
// 1. 建立 buffered channel（容量 = 併發數）和 WaitGroup
// 2. fan-out：啟動 N 個 goroutine 同時搶票，每個結果送入 channel
// 3. 獨立 goroutine 等待 WaitGroup 完成後關閉 channel
// 4. fan-in：主 goroutine 從 channel 收集結果並計算統計
func (ss *SimulatorService) RunSimulation(req *model.SimulateRequest) (*model.SimulationResult, error) {
	// 驗證活動是否存在
	event := ss.store.GetEvent(req.EventID)
	if event == nil {
		return nil, fmt.Errorf("活動 %s 不存在", req.EventID)
	}

	concurrency := req.Concurrency
	perUser := req.PerUser

	// 建立 buffered channel，容量等於併發數，避免 goroutine 阻塞
	resultCh := make(chan model.GrabResult, concurrency)

	// WaitGroup 追蹤所有搶票 goroutine 的完成狀態
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// fan-out：啟動 N 個 goroutine 同時搶票
	for i := 0; i < concurrency; i++ {
		go func(userIndex int) {
			defer wg.Done()
			userID := fmt.Sprintf("sim-user-%d", userIndex+1)

			// 呼叫搶票服務執行實際搶票
			result, _ := ss.ticketService.GrabTicket(req.EventID, userID, perUser)
			if result != nil {
				resultCh <- *result
			}
		}(i)
	}

	// 獨立 goroutine：等待所有搶票完成後關閉 channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// fan-in：從 channel 收集所有結果
	var results []model.GrabResult
	var successCount, failCount int
	var totalResponseTime, minResponseTime, maxResponseTime float64
	minResponseTime = math.MaxFloat64

	for result := range resultCh {
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failCount++
		}

		totalResponseTime += result.ResponseTime

		if result.ResponseTime < minResponseTime {
			minResponseTime = result.ResponseTime
		}
		if result.ResponseTime > maxResponseTime {
			maxResponseTime = result.ResponseTime
		}
	}

	// 計算統計數據
	totalRequests := len(results)
	var successRate, avgResponseTime float64

	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests) * 100
		avgResponseTime = totalResponseTime / float64(totalRequests)
	}

	// 若無結果，將最小回應時間設為 0
	if minResponseTime == math.MaxFloat64 {
		minResponseTime = 0
	}

	return &model.SimulationResult{
		EventID:         req.EventID,
		TotalRequests:   totalRequests,
		SuccessCount:    successCount,
		FailCount:       failCount,
		SuccessRate:     successRate,
		AvgResponseTime: avgResponseTime,
		MinResponseTime: minResponseTime,
		MaxResponseTime: maxResponseTime,
		Results:         results,
	}, nil
}
