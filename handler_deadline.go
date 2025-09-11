package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// DeadlineDocument search_deadlineコレクション用の構造体
type DeadlineDocument struct {
	UserID      string    `firestore:"userId"`
	Deadline    time.Time `firestore:"deadline"`
	Status      string    `firestore:"status"`
	MQTTTopic   string    `firestore:"mqttTopic"`
	MQTTMessage string    `firestore:"mqttMessage"`
	CreatedAt   time.Time `firestore:"createdAt"`
}

// createDeadlineFromNotification 通知データからdeadlineを生成する関数
func createDeadlineFromNotification(date string, notificationTime string, title string, userID string) (time.Time, error) {
	// 日付と時間を結合してパース
	dateTimeStr := fmt.Sprintf("%s %s", date, notificationTime)
	
	// 日本時間（UTC+9）でパース
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load timezone: %v", err)
	}
	
	// 日付と時間をパース（"2025-09-16 09:00"形式）
	deadline, err := time.ParseInLocation("2006-01-02 15:04", dateTimeStr, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse datetime: %v", err)
	}
	
	return deadline, nil
}

// saveDeadlineToFirestore search_deadlineコレクションにデータを保存する関数
func saveDeadlineToFirestore(ctx context.Context, client *firestore.Client, userID string, events map[string][]TaskSlot, notifications map[string][]NotificationSlot) error {
	log.Printf("DEBUG: saveDeadlineToFirestore called for UID %s", userID)
	
	// 通知データを処理
	for date, notificationSlots := range notifications {
		// 該当日のタスクを取得
		taskSlots, exists := events[date]
		if !exists || len(taskSlots) == 0 {
			log.Printf("DEBUG: No tasks found for date %s, skipping notifications", date)
			continue
		}
		
		// 最初のタスクのタイトルを使用（複数タスクがある場合は最初のものを使用）
		title := taskSlots[0].Title
		if title == "" {
			title = "タスクリマインダー" // デフォルトタイトル
		}
		
		// 各通知時間に対してドキュメントを作成
		for _, notification := range notificationSlots {
			// deadlineを生成
			deadline, err := createDeadlineFromNotification(date, notification.Time, title, userID)
			if err != nil {
				log.Printf("ERROR: Failed to create deadline for date %s, time %s: %v", date, notification.Time, err)
				continue
			}
			
			// MQTTトピックとメッセージを生成
			mqttTopic := "calendar/test/reminders"
			mqttMessage := fmt.Sprintf(`{"title":"%s","time":"%s"}`, title, notification.Time)
			
			// DeadlineDocumentを作成
			deadlineDoc := DeadlineDocument{
				UserID:      userID,
				Deadline:    deadline,
				Status:      "pending",
				MQTTTopic:   mqttTopic,
				MQTTMessage: mqttMessage,
				CreatedAt:   time.Now(),
			}
			
			// Firestoreに保存
			_, _, err = client.Collection("search_deadline").Add(ctx, deadlineDoc)
			if err != nil {
				log.Printf("ERROR: Failed to save deadline document: %v", err)
				return fmt.Errorf("failed to save deadline document: %v", err)
			}
			
			log.Printf("DEBUG: Saved deadline document for UID %s, date %s, time %s", userID, date, notification.Time)
		}
	}
	
	log.Printf("DEBUG: Successfully saved all deadline documents for UID %s", userID)
	return nil
}
