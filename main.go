package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// 各日のデータ構造
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Day  int    `json:"day"`  // 数値の「日」
}

// レスポンス全体の構造
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Days  []Day `json:"days"`  // 各日の配列
}

// APIのエンドポイントのハンドラー関数
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	// CORSの許可とレスポンスのContent-Typeを設定
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// クエリパラメータから年・月・移動方向を取得
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	moveStr := r.URL.Query().Get("move")

	var baseYear int
	var baseMonth int

	// クエリに年と月が指定されている場合、それを使用
	if yearStr != "" && monthStr != "" {
		baseYear, _ = strconv.Atoi(yearStr)
		baseMonth, _ = strconv.Atoi(monthStr)
	} else {
		// 指定がなければ現在の年月を使用
		now := time.Now()
		baseYear = now.Year()
		baseMonth = int(now.Month())
	}

	// "next" または "prev" の指示があれば月を移動
	switch moveStr {
	case "next":
		if baseMonth == 12 {
			baseMonth = 1
			baseYear++
		} else {
			baseMonth++
		}
	case "prev":
		if baseMonth == 1 {
			baseMonth = 12
			baseYear--
		} else {
			baseMonth--
		}
	}

	// 指定された月の日数を取得
	daysInMonth := time.Date(baseYear, time.Month(baseMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	var days []Day
	// 1日から月末までループして日付を作成
	for i := 1; i <= daysInMonth; i++ {
		date := time.Date(baseYear, time.Month(baseMonth), i, 0, 0, 0, 0, time.UTC)
		days = append(days, Day{
			Date: date.Format("2006-01-02"), // Go独特の日時フォーマット
			Day:  i,
		})
	}

	// レスポンス用の構造体を生成
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Days:  days,
	}

	// JSONにエンコードしてレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// サーバーのエントリーポイント
func main() {
	http.HandleFunc("/api/calendar", calendarHandler) // エンドポイントを登録
	fmt.Println("Listening on :8080")                 // 起動ログ
	http.ListenAndServe(":8080", nil)                 // ポート8080でリクエストを待機
}
