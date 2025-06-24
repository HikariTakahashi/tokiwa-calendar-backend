// post_transfer.go (修正後)

package main

import (
	"fmt"
	"log"
	"time"
)

// TimeEntry構造体は変更なし
type TimeEntry struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Order     *int   `json:"order,omitempty"`
	Username  string `json:"username"`
	UserColor string `json:"userColor"`
}

// ★★★ 変更点 ★★★
// Firestoreに保存するドキュメント全体の構造を定義します。
// StartDateとEndDateをポインタ型(*string)にし、omitemptyタグを追加します。
// これにより、値がnil(null)の場合にFirestoreのフィールド自体が作られなくなります。
type ScheduleDocument struct {
	AllowOtherEdit bool                   `firestore:"allowOtherEdit"`
	StartDate      *string                `firestore:"startDate,omitempty"`
	EndDate        *string                `firestore:"endDate,omitempty"`
	Events         map[string][]TimeEntry `firestore:"events"`
}

// response-post.goで受け取ったPOSTデータを解析・加工する処理
func transformScheduleData(requestData map[string]interface{}) (string, *ScheduleDocument, error) {
	// spaceIdの取得は変更なし
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		return "", nil, fmt.Errorf("'spaceId' がリクエストデータに含まれていません")
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		return "", nil, fmt.Errorf("'spaceId' が無効です")
	}

	// メタデータを取得
	allowEdit, _ := requestData["allowOtherEdit"].(bool)

	// ★★★ 変更点 ★★★
	// startDateとendDateをポインタとして処理する
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

	// "events" オブジェクトを取得
	eventsToStore := make(map[string][]TimeEntry)
	eventsInterface, ok := requestData["events"]
	if ok { // eventsキーが存在する場合のみ処理
		eventsMap, ok := eventsInterface.(map[string]interface{})
		if !ok {
			return "", nil, fmt.Errorf("'events' の形式が不正です")
		}

		// eventsMapの中をループ処理する
		for key, value := range eventsMap {
			// ★★★ 追加: 安全のため、キーが日付形式か簡易チェック ★★★
			// これで "allowOtherEdit" のようなキーが混入しても無視される
			_, err := time.Parse("2006-01-02", key)
			if err != nil {
				log.Printf("INFO: 'events' 内のキー '%s' は日付形式ではないため、スキップします。", key)
				continue
			}

			eventList, ok := value.([]interface{})
			if !ok {
				log.Printf("WARN: キー '%s' の値がイベントの配列ではないため、スキップします。", key)
				continue
			}

			// 以下のイベント解析ロジックは変更なし
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

	// 最終的なドキュメント構造を作成
	scheduleDoc := &ScheduleDocument{
		AllowOtherEdit: allowEdit,
		StartDate:      startDatePtr,
		EndDate:        endDatePtr,
		Events:         eventsToStore,
	}

	return spaceId, scheduleDoc, nil
}