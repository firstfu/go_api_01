// service/simulator.go
// 併發模擬器業務邏輯層
// 使用 goroutine + WaitGroup + buffered channel 實現 fan-out/fan-in 模式。
// fan-out：啟動 N 個 goroutine 同時搶票
// fan-in：透過 reporter goroutine 消費結果，匯報進度並收集統計
// reporter 每完成 10% 輸出 console log，並推送 ProgressEvent 至 progressCh（若有）
// 此模組用於模擬大量用戶同時搶票的場景，測試系統的併發處理能力。

package service

import (
	"fmt"
	"go_api_01/model"
	"go_api_01/store"
	"log"
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
// 使用 fan-out/fan-in 模式，並透過 reporter goroutine 匯報進度：
// 1. fan-out：啟動 N 個 goroutine 同時搶票，結果送入 resultCh
// 2. reporter goroutine：消費 resultCh，每完成 10% 輸出 console log + 推送 progressCh
// 3. 主流程等待 reporter 完成後彙整最終結果
//
// progressCh 可為 nil（向後相容原有 POST /api/simulate 呼叫）
func (ss *SimulatorService) RunSimulation(req *model.SimulateRequest, progressCh chan<- model.ProgressEvent) (*model.SimulationResult, error) {
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

	// reporterResult 匯報器結果結構（非匯出）
	// 用於 reporter goroutine 透過 doneCh 將彙整結果傳回主流程
	type reporterResult struct {
		results       []model.GrabResult
		successCount  int
		failCount     int
		totalRespTime float64
		minRespTime   float64
		maxRespTime   float64
	}

	// doneCh 用於接收 reporter goroutine 彙整完成的結果
	doneCh := make(chan reporterResult, 1)

	// 計算匯報間隔：每完成 10%（至少每 1 筆）匯報一次
	reportInterval := concurrency / 10
	if reportInterval < 1 {
		reportInterval = 1
	}

	// reporter goroutine：消費 resultCh，匯報進度並收集結果
	go func() {
		var rr reporterResult
		rr.minRespTime = math.MaxFloat64
		count := 0

		for result := range resultCh {
			// 收集結果
			rr.results = append(rr.results, result)
			count++

			if result.Success {
				rr.successCount++
			} else {
				rr.failCount++
			}

			rr.totalRespTime += result.ResponseTime

			if result.ResponseTime < rr.minRespTime {
				rr.minRespTime = result.ResponseTime
			}
			if result.ResponseTime > rr.maxRespTime {
				rr.maxRespTime = result.ResponseTime
			}

			// 每 N 筆匯報一次進度
			if count%reportInterval == 0 || count == concurrency {
				// 取得目前活動剩餘票數
				snapshot := ss.store.GetEventSnapshot(req.EventID)
				remaining := 0
				if snapshot != nil {
					remaining = snapshot.Remaining
				}

				pct := float64(count) / float64(concurrency) * 100

				// console log 輸出
				if count == concurrency {
					log.Printf("[模擬進度] [%d/%d] 成功: %d | 失敗: %d | 剩餘票數: %d ✓ 完成",
						count, concurrency, rr.successCount, rr.failCount, remaining)
				} else {
					log.Printf("[模擬進度] [%d/%d] 成功: %d | 失敗: %d | 剩餘票數: %d",
						count, concurrency, rr.successCount, rr.failCount, remaining)
				}

				// 推送至 progressCh（若有）
				if progressCh != nil {
					progressCh <- model.ProgressEvent{
						Current:      count,
						Total:        concurrency,
						SuccessCount: rr.successCount,
						FailCount:    rr.failCount,
						Remaining:    remaining,
						Percentage:   pct,
						Done:         count == concurrency,
					}
				}
			}
		}

		// 若無結果，將最小回應時間設為 0
		if rr.minRespTime == math.MaxFloat64 {
			rr.minRespTime = 0
		}

		doneCh <- rr
	}()

	// 等待 reporter 完成
	rr := <-doneCh

	// 若 progressCh 不為 nil，關閉它
	if progressCh != nil {
		close(progressCh)
	}

	// 計算統計數據
	totalRequests := len(rr.results)
	var successRate, avgResponseTime float64

	if totalRequests > 0 {
		successRate = float64(rr.successCount) / float64(totalRequests) * 100
		avgResponseTime = rr.totalRespTime / float64(totalRequests)
	}

	return &model.SimulationResult{
		EventID:         req.EventID,
		TotalRequests:   totalRequests,
		SuccessCount:    rr.successCount,
		FailCount:       rr.failCount,
		SuccessRate:     successRate,
		AvgResponseTime: avgResponseTime,
		MinResponseTime: rr.minRespTime,
		MaxResponseTime: rr.maxRespTime,
		Results:         rr.results,
	}, nil
}
