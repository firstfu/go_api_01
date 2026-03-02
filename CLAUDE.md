# Go 搶票模擬系統 — Claude Code 專案指引

## 專案概述

這是一個 Go 搶票模擬後端系統，學習 Go 高併發處理模式。使用 Gin 框架 + 純記憶體儲存。

## 專案路徑

- 專案根目錄: `D:\myCodeProject\go_api_01`
- 文件目錄: `docs/`（prd.md、todo.md、usage.md）

## 技術棧

- Go 1.22+ / Gin 框架
- 純記憶體儲存（無資料庫）
- 併發控制: sync.Mutex、sync.RWMutex、sync/atomic、goroutine、channel、WaitGroup

## 架構分層

```
Handler（HTTP 處理）→ Service（業務邏輯）→ Store（記憶體存儲）
```

- `model/model.go` — 所有資料結構（Event, Order, GrabRequest 等）
- `store/store.go` — 線程安全的記憶體存儲（RWMutex + atomic）
- `service/event.go` — 活動 CRUD 邏輯
- `service/ticket.go` — **搶票核心**（Mutex 保護原子性，防止超賣）
- `service/simulator.go` — 併發模擬器（fan-out/fan-in 模式）
- `handler/` — HTTP 請求處理與參數驗證
- `router/router.go` — 路由註冊與 CORS 中間件
- `main.go` — 依賴組裝與伺服器啟動

## API 端點

| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/events` | `handler.EventHandler.CreateEvent` |
| GET | `/api/events/:id` | `handler.EventHandler.GetEvent` |
| POST | `/api/events/:id/grab` | `handler.TicketHandler.GrabTicket` |
| GET | `/api/events/:id/orders` | `handler.TicketHandler.GetOrders` |
| POST | `/api/simulate` | `handler.SimulateHandler.StartSimulation` |
| GET | `/api/stats` | `handler.SimulateHandler.GetStats` |

## 開發規範

1. 所有檔案頂部加上繁體中文註解，說明該檔案的職責。
2. 所有 struct、方法、函數都加上繁體中文註解。
3. 統一回應格式: `model.APIResponse{Code, Message, Data}`。
4. 文件管理: prd.md（需求）、todo.md（待辦）、usage.md（使用說明）放在 `docs/` 下。
5. 完成功能後在 `docs/todo.md` 中劃掉對應項目。

## 啟動方式

```bash
go run main.go    # 監聽 :8080
```

## 併發設計要點

- **兩層鎖分離**: store.mu（RWMutex）保護 map，ticketService.mu（Mutex）保護搶票原子性。
- **不用 defer Unlock**: ticket.go 中手動 Unlock 以減少鎖持有時間。
- **atomic 統計**: 計數器用 sync/atomic，不需額外加鎖。
- **fan-out/fan-in**: 模擬器用 buffered channel + WaitGroup 收集結果。
