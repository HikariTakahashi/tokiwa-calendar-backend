package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func processGetRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	switch r := req.(type) {
	case *http.Request:
		return handleGetRequest(ctx, r)
	case events.APIGatewayV2HTTPRequest:
		return handleGetLambdaRequest(ctx, r)
	default:
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}
}

func handleGetRequest(ctx context.Context, req *http.Request) (map[string]interface{}, int) {
	// メール設定確認エンドポイント
	if req.URL.Path == "/email-config" {
		return checkEmailConfig()
	}
	
	return map[string]interface{}{"error": "エンドポイントが見つかりません"}, http.StatusNotFound
}

func handleGetLambdaRequest(ctx context.Context, req events.APIGatewayV2HTTPRequest) (map[string]interface{}, int) {
	// メール設定確認エンドポイント
	if req.RequestContext.HTTP.Path == "/email-config" {
		return checkEmailConfig()
	}
	
	return map[string]interface{}{"error": "エンドポイントが見つかりません"}, http.StatusNotFound
}

// checkEmailConfig はメール設定の状況を確認します
func checkEmailConfig() (map[string]interface{}, int) {
	config := getEmailConfig()
	
	// パスワードは長さのみ表示（セキュリティのため）
	passwordInfo := "設定されていません"
	if config.SMTPPassword != "" {
		passwordInfo = fmt.Sprintf("%d文字", len(config.SMTPPassword))
	}
	
	configInfo := map[string]interface{}{
		"smtp_host":     config.SMTPHost,
		"smtp_port":     config.SMTPPort,
		"smtp_username": config.SMTPUsername,
		"smtp_password": passwordInfo,
		"from_email":    config.FromEmail,
		"from_name":     config.FromName,
	}
	
	// 設定の妥当性をチェック
	if err := validateEmailConfig(); err != nil {
		configInfo["validation_error"] = err.Error()
		configInfo["is_valid"] = false
		return configInfo, http.StatusOK
	}
	
	configInfo["is_valid"] = true
	return configInfo, http.StatusOK
}

// 既存のスケジュール取得機能
func processGetScheduleRequest(ctx context.Context, spaceId string) (map[string]interface{}, int) {
	data, err := getScheduleFromFirestore(ctx, spaceId)
	if err != nil {
		fmt.Printf("Firestore Getエラー (spaceId: %s): %v\n", spaceId, err)
		return map[string]interface{}{"error": "データの取得に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	if data == nil {
		return map[string]interface{}{"message": "指定されたspaceIdのデータが見つかりません"}, http.StatusNotFound
	}

	return data, http.StatusOK
}