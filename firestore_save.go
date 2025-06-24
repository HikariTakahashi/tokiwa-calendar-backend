// firestore_save.go

package main

import (
	"context"
)

// ★★★ 変更点 ★★★
// 第3引数を interface{} または 具体的な *ScheduleDocument 型で受け取る
func saveScheduleToFirestore(ctx context.Context, spaceId string, data *ScheduleDocument) error {
	docRef := client.Collection(firestoreCollectionName).Doc(spaceId)
	
	// .Set() メソッドは構造体を渡すと、自動的にFirestoreのドキュメント形式に変換してくれる
	// 構造体に `firestore` タグが付いているため、その通りのフィールド名で保存される
	_, err := docRef.Set(ctx, data)
	return err
}