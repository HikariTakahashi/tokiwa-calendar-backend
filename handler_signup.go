package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			fmt.Printf("リクエストボディの読み取りに失敗: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayV2HTTPRequest:
		bodyBytes = []byte(r.Body)
	default:
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var signupData SignupRequest
	if err := json.Unmarshal(bodyBytes, &signupData); err != nil {
		return map[string]interface{}{"error": "JSONの解析に失敗しました: " + err.Error()}, http.StatusBadRequest
	}

	// セキュリティログ（パスワードは隠蔽）
	fmt.Printf("サインアップリクエスト受信: email=%s, encrypted_password=***\n", signupData.Email)

	// バリデーション
	if signupData.Email == "" {
		return map[string]interface{}{"error": "メールアドレスが入力されていません"}, http.StatusBadRequest
	}
	if signupData.Password == "" {
		return map[string]interface{}{"error": "パスワードが入力されていません"}, http.StatusBadRequest
	}

	// 暗号化されたパスワードを復号化
	fmt.Printf("復号化対象の暗号化パスワード: %s\n", signupData.Password[:50] + "...")
	decryptedPassword, err := DecryptPassword(signupData.Password)
	if err != nil {
		fmt.Printf("パスワード復号化エラー: %v\n", err)
		return map[string]interface{}{"error": "パスワードの復号化に失敗しました: " + err.Error()}, http.StatusBadRequest
	}
	fmt.Printf("復号化成功: %s\n", decryptedPassword)
	
	// メールアドレスの前処理と検証
	cleanEmail := strings.TrimSpace(strings.ToLower(signupData.Email))
	fmt.Printf("元のメールアドレス: %s\n", signupData.Email)
	fmt.Printf("処理後のメールアドレス: %s\n", cleanEmail)
	fmt.Printf("メールアドレス長: %d\n", len(cleanEmail))
	
	if cleanEmail == "" {
		return map[string]interface{}{"error": "メールアドレスが空です"}, http.StatusBadRequest
	}
	
	// メールアドレス形式の検証
	if !strings.Contains(cleanEmail, "@") || !strings.Contains(cleanEmail, ".") {
		return map[string]interface{}{"error": "メールアドレスの形式が不正です"}, http.StatusBadRequest
	}

	// パスワード強度チェック
	passwordStrength := checkPasswordStrength(decryptedPassword)
	if !passwordStrength.IsValid {
		errorMessage := "パスワードの強度が不足しています: " + strings.Join(passwordStrength.Errors, ", ")
		return map[string]interface{}{"error": errorMessage}, http.StatusBadRequest
	}

	// Firebase Authenticationでユーザーを作成
	fmt.Printf("Firebaseに送信するメールアドレス: %s\n", cleanEmail)
	fmt.Printf("Firebaseに送信するパスワード長: %d\n", len(decryptedPassword))
	
	params := (&auth.UserToCreate{}).
		Email(cleanEmail).
		Password(decryptedPassword).
		EmailVerified(false)

	userRecord, err := authClient.CreateUser(ctx, params)
	if err != nil {
		fmt.Printf("Firebase Authenticationでのユーザー作成エラー: %v\n", err)
		fmt.Printf("エラーの詳細: %+v\n", err)
		
		// Firebaseのエラーメッセージをより詳細に表示
		fmt.Printf("エラータイプ: %T\n", err)
		fmt.Printf("エラー文字列: %s\n", err.Error())
		
		return map[string]interface{}{"error": "ユーザーの作成に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	fmt.Printf("ユーザーが正常に作成されました。UID: %s\n", userRecord.UID)

	return map[string]interface{}{
		"message": "ユーザーが正常に作成されました",
		"uid":     userRecord.UID,
	}, http.StatusCreated
}

// handleSignupRequest はサインアップPOSTリクエストを処理するハンドラです
func handleSignupRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processSignupRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
} 