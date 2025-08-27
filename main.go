//go:build local

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
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

// authMiddleware はHTTPリクエストの認証を"必須"で行うミドルウェアです。
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		
		// デバッグ情報: リクエストヘッダーの確認
		log.Printf("DEBUG: authMiddleware - URL: %s, Method: %s", r.URL.Path, r.Method)
		log.Printf("DEBUG: authMiddleware - Authorization header exists: %v", authHeader != "")
		if authHeader != "" {
			log.Printf("DEBUG: authMiddleware - Authorization header preview: %s", authHeader[:min(len(authHeader), 30)]+"...")
		}
		
		if authHeader == "" {
			log.Printf("ERROR: authMiddleware - No authorization header provided")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
			return
		}

		// "Bearer " プレフィックスを検証・削除
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Printf("ERROR: authMiddleware - Invalid authorization header format: %s", authHeader[:min(len(authHeader), 20)])
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証ヘッダーの形式が正しくありません"})
			return
		}
		sessionToken := parts[1]
		log.Printf("DEBUG: authMiddleware - Session token length: %d", len(sessionToken))

		// セッショントークンを検証
		userSession, err := validateSessionToken(sessionToken)
		if err != nil {
			log.Printf("ERROR: authMiddleware - Failed to verify session token: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証に失敗しました"})
			return
		}
		
		log.Printf("DEBUG: authMiddleware - Session validation successful for UID: %s", userSession.UID)

		// 認証成功 - ユーザー情報をリクエストコンテキストに追加
		// Firebase Auth Tokenの形式に合わせてコンテキストに格納
		mockToken := struct {
			UID string
		}{
			UID: userSession.UID,
		}
		ctx := context.WithValue(r.Context(), "token", mockToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// min は2つの整数の最小値を返すヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// handleVerifyRequest は認証POSTリクエストを処理するハンドラです
func handleVerifyRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := ProcessVerifyRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleCleanupRequest はクリーンアップPOSTリクエストを処理するハンドラです
func handleCleanupRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := ProcessCleanupRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleGoogleAuthRequest はGoogle OAuth2.0認証POSTリクエストを処理するハンドラです
func handleGoogleAuthRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processGoogleAuthRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleGitHubAuthRequest はGitHub OAuth2.0認証POSTリクエストを処理するハンドラです
func handleGitHubAuthRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processGitHubAuthRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleTwitterAuthRequest はTwitter OAuth2.0認証POSTリクエストを処理するハンドラです
func handleTwitterAuthRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processTwitterAuthRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleUserProvidersRequest はユーザープロバイダー情報取得リクエストを処理するハンドラです
func handleUserProvidersRequest(w http.ResponseWriter, r *http.Request) {
	// 認証情報をコンテキストから取得
	tokenValue := r.Context().Value("token")
	if tokenValue == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	mockToken, ok := tokenValue.(struct{ UID string })
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	// Firebase auth.Token形式にマッピング
	token := &auth.Token{
		UID: mockToken.UID,
	}

	response, statusCode := processUserProvidersRequest(r.Context(), token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleUserProvidersDetailRequest はユーザープロバイダー詳細情報取得リクエストを処理するハンドラです
func handleUserProvidersDetailRequest(w http.ResponseWriter, r *http.Request) {
	// 認証情報をコンテキストから取得
	tokenValue := r.Context().Value("token")
	if tokenValue == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	mockToken, ok := tokenValue.(struct{ UID string })
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	// Firebase auth.Token形式にマッピング
	token := &auth.Token{
		UID: mockToken.UID,
	}

	log.Printf("DEBUG: handleUserProvidersDetailRequest called for UID: %s", token.UID)
	response, statusCode := processUserProvidersDetailRequest(r.Context(), token)
	log.Printf("DEBUG: processUserProvidersDetailRequest returned response: %+v", response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleLinkAccountRequest はアカウントリンクリクエストを処理するハンドラです
func handleLinkAccountRequest(w http.ResponseWriter, r *http.Request) {
	// 認証情報をコンテキストから取得
	tokenValue := r.Context().Value("token")
	if tokenValue == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	mockToken, ok := tokenValue.(struct{ UID string })
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	// Firebase auth.Token形式にマッピング
	token := &auth.Token{
		UID: mockToken.UID,
	}

	response, statusCode := processLinkAccountRequest(r.Context(), r, token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleUnlinkAccountRequest はアカウント解除リクエストを処理するハンドラです
func handleUnlinkAccountRequest(w http.ResponseWriter, r *http.Request) {
	// 認証情報をコンテキストから取得
	tokenValue := r.Context().Value("token")
	if tokenValue == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	mockToken, ok := tokenValue.(struct{ UID string })
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "認証が必要です"})
		return
	}

	// Firebase auth.Token形式にマッピング
	token := &auth.Token{
		UID: mockToken.UID,
	}

	response, statusCode := processUnlinkAccountRequest(r.Context(), r, token)
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
	} else if strings.HasPrefix(r.URL.Path, "/api/verify") {
		handleVerifyRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/cleanup") {
		handleCleanupRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/user-data") {
		authMiddleware(http.HandlerFunc(handleUserDataRequest)).ServeHTTP(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/auth/google") {
		handleGoogleAuthRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/auth/github") {
		handleGitHubAuthRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/auth/twitter") {
		handleTwitterAuthRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/user-providers-detail") {
		authMiddleware(http.HandlerFunc(handleUserProvidersDetailRequest)).ServeHTTP(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/user-providers") {
		authMiddleware(http.HandlerFunc(handleUserProvidersRequest)).ServeHTTP(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/link-account") {
		authMiddleware(http.HandlerFunc(handleLinkAccountRequest)).ServeHTTP(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/unlink-account") {
		authMiddleware(http.HandlerFunc(handleUnlinkAccountRequest)).ServeHTTP(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/api/task") {
		// パスが/api/taskの場合は、メソッドに応じて処理を分岐
		if r.Method == "GET" {
			authMiddleware(http.HandlerFunc(handleTaskGet)).ServeHTTP(w, r)
		} else if r.Method == "POST" {
			authMiddleware(http.HandlerFunc(handleTaskSave)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
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
		response, statusCode := processGetScheduleRequest(r.Context(), spaceId)
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