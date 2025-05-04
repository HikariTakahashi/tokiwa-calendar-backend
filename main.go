package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// レスポンス用の構造体
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Days  []Day `json:"days"`  // 各日の配列
}

// ループ用の構造体
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Day  int    `json:"day"`  // 数値の「日」
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

	// レスポンス用の構造体の作成：CalendarResponse に実際のデータを入れる
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Days:  days,
	}
     
	// クライアントへ、JSON形式でレスポンスを返す
	if err := sendJSONResponse(w, resp); err != nil {
		http.Error(w, "JSON encoding failed", http.StatusInternalServerError)
		fmt.Println("JSONエンコードエラー:", err)
		return
	}
}

// クエリパラメータを解析して、年・月・移動方向を取得する関数
func parseQueryParams(r *http.Request)(int, int, string, error){
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	moveStr := r.URL.Query().Get("move")

	// 年と月の基準値を格納する変数
	var baseYear, baseMonth int

		// クエリパラメータがあればそれを基準に、なければ現在日時を基準に年と月を設定
	if yearStr != "" && monthStr != "" {
		var err error
		baseYear, err = strconv.Atoi(yearStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid year parameter")
		}

		baseMonth, err = strconv.Atoi(monthStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid month parameter")
		}

		if baseMonth < 1 || baseMonth > 12 {
			return 0, 0, "", fmt.Errorf("month must be between 1 and 12")
		}

	} else {
		now := time.Now()
		baseYear = now.Year()
		baseMonth = int(now.Month())
	}

	// moveStr の値チェック
	if moveStr != "" && moveStr != "next" && moveStr != "prev" {
		return 0, 0, "", fmt.Errorf("invalid move parameter")
	}
	return baseYear, baseMonth, moveStr, nil
}