# Go 搶票模擬系統 — 產品需求文件（PRD）

## 1. 專案概述

建立一個 Go 搶票模擬後端系統，用於學習 Go 的高併發處理模式，包含 goroutine、sync.Mutex、channel、WaitGroup、sync/atomic 等核心機制。

## 2. 技術架構

- **語言**: Go 1.26
- **HTTP 框架**: Gin
- **儲存**: 純記憶體（sync.RWMutex 保護的 map）
- **併發控制**: sync.Mutex（搶票）、sync/atomic（統計計數器）
- **模擬器**: goroutine + WaitGroup + buffered channel（fan-out/fan-in 模式）

## 3. 功能需求

### 3.1 活動管理
- 建立售票活動（名稱、描述、總票數、票價）
- 查詢活動詳情（含剩餘票數）

### 3.2 搶票功能
- 用戶搶票（指定活動、用戶 ID、購買數量）
- 併發安全：透過 Mutex 確保不超賣
- 搶票成功自動建立訂單

### 3.3 訂單查詢
- 依活動 ID 查詢所有訂單

### 3.4 模擬功能
- 啟動 N 個 goroutine 同時搶票
- 統計模擬結果（成功率、平均回應時間等）

### 3.5 全域統計
- 查看累計的搶票嘗試、成功、失敗次數
- 查看成功率與平均回應時間

## 4. API 端點

| Method | Path | 說明 |
|--------|------|------|
| POST | /api/events | 建立活動 |
| GET | /api/events/:id | 查詢活動 |
| POST | /api/events/:id/grab | 搶票 |
| GET | /api/events/:id/orders | 查詢訂單 |
| POST | /api/simulate | 啟動搶票模擬 |
| GET | /api/stats | 查看全域統計 |

## 5. 核心併發設計

### 5.1 兩層鎖分離
- **store.mu (RWMutex)**: 保護 map 結構安全
- **ticketService.mu (Mutex)**: 保護搶票業務原子性

### 5.2 Atomic 操作
- 統計計數器使用 sync/atomic 進行高效累加

### 5.3 模擬器 fan-out/fan-in
1. 主 goroutine 建立 buffered channel + WaitGroup
2. 啟動 N 個 goroutine 同時搶票
3. 獨立 goroutine 等待完成後關閉 channel
4. 主 goroutine 收集結果並計算統計

## 6. 統一回應格式

```json
{"code": 0, "message": "成功訊息", "data": {...}}
```
