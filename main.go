package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var client *firestore.Client
func main() {

	// .envファイルを読み込む
	err := godotenv.Load() 
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err) 
	}

	// Firestoreクライアントの初期化
	ctx := context.Background()

	// サービスアカウントキーのJSONを環境変数から読み込む
	serviceAccountJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	if serviceAccountJSON == "" {
		log.Fatalf("環境変数 GOOGLE_APPLICATION_CREDENTIALS_JSON が設定されていません。")
	}

	// 環境変数から読み込んだJSON文字列を使って認証情報を設定
	authOption := option.WithCredentialsJSON([]byte(serviceAccountJSON))

	// セッション作成（認証情報を使ってFirebase Admin SDKを初期化し、Firebaseサービスへのアクセス許可）
	app, err := firebase.NewApp(ctx, nil, authOption)
	if err != nil {
		log.Fatalf("Firebase Admin SDKの初期化に失敗しました: %v", err)
	}

	// Firestoreクライアントの作成
	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Firestoreクライアントの取得に失敗しました: %v", err)
	}

	//　セッションを閉じる
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Firestoreクライアントのクローズに失敗しました: %v", err)
		}
	}()

	fmt.Println("Firebase Admin SDKとFirestoreクライアントが正常に初期化されました。")

	http.HandleFunc("/api/time", postHandler) // postHandlerは別途定義が必要です
	http.HandleFunc("/api/time/", getHandler) // getHandlerは別途定義が必要です
	fmt.Println("Listening on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}