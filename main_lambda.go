//go:build !local

package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var responseData map[string]interface{}
	var statusCode int

	switch request.HTTPMethod {
	case "POST":
		if request.Path == "/api/time" {
			responseData, statusCode = processPostRequest(ctx, request)
		}
	case "GET":
		if spaceId, ok := request.PathParameters["spaceId"]; ok {
			responseData, statusCode = processGetRequest(ctx, spaceId)
		}
	case "OPTIONS":
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    getCorsHeaders(),
		}, nil
	}

	if responseData == nil {
		responseData = map[string]interface{}{"error": "Not Found"}
		statusCode = http.StatusNotFound
	}

	body, err := json.Marshal(responseData)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: "Failed to marshal response", StatusCode: http.StatusInternalServerError}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    getCorsHeaders(),
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}