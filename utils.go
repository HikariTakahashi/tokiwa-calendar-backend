// utils.go
package main

import (
	"context"
	"net/http"
)

// ローカルサーバー用: http.ResponseWriterに直接CORSヘッダーを設定
// (本番ビルドでは使われない)
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

// Lambda用: CORSヘッダーのマップを返す
// (main_lambda.go から呼ばれる)
func getCorsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE",
		"Access-Control-Allow-Headers": "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
	}
}

// userContextKey は、コンテキストキーの衝突を避けるためのカスタム型です。
type userContextKey string

// uidContextKey は、コンテキスト内でユーザーIDを保存・取得するためのキーです。
const uidContextKey userContextKey = "userUID"

// setUIDInContext は、ユーザーIDを含む新しいコンテキストを生成します。
func setUIDInContext(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, uidContextKey, uid)
}

// getUIDFromContext は、コンテキストからユーザーIDを取得します。
func getUIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey).(string)
	return uid, ok
}