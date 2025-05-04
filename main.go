package main

import (
	"fmt"
	"net/http"
)

// カレンダーのレスポンスに使う構造体
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Days  []Day `json:"days"`  // 各日の配列
}

// サーバーのエントリーポイント
func main() {
	// api/calendar というURLに来たリクエストは、calendarHandler という関数で処理する設定
	http.HandleFunc("/api/calendar", handler) // エンドポイントを登録
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)                 // ポート8080で待機、リクエスト時にHandleFuncが実行される
}

// calendar にアクセスされたとき実行される処理
func handler(w http.ResponseWriter, r *http.Request) {
	// CORS設定
	setCORS(w)
	
	// クエリパラメータの解析
    baseYear, baseMonth, moveStr, err := parseQueryParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 日付の調整
	baseYear, baseMonth = adjustDate(baseYear, baseMonth, moveStr)

	// 月末計算
	endOfMonth := getEndOfMonth(baseYear, baseMonth) 

	// 日付データの作成
	days := generateDays(baseYear, baseMonth, endOfMonth)

	// カレンダーのレスポンスを作成して送信
	if err := sendCalendarResponse(w, baseYear, baseMonth, days); err != nil {
		http.Error(w, "JSON encoding failed", http.StatusInternalServerError)
		fmt.Println("JSONエンコードエラー:", err)
		return
	}
}

// カレンダーのレスポンスを作成して送信する処理
func sendCalendarResponse(w http.ResponseWriter, baseYear, baseMonth int, days []Day) error {
	// レスポンス用の構造体の作成
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Days:  days,
	}

	// クライアントへJSON形式でレスポンスを返す
	return sendJSONResponse(w, resp)
}