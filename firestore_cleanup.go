package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/api/iterator"
)

// CleanupExpiredUsers は期限切れの未認証ユーザーとトークンを削除します
func CleanupExpiredUsers(ctx context.Context) error {
	log.Printf("INFO: Starting cleanup of expired users and tokens")
	
	// 期限切れトークンの削除
	if err := cleanupExpiredTokens(ctx); err != nil {
		log.Printf("ERROR: Failed to cleanup expired tokens: %v", err)
		return err
	}
	
	// 期限切れ未認証ユーザーの削除
	if err := cleanupExpiredUnverifiedUsers(ctx); err != nil {
		log.Printf("ERROR: Failed to cleanup expired unverified users: %v", err)
		return err
	}
	
	log.Printf("INFO: Cleanup completed successfully")
	return nil
}

// cleanupExpiredTokens は期限切れの認証トークンを削除します
func cleanupExpiredTokens(ctx context.Context) error {
	collectionName := firestoreCollectionName + "_verification_tokens"
	now := time.Now()
	
	// 期限切れトークンを検索
	iter := firestoreClient.Collection(collectionName).Where("expires_at", "<", now).Documents(ctx)
	
	var deletedCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("ERROR: Failed to iterate over expired tokens: %v", err)
			continue
		}
		
		// トークンを削除
		if _, err := doc.Ref.Delete(ctx); err != nil {
			log.Printf("ERROR: Failed to delete expired token %s: %v", doc.Ref.ID, err)
			continue
		}
		
		deletedCount++
		log.Printf("INFO: Deleted expired token: %s", doc.Ref.ID)
	}
	
	log.Printf("INFO: Deleted %d expired verification tokens", deletedCount)
	return nil
}

// cleanupExpiredUnverifiedUsers は期限切れの未認証ユーザーを削除します
func cleanupExpiredUnverifiedUsers(ctx context.Context) error {
	collectionName := firestoreCollectionName + "_verification_tokens"
	cutoffTime := time.Now().Add(-24 * time.Hour) // 24時間前
	
	// 24時間以上前のトークンを検索
	iter := firestoreClient.Collection(collectionName).Where("created_at", "<", cutoffTime).Documents(ctx)
	
	var deletedUserCount int
	processedUIDs := make(map[string]bool) // 重複削除を防ぐため
	
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("ERROR: Failed to iterate over old tokens: %v", err)
			continue
		}
		
		// トークンデータを取得
		var token VerificationToken
		if err := doc.DataTo(&token); err != nil {
			log.Printf("ERROR: Failed to parse token data for %s: %v", doc.Ref.ID, err)
			continue
		}
		
		// 既に処理済みのUIDはスキップ
		if processedUIDs[token.UID] {
			continue
		}
		processedUIDs[token.UID] = true
		
		// ユーザーが未認証かどうかをチェック
		isVerified, err := isUserEmailVerified(ctx, token.UID)
		if err != nil {
			log.Printf("ERROR: Failed to check verification status for UID %s: %v", token.UID, err)
			continue
		}
		
		// 未認証ユーザーの場合、削除
		if !isVerified {
			if err := deleteUnverifiedUser(ctx, token.UID); err != nil {
				log.Printf("ERROR: Failed to delete unverified user %s: %v", token.UID, err)
				continue
			}
			deletedUserCount++
			log.Printf("INFO: Deleted unverified user: %s (email: %s)", token.UID, token.Email)
		}
	}
	
	log.Printf("INFO: Deleted %d expired unverified users", deletedUserCount)
	return nil
}

// deleteUnverifiedUser は未認証ユーザーをFirebase Authenticationから削除します
func deleteUnverifiedUser(ctx context.Context, uid string) error {
	// Firebase Authenticationからユーザーを削除
	if err := authClient.DeleteUser(ctx, uid); err != nil {
		return err
	}
	
	// 関連するトークンも削除
	if err := deleteAllTokensForUser(ctx, uid); err != nil {
		log.Printf("WARN: Failed to delete tokens for user %s: %v", uid, err)
		// ユーザー削除は成功しているので、トークン削除の失敗は警告として記録
	}
	
	return nil
}

// deleteAllTokensForUser は指定されたユーザーのすべてのトークンを削除します
func deleteAllTokensForUser(ctx context.Context, uid string) error {
	collectionName := firestoreCollectionName + "_verification_tokens"
	
	// ユーザーのトークンを検索
	iter := firestoreClient.Collection(collectionName).Where("uid", "==", uid).Documents(ctx)
	
	var deletedCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("ERROR: Failed to iterate over user tokens: %v", err)
			continue
		}
		
		// トークンを削除
		if _, err := doc.Ref.Delete(ctx); err != nil {
			log.Printf("ERROR: Failed to delete token %s for user %s: %v", doc.Ref.ID, uid, err)
			continue
		}
		
		deletedCount++
	}
	
	log.Printf("INFO: Deleted %d tokens for user %s", deletedCount, uid)
	return nil
} 