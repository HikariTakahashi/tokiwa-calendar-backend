package main

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// userContextKey はコンテキスト内でユーザーIDを格納するためのキーです。
// 文字列リテラルを直接使うのを避け、型安全性を高めるためのテクニックです。
type userContextKey string

const uidContextKey = userContextKey("uid")

// optionalAuthMiddleware はHTTPリクエストを"オプショナル"で認証するミドルウェアです。
func optionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// ヘッダーがなければ、非ログインユーザーとして次の処理へ
			next.ServeHTTP(w, r)
			return
		}

		// "Bearer " プレフィックスを検証・削除
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Println("WARN: Authorization header format is invalid, proceeding as anonymous.")
			next.ServeHTTP(w, r) // フォーマット不正でもエラーにせず、非ログインユーザーとして続行
			return
		}
		idToken := parts[1]

		token, err := authClient.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			log.Printf("WARN: Failed to verify ID token, proceeding as anonymous: %v\n", err)
			next.ServeHTTP(w, r) // トークン検証失敗でもエラーにせず、非ログインユーザーとして続行
			return
		}

		// 検証成功: コンテキストにUIDを保存して次のハンドラを呼び出す
		ctx := context.WithValue(r.Context(), uidContextKey, token.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUIDFromContext はコンテキストからUIDを取得します。後のステップで使います。
func getUIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey).(string)
	return uid, ok
}