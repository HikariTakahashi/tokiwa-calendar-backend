//go:build !local

package main

import "log"

// デフォルトのビルド（go build .）でビルドされる
func init() {
	// ユーザー指定の本番用コレクション名を設定
	firestoreCollectionName = "schedules_prod"

	log.Println("========================================")
	log.Println("    RUNNING IN PRODUCTION MODE")
	log.Printf("    Firestore Collection: %s\n", firestoreCollectionName)
	log.Println("========================================")
}