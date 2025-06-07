package main

import (
	"context"
	"fmt"
)

// TimeEntry は各イベントの開始時刻、終了時刻、順序を格納する構造体
type TimeEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Order *int   `json:"order,omitempty"`
}

// processAndSaveSchedule: 受け取ったデータを解析・整形し、Firestoreに保存する
func processAndSaveSchedule(ctx context.Context, requestData map[string]interface{}) (string, map[string][]TimeEntry, error) {
	// Firestoreクライアントの存在確認
	if client == nil {
		return "", nil, fmt.Errorf("重大なエラー: Firestoreクライアントが初期化されていません")
	}

	// requestDateマップから"spaceID"というキーで値を取得
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		return "", nil, fmt.Errorf("'spaceId' がリクエストデータに含まれていません")
	}
	spaceId := spaceIdInterface.(string)

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
		eventList := value.([]interface{})

		var dateEvents []TimeEntry
		for i, eventInterface := range eventList {
			eventMap, _ := eventInterface.(map[string]interface{})
			startStr, _ := eventMap["start"].(string)
			endStr, _ := eventMap["end"].(string)

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
			dateEvents = append(dateEvents, TimeEntry{Start: startStr, End: endStr, Order: orderPtr})
		}
		eventsToStore[key] = dateEvents
	}

	// 保存すべきイベントがない場合は、処理を中断してその旨を返す
	if len(eventsToStore) == 0 {
		return spaceId, nil, nil
	}

	// Firestoreにデータを保存
	docRef := client.Collection("schedules").Doc(spaceId)
	_, err := docRef.Set(ctx, eventsToStore)
	if err != nil {
		return "", nil, fmt.Errorf("Firestoreへのデータ保存に失敗しました: %w", err)
	}

	return spaceId, eventsToStore, nil
}