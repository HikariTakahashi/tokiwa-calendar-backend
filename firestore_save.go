// firestore_save.go

package main

import (
	"context"
)

// saveScheduleToFirestore は、指定されたspaceIdのドキュメントとしてデータをFirestoreに保存します。
// 既存のドキュメントがある場合は、完全に上書きされます。
func saveScheduleToFirestore(ctx context.Context, spaceId string, data *ScheduleDocument) error {
	docRef := client.Collection(firestoreCollectionName).Doc(spaceId)
	
	// .Set() メソッドは構造体を渡すと、自動的にFirestoreのドキュメント形式に変換します。
	_, err := docRef.Set(ctx, data)
	return err
}