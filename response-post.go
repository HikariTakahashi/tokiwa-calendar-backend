package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// /api/time にPOSTでリクエストが来た時の処理
func postHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POSTリクエストを受信しました")
	setCORS(w) // CORSヘッダーを設定

	// 正当なリクエストだけ受け付ける処理
	// OPTIONSメソッドのチェック
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTメソッド以外はエラー
	if r.Method != http.MethodPost {
		http.Error(w, "許可されていないメソッドです (POSTのみ許可)", http.StatusMethodNotAllowed)
		return
	}

	// JSONデータの受信と解析
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "JSONの解析に失敗しました: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析したデータを渡し、Firestoreへの保存処理を呼び出す
	spaceId, savedEvents, err := processAndSaveSchedule(r.Context(), requestData)
	if err != nil {
		http.Error(w, "データの処理または保存に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("処理エラー: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 保存する有効なイベントがなかった場合のレスポンス
	if len(savedEvents) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "有効なカレンダーイベントデータが見つかりませんでした。",
			"spaceId": spaceId,
		})
		return
	}

	// 成功した場合のレスポンス
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":   "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":   spaceId,
		"savedEvents": savedEvents,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
	fmt.Printf("データがFirestoreに正常に保存されました。Document ID: %s\n", spaceId)
}