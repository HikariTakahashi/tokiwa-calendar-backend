package main

import (
	"encoding/json"
	"net/http"
)

// フロントとバックで別々のドメインよりCORS設定
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// レスポンスのデータ型をJSONに、resp(データ) → Encode(変換) → レスポンス出力
func sendJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}