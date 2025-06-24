package main

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// response-get.goを用いて、DBを参照
// データベースの特定の情報をフロントに返す処理
func getScheduleFromFirestore(ctx context.Context, spaceId string) (map[string]interface{}, error) {
	// Firestoreからデータを取得
	docRef := client.Collection("schedules").Doc(spaceId)
	doc, err := docRef.Get(ctx)
	if err != nil {
		// データが見つからないエラーの場合、データなし(nil)とエラーなしを返してハンドラ側で404を処理させる
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		// その他のエラーの場合は、エラーをそのまま返す
		return nil, err
	}

	return doc.Data(), nil
}