package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// POSTで受け取るデータの構造体
type TimeData struct {
	Date  string `json:"date"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// /api/time にPOSTされた時の処理
func postHandler(w http.ResponseWriter, r *http.Request) {
	// フロントエンドからのアクセスを許可（CORS対応）
	setCORS(w)

	// POSTメソッド以外は405エラーを返す
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	// リクエストのJSONをTimeData構造体に変換（デコード）
	var data TimeData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "JSONの解析に失敗しました", http.StatusBadRequest)
		fmt.Println("デコードエラー:", err)
		return
	}

	// 受け取ったデータをターミナルに出力（確認用）
	fmt.Printf("受信したデータ: %+v\n", data)

	// データをJSONに変換してレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}