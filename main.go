//go:build local

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// setCORS はローカルサーバー用のCORSヘッダーを設定します。
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

// corsMiddleware は、CORSヘッダーを設定し、OPTIONSリクエストを処理するミドルウェアです。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		ctx := setUIDInContext(r.Context(), token.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// handlePostRequest はPOSTリクエストを処理するハンドラです。
func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processPostRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleSignupRequest はサインアップPOSTリクエストを処理するハンドラです
func handleSignupRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processSignupRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleLoginRequest はログインPOSTリクエストを処理するハンドラです
func handleLoginRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processLoginRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// apiRouter は、HTTPメソッドに基づいてリクエストを適切なハンドラに振り分けるルーターです。
func apiRouter(w http.ResponseWriter, r *http.Request) {
	// パスに基づいて処理を分岐
	if strings.HasPrefix(r.URL.Path, "/api/time") {
		handleTimeRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/signup") {
		handleSignupRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/login") {
		handleLoginRequest(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func handleTimeRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// GETリクエストの処理: /api/time/{spaceId}
		spaceId := strings.TrimPrefix(r.URL.Path, "/api/time/")
		if spaceId == "" {
			http.Error(w, "spaceId is missing in the URL path", http.StatusBadRequest)
			return
		}
		response, statusCode := processGetRequest(r.Context(), spaceId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// POSTリクエストの処理: /api/time
		optionalAuthMiddleware(http.HandlerFunc(handlePostRequest)).ServeHTTP(w, r)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	apiHandler := http.HandlerFunc(apiRouter)

	// CORSミドルウェアでapiHandlerをラップし、/api/ パス以下すべてに登録
	http.Handle("/api/", corsMiddleware(apiHandler))

	log.Println("Starting local server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}