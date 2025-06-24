// handler_post.go (完全な修正版)

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

	// ★★★ この部分が抜けていました ★★★
	// リクエストの種類に応じてJSONをパースし、「requestData」変数に格納します
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
    // ★★★ ここまで ★★★

	// データ形式を変換します
	spaceId, scheduleDoc, err := transformScheduleData(requestData)
	if err != nil {
		return map[string]interface{}{"error": "データ形式の解析に失敗しました: " + err.Error()}, http.StatusBadRequest
	}

	if len(scheduleDoc.Events) == 0 {
		fmt.Printf("イベントデータは空ですが、メタデータを保存します。 Document ID: %s\n", spaceId)
	}

	// データをFirestoreに保存します
	if err := saveScheduleToFirestore(ctx, spaceId, scheduleDoc); err != nil {
		fmt.Printf("処理エラー: %v\n", err)
		return map[string]interface{}{"error": "データの処理または保存に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	fmt.Printf("データがFirestoreに正常に保存されました。Document ID: %s\n", spaceId)
	
	// クライアントへのレスポンスを返します
	return map[string]interface{}{
		"message":   "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":   spaceId,
		"savedData": scheduleDoc,
	}, http.StatusOK
}