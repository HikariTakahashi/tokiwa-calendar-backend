package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// POSTで受け取るデータの構造体
type TimeData struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// /api/time にPOSTされた時の処理
func postHandler(w http.ResponseWriter, r *http.Request) {
	// OPTIONSリクエストの場合はCORSプリフライトとして処理
	if r.Method == "OPTIONS" {
		setCORS(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTメソッド以外はエラー
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// フロントエンドからのアクセスを許可（CORS対応）
	setCORS(w)

	// 複数日付データを格納するマップを用意
	var data map[string]TimeData

	// リクエストのJSONをTimeData構造体に変換（デコード）
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "JSONの解析に失敗しました", http.StatusBadRequest)
		fmt.Println("デコードエラー:", err)
		return
	}

	// 受け取ったデータをターミナルに出力（確認用）
	fmt.Printf("受信したデータ: %+v\n", data)

	// データをJSONに変換してレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "JSONのエンコードに失敗しました", http.StatusInternalServerError)
		fmt.Println("エンコードエラー:", err)
		return
	}
}