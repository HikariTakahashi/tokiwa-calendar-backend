package main

import (
	"fmt"
)

// TimeEntry は各イベントの開始時刻、終了時刻、順序を格納する構造体
type TimeEntry struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Order     *int   `json:"order,omitempty"`
	Username  string `json:"username"`
	UserColor string `json:"userColor"`
}

// response-post.goで受け取ったPOSTデータを解析・加工する処理
func transformScheduleData(requestData map[string]interface{}) (string, map[string][]TimeEntry, error) {
	// requestDateマップから"spaceID"というキーで値を取得
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		return "", nil, fmt.Errorf("'spaceId' がリクエストデータに含まれていません")
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		return "", nil, fmt.Errorf("'spaceId' が無効です")
	}

	// スケジュールデータの整理
	// eventsToStore：Firestoreに保存するための、最終的なきれいなデータを格納する変数
	eventsToStore := make(map[string][]TimeEntry)

	// 外側ループ：requestDateからkeyが"spaceId"以外の者をループ処理
	// key:"2025-06-12"等が、value： [{"start":...}] ような配列が入る
	for key, value := range requestData {
		if key == "spaceId" {
			continue
		}

		// valueをinterface{}型から[]interface{}型に変換
		eventList, ok := value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("キー '%s' の値がイベントの配列ではありません", key)
		}

		var dateEvents []TimeEntry
		for i, eventInterface := range eventList {
			eventMap, ok := eventInterface.(map[string]interface{})
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントが無効な形式です", key, i)
			}

			// 必須フィールドの取得
			startStr, ok := eventMap["start"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'start' が無効です", key, i)
			}
			endStr, ok := eventMap["end"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'end' が無効です", key, i)
			}
			username, ok := eventMap["username"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'username' が無効です", key, i)
			}
			userColor, ok := eventMap["userColor"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'userColor' が無効です", key, i)
			}

			// orderの処理 (orderが存在しない場合、デフォルト値 1 を設定)
			var orderPtr *int
			defaultValue := 1
			if orderVal, exists := eventMap["order"]; exists {
				if orderFloat, isFloat := orderVal.(float64); isFloat {
					val := int(orderFloat)
					orderPtr = &val
				} else if orderInt, isInt := orderVal.(int); isInt {
					orderPtr = &orderInt
				} else {
					return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'order' が無効な形式です", key, i)
				}
			} else {
				orderPtr = &defaultValue
			}

			dateEvents = append(dateEvents, TimeEntry{
				Start:     startStr,
				End:       endStr,
				Order:     orderPtr,
				Username:  username,
				UserColor: userColor,
			})
		}
		eventsToStore[key] = dateEvents
	}

	return spaceId, eventsToStore, nil
}
