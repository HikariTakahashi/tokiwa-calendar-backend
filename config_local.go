//go:build local

package main

import "fmt"

// ローカル実行時（go run --tags=local .）にのみビルドされる
func init() {
	// ユーザー指定のテスト用コレクション名を設定
	firestoreCollectionName = "schedules_test"

	fmt.Println("========================================")
	fmt.Println("    RUNNING IN LOCAL MODE")
	fmt.Printf("    Firestore Collection: %s\n", firestoreCollectionName)
	fmt.Println("========================================")
}