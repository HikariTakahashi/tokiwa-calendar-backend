package main

import (
	"context" // Firestore操作に必要
	"encoding/json"
	"fmt"
	"net/http"
	// "time" // 日付や時刻のバリデーションを行う場合に必要
)

// 各イベントの開始時刻、終了時刻、および順序を格納する構造体
type TimeEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Order *int   `json:"order,omitempty"` // orderはオプションなのでポインタ型
}

// /api/time にPOSTされた時の処理
func postHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POSTリクエストを受信しました")

	// OPTIONSリクエストの場合はCORSプリフライトとして処理
	if r.Method == http.MethodOptions {
		setCORS(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTメソッド以外はエラー
	if r.Method != http.MethodPost {
		http.Error(w, "許可されていないメソッドです (POSTのみ許可)", http.StatusMethodNotAllowed)
		return
	}

	setCORS(w)

	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "JSONの解析に失敗しました: "+err.Error(), http.StatusBadRequest)
		fmt.Println("デコードエラー:", err)
		return
	}
	defer r.Body.Close()

	fmt.Printf("受信した生データ: %+v\n", requestData)

	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		http.Error(w, "'spaceId' がリクエストデータに含まれていません", http.StatusBadRequest)
		fmt.Println("エラー: 'spaceId' が見つかりません")
		return
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		http.Error(w, "'spaceId' は空でない文字列である必要があります", http.StatusBadRequest)
		fmt.Println("エラー: 'spaceId' が文字列でないか、空です")
		return
	}

	eventsToStore := make(map[string][]TimeEntry)

	for key, value := range requestData {
		if key == "spaceId" {
			continue
		}

		eventListInterface, ok := value.([]interface{})
		if !ok {
			errMsg := fmt.Sprintf("キー '%s' のデータ形式が無効です。イベントの配列形式である必要があります。", key)
			http.Error(w, errMsg, http.StatusBadRequest)
			fmt.Printf("エラー: キー '%s' の値が予期した配列形式ではありません: %+v\n", key, value)
			return
		}

		var dateEvents []TimeEntry

		for i, eventInterface := range eventListInterface {
			eventMap, ok := eventInterface.(map[string]interface{})
			if !ok {
				errMsg := fmt.Sprintf("キー '%s' の配列の %d 番目の要素のデータ形式が無効です。オブジェクト形式である必要があります。", key, i)
				http.Error(w, errMsg, http.StatusBadRequest)
				fmt.Printf("エラー: キー '%s' の配列の %d 番目の要素がマップ形式ではありません: %+v\n", key, i, eventInterface)
				return
			}

			startStrInterface, startExists := eventMap["start"]
			endStrInterface, endExists := eventMap["end"]

			if !startExists || !endExists {
				errMsg := fmt.Sprintf("キー '%s' の配列の %d 番目のイベントに 'start' または 'end' が含まれていません。", key, i)
				http.Error(w, errMsg, http.StatusBadRequest)
				fmt.Printf("エラー: キー '%s' の配列の %d 番目のイベントデータに 'start' または 'end' がありません: %+v\n", key, i, eventMap)
				return
			}

			startStr, startOk := startStrInterface.(string)
			endStr, endOk := endStrInterface.(string)

			if !startOk || !endOk {
				errMsg := fmt.Sprintf("キー '%s' の配列の %d 番目のイベントの 'start' または 'end' の形式が無効です。文字列である必要があります。", key, i)
				http.Error(w, errMsg, http.StatusBadRequest)
				fmt.Printf("エラー: キー '%s' の配列の %d 番目のイベントの 'start' または 'end' が文字列ではありません: %+v\n", key, i, eventMap)
				return
			}

			var orderPtr *int
			if orderInterface, orderExists := eventMap["order"]; orderExists {
				orderFloat, orderIsFloat := orderInterface.(float64)
				if orderIsFloat {
					orderVal := int(orderFloat)
					orderPtr = &orderVal
				} else {
					orderInt, orderIsInt := orderInterface.(int)
					if orderIsInt {
						orderVal := int(orderInt)
						orderPtr = &orderVal
					} else {
						errMsg := fmt.Sprintf("キー '%s' の配列の %d 番目のイベントの 'order' の形式が無効です。数値である必要があります。", key, i)
						http.Error(w, errMsg, http.StatusBadRequest)
						fmt.Printf("エラー: キー '%s' の配列の %d 番目のイベントの 'order' が数値ではありません (型: %T): %+v\n", key, i, orderInterface, eventMap)
						return
					}
				}
			} else {
				// ★★★ 変更点: orderが存在しない場合、デフォルト値 1 を設定 ★★★
				defaultValue := 1
				orderPtr = &defaultValue
			}

			dateEvents = append(dateEvents, TimeEntry{Start: startStr, End: endStr, Order: orderPtr})
		}
		eventsToStore[key] = dateEvents
	}

	fmt.Printf("抽出した spaceId: %s\n", spaceId)
	fmt.Printf("Firestoreに保存するイベントデータ: %+v\n", eventsToStore)

	if client == nil {
		http.Error(w, "Firestoreクライアントが初期化されていません", http.StatusInternalServerError)
		fmt.Println("重大なエラー: Firestoreクライアント (global 'client') がnilです")
		return
	}

	if len(eventsToStore) == 0 {
		fmt.Printf("spaceId '%s' に関連する有効なカレンダーイベントが見つかりませんでした。Firestoreへの保存は行いません。\n", spaceId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "有効なカレンダーイベントデータが見つかりませんでした。",
			"spaceId": spaceId,
		})
		return
	}

	docRef := client.Collection("schedules").Doc(spaceId)
	_, err := docRef.Set(context.Background(), eventsToStore)
	if err != nil {
		http.Error(w, "Firestoreへのデータ保存に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("Firestore Setエラー (spaceId: %s, Collection: schedules): %v\n", spaceId, err)
		return
	}

	fmt.Printf("データがFirestoreに正常に保存されました。Collection: schedules, Document ID: %s\n", spaceId)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":     "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":     spaceId,
		"savedEvents": eventsToStore,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
}