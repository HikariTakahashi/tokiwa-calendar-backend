package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// UserData はユーザーデータの構造体です
type UserData struct {
	UserName  string `json:"userName" firestore:"userName"`
	UserColor string `json:"userColor" firestore:"userColor"`
	UID       string `json:"uid" firestore:"uid"`
}

// UserDataRequest はユーザーデータ更新リクエストの構造体です
type UserDataRequest struct {
	UserName  string `json:"userName"`
	UserColor string `json:"userColor"`
}

// UserDataResponse はユーザーデータレスポンスの構造体です
type UserDataResponse struct {
	UserName  string `json:"userName,omitempty"`
	UserColor string `json:"userColor,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// processUserDataSaveRequest はユーザーデータ保存リクエストを処理します
func processUserDataSaveRequest(ctx context.Context, req interface{}, token *auth.Token) (map[string]interface{}, int) {
	var bodyBytes []byte
	var err error

	// リクエストソース（ローカルサーバー or Lambda）に応じてリクエストボディをバイトスライスとして取得
	switch r := req.(type) {
	case *http.Request:
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: Failed to read request body: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayV2HTTPRequest:
		bodyBytes = []byte(r.Body)
	default:
		log.Printf("ERROR: Unknown request type: %T\n", r)
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var userData UserDataRequest
	if err := json.Unmarshal(bodyBytes, &userData); err != nil {
		log.Printf("ERROR: Failed to decode JSON: %v\n", err)
		return map[string]interface{}{"error": "JSONの解析に失敗しました"}, http.StatusBadRequest
	}

	// バリデーション
	if userData.UserName == "" {
		log.Printf("ERROR: UserName is required")
		return map[string]interface{}{"error": "ユーザー名は必須です"}, http.StatusBadRequest
	}

	if userData.UserColor == "" {
		log.Printf("ERROR: UserColor is required")
		return map[string]interface{}{"error": "ユーザーカラーは必須です"}, http.StatusBadRequest
	}

	// UserDataを作成
	userDataToSave := &UserData{
		UserName:  userData.UserName,
		UserColor: userData.UserColor,
		UID:       token.UID,
	}

	// Firestoreに保存
	if err := saveUserDataToFirestore(ctx, token.UID, userDataToSave); err != nil {
		log.Printf("ERROR: Failed to save user data to Firestore: %v", err)
		return map[string]interface{}{"error": "ユーザーデータの保存に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: User data saved successfully for UID: %s", token.UID)
	return map[string]interface{}{
		"message":   "ユーザーデータを保存しました",
		"userName":  userData.UserName,
		"userColor": userData.UserColor,
	}, http.StatusOK
}

// processUserDataGetRequest はユーザーデータ取得リクエストを処理します
func processUserDataGetRequest(ctx context.Context, token *auth.Token) (map[string]interface{}, int) {
	// Firestoreからユーザーデータを取得
	userData, err := getUserDataFromFirestore(ctx, token.UID)
	if err != nil {
		// データが見つからない場合はデフォルト値を返す
		log.Printf("INFO: User data not found for UID: %s, returning default values", token.UID)
		return map[string]interface{}{
			"userName":  "",
			"userColor": "#3b82f6",
		}, http.StatusOK
	}

	log.Printf("INFO: User data retrieved successfully for UID: %s", token.UID)
	return map[string]interface{}{
		"userName":  userData.UserName,
		"userColor": userData.UserColor,
	}, http.StatusOK
}

// handleUserDataRequest はユーザーデータリクエストのエントリーポイントです
func handleUserDataRequest(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{}
	var statusCode int

	// 認証情報の取得 - セッショントークン用の構造体からUIDを取得
	tokenValue := r.Context().Value("token")
	if tokenValue == nil {
		log.Printf("ERROR: User not authenticated - no token in context")
		result = map[string]interface{}{"error": "認証が必要です"}
		statusCode = http.StatusUnauthorized
	} else {
		// セッショントークン用のmockToken構造体からUIDを取得
		mockToken, ok := tokenValue.(struct{ UID string })
		if !ok {
			log.Printf("ERROR: Invalid token type in context")
			result = map[string]interface{}{"error": "認証が必要です"}
			statusCode = http.StatusUnauthorized
		} else {
			// Firebase auth.Token形式にマッピング（既存のコードとの互換性のため）
			token := &auth.Token{
				UID: mockToken.UID,
			}
			log.Printf("DEBUG: handleUserDataRequest - Successfully extracted UID: %s", token.UID)
			
			switch r.Method {
			case http.MethodPost:
				result, statusCode = processUserDataSaveRequest(r.Context(), r, token)
			case http.MethodGet:
				result, statusCode = processUserDataGetRequest(r.Context(), token)
			default:
				log.Printf("ERROR: Method not allowed: %s", r.Method)
				result = map[string]interface{}{"error": "許可されていないメソッドです"}
				statusCode = http.StatusMethodNotAllowed
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}

// lambdaUserDataHandler はLambda環境でのユーザーデータリクエストを処理します
func lambdaUserDataHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var result map[string]interface{}
	var statusCode int

	// 認証処理
	authHeader := ""
	if auth, exists := request.Headers["authorization"]; exists {
		authHeader = auth
	} else if auth, exists := request.Headers["Authorization"]; exists {
		authHeader = auth
	}

	if authHeader == "" {
		log.Printf("ERROR: Authorization header missing")
		result = map[string]interface{}{"error": "認証が必要です"}
		statusCode = http.StatusUnauthorized
	} else {
		// Bearerトークンの検証
		token, err := validateAuthHeader(ctx, authHeader)
		if err != nil {
			log.Printf("ERROR: Token validation failed: %v", err)
			result = map[string]interface{}{"error": "認証に失敗しました"}
			statusCode = http.StatusUnauthorized
		} else {
			switch request.RequestContext.HTTP.Method {
			case http.MethodPost:
				result, statusCode = processUserDataSaveRequest(ctx, request, token)
			case http.MethodGet:
				result, statusCode = processUserDataGetRequest(ctx, token)
			default:
				log.Printf("ERROR: Method not allowed: %s", request.RequestContext.HTTP.Method)
				result = map[string]interface{}{"error": "許可されていないメソッドです"}
				statusCode = http.StatusMethodNotAllowed
			}
		}
	}

	body, _ := json.Marshal(result)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

// validateAuthHeader はAuthorizationヘッダーを検証してFirebase Auth Tokenを返します
func validateAuthHeader(ctx context.Context, authHeader string) (*auth.Token, error) {
	// "Bearer " プレフィックスを検証・削除
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.New("invalid authorization header format")
	}
	idToken := parts[1]

	// Firebase Auth でトークンを検証
	token, err := authClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Printf("ERROR: Token validation failed: %v", err)
		return nil, err
	}

	return token, nil
}
