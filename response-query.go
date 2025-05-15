// response.go

package main

import (
	"encoding/json"
	"fmt"
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

// calendar にアクセスされたとき実行される処理
func gethandler(w http.ResponseWriter, r *http.Request) {

    // CORS設定
	setCORS(w)
	
	// クエリパラメータを取得・解析（?year=2024&month=5&move=prev など）
    baseYear, baseMonth, moveStr, err := parseQueryParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 指定された年・月を調整（前月・翌月など）
	baseYear, baseMonth = adjustDate(baseYear, baseMonth, moveStr)

	// 月末の日付計算
	endOfMonth := getEndOfMonth(baseYear, baseMonth) 

	// 指定月の日付データ作成
	days := generateDays(baseYear, baseMonth, endOfMonth)

	// カレンダーのレスポンスを作成して送信
	if err := sendCalendarResponse(w, baseYear, baseMonth, days); err != nil {
		http.Error(w, "JSONエンコードに失敗しました", http.StatusInternalServerError)
		fmt.Println("JSONエンコードエラー:", err)
		return
	}
}

// カレンダーのレスポンスを作成して送信する処理
func sendCalendarResponse(w http.ResponseWriter, baseYear, baseMonth int, days []Day) error {
	today := time.Now().In(time.UTC)

	// レスポンス用の構造体の作成
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Day:   today.Day(),                      // 今日の日にち
		Week:  today.Weekday().String()[:3],     // 今日の曜日
		Days:  days,
	}

	// JSONでレスポンスを返す
	return sendJSONResponse(w, resp)
}

// 任意のデータをJSON形式に変換してレスポンスとして返す
func sendJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}