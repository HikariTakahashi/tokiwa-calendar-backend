package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// VerifyRequest は認証リクエストの構造体です
type VerifyRequest struct {
	Token string `json:"token"`
}

// VerifyResponse は認証レスポンスの構造体です
type VerifyResponse struct {
	Message      string `json:"message"`
	Success      bool   `json:"success"`
	AlreadyVerified bool `json:"already_verified,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ProcessVerifyRequest は認証リクエストを処理します
func ProcessVerifyRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
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
	var verifyData VerifyRequest
	if err := json.Unmarshal(bodyBytes, &verifyData); err != nil {
		log.Printf("WARN: Failed to parse verify JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// セキュリティログ
	log.Printf("INFO: Verify request received for token=%s\n", verifyData.Token)

	// バリデーション
	if verifyData.Token == "" {
		return map[string]interface{}{"error": "認証トークンが入力されていません"}, http.StatusBadRequest
	}

	// 認証トークンを検証
	success, alreadyVerified, err := verifyEmailToken(ctx, verifyData.Token)
	if err != nil {
		log.Printf("ERROR: Failed to verify token: %v\n", err)
		// トークンが見つからない場合、既に認証済みの可能性があるため、
		// エラーメッセージをより適切なものに変更
		return map[string]interface{}{
			"message": "認証トークンが見つからないか、既に使用済みです",
			"success": false,
		}, http.StatusBadRequest
	}

	if !success {
		return map[string]interface{}{
			"message": "認証トークンが無効です",
			"success": false,
		}, http.StatusBadRequest
	}

	response := map[string]interface{}{
		"success": true,
	}

	if alreadyVerified {
		response["message"] = "メールアドレスは既に認証済みです"
		response["already_verified"] = true
	} else {
		response["message"] = "メールアドレスの認証が完了しました"
		response["already_verified"] = false
	}

	return response, http.StatusOK
}

 