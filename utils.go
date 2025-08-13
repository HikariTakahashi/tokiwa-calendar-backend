// utils.go
package main

import (
	"context"
	"os"
	"strings"
)

// Lambda用: CORSヘッダーのマップを返す
// (main_lambda.go から呼ばれる)
func getCorsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Methods":     "POST, GET, OPTIONS, PUT, DELETE",
		"Access-Control-Allow-Headers":     "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
		"Access-Control-Allow-Credentials": "true",
	}
}

// userContextKey は、コンテキストキーの衝突を避けるためのカスタム型です。
type userContextKey string

// uidContextKey は、コンテキスト内でユーザーIDを保存・取得するためのキーです。
const uidContextKey userContextKey = "userUID"

// setUIDInContext は、ユーザーIDを含む新しいコンテキストを生成します。
func setUIDInContext(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, uidContextKey, uid)
}

// getUIDFromContext は、コンテキストからユーザーIDを取得します。
func getUIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey).(string)
	return uid, ok
}

// isLambdaEnvironment は現在の環境がAWS Lambdaかどうかを判定します
func isLambdaEnvironment() bool {
	// AWS Lambda環境では以下の環境変数が設定されます
	lambdaEnvVars := []string{
		"AWS_LAMBDA_FUNCTION_NAME",
		"AWS_LAMBDA_FUNCTION_VERSION",
		"AWS_LAMBDA_LOG_GROUP_NAME",
		"AWS_LAMBDA_LOG_STREAM_NAME",
		"AWS_LAMBDA_RUNTIME_API",
	}
	
	for _, envVar := range lambdaEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	
	return false
}

// shouldSkipEmailVerification はLambda環境でメール送信が失敗した場合にメール認証をスキップするかどうかを判定します
func shouldSkipEmailVerification() bool {
	// Lambda環境でのみメール認証スキップを許可
	if !isLambdaEnvironment() {
		return false
	}
	
	// 環境変数で明示的にスキップを許可する場合
	skipEmail := os.Getenv("SKIP_EMAIL_VERIFICATION")
	return strings.ToLower(skipEmail) == "true"
}