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
)

// ★★★【最重要変更点】★★★
// ハンドラが受け取るrequestの型を、正しい events.APIGatewayV2HTTPRequest に変更します。
func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {

	// デバッグログ
	log.Printf("Received request: %+v", request)

	var responseData map[string]interface{}
	var statusCode int

	// 正しい型になったので、シンプルな判定に戻します
	switch request.RequestContext.HTTP.Method {
	case "POST":
		if request.RequestContext.HTTP.Path == "/api/time" {
			// events.APIGatewayProxyRequest 型に変換して渡す
			// (processPostRequestがこの型を期待しているため)
			proxyReq := events.APIGatewayProxyRequest{
				Body: request.Body,
			}
			responseData, statusCode = processPostRequest(ctx, proxyReq)
		}
	case "GET":
		// パス `/api/time/{spaceId}` にマッチするかどうかを判定
		path := request.RequestContext.HTTP.Path
		if strings.HasPrefix(path, "/api/time/") {
			// パスを / で分割して、4番目の要素（spaceId）を取得
			parts := strings.Split(path, "/")
			if len(parts) >= 4 && parts[3] != "" {
				spaceId := parts[3]
				responseData, statusCode = processGetRequest(ctx, spaceId)
			}
		}
	case "OPTIONS":
		// getCorsHeadersは utils.go にあるものを使用します
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    getCorsHeaders(),
		}, nil
	}

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
			Headers:    getCorsHeaders(),
			Body:       "{\"error\":\"Failed to process the response\"}",
		}, nil
	}

	log.Printf("Responding with status code %d.", statusCode)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    getCorsHeaders(),
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}