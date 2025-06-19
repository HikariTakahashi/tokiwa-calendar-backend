package main

import (
	"fmt"
)

// TimeEntry は各イベントの開始時刻、終了時刻、順序を格納する構造体
type TimeEntry struct {
	Start     string `json:"Start"`
	End       string `json:"End"`
	Order     *int   `json:"Order,omitempty"`
	Username  string `json:"Username"`
	UserColor string `json:"UserColor"`
}

// ScheduleData はスケジュールデータ全体を表す構造体
type ScheduleData struct {
	Events         map[string][]TimeEntry `json:"Events"`
	StartDate      *string                `json:"StartDate,omitempty"`
	EndDate        *string                `json:"EndDate,omitempty"`
	AllowOtherEdit bool                   `json:"AllowOtherEdit"`
}

// response-post.goで受け取ったPOSTデータを解析・加工する処理
func transformScheduleData(requestData map[string]interface{}) (string, *ScheduleData, error) {
	// requestDateマップから"spaceID"というキーで値を取得
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		return "", nil, fmt.Errorf("'spaceId' がリクエストデータに含まれていません")
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		return "", nil, fmt.Errorf("'spaceId' が無効です")
	}

	// AllowOtherEditの処理
	allowOtherEdit := false
	if allowOtherEditVal, exists := requestData["AllowOtherEdit"]; exists {
		if allowOtherEditBool, isBool := allowOtherEditVal.(bool); isBool {
			allowOtherEdit = allowOtherEditBool
		}
	}

	// eventsデータの取得
	eventsInterface, ok := requestData["events"]
	if !ok {
		return "", nil, fmt.Errorf("'events' がリクエストデータに含まれていません")
	}

	eventsMap, ok := eventsInterface.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'events' が無効な形式です")
	}

	// Eventsデータの取得
	eventsDataInterface, ok := eventsMap["Events"]
	if !ok {
		return "", nil, fmt.Errorf("'Events' がeventsデータに含まれていません")
	}

	eventsData, ok := eventsDataInterface.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'Events' が無効な形式です")
	}

	// スケジュールデータの整理
	// eventsToStore：Firestoreに保存するための、最終的なきれいなデータを格納する変数
	eventsToStore := make(map[string][]TimeEntry)

	// startDateとendDateの処理（events内から取得）
	var startDate *string
	var endDate *string

	if startDateVal, exists := eventsMap["StartDate"]; exists {
		if startDateStr, isString := startDateVal.(string); isString && startDateStr != "" {
			startDate = &startDateStr
		}
	}

	if endDateVal, exists := eventsMap["EndDate"]; exists {
		if endDateStr, isString := endDateVal.(string); isString && endDateStr != "" {
			endDate = &endDateStr
		}
	}

	// 外側ループ：eventsDataからkeyが"StartDate"、"EndDate"以外の者をループ処理
	// key:"2025-06-12"等が、value： [{"Start":...}] ような配列が入る
	for key, value := range eventsData {
		if key == "StartDate" || key == "EndDate" {
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
			startStr, ok := eventMap["Start"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'Start' が無効です", key, i)
			}
			endStr, ok := eventMap["End"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'End' が無効です", key, i)
			}
			username, ok := eventMap["Username"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'Username' が無効です", key, i)
			}
			userColor, ok := eventMap["UserColor"].(string)
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'UserColor' が無効です", key, i)
			}

			// orderの処理 (orderが存在しない場合、デフォルト値 1 を設定)
			var orderPtr *int
			defaultValue := 1
			if orderVal, exists := eventMap["Order"]; exists {
				if orderFloat, isFloat := orderVal.(float64); isFloat {
					val := int(orderFloat)
					orderPtr = &val
				} else if orderInt, isInt := orderVal.(int); isInt {
					orderPtr = &orderInt
				} else {
					return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントの 'Order' が無効な形式です", key, i)
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

	scheduleData := &ScheduleData{
		Events:         eventsToStore,
		AllowOtherEdit: allowOtherEdit,
	}

	// startDateとendDateが存在する場合のみ追加
	if startDate != nil {
		scheduleData.StartDate = startDate
	}
	if endDate != nil {
		scheduleData.EndDate = endDate
	}

	return spaceId, scheduleData, nil
}
