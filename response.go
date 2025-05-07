// response.go

package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// カレンダーのレスポンスに使う構造体
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Day   int   `json:"day"`   // 今日の日にち
	Week  string `json:"week"` // 今日の曜日
	Days  []Day `json:"days"`  // 各日の配列
}

// カレンダーのレスポンスを作成して送信する処理
func sendCalendarResponse(w http.ResponseWriter, baseYear, baseMonth int, days []Day) error {
	
	now := time.Now()
	today := now.In(time.UTC) // UTCに統一

	// レスポンス用の構造体の作成
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Day:   today.Day(),                      // 追加：今日の日
		Week:  today.Weekday().String()[:3],
		Days:  days,
	}

	// クライアントへJSON形式でレスポンスを返す
	return sendJSONResponse(w, resp)
}

// レスポンスのデータ型をJSONに、resp(データ) → Encode(変換) → レスポンス出力
func sendJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}