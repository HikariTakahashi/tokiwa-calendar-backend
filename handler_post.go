package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// /api/time にPOSTでリクエストが来た時の処理
// フロントからのJSONリクエスト（POST）を受け取る処理
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

	// 受け取ったPOSTデータを解析・加工する処理を呼び出す
	spaceId, scheduleData, err := transformScheduleData(requestData)
	if err != nil {
		http.Error(w, "データ形式の解析に失敗しました: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 保存する有効なイベントがなかった場合のレスポンス
	if len(scheduleData.Events) == 0 {
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"message": "有効なカレンダーイベントデータが見つかりませんでした。",
			"spaceId": spaceId,
		}
		// startDateとendDateが存在する場合のみレスポンスに含める
		if scheduleData.StartDate != nil && *scheduleData.StartDate != "" {
			response["startDate"] = *scheduleData.StartDate
		}
		if scheduleData.EndDate != nil && *scheduleData.EndDate != "" {
			response["endDate"] = *scheduleData.EndDate
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// DBに保存する処理を呼び出す
	if err := saveScheduleToFirestore(r.Context(), spaceId, scheduleData); err != nil {
		http.Error(w, "データの処理または保存に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("処理エラー: %v\n", err)
		return
	}

	// 成功した場合のレスポンス
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":     "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":     spaceId,
		"savedEvents": scheduleData.Events,
	}
	// startDateとendDateが存在する場合のみレスポンスに含める
	if scheduleData.StartDate != nil && *scheduleData.StartDate != "" {
		response["startDate"] = *scheduleData.StartDate
	}
	if scheduleData.EndDate != nil && *scheduleData.EndDate != "" {
		response["endDate"] = *scheduleData.EndDate
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
	fmt.Printf("データがFirestoreに正常に保存されました。Document ID: %s\n", spaceId)
}