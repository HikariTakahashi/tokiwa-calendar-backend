// firestore_client.go
package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// この 'client' 変数がプロジェクト全体で共有される
var client *firestore.Client

func init() {
	_ = godotenv.Load() // .envファイルはローカル開発でのみ使用。エラーは無視。

	log.Println("Initializing Firestore client...")

	ctx := context.Background()

	serviceAccountJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	if serviceAccountJSON == "" {
		log.Fatalf("環境変数 GOOGLE_APPLICATION_CREDENTIALS_JSON が設定されていません。")
	}

	authOption := option.WithCredentialsJSON([]byte(serviceAccountJSON))
	app, err := firebase.NewApp(ctx, nil, authOption)
	if err != nil {
		log.Fatalf("Firebase Admin SDKの初期化に失敗しました: %v", err)
	}

	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Firestoreクライアントの取得に失敗しました: %v", err)
	}
	log.Println("Firestore client initialized successfully.")
}