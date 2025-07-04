// firestore_client.go
package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// FirestoreとAuthのクライアントをプロジェクト全体で共有する
var firestoreClient *firestore.Client
var authClient *auth.Client

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

	firestoreClient, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Firestoreクライアントの取得に失敗しました: %v", err)
	}

	authClient, err = app.Auth(ctx)
	if err != nil {
		log.Fatalf("Authクライアントの取得に失敗しました: %v", err)
	}

	log.Println("Firestore and Auth clients initialized successfully.")
}