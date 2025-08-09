package main

import (
	"context"
	"log"
	"net/http"
)

// CleanupResponse はクリーンアップレスポンスの構造体です
type CleanupResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ProcessCleanupRequest はクリーンアップリクエストを処理します
func ProcessCleanupRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	// セキュリティログ
	log.Printf("INFO: Cleanup request received")

	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Cleanup failed: %v", err)
		return map[string]interface{}{
			"message": "クリーンアップの実行に失敗しました",
			"success": false,
			"error":   err.Error(),
		}, http.StatusInternalServerError
	}

	response := map[string]interface{}{
		"message": "クリーンアップが正常に完了しました",
		"success": true,
	}

	return response, http.StatusOK
} 