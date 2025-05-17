package main

import (
	"fmt"
	"net/http"
)

// サーバーのエントリーポイント
func main() {
	http.HandleFunc("/api/time", postHandler) // POST用エンドポイント
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)         // ポート8080で待機、リクエスト時にHandleFuncが実行
}