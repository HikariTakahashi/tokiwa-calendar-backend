package main

import (
	"context"
	"log"
	"time"

	"firebase.google.com/go/v4/auth"
)

func getScheduleFromFirestore(ctx context.Context, spaceId string) (map[string]interface{}, error) {
	doc, err := firestoreClient.Collection(firestoreCollectionName).Doc(spaceId).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := doc.DataTo(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// getVerificationToken は認証トークンをFirestoreから取得します
func getVerificationToken(ctx context.Context, token string) (*VerificationToken, error) {
	// 認証トークン用のコレクション名
	collectionName := firestoreCollectionName + "_verification_tokens"
	
	// Firestoreから取得
	doc, err := firestoreClient.Collection(collectionName).Doc(token).Get(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to get verification token from Firestore: %v", err)
		return nil, err
	}
	
	var verificationToken VerificationToken
	if err := doc.DataTo(&verificationToken); err != nil {
		log.Printf("ERROR: Failed to convert verification token data: %v", err)
		return nil, err
	}
	
	log.Printf("INFO: Verification token retrieved from Firestore for email: %s", verificationToken.Email)
	return &verificationToken, nil
}

// verifyEmailToken は認証トークンを検証し、メールアドレスを認証します
// 戻り値: (成功したかどうか, 既に認証済みかどうか, エラー)
func verifyEmailToken(ctx context.Context, token string) (bool, bool, error) {
	// トークンを取得
	verificationToken, err := getVerificationToken(ctx, token)
	if err != nil {
		// トークンが見つからない場合、既に認証済みの可能性がある
		log.Printf("WARN: Token not found, checking if user might be already verified: %v", err)
		return false, false, err
	}
	
	// トークンの有効期限をチェック
	if time.Now().After(verificationToken.ExpiresAt) {
		log.Printf("WARN: Verification token expired for email: %s", verificationToken.Email)
		
		// 期限切れトークンの場合、未認証ユーザーを削除
		isVerified, err := isUserEmailVerified(ctx, verificationToken.UID)
		if err != nil {
			log.Printf("ERROR: Failed to check verification status for expired token: %v", err)
		} else if !isVerified {
			// 未認証ユーザーの場合、削除を試行
			if err := deleteUnverifiedUser(ctx, verificationToken.UID); err != nil {
				log.Printf("ERROR: Failed to delete unverified user with expired token: %v", err)
			} else {
				log.Printf("INFO: Deleted unverified user with expired token: %s", verificationToken.UID)
			}
		}
		
		return false, false, nil
	}
	
	// 既に認証済みかどうかをチェック
	isVerified, err := isUserEmailVerified(ctx, verificationToken.UID)
	if err != nil {
		log.Printf("ERROR: Failed to check email verification status: %v", err)
		return false, false, err
	}
	
	if isVerified {
		log.Printf("INFO: User email is already verified for UID: %s", verificationToken.UID)
		// 既に認証済みの場合は、トークンを削除して成功として返す
		if err := deleteVerificationToken(ctx, token); err != nil {
			log.Printf("WARN: Failed to delete verification token for already verified user: %v", err)
		}
		return true, true, nil
	}
	
	// Firebase Authenticationでメールアドレスを認証
	params := (&auth.UserToUpdate{}).
		EmailVerified(true)
	
	_, err = authClient.UpdateUser(ctx, verificationToken.UID, params)
	if err != nil {
		log.Printf("ERROR: Failed to update user email verification status: %v", err)
		return false, false, err
	}
	
	// 認証済みトークンを削除
	if err := deleteVerificationToken(ctx, token); err != nil {
		log.Printf("WARN: Failed to delete verification token after successful verification: %v", err)
		// トークン削除に失敗しても認証は成功しているので、警告として記録
	}
	
	log.Printf("INFO: Email verification completed successfully for UID: %s", verificationToken.UID)
	return true, false, nil
}

// deleteVerificationToken は認証トークンをFirestoreから削除します
func deleteVerificationToken(ctx context.Context, token string) error {
	// 認証トークン用のコレクション名
	collectionName := firestoreCollectionName + "_verification_tokens"
	
	// Firestoreから削除
	_, err := firestoreClient.Collection(collectionName).Doc(token).Delete(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to delete verification token from Firestore: %v", err)
		return err
	}
	
	log.Printf("INFO: Verification token deleted from Firestore")
	return nil
}

// isUserEmailVerified はユーザーのメールアドレスが既に認証済みかどうかをチェックします
func isUserEmailVerified(ctx context.Context, uid string) (bool, error) {
	userRecord, err := authClient.GetUser(ctx, uid)
	if err != nil {
		log.Printf("ERROR: Failed to get user record: %v", err)
		return false, err
	}
	
	return userRecord.EmailVerified, nil
}