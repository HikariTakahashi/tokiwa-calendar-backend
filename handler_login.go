package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// LoginRequest はログインリクエストの構造体です
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse はログインレスポンスの構造体です
type LoginResponse struct {
	Message     string `json:"message"`
	UID         string `json:"uid,omitempty"`
	Email       string `json:"email,omitempty"`
	CustomToken string `json:"customToken,omitempty"`
	Error       string `json:"error,omitempty"`
}

// getFirebaseAPIKey は環境変数からFirebase APIキーを取得します
func getFirebaseAPIKey() string {
	// 環境変数からAPIキーを取得
	apiKey := os.Getenv("FIREBASE_API_KEY")
	if apiKey != "" {
		return apiKey
	}
	
	// ローカル開発用のデフォルト値（本番環境では必ず環境変数を設定してください）
	// 注意: この値は実際のFirebaseプロジェクトのAPIキーに置き換える必要があります
	return "AIzaSyBxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}

// verifyPasswordWithFirebase はFirebase Auth REST APIを使用してパスワード認証を行います
func verifyPasswordWithFirebase(email, password string) (map[string]interface{}, error) {
	apiKey := getFirebaseAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("firebase APIキーが設定されていません")
	}
	
	url := "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=" + apiKey
	
	// 認証リクエストの作成
	authRequest := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}
	
	authRequestJSON, err := json.Marshal(authRequest)
	if err != nil {
		return nil, fmt.Errorf("認証リクエストのJSON化エラー: %v", err)
	}
	
	// Firebase Auth REST APIにリクエストを送信
	resp, err := http.Post(url, "application/json", strings.NewReader(string(authRequestJSON)))
	if err != nil {
		return nil, fmt.Errorf("firebase auth APIリクエストエラー: %v", err)
	}
	defer resp.Body.Close()
	
	// レスポンスを読み取り
	var authResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, fmt.Errorf("認証レスポンスの解析エラー: %v", err)
	}
	
	return authResponse, nil
}

func processLoginRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	var bodyBytes []byte
	var err error

	// リクエストソース（ローカルサーバー or Lambda）に応じてリクエストボディをバイトスライスとして取得
	switch r := req.(type) {
	case *http.Request:
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: Failed to read request body: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayV2HTTPRequest:
		bodyBytes = []byte(r.Body)
	default:
		log.Printf("ERROR: Unknown request type: %T\n", r)
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var loginData LoginRequest
	if err := json.Unmarshal(bodyBytes, &loginData); err != nil {
		log.Printf("WARN: Failed to parse login JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// セキュリティログ（パスワードは隠蔽）
	log.Printf("INFO: Login request received for email=%s\n", loginData.Email)

	// バリデーション
	if loginData.Email == "" {
		return map[string]interface{}{"error": "メールアドレスが入力されていません"}, http.StatusBadRequest
	}
	if loginData.Password == "" {
		return map[string]interface{}{"error": "パスワードが入力されていません"}, http.StatusBadRequest
	}

	// 暗号化されたパスワードを復号化
	decryptedPassword, err := DecryptPassword(loginData.Password)
	if err != nil {
		log.Printf("WARN: Failed to decrypt password for email=%s: %v\n", loginData.Email, err)
		// クライアントには一般的なエラーを返す
		return map[string]interface{}{"error": "メールアドレスまたはパスワードが正しくありません"}, http.StatusUnauthorized
	}

	// メールアドレスの前処理と検証
	cleanEmail := strings.TrimSpace(strings.ToLower(loginData.Email))

	if cleanEmail == "" {
		return map[string]interface{}{"error": "メールアドレスが空です"}, http.StatusBadRequest
	}
	
	// メールアドレス形式の検証
	if !strings.Contains(cleanEmail, "@") || !strings.Contains(cleanEmail, ".") {
		return map[string]interface{}{"error": "メールアドレスの形式が不正です"}, http.StatusBadRequest
	}

	// Firebase Auth REST APIを使用してパスワード認証を実行
	log.Printf("INFO: Verifying password with Firebase Auth for email=%s\n", cleanEmail)

	authResponse, err := verifyPasswordWithFirebase(cleanEmail, decryptedPassword)
	if err != nil {
		log.Printf("ERROR: Firebase auth API request failed: %v\n", err)
		return map[string]interface{}{"error": "認証サービスへの接続に失敗しました"}, http.StatusInternalServerError
	}

	// エラーチェック
	if authResponse["error"] != nil {
		errorInfo := authResponse["error"].(map[string]interface{})
		errorMessage := errorInfo["message"].(string)
		log.Printf("WARN: Firebase auth failed for email=%s. Reason: %s\n", cleanEmail, errorMessage)

		// エラーメッセージを最小限に統一（セキュリティのため）
		switch errorMessage {
		case "TOO_MANY_ATTEMPTS_TRY_LATER":
			return map[string]interface{}{"error": "ログイン試行回数が多すぎます。しばらく時間をおいてから再試行してください"}, http.StatusTooManyRequests
		case "USER_DISABLED":
			return map[string]interface{}{"error": "アカウントが無効化されています"}, http.StatusUnauthorized
		default:
			return map[string]interface{}{"error": "メールアドレスまたはパスワードが正しくありません"}, http.StatusUnauthorized
		}
	}

	// 認証成功
	idToken := authResponse["idToken"].(string)
	localId := authResponse["localId"].(string)
	email := authResponse["email"].(string)

	log.Printf("INFO: Firebase auth successful. UID: %s\n", localId)

	// カスタムトークンを生成（フロントエンドでの認証用）
	customToken, err := authClient.CustomToken(ctx, localId)
	if err != nil {
		log.Printf("ERROR: Failed to create custom token for UID %s: %v\n", localId, err)
		return map[string]interface{}{"error": "認証トークンの生成に失敗しました"}, http.StatusInternalServerError
	}

	return map[string]interface{}{
		"message":     "ログインが成功しました",
		"uid":         localId,
		"email":       email,
		"customToken": customToken,
		"idToken":     idToken,
	}, http.StatusOK
}