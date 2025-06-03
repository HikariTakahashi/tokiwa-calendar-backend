package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// /api/time/{spaceId} にGETされた時の処理
func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GETリクエストを受信しました")

	// OPTIONSリクエストの場合はCORSプリフライトとして処理
	if r.Method == http.MethodOptions {
		setCORS(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	// GETメソッド以外はエラー
	if r.Method != http.MethodGet {
		http.Error(w, "許可されていないメソッドです (GETのみ許可)", http.StatusMethodNotAllowed)
		return
	}

	setCORS(w)

	// URLからspaceIdを抽出
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(w, "無効なURLです", http.StatusBadRequest)
		return
	}
	spaceId := parts[3]

	if spaceId == "" {
		http.Error(w, "spaceIdが指定されていません", http.StatusBadRequest)
		return
	}

	// Firestoreからデータを取得
	docRef := client.Collection("schedules").Doc(spaceId)
	doc, err := docRef.Get(context.Background())
	if err != nil {
		http.Error(w, "データの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("Firestore Getエラー (spaceId: %s): %v\n", spaceId, err)
		return
	}

	// データが存在しない場合
	if !doc.Exists() {
		http.Error(w, "指定されたspaceIdのデータが見つかりません", http.StatusNotFound)
		return
	}

	// データをJSONとして返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(doc.Data()); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
} 