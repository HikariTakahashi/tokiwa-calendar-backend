//go:build scheduler

package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
)

// schedulerHandler はCloudWatch Eventsから呼び出される定期クリーンアップハンドラーです
func schedulerHandler(ctx context.Context, event events.CloudWatchEvent) error {
	log.Printf("INFO: Scheduler Lambda triggered by CloudWatch Event: %s", event.ID)
	
	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Scheduled cleanup failed: %v", err)
		return err
	}
	
	log.Printf("INFO: Scheduled cleanup completed successfully")
	return nil
}

func main() {
	// 環境変数ファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}
	
	// Firebaseはinit()関数で自動初期化されるため、ここでは何もしない
	
	lambda.Start(schedulerHandler)
} 