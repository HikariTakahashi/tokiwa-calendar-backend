// handler_post.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func processPostRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	var bodyBytes []byte
	var err error

	// リクエストソース（ローカルサーバー or Lambda）に応じてリクエストボディをバイトスライスとして取得
	switch r := req.(type) {
	case *http.Request:
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("リクエストボディの読み取りに失敗: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayProxyRequest:
		bodyBytes = []byte(r.Body)
	default:
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var postData SchedulePostRequest
	if err := json.Unmarshal(bodyBytes, &postData); err != nil {
		return map[string]interface{}{"error": "JSONの解析に失敗しました: " + err.Error()}, http.StatusBadRequest
	}

	// バリデーション
	if postData.SpaceID == "" {
		return map[string]interface{}{"error": "'spaceId' がリクエストデータに含まれていないか、空です"}, http.StatusBadRequest
	}

	// Events内の各TimeEntryでOrderが指定されていない場合にデフォルト値を設定
	defaultValue := 1
	if postData.Events != nil {
		for _, eventList := range postData.Events {
			for i := range eventList {
				if eventList[i].Order == nil {
					eventList[i].Order = &defaultValue
				}
			}
		}
	}

	// Firestoreに保存するドキュメントを作成
	scheduleDoc := &ScheduleDocument{
		AllowOtherEdit: postData.AllowOtherEdit,
		StartDate:      postData.StartDate,
		EndDate:        postData.EndDate,
		Events:         postData.Events,
	}

	// コンテキストからUIDを取得し、存在すればドキュメントにセットする
	// ミドルウェアにより、ログインユーザーの場合のみUIDがセットされている
	if uid, ok := getUIDFromContext(ctx); ok {
		scheduleDoc.OwnerUID = uid
	}

	if err := saveScheduleToFirestore(ctx, postData.SpaceID, scheduleDoc); err != nil {
		fmt.Printf("Firestoreへの保存エラー: %v\n", err)
		return map[string]interface{}{"error": "データの保存に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	fmt.Printf("データがFirestoreに正常に保存されました。Document ID: %s\n", postData.SpaceID)

	return map[string]interface{}{
		"message":   "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":   postData.SpaceID,
		"savedData": scheduleDoc,
	}, http.StatusOK
}