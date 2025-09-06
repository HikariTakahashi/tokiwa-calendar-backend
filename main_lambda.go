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

func main() {
	// 環境変数ファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}
	
	lambda.Start(handler)
}