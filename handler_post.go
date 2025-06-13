package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func processPostRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	var requestData map[string]interface{}
	var err error

	switch r := req.(type) {
	case *http.Request:
		err = json.NewDecoder(r.Body).Decode(&requestData)
		defer r.Body.Close()
	case events.APIGatewayProxyRequest:
		err = json.Unmarshal([]byte(r.Body), &requestData)
	default:
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	if err != nil {
		return map[string]interface{}{"error": "JSONの解析に失敗しました: " + err.Error()}, http.StatusBadRequest
	}

	spaceId, eventsToStore, err := transformScheduleData(requestData)
	if err != nil {
		return map[string]interface{}{"error": "データ形式の解析に失敗しました: " + err.Error()}, http.StatusBadRequest
	}

	if len(eventsToStore) == 0 {
		return map[string]interface{}{
			"message": "有効なカレンダーイベントデータが見つかりませんでした。", "spaceId": spaceId,
		}, http.StatusOK
	}

	if err := saveScheduleToFirestore(ctx, spaceId, eventsToStore); err != nil {
		fmt.Printf("処理エラー: %v\n", err)
		return map[string]interface{}{"error": "データの処理または保存に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	fmt.Printf("データがFirestoreに正常に保存されました。Document ID: %s\n", spaceId)
	return map[string]interface{}{
		"message": "データは正常に受信され、Firestoreに保存されました。", "spaceId": spaceId, "savedEvents": eventsToStore,
	}, http.StatusOK
}