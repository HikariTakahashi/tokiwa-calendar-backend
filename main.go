package main

import (
	"fmt"
	"net/http"
)

// サーバーのエントリーポイント
func main() {
	// api/calendar のURLに来たリクエストは、handler関数で処理する設定
	http.HandleFunc("/api/calendar", handler) // エンドポイントを登録
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)         // ポート8080で待機、リクエスト時にHandleFuncが実行
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