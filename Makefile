# Go 搶票模擬系統 — Makefile
# 提供常用指令的快捷方式，方便開發與測試使用

# 變數設定
APP_NAME := go_api_01
PORT := 8080
BASE_URL := http://localhost:$(PORT)

# ─────────────────────────────────────────────
# 開發指令
# ─────────────────────────────────────────────

## 啟動伺服器
.PHONY: run
run:
	go run main.go

## 編譯執行檔
.PHONY: build
build:
	go build -o $(APP_NAME).exe main.go

## 執行測試
.PHONY: test
test:
	go test ./... -v

## 整理依賴
.PHONY: tidy
tidy:
	go mod tidy

# ─────────────────────────────────────────────
# API 操作（需先 make run 啟動伺服器）
# ─────────────────────────────────────────────

## 建立活動（100 張票，票價 2800）
.PHONY: create-event
create-event:
	curl -s -X POST $(BASE_URL)/api/events \
	  -H "Content-Type: application/json" \
	  -d '{"name":"五月天演唱會","description":"2026巡迴","total_tickets":100,"price":2800}' | python -m json.tool

## 查詢活動（預設 evt-1，可用 make query-event ID=evt-2）
.PHONY: query-event
query-event:
	curl -s $(BASE_URL)/api/events/$(or $(ID),evt-1) | python -m json.tool

## 單人搶票（預設 evt-1 / user-001 / 1張）
.PHONY: grab
grab:
	curl -s -X POST $(BASE_URL)/api/events/$(or $(ID),evt-1)/grab \
	  -H "Content-Type: application/json" \
	  -d '{"user_id":"$(or $(USER),user-001)","quantity":$(or $(QTY),1)}' | python -m json.tool

## 查詢訂單（預設 evt-1）
.PHONY: orders
orders:
	curl -s $(BASE_URL)/api/events/$(or $(ID),evt-1)/orders | python -m json.tool

## 查看全域統計
.PHONY: stats
stats:
	curl -s $(BASE_URL)/api/stats | python -m json.tool

# ─────────────────────────────────────────────
# 模擬搶票
# ─────────────────────────────────────────────

## 模擬 500 人搶票（預設 evt-1，每人 1 張）
.PHONY: sim
sim:
	curl -s -X POST $(BASE_URL)/api/simulate \
	  -H "Content-Type: application/json" \
	  -d '{"event_id":"$(or $(ID),evt-1)","concurrency":$(or $(N),500),"per_user":$(or $(QTY),1)}' | python -m json.tool

## 完整流程：建立活動 → 模擬搶票 → 查看統計
.PHONY: demo
demo:
	@echo "=== 1. 建立活動（100 張票）==="
	@curl -s -X POST $(BASE_URL)/api/events \
	  -H "Content-Type: application/json" \
	  -d '{"name":"Demo 活動","total_tickets":100,"price":500}' | python -m json.tool
	@echo ""
	@echo "=== 2. 模擬 500 人搶票 ==="
	@curl -s -X POST $(BASE_URL)/api/simulate \
	  -H "Content-Type: application/json" \
	  -d '{"event_id":"evt-1","concurrency":500,"per_user":1}' | python -m json.tool
	@echo ""
	@echo "=== 3. 查看統計 ==="
	@curl -s $(BASE_URL)/api/stats | python -m json.tool

# ─────────────────────────────────────────────
# 瀏覽器開啟（需先啟動伺服器）
# ─────────────────────────────────────────────

## 用瀏覽器開啟活動資訊
.PHONY: open-event
open-event:
	start $(BASE_URL)/api/events/$(or $(ID),evt-1)

## 用瀏覽器開啟訂單列表
.PHONY: open-orders
open-orders:
	start $(BASE_URL)/api/events/$(or $(ID),evt-1)/orders

## 用瀏覽器開啟全域統計
.PHONY: open-stats
open-stats:
	start $(BASE_URL)/api/stats

# ─────────────────────────────────────────────
# 說明
# ─────────────────────────────────────────────

## 顯示所有可用指令
.PHONY: help
help:
	@echo "Go 搶票模擬系統 — 可用指令"
	@echo "────────────────────────────────────────"
	@echo ""
	@echo "  開發："
	@echo "    make run             啟動伺服器"
	@echo "    make build           編譯執行檔"
	@echo "    make test            執行測試"
	@echo "    make tidy            整理依賴"
	@echo ""
	@echo "  API 操作（需先啟動伺服器）："
	@echo "    make create-event    建立活動（100張票）"
	@echo "    make query-event     查詢活動        ID=evt-2"
	@echo "    make grab            單人搶票        ID=evt-1 USER=user-001 QTY=1"
	@echo "    make orders          查詢訂單        ID=evt-1"
	@echo "    make stats           查看全域統計"
	@echo ""
	@echo "  模擬："
	@echo "    make sim             模擬搶票        ID=evt-1 N=500 QTY=1"
	@echo "    make demo            完整流程展示（建立→模擬→統計）"
	@echo ""
	@echo "  瀏覽器開啟："
	@echo "    make open-event      開啟活動資訊    ID=evt-1"
	@echo "    make open-orders     開啟訂單列表    ID=evt-1"
	@echo "    make open-stats      開啟全域統計"
	@echo ""
	@echo "  範例："
	@echo "    make sim N=1000              1000人搶票"
	@echo "    make sim ID=evt-2 N=200      指定活動、200人"
	@echo "    make grab ID=evt-3 QTY=2     搶 evt-3 的 2 張票"
