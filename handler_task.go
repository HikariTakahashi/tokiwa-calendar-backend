package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TaskSlot タスクスロットの構造体
type TaskSlot struct {
	Description string `json:"description"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Title       string `json:"title"`
	UserColor   string `json:"userColor"`
	Order       int    `json:"order"`
}

// TaskSaveRequest タスク保存リクエストの構造体
type TaskSaveRequest struct {
	UserUID string                    `json:"useruid"`
	Events  map[string][]TaskSlot `json:"events"`
}

// TaskSaveResponse タスク保存レスポンスの構造体
type TaskSaveResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// TaskGetResponse タスク取得レスポンスの構造体
type TaskGetResponse struct {
	Events  map[string][]TaskSlot `json:"events"`
	Message string                `json:"message"`
	Success bool                  `json:"success"`
	Error   string                `json:"error,omitempty"`
}

// handleTaskSave タスク保存ハンドラー
func handleTaskSave(w http.ResponseWriter, r *http.Request) {
	// CORSヘッダーを設定
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// OPTIONSリクエストの場合は早期リターン
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTメソッドのみ許可
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 認証トークンを取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Bearerトークンを抽出
	token := extractBearerToken(authHeader)
	if token == "" {
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	// トークンを検証してUIDを取得
	session, err := validateSessionToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	uid := session.UID

	// リクエストボディをパース
	var request TaskSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Firestoreクライアントを取得
	ctx := context.Background()
	client := firestoreClient
	if client == nil {
		log.Printf("Firestore client is not initialized")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// リクエストのUserUIDとトークンから取得したUIDが一致するかチェック
	if request.UserUID != uid {
		log.Printf("UserUID mismatch: request.UserUID=%s, token.UID=%s", request.UserUID, uid)
		response := TaskSaveResponse{
			Message: "ユーザーIDが一致しません",
			Success: false,
			Error:   "UserUID mismatch",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// タスクデータを保存
	if err := saveTaskDataToFirestore(ctx, client, uid, request.Events); err != nil {
		log.Printf("Failed to save task data: %v", err)
		response := TaskSaveResponse{
			Message: "タスクの保存に失敗しました",
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 成功レスポンス
	response := TaskSaveResponse{
		Message: "タスクが正常に保存されました",
		Success: true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTaskGet タスク取得ハンドラー
func handleTaskGet(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleTaskGet called - Method: %s, URL: %s", r.Method, r.URL.String())
	
	// CORSヘッダーを設定
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// OPTIONSリクエストの場合は早期リターン
	if r.Method == "OPTIONS" {
		log.Printf("DEBUG: OPTIONS request, returning early")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GETメソッドのみ許可
	if r.Method != "GET" {
		log.Printf("DEBUG: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 認証トークンを取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("DEBUG: Authorization header required")
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Bearerトークンを抽出
	token := extractBearerToken(authHeader)
	if token == "" {
		log.Printf("DEBUG: Invalid authorization header")
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	// トークンを検証してUIDを取得
	session, err := validateSessionToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	uid := session.UID
	log.Printf("DEBUG: Token validated successfully for UID: %s", uid)

	// Firestoreクライアントを取得
	ctx := context.Background()
	client := firestoreClient
	if client == nil {
		log.Printf("Firestore client is not initialized")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// タスクデータを取得
	log.Printf("DEBUG: Getting task data for UID: %s", uid)
	events, err := getExistingTaskData(ctx, client, uid)
	if err != nil {
		log.Printf("Failed to get task data: %v", err)
		response := TaskGetResponse{
			Message: "タスクデータの取得に失敗しました",
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("DEBUG: Task data retrieved successfully for UID: %s, events count: %d", uid, len(events))
	log.Printf("DEBUG: Events data: %+v", events)

	// 成功レスポンス
	response := TaskGetResponse{
		Events:  events,
		Message: "タスクデータを正常に取得しました",
		Success: true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("DEBUG: Task get response sent successfully")
}

// saveTaskDataToFirestore Firestoreにタスクデータを保存
func saveTaskDataToFirestore(ctx context.Context, client *firestore.Client, uid string, events map[string][]TaskSlot) error {
	log.Printf("DEBUG: saveTaskDataToFirestore called for UID %s", uid)
	
	// 既存のタスクデータを取得
	existingTasks, err := getExistingTaskData(ctx, client, uid)
	if err != nil {
		return fmt.Errorf("failed to get existing task data: %v", err)
	}

	// 新しいイベントデータを既存データにマージ
	for date, tasks := range events {
		existingTasks[date] = tasks
	}

	log.Printf("DEBUG: Saving task data to Firestore for UID %s", uid)
	log.Printf("DEBUG: Task data to save: %+v", existingTasks)

	// Firestoreに保存
	docRef := client.Collection("task").Doc(uid)
	_, err = docRef.Set(ctx, map[string]interface{}{
		"events":    existingTasks,
		"updatedAt": time.Now(),
		"uid":       uid,
	})
	if err != nil {
		log.Printf("ERROR: Failed to save task data to Firestore: %v", err)
		return fmt.Errorf("failed to save to Firestore: %v", err)
	}

	log.Printf("DEBUG: Task data saved successfully for UID %s", uid)
	return nil
}

// getExistingTaskData 既存のタスクデータを取得
func getExistingTaskData(ctx context.Context, client *firestore.Client, uid string) (map[string][]TaskSlot, error) {
	docRef := client.Collection("task").Doc(uid)
	doc, err := docRef.Get(ctx)
	if err != nil {
		// デバッグ情報を出力
		log.Printf("DEBUG: getExistingTaskData error for UID %s: %v", uid, err)
		
		// ドキュメントが存在しない場合は空のマップを返す
		// Firestoreでは、ドキュメントが存在しない場合にNotFoundエラーが返される
		if status.Code(err) == codes.NotFound || strings.Contains(err.Error(), "NotFound") || err == iterator.Done {
			log.Printf("DEBUG: Document not found for UID %s, returning empty map", uid)
			return make(map[string][]TaskSlot), nil
		}
		return nil, err
	}

	var data map[string]interface{}
	if err := doc.DataTo(&data); err != nil {
		log.Printf("DEBUG: Failed to convert document data: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: Raw Firestore data for UID %s: %+v", uid, data)

	// eventsフィールドを取得
	eventsData, ok := data["events"]
	if !ok {
		log.Printf("DEBUG: No events field found in document for UID %s", uid)
		return make(map[string][]TaskSlot), nil
	}

	log.Printf("DEBUG: Events data for UID %s: %+v", uid, eventsData)

	// 型アサーション
	eventsMap, ok := eventsData.(map[string]interface{})
	if !ok {
		return make(map[string][]TaskSlot), nil
	}

	// TaskSlotの配列に変換
	result := make(map[string][]TaskSlot)
	for date, tasksData := range eventsMap {
		log.Printf("DEBUG: Processing date %s, tasksData: %+v", date, tasksData)
		
		tasksArray, ok := tasksData.([]interface{})
		if !ok {
			log.Printf("DEBUG: Failed to convert tasksData to array for date %s", date)
			continue
		}

		var tasks []TaskSlot
		for i, taskData := range tasksArray {
			log.Printf("DEBUG: Processing task %d for date %s: %+v", i, date, taskData)
			taskMap, ok := taskData.(map[string]interface{})
			if !ok {
				continue
			}

			task := TaskSlot{}
			log.Printf("DEBUG: Task map for date %s, task %d: %+v", date, i, taskMap)
			
			// フィールド名の大文字小文字を考慮して取得
			if description, ok := taskMap["description"].(string); ok {
				task.Description = description
				log.Printf("DEBUG: Set description: %s", description)
			} else if description, ok := taskMap["Description"].(string); ok {
				task.Description = description
				log.Printf("DEBUG: Set Description: %s", description)
			}
			
			if start, ok := taskMap["start"].(string); ok {
				task.Start = start
				log.Printf("DEBUG: Set start: %s", start)
			} else if start, ok := taskMap["Start"].(string); ok {
				task.Start = start
				log.Printf("DEBUG: Set Start: %s", start)
			}
			
			if end, ok := taskMap["end"].(string); ok {
				task.End = end
				log.Printf("DEBUG: Set end: %s", end)
			} else if end, ok := taskMap["End"].(string); ok {
				task.End = end
				log.Printf("DEBUG: Set End: %s", end)
			}
			
			if title, ok := taskMap["title"].(string); ok {
				task.Title = title
				log.Printf("DEBUG: Set title: %s", title)
			} else if title, ok := taskMap["Title"].(string); ok {
				task.Title = title
				log.Printf("DEBUG: Set Title: %s", title)
			}
			
			if userColor, ok := taskMap["userColor"].(string); ok {
				task.UserColor = userColor
				log.Printf("DEBUG: Set userColor: %s", userColor)
			} else if userColor, ok := taskMap["UserColor"].(string); ok {
				task.UserColor = userColor
				log.Printf("DEBUG: Set UserColor: %s", userColor)
			}
			
			if order, ok := taskMap["order"].(float64); ok {
				task.Order = int(order)
				log.Printf("DEBUG: Set order: %d", int(order))
			} else if order, ok := taskMap["Order"].(float64); ok {
				task.Order = int(order)
				log.Printf("DEBUG: Set Order: %d", int(order))
			}

			log.Printf("DEBUG: Final task object: %+v", task)
			tasks = append(tasks, task)
		}
		result[date] = tasks
		log.Printf("DEBUG: Final result for date %s: %+v", date, tasks)
	}

	log.Printf("DEBUG: Final result map: %+v", result)
	return result, nil
}

// extractBearerToken Bearerトークンを抽出
func extractBearerToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}
