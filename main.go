//go:build local

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/joho/godotenv"
)

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

// handlePostRequest はPOSTリクエストを処理するハンドラです。
// この関数は、ミドルウェアによってラップされて呼び出されます。
func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processPostRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// apiRouter は、HTTPメソッドに基づいてリクエストを適切なハンドラに振り分けるルーターです。
func apiRouter(w http.ResponseWriter, r *http.Request) {
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
	// 環境変数ファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}
	
	apiHandler := http.HandlerFunc(apiRouter)
	signupHandler := http.HandlerFunc(handleSignupRequest)
	loginHandler := http.HandlerFunc(handleLoginRequest)

	// CORSミドルウェアでapiHandlerをラップし、/api/time/ パスに登録
	http.Handle("/api/time/", corsMiddleware(apiHandler))
	
	// サインアップ用のエンドポイントを追加
	http.Handle("/api/signup", corsMiddleware(signupHandler))
	
	// ログイン用のエンドポイントを追加
	http.Handle("/api/login", corsMiddleware(loginHandler))

	log.Println("Starting local server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}