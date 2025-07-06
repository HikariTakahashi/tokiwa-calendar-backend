package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// SignupRequest はサインアップリクエストの構造体です
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignupResponse はサインアップレスポンスの構造体です
type SignupResponse struct {
	Message string `json:"message"`
	UID     string `json:"uid,omitempty"`
	Error   string `json:"error,omitempty"`
}

func processSignupRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
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
	var signupData SignupRequest
	if err := json.Unmarshal(bodyBytes, &signupData); err != nil {
		log.Printf("WARN: Failed to parse signup JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// セキュリティログ（パスワードは隠蔽）
	log.Printf("INFO: Signup request received for email=%s\n", signupData.Email)

	// バリデーション
	if signupData.Email == "" {
		return map[string]interface{}{"error": "メールアドレスが入力されていません"}, http.StatusBadRequest
	}
	if signupData.Password == "" {
		return map[string]interface{}{"error": "パスワードが入力されていません"}, http.StatusBadRequest
	}

	// ★修正点: クライアントから送られてきたパスワードをそのまま使用します
	password := signupData.Password

	// メールアドレスの前処理と検証
	cleanEmail := strings.TrimSpace(strings.ToLower(signupData.Email))
	log.Printf("INFO: Processing signup for clean email: %s\n", cleanEmail)

	if cleanEmail == "" {
		return map[string]interface{}{"error": "メールアドレスが空です"}, http.StatusBadRequest
	}
	
	// メールアドレス形式の検証
	if !strings.Contains(cleanEmail, "@") || !strings.Contains(cleanEmail, ".") {
		return map[string]interface{}{"error": "メールアドレスの形式が不正です"}, http.StatusBadRequest
	}

	// パスワード強度チェック
	passwordStrength := checkPasswordStrength(password)
	if !passwordStrength.IsValid {
		errorMessage := "パスワードの強度が不足しています: " + strings.Join(passwordStrength.Errors, ", ")
		return map[string]interface{}{"error": errorMessage}, http.StatusBadRequest
	}

	// Firebase Authenticationでユーザーを作成
	params := (&auth.UserToCreate{}).
		Email(cleanEmail).
		Password(password).
		EmailVerified(false)

	userRecord, err := authClient.CreateUser(ctx, params)
	if err != nil {
		log.Printf("ERROR: Failed to create user in Firebase Auth for email=%s: %v\n", cleanEmail, err)
		// Firebaseからのエラーコードに基づいて、より親切なメッセージを返す
		if auth.IsEmailAlreadyExists(err) {
			return map[string]interface{}{"error": "このメールアドレスは既に使用されています。"}, http.StatusConflict
		}
		return map[string]interface{}{"error": "ユーザーの作成に失敗しました。"}, http.StatusInternalServerError
	}

	log.Printf("INFO: User successfully created. UID: %s\n", userRecord.UID)

	return map[string]interface{}{
		"message": "ユーザーが正常に作成されました",
		"uid":     userRecord.UID,
	}, http.StatusCreated
}