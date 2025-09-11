//go:build !local

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	// ★★★【最重要変更点】★★★ 使用するイベントの型をV2に変更します
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
)

// ★★★【最重要変更点】★★★
// ハンドラが受け取るrequestの型を、正しい events.APIGatewayV2HTTPRequest に変更します。
func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {

	// デバッグログ
	log.Printf("Received request: %+v", request)

	var responseData map[string]interface{}
	var statusCode int

	// ルーティングロジックを整理
	path := request.RequestContext.HTTP.Path
	method := request.RequestContext.HTTP.Method

	if strings.HasPrefix(path, "/api/signup") && method == "POST" {
		responseData, statusCode = processSignupRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/login") && method == "POST" {
		responseData, statusCode = processLoginRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/verify") && method == "POST" {
		responseData, statusCode = ProcessVerifyRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/cleanup") && method == "POST" {
		responseData, statusCode = ProcessCleanupRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/user-data") {
		response, err := lambdaUserDataHandler(ctx, request)
		if err != nil {
			log.Printf("ERROR: Lambda user data handler error: %v", err)
			responseData = map[string]interface{}{"error": "Internal server error"}
			statusCode = http.StatusInternalServerError
		} else {
			json.Unmarshal([]byte(response.Body), &responseData)
			statusCode = response.StatusCode
		}
	} else if strings.HasPrefix(path, "/api/auth/google") && method == "POST" {
		responseData, statusCode = processGoogleAuthRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/auth/github") && method == "POST" {
		responseData, statusCode = processGitHubAuthRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/auth/twitter") && method == "POST" {
		responseData, statusCode = processTwitterAuthRequest(ctx, request)
	} else if strings.HasPrefix(path, "/api/user-providers") && method == "GET" {
		response, err := lambdaUserProvidersHandler(ctx, request)
		if err != nil {
			log.Printf("ERROR: Lambda user providers handler error: %v", err)
			responseData = map[string]interface{}{"error": "Internal server error"}
			statusCode = http.StatusInternalServerError
		} else {
			json.Unmarshal([]byte(response.Body), &responseData)
			statusCode = response.StatusCode
		}
	} else if strings.HasPrefix(path, "/api/user-profile") && method == "GET" {
		response, err := lambdaUserProfileHandler(ctx, request)
		if err != nil {
			log.Printf("ERROR: Lambda user profile handler error: %v", err)
			responseData = map[string]interface{}{"error": "Internal server error"}
			statusCode = http.StatusInternalServerError
		} else {
			json.Unmarshal([]byte(response.Body), &responseData)
			statusCode = response.StatusCode
		}
	} else if strings.HasPrefix(path, "/api/user-providers-detail") && method == "GET" {
		response, err := lambdaUserProvidersDetailHandler(ctx, request)
		if err != nil {
			log.Printf("ERROR: Lambda user providers detail handler error: %v", err)
			responseData = map[string]interface{}{"error": "Internal server error"}
			statusCode = http.StatusInternalServerError
		} else {
			json.Unmarshal([]byte(response.Body), &responseData)
			statusCode = response.StatusCode
		}
	} else if strings.HasPrefix(path, "/email-config") && method == "GET" {
		responseData, statusCode = checkEmailConfig()
	} else if strings.HasPrefix(path, "/email-debug") && method == "GET" {
		responseData, statusCode = checkEmailDebug()
	} else if strings.HasPrefix(path, "/api/task") {
		// タスク関連のエンドポイント
		authHeader := request.Headers["authorization"]
		if authHeader == "" {
			responseData = map[string]interface{}{"error": "認証が必要です"}
			statusCode = http.StatusUnauthorized
		} else {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				sessionToken := parts[1]
				userSession, err := validateSessionToken(sessionToken)
				if err != nil {
					log.Printf("ERROR: Lambda - Failed to verify session token: %v", err)
					responseData = map[string]interface{}{"error": "認証に失敗しました"}
					statusCode = http.StatusUnauthorized
				} else {
					log.Printf("INFO: Lambda - Authenticated user: %s", userSession.UID)
					newCtx := context.WithValue(ctx, "token", struct{ UID string }{UID: userSession.UID})
					
					if method == "GET" {
						responseData, statusCode = processTaskGetLambda(newCtx, userSession.UID)
					} else if method == "POST" {
						responseData, statusCode = processTaskSaveLambda(newCtx, request)
					} else {
						responseData = map[string]interface{}{"error": "Method not allowed"}
						statusCode = http.StatusMethodNotAllowed
					}
				}
			} else {
				responseData = map[string]interface{}{"error": "認証ヘッダーの形式が正しくありません"}
				statusCode = http.StatusUnauthorized
			}
		}
	} else if strings.HasPrefix(path, "/api/time") {
		if method == "POST" {
			// POST /api/time の処理
			authHeader := request.Headers["authorization"]
			newCtx := ctx
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					idToken := parts[1]
					token, err := authClient.VerifyIDToken(ctx, idToken)
					if err == nil {
						log.Printf("INFO: Lambda - Authenticated user: %s", token.UID)
						newCtx = setUIDInContext(ctx, token.UID)
					}
				}
			}
			proxyReq := events.APIGatewayProxyRequest{
				Body: request.Body,
			}
			responseData, statusCode = processPostRequest(newCtx, proxyReq)
		} else if method == "GET" {
			// GET /api/time/{spaceId} の処理
			// パスを / で分割して、4番目の要素（spaceId）を取得
			parts := strings.Split(path, "/")
			if len(parts) >= 4 && parts[3] != "" {
				spaceId := parts[3]
				responseData, statusCode = processGetRequest(ctx, spaceId)
			}
		}
	}

	// OPTIONSリクエストはすべてのパスで許可
	if method == "OPTIONS" {
		// getCorsHeadersは utils.go にあるものを使用します
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    getCorsHeaders(),
		}, nil
	}

	// すべてのレスポンスにCORSヘッダーを追加
	corsHeaders := getCorsHeaders()

	if responseData == nil {
		log.Printf("No route matched for method [%s] and path [%s]", request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path)
		responseData = map[string]interface{}{"error": "Not Found", "requestedPath": request.RequestContext.HTTP.Path}
		statusCode = http.StatusNotFound
	}

	body, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("ERROR: Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders,
			Body:       "{\"error\":\"Failed to process the response\"}",
		}, nil
	}

	log.Printf("Responding with status code %d.", statusCode)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    corsHeaders,
		Body:       string(body),
	}, nil
}

// processTaskSaveLambda はLambda用のタスク保存処理です
func processTaskSaveLambda(ctx context.Context, request events.APIGatewayV2HTTPRequest) (map[string]interface{}, int) {
	// リクエストボディをパース
	var taskRequest TaskSaveRequest
	if err := json.Unmarshal([]byte(request.Body), &taskRequest); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		return map[string]interface{}{"error": "Invalid request body"}, http.StatusBadRequest
	}

	// Firestoreクライアントを使用
	if firestoreClient == nil {
		log.Printf("Firestore client is not initialized")
		return map[string]interface{}{"error": "Internal server error"}, http.StatusInternalServerError
	}

	// タスクデータを保存
	if err := saveTaskDataToFirestore(ctx, firestoreClient, taskRequest.UserUID, taskRequest.Events, taskRequest.Notifications); err != nil {
		log.Printf("Failed to save task data: %v", err)
		return map[string]interface{}{"error": "Failed to save task data"}, http.StatusInternalServerError
	}

	return map[string]interface{}{
		"message": "タスクデータが正常に保存されました",
		"success": true,
	}, http.StatusOK
}

// processTaskGetLambda はLambda用のタスク取得処理です
func processTaskGetLambda(ctx context.Context, uid string) (map[string]interface{}, int) {
	// Firestoreクライアントを使用
	if firestoreClient == nil {
		log.Printf("Firestore client is not initialized")
		return map[string]interface{}{"error": "Internal server error"}, http.StatusInternalServerError
	}

	// 既存のタスクデータを取得
	events, err := getExistingTaskData(ctx, firestoreClient, uid)
	if err != nil {
		log.Printf("Failed to get existing task data: %v", err)
		return map[string]interface{}{"error": "Failed to get task data"}, http.StatusInternalServerError
	}

	// 通知データを取得（空のマップを返す）
	notifications := make(map[string][]NotificationSlot)

	return map[string]interface{}{
		"events":        events,
		"notifications": notifications,
		"message":       "タスクデータが正常に取得されました",
		"success":       true,
	}, http.StatusOK
}

func main() {
	// 環境変数ファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}
	
	lambda.Start(handler)
}