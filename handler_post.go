package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
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
			log.Printf("ERROR: Failed to read request body: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayProxyRequest:
		bodyBytes = []byte(r.Body)
	default:
		log.Printf("ERROR: Unknown request type: %T\n", req)
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var postData SchedulePostRequest
	if err := json.Unmarshal(bodyBytes, &postData); err != nil {
		log.Printf("WARN: Failed to parse JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
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

	// 既存のspaceIdが指定されているかチェック
	var targetSpaceId string
	var isUpdate bool
	
	if postData.SpaceId != nil && *postData.SpaceId != "" {
		// 既存のspaceIdが指定されている場合（再同期時）
		targetSpaceId = *postData.SpaceId
		isUpdate = true
		log.Printf("INFO: Updating existing spaceId: %s", targetSpaceId)
	} else {
		// 新規作成時
		targetSpaceId = firestoreClient.Collection(firestoreCollectionName).NewDoc().ID
		isUpdate = false
		log.Printf("INFO: Creating new spaceId: %s", targetSpaceId)
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
		if isUpdate {
			log.Printf("INFO: Updating spaceId %s with owner UID %s\n", targetSpaceId, uid)
		} else {
			log.Printf("INFO: Associating new spaceId %s with owner UID %s\n", targetSpaceId, uid)
		}
	} else {
		if isUpdate {
			log.Printf("INFO: Updating spaceId %s for anonymous user.\n", targetSpaceId)
		} else {
			log.Printf("INFO: Creating new spaceId %s for anonymous user.\n", targetSpaceId)
		}
	}

	if err := saveScheduleToFirestore(ctx, targetSpaceId, scheduleDoc); err != nil {
		log.Printf("ERROR: Failed to save to Firestore: %v\n", err)
		return map[string]interface{}{"error": "データの保存に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	if isUpdate {
		log.Printf("INFO: Data successfully updated in Firestore. Document ID: %s\n", targetSpaceId)
		return map[string]interface{}{
			"message":   "データは正常に更新され、Firestoreに保存されました。",
			"spaceId":   targetSpaceId,
		}, http.StatusOK
	} else {
		log.Printf("INFO: Data successfully saved to Firestore. Document ID: %s\n", targetSpaceId)
		return map[string]interface{}{
			"message":   "データは正常に受信され、Firestoreに保存されました。",
			"spaceId":   targetSpaceId,
		}, http.StatusOK
	}
}