// post_transfer.go

package main

import (
	"fmt"
	"log"
	"time"
)

// ★★★ 変更点 ★★★
// Firestoreに保存する際のキー名を小文字にするため、`firestore`タグを追加
type TimeEntry struct {
	Start     string `json:"start"     firestore:"start"`
	End       string `json:"end"       firestore:"end"`
	Order     *int   `json:"order,omitempty" firestore:"order,omitempty"`
	Username  string `json:"username"  firestore:"username"`
	UserColor string `json:"userColor" firestore:"userColor"`
}

// ★★★ 変更点 ★★★
// StartDateとEndDateをポインタ型(*string)にし、omitemptyタグを追加
type ScheduleDocument struct {
	AllowOtherEdit bool                   `firestore:"allowOtherEdit"`
	StartDate      *string                `firestore:"startDate,omitempty"`
	EndDate        *string                `firestore:"endDate,omitempty"`
	Events         map[string][]TimeEntry `firestore:"events"`
}

func transformScheduleData(requestData map[string]interface{}) (string, *ScheduleDocument, error) {
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		return "", nil, fmt.Errorf("'spaceId' がリクエストデータに含まれていません")
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		return "", nil, fmt.Errorf("'spaceId' が無効です")
	}

	allowEdit, _ := requestData["allowOtherEdit"].(bool)

	// ★★★ 変更点 ★★★
	// startDateとendDateをポインタとして処理
	var startDatePtr, endDatePtr *string
	if val, ok := requestData["startDate"]; ok && val != nil {
		if strVal, isString := val.(string); isString {
			startDatePtr = &strVal
		}
	}
	if val, ok := requestData["endDate"]; ok && val != nil {
		if strVal, isString := val.(string); isString {
			endDatePtr = &strVal
		}
	}

	eventsToStore := make(map[string][]TimeEntry)
	if eventsInterface, ok := requestData["events"]; ok {
		if eventsMap, isMap := eventsInterface.(map[string]interface{}); isMap {
			for key, value := range eventsMap {
				// キーが日付形式か簡易チェック
				if _, err := time.Parse("2006-01-02", key); err != nil {
					log.Printf("INFO: 'events' 内のキー '%s' は日付形式ではないため、スキップします。", key)
					continue
				}

				eventList, ok := value.([]interface{})
				if !ok {
					log.Printf("WARN: キー '%s' の値がイベントの配列ではないため、スキップします。", key)
					continue
				}

				var dateEvents []TimeEntry
				for i, eventInterface := range eventList {
					eventMap, ok := eventInterface.(map[string]interface{})
					if !ok {
						log.Printf("WARN: キー '%s' の %d 番目のイベントデータ形式が不正なため、スキップします。", key, i)
						continue
					}

					startStr, _ := eventMap["start"].(string)
					endStr, _ := eventMap["end"].(string)
					username, _ := eventMap["username"].(string)
					userColor, _ := eventMap["userColor"].(string)

					var orderPtr *int
					defaultValue := 1
					if orderVal, exists := eventMap["order"]; exists {
						if orderFloat, isFloat := orderVal.(float64); isFloat {
							val := int(orderFloat)
							orderPtr = &val
						} else if orderInt, isInt := orderVal.(int); isInt {
							orderPtr = &orderInt
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
		}
	}

	scheduleDoc := &ScheduleDocument{
		AllowOtherEdit: allowEdit,
		StartDate:      startDatePtr,
		EndDate:        endDatePtr,
		Events:         eventsToStore,
	}

	return spaceId, scheduleDoc, nil
}