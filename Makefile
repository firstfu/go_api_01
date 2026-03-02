# Go 搶票模擬系統 — Makefile
# 提供常用指令的快捷方式，方便開發與測試使用
#
# Windows 相容說明：
#   所有中文輸出使用 \uXXXX 跳脫序列，透過 Python 解碼顯示，
#   避免 Windows 終端機 codepage 造成亂碼。

# 變數設定
APP_NAME := go_api_01
PORT := 8080
BASE_URL := http://localhost:$(PORT)

# JSON 格式化指令（支援中文顯示）
JSON_FMT = python -c "import sys,json;print(json.dumps(json.load(sys.stdin),ensure_ascii=False,indent=4))"

# 環境變數：確保 Python 使用 UTF-8
export PYTHONUTF8 := 1

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
	  -d "{\"name\":\"\\u4e94\\u6708\\u5929\\u6f14\\u5531\\u6703\",\"description\":\"2026\\u5de1\\u8ff4\",\"total_tickets\":100,\"price\":2800}" | $(JSON_FMT)

## 查詢活動（預設 evt-1，可用 make query-event ID=evt-2）
.PHONY: query-event
query-event:
	curl -s $(BASE_URL)/api/events/$(or $(ID),evt-1) | $(JSON_FMT)

## 單人搶票（預設 evt-1 / user-001 / 1張）
.PHONY: grab
grab:
	curl -s -X POST $(BASE_URL)/api/events/$(or $(ID),evt-1)/grab \
	  -H "Content-Type: application/json" \
	  -d "{\"user_id\":\"$(or $(USER),user-001)\",\"quantity\":$(or $(QTY),1)}" | $(JSON_FMT)

## 查詢訂單（預設 evt-1）
.PHONY: orders
orders:
	curl -s $(BASE_URL)/api/events/$(or $(ID),evt-1)/orders | $(JSON_FMT)

## 查看全域統計
.PHONY: stats
stats:
	curl -s $(BASE_URL)/api/stats | $(JSON_FMT)

# ─────────────────────────────────────────────
# 模擬搶票
# ─────────────────────────────────────────────

## 模擬 500 人搶票（預設 evt-1，每人 1 張）
.PHONY: sim
sim:
	curl -s -X POST $(BASE_URL)/api/simulate \
	  -H "Content-Type: application/json" \
	  -d "{\"event_id\":\"$(or $(ID),evt-1)\",\"concurrency\":$(or $(N),500),\"per_user\":$(or $(QTY),1)}" | $(JSON_FMT)

## 完整流程：建立活動 → 模擬搶票 → 查看統計
.PHONY: demo
demo:
	@python -c "print('=== 1. \\u5efa\\u7acb\\u6d3b\\u52d5\\uff08100 \\u5f35\\u7968\\uff09===')"
	@curl -s -X POST $(BASE_URL)/api/events \
	  -H "Content-Type: application/json" \
	  -d "{\"name\":\"Demo\",\"total_tickets\":100,\"price\":500}" | $(JSON_FMT)
	@python -c "print()"
	@python -c "print('=== 2. \\u6a21\\u64ec 500 \\u4eba\\u6436\\u7968 ===')"
	@curl -s -X POST $(BASE_URL)/api/simulate \
	  -H "Content-Type: application/json" \
	  -d "{\"event_id\":\"evt-1\",\"concurrency\":500,\"per_user\":1}" | $(JSON_FMT)
	@python -c "print()"
	@python -c "print('=== 3. \\u67e5\\u770b\\u7d71\\u8a08 ===')"
	@curl -s $(BASE_URL)/api/stats | $(JSON_FMT)

# ─────────────────────────────────────────────
# 瀏覽器開啟（需先啟動伺服器）
# ─────────────────────────────────────────────

## 用瀏覽器開啟活動資訊
.PHONY: open-event
open-event:
	cmd /c start $(BASE_URL)/api/events/$(or $(ID),evt-1)

## 用瀏覽器開啟訂單列表
.PHONY: open-orders
open-orders:
	cmd /c start $(BASE_URL)/api/events/$(or $(ID),evt-1)/orders

## 用瀏覽器開啟全域統計
.PHONY: open-stats
open-stats:
	cmd /c start $(BASE_URL)/api/stats

# ─────────────────────────────────────────────
# 說明
# ─────────────────────────────────────────────

## 顯示所有可用指令
.PHONY: help
help:
	@python cmd/help.py
