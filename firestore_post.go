package main

import (
	"context"
	"fmt"
)

// response-post.goを用いて、DBに保存する処理
func saveScheduleToFirestore(ctx context.Context, spaceId string, scheduleData *ScheduleData) error {
	// Firestoreクライアントの存在確認
	if client == nil {
		return fmt.Errorf("重大なエラー: Firestoreクライアントが初期化されていません")
	}

	// Firestoreにデータを保存
	docRef := client.Collection("schedules").Doc(spaceId)
	_, err := docRef.Set(ctx, scheduleData)
	if err != nil {
		return fmt.Errorf("firestoreへのデータ保存に失敗しました: %w", err)
	}

	return nil
}