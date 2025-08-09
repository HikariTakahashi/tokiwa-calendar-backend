package main

import (
	"context"
	"log"
)

// saveScheduleToFirestore は、指定されたspaceIdのドキュメントとしてデータをFirestoreに保存します。
// 既存のドキュメントがある場合は、完全に上書きされます。
func saveScheduleToFirestore(ctx context.Context, spaceId string, data *ScheduleDocument) error {
	docRef := firestoreClient.Collection(firestoreCollectionName).Doc(spaceId)
	
	// .Set() メソッドは構造体を渡すと、自動的にFirestoreのドキュメント形式に変換します。
	_, err := docRef.Set(ctx, data)
	return err
}

// saveVerificationToken は認証トークンをFirestoreに保存します
func saveVerificationToken(ctx context.Context, token *VerificationToken) error {
	// 認証トークン用のコレクション名
	collectionName := firestoreCollectionName + "_verification_tokens"
	
	// Firestoreに保存
	_, err := firestoreClient.Collection(collectionName).Doc(token.Token).Set(ctx, token)
	if err != nil {
		log.Printf("ERROR: Failed to save verification token to Firestore: %v", err)
		return err
	}
	
	log.Printf("INFO: Verification token saved to Firestore for email: %s", token.Email)
	return nil
}

// saveUserDataToFirestore はユーザーデータをFirestoreに保存します
func saveUserDataToFirestore(ctx context.Context, uid string, userData *UserData) error {
	// ユーザーデータ用のコレクション名（環境に依存しない固定名）
	collectionName := "user_data"
	
	// Firestoreに保存
	_, err := firestoreClient.Collection(collectionName).Doc(uid).Set(ctx, userData)
	if err != nil {
		log.Printf("ERROR: Failed to save user data to Firestore: %v", err)
		return err
	}
	
	log.Printf("INFO: User data saved to Firestore for UID: %s", uid)
	return nil
}