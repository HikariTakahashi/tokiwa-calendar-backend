// post_transfer.go (修正後)

package main

import (
	"fmt"
)

// TimeEntry 構造体は、JSONリクエストの各イベントオブジェクトに対応します。
// タグはjsonのままにしておきます。
type TimeEntry struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Order     *int   `json:"order,omitempty"`
	Username  string `json:"username"`
	UserColor string `json:"userColor"`
}

// ★★★ 変更点 ★★★
// Firestoreに保存するドキュメント全体の構造を定義します。
// firestoreタグを使うことで、Firestore上でのフィールド名を明示できます。
type ScheduleDocument struct {
	AllowOtherEdit bool                   `firestore:"allowOtherEdit"`
	StartDate      string                 `firestore:"startDate"`
	EndDate        string                 `firestore:"endDate"`
	Events         map[string][]TimeEntry `firestore:"events"`
}

// ★★★ 変更点 ★★★
// response-post.goで受け取ったPOSTデータを解析・加工する処理
// 戻り値を、新しい ScheduleDocument 構造体のポインタに変更します。
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

	// ★★★ 追加: 新しいメタデータを取得 ★★★
	allowEdit, _ := requestData["allowOtherEdit"].(bool) // 存在しない場合はfalseになる
	startDate, ok := requestData["startDate"].(string)
	if !ok {
		return "", nil, fmt.Errorf("'startDate' がリクエストデータに含まれていません")
	}
	endDate, ok := requestData["endDate"].(string)
	if !ok {
		return "", nil, fmt.Errorf("'endDate' がリクエストデータに含まれていません")
	}

	// ★★★ 変更点: "events" オブジェクトを取得 ★★★
	eventsInterface, ok := requestData["events"]
	if !ok {
		// eventsが無くても、メタデータ更新として許容する場合はこのエラーチェックを外す
		return "", nil, fmt.Errorf("'events' がリクエストデータに含まれていません")
	}
	eventsMap, ok := eventsInterface.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'events' の形式が不正です")
	}

	// スケジュールデータの整理
	eventsToStore := make(map[string][]TimeEntry)

	// eventsMapの中をループ処理する
	for key, value := range eventsMap {
		eventList, ok := value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("キー '%s' の値がイベントの配列ではありません", key)
		}

		var dateEvents []TimeEntry
		// ここのループは以前のロジックとほぼ同じ
		for i, eventInterface := range eventList {
			eventMap, ok := eventInterface.(map[string]interface{})
			if !ok {
				return "", nil, fmt.Errorf("キー '%s' の %d 番目のイベントデータ形式が不正です", key, i)
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

	// ★★★ 変更点: 最終的なドキュメント構造を作成 ★★★
	scheduleDoc := &ScheduleDocument{
		AllowOtherEdit: allowEdit,
		StartDate:      startDate,
		EndDate:        endDate,
		Events:         eventsToStore,
	}

	return spaceId, scheduleDoc, nil
}