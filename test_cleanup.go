//go:build local

package main

import (
	"context"
	"log"
	"time"
)

// TestCleanup はローカル開発時にクリーンアップ機能をテストするための関数です
func TestCleanup() {
	log.Println("=== Starting Cleanup Test ===")
	
	ctx := context.Background()
	
	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Test cleanup failed: %v", err)
		return
	}
	
	log.Println("=== Cleanup Test Completed Successfully ===")
}

// TestCleanupWithDelay は指定された時間後にクリーンアップを実行するテスト関数です
func TestCleanupWithDelay(delay time.Duration) {
	log.Printf("=== Starting Delayed Cleanup Test (delay: %v) ===", delay)
	
	time.Sleep(delay)
	
	ctx := context.Background()
	
	// クリーンアップを実行
	if err := CleanupExpiredUsers(ctx); err != nil {
		log.Printf("ERROR: Delayed test cleanup failed: %v", err)
		return
	}
	
	log.Println("=== Delayed Cleanup Test Completed Successfully ===")
} 