package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// /api/time/{spaceId} にGETされた時の処理
// フロントからのクエリ（Get）を受け取る処理
// 受け取ったクエリを解析・加工する処理
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

	// DBを参照し、データを取得する処理を呼び出す
	data, err := getScheduleFromFirestore(r.Context(), spaceId)
	if err != nil {
		http.Error(w, "データの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("Firestore Getエラー (spaceId: %s): %v\n", spaceId, err)
		return
	}

	// データが存在しない場合 (getScheduleFromFirestoreがnilを返却する)
	if data == nil {
		http.Error(w, "指定されたspaceIdのデータが見つかりません", http.StatusNotFound)
		return
	}

	// レスポンスデータの構築
	response := make(map[string]interface{})

	// イベントデータの処理
	if events, exists := data["Events"]; exists {
		response["events"] = map[string]interface{}{
			"Events": events,
		}
	} else {
		// 古い形式のデータの場合（Eventsキーがない場合）、データ全体をEventsとして扱う
		response["events"] = map[string]interface{}{
			"Events": data,
		}
	}

	// startDateとendDateが存在する場合のみレスポンスに含める
	if startDate, exists := data["StartDate"]; exists && startDate != nil {
		if startDateStr, isString := startDate.(string); isString && startDateStr != "" {
			response["startDate"] = startDateStr
		}
	}
	if endDate, exists := data["EndDate"]; exists && endDate != nil {
		if endDateStr, isString := endDate.(string); isString && endDateStr != "" {
			response["endDate"] = endDateStr
		}
	}

	// AllowOtherEditが存在する場合のみレスポンスに含める
	if allowOtherEdit, exists := data["AllowOtherEdit"]; exists {
		if allowOtherEditBool, isBool := allowOtherEdit.(bool); isBool {
			response["allowOtherEdit"] = allowOtherEditBool
		}
	}

	// データをJSONとして返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
}
