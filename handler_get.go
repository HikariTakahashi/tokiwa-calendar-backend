package main

import (
	"context"
	"fmt"
	"net/http"
)

func processGetRequest(ctx context.Context, spaceId string) (map[string]interface{}, int) {
	data, err := getScheduleFromFirestore(ctx, spaceId)
	if err != nil {
		fmt.Printf("Firestore Getエラー (spaceId: %s): %v\n", spaceId, err)
		return map[string]interface{}{"error": "データの取得に失敗しました: " + err.Error()}, http.StatusInternalServerError
	}

	if data == nil {
		return map[string]interface{}{"message": "指定されたspaceIdのデータが見つかりません"}, http.StatusNotFound
	}

	return data, http.StatusOK
}