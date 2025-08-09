package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

// ScheduledCleanupHandler はCloudWatch Eventsから呼び出される定期クリーンアップハンドラーです
func ScheduledCleanupHandler(ctx context.Context, event events.CloudWatchEvent) error {
	log.Printf("INFO: Scheduled cleanup triggered by CloudWatch Event: %s", event.ID)
	
	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Scheduled cleanup failed: %v", err)
		return err
	}
	
	log.Printf("INFO: Scheduled cleanup completed successfully")
	return nil
}

// ManualCleanupHandler は手動でクリーンアップを実行するためのハンドラーです
func ManualCleanupHandler(ctx context.Context) error {
	log.Printf("INFO: Manual cleanup started")
	
	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Manual cleanup failed: %v", err)
		return err
	}
	
	log.Printf("INFO: Manual cleanup completed successfully")
	return nil
} 