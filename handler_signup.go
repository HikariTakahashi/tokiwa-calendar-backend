package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	// クライアントから送られてきた暗号化されたパスワードを復号化
	// まず簡易暗号化方式を試す
	decryptedPassword, err := decryptPassword(signupData.Password)
	if err != nil {
		log.Printf("DEBUG: Simple decryption failed, trying AES-CBC: %v\n", err)
		// 簡易暗号化が失敗した場合、AES-CBC暗号化を試す
		decryptedPassword, err = DecryptPassword(signupData.Password)
		if err != nil {
			log.Printf("ERROR: Both decryption methods failed: %v\n", err)
			return map[string]interface{}{"error": "パスワードの復号化に失敗しました"}, http.StatusBadRequest
		}
	}

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

	// 復号化されたパスワードで強度チェック
	log.Printf("DEBUG: Checking password strength for password length: %d", len(decryptedPassword))
	passwordStrength := checkPasswordStrength(decryptedPassword)
	if !passwordStrength.IsValid {
		log.Printf("DEBUG: Password strength check failed: %v", passwordStrength.Errors)
		errorMessage := "パスワードの強度が不足しています: " + strings.Join(passwordStrength.Errors, ", ")
		return map[string]interface{}{"error": errorMessage}, http.StatusBadRequest
	}
	log.Printf("DEBUG: Password strength check passed")

	// Firebase Authenticationでユーザーを作成
	params := (&auth.UserToCreate{}).
		Email(cleanEmail).
		Password(decryptedPassword).
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

	// 認証トークンを生成
	verificationToken, err := generateVerificationToken(cleanEmail, userRecord.UID)
	if err != nil {
		log.Printf("ERROR: Failed to generate verification token: %v\n", err)
		// トークン生成に失敗してもユーザー作成は成功しているので、警告として記録
		log.Printf("WARN: User created but verification email could not be sent")
		return map[string]interface{}{
			"message": "ユーザーが正常に作成されました（認証メールの送信に失敗しました）",
			"uid":     userRecord.UID,
		}, http.StatusCreated
	}

	// 認証トークンをFirestoreに保存
	if err := saveVerificationToken(ctx, verificationToken); err != nil {
		log.Printf("ERROR: Failed to save verification token: %v\n", err)
		// トークン保存に失敗してもユーザー作成は成功しているので、警告として記録
		log.Printf("WARN: User created but verification token could not be saved")
		return map[string]interface{}{
			"message": "ユーザーが正常に作成されました（認証メールの送信に失敗しました）",
			"uid":     userRecord.UID,
		}, http.StatusCreated
	}

	// メール設定の検証
	log.Printf("DEBUG: Starting email configuration validation")
	if err := validateEmailConfig(); err != nil {
		log.Printf("ERROR: Email configuration validation failed: %v\n", err)
		log.Printf("WARN: User created but email configuration is invalid")
		return map[string]interface{}{
			"message": "ユーザーが正常に作成されました（メール設定が無効です）",
			"uid":     userRecord.UID,
			"debug": map[string]interface{}{
				"emailConfigError": err.Error(),
				"emailConfig":      getEmailConfigForDebug(),
			},
		}, http.StatusCreated
	}
	log.Printf("DEBUG: Email configuration validation passed")

	// 認証メールを送信
	log.Printf("DEBUG: Starting verification email send to: %s", cleanEmail)
	if err := sendVerificationEmail(cleanEmail, verificationToken.Token); err != nil {
		log.Printf("ERROR: Failed to send verification email: %v\n", err)
		
		// Lambda環境でメール認証スキップが許可されている場合
		if shouldSkipEmailVerification() {
			log.Printf("INFO: Lambda environment detected, skipping email verification for user: %s", userRecord.UID)
			
			// ユーザーを自動的にメール認証済みとしてマーク
			if err := markUserAsVerified(ctx, userRecord.UID); err != nil {
				log.Printf("WARN: Failed to mark user as verified: %v", err)
			}
			
			return map[string]interface{}{
				"message": "ユーザーが正常に作成されました（Lambda環境のためメール認証をスキップしました）",
				"uid":     userRecord.UID,
				"lambdaMode": true,
				"debug": map[string]interface{}{
					"emailSendError": err.Error(),
					"emailConfig":    getEmailConfigForDebug(),
					"targetEmail":    cleanEmail,
					"lambdaEnvironment": true,
				},
			}, http.StatusCreated
		}
		
		// 通常の環境では警告として記録
		log.Printf("WARN: User created but verification email could not be sent")
		return map[string]interface{}{
			"message": "ユーザーが正常に作成されました（認証メールの送信に失敗しました）",
			"uid":     userRecord.UID,
			"debug": map[string]interface{}{
				"emailSendError": err.Error(),
				"emailConfig":    getEmailConfigForDebug(),
				"targetEmail":    cleanEmail,
			},
		}, http.StatusCreated
	}
	log.Printf("DEBUG: Verification email sent successfully")

	return map[string]interface{}{
		"message": "ユーザーが正常に作成されました。認証メールをお送りしました。",
		"uid":     userRecord.UID,
	}, http.StatusCreated
}

// markUserAsVerified はユーザーをメール認証済みとしてマークします
func markUserAsVerified(ctx context.Context, uid string) error {
	// Firebase Authでユーザーのメール認証状態を更新
	params := (&auth.UserToUpdate{}).
		EmailVerified(true)
	
	_, err := authClient.UpdateUser(ctx, uid, params)
	if err != nil {
		return fmt.Errorf("ユーザーの認証状態の更新に失敗しました: %v", err)
	}
	
	log.Printf("INFO: User %s marked as email verified", uid)
	return nil
}