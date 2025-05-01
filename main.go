package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// 構造体Dayの作成（１日のデータをひとまとめ）
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Day  int    `json:"day"`  // 数値の「日」
}

// 構造体CalendarResponseの作成（レスポンス用に年・月・Dayリストを一元管理）
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Days  []Day `json:"days"`  // 各日の配列
}

// APIのエンドポイントのハンドラー関数
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")           // TSからリクエストを送れるようにする許可

	// クエリパラメータから年・月・移動方向を取得
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	moveStr := r.URL.Query().Get("move")

	// カレンダーの基準になる年・月
	var baseYear, baseMonth int
	
	// クエリパラメータより指定の日時・現在時間を格納
	if yearStr != "" && monthStr != "" {
		var err error
		baseYear, err = strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year parameter", http.StatusBadRequest)
			return
		}

		baseMonth, err = strconv.Atoi(monthStr)
		if err != nil {
			http.Error(w, "Invalid month parameter", http.StatusBadRequest)
			return
		}

		if baseMonth < 1 || baseMonth > 12 {
			http.Error(w, "Month must be between 1 and 12", http.StatusBadRequest)
			return
		}

	} else {
		now := time.Now()
		baseYear = now.Year()
		baseMonth = int(now.Month())
	}

	// moveStr の値チェック
	if moveStr != "" && moveStr != "next" && moveStr != "prev" {
		http.Error(w, "Invalid move parameter. Use 'next', 'prev', or leave empty.", http.StatusBadRequest)
		return
	}

	// 現在の年月を「1日」で作る（AddDateでズレないように）
	baseDate := time.Date(baseYear, time.Month(baseMonth), 1, 0, 0, 0, 0, time.UTC)

	// 月移動処理
	switch moveStr {
	case "next":
    	baseDate = baseDate.AddDate(0, 1, 0) // 1ヶ月進める
	case "prev":
  	  baseDate = baseDate.AddDate(0, -1, 0) // 1ヶ月戻す
	}

	// 加算後の年月を再代入
	baseYear = baseDate.Year()
	baseMonth = int(baseDate.Month())

	daysInMonth := time.Date(baseYear, time.Month(baseMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()  // 指定された月の月末までの日数を計算

	var days []Day
	for i := 1; i <= daysInMonth; i++ {                                                         // 1日～月末までループし、１日ずつDay型でデータを作成
		date := time.Date(baseYear, time.Month(baseMonth), i, 0, 0, 0, 0, time.UTC)
		days = append(days, Day{                                                                // スライスに格納
			Date: date.Format("2006-01-02"), // Go特有の日時フォーマット
			Day:  i,
		})
	}

	// レスポンス用の構造体の作成：CalendarResponse に実際のデータを入れる
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Days:  days,
	}
    
	// レスポンスのデータ型をJSONに、resp(データ) → Encode(変換) → レスポンス出力
	w.Header().Set("Content-Type", "application/json")           
	// クライアントへ、JSON形式でレスポンスを返す
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "JSON encoding failed", http.StatusInternalServerError)
		fmt.Println("JSONエンコードエラー:", err)
		return
	}
}

// サーバーのエントリーポイント
func main() {
	// api/calendar というURLに来たリクエストは、calendarHandler という関数で処理する設定
	http.HandleFunc("/api/calendar", calendarHandler) // エンドポイントを登録
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)                 // ポート8080で待機、リクエスト時にHandleFuncが実行される
}
