// utils.go
package main

import "net/http"

// ローカルサーバー用: http.ResponseWriterに直接CORSヘッダーを設定
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Lambda用: CORSヘッダーのマップを返す
// この関数が main_lambda.go から呼ばれる
func getCorsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
	}
}