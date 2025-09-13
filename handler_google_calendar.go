package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// GoogleAccessTokenResponse はGoogleアクセストークン取得レスポンスの構造体です
type GoogleAccessTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Error        string `json:"error,omitempty"`
}

// GoogleTokenRequest はGoogleトークンリクエストの構造体です
type GoogleTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

// GoogleCalendarToken はGoogle Calendar用のトークン情報を保存する構造体です
type GoogleCalendarToken struct {
	UserUID      string    `json:"userUID" firestore:"userUID"`
	AccessToken  string    `json:"accessToken" firestore:"accessToken"`
	RefreshToken string    `json:"refreshToken" firestore:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt" firestore:"expiresAt"`
	TokenType    string    `json:"tokenType" firestore:"tokenType"`
	CreatedAt    time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// processGoogleAccessTokenRequest はGoogleアクセストークン取得リクエストを処理します
func processGoogleAccessTokenRequest(ctx context.Context, req interface{}, token *auth.Token) (map[string]interface{}, int) {
	// ユーザーのGoogle Calendarトークン情報を取得
	calendarToken, err := getGoogleCalendarToken(ctx, token.UID)
	if err != nil {
		log.Printf("ERROR: Failed to get Google Calendar token for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "Google Calendarトークンの取得に失敗しました"}, http.StatusInternalServerError
	}

	// トークンが存在しない場合
	if calendarToken == nil {
		return map[string]interface{}{"error": "Google Calendarが連携されていません"}, http.StatusNotFound
	}

	// トークンの有効性をチェック
	now := time.Now()
	if calendarToken.ExpiresAt.Before(now) {
		// トークンが期限切れの場合、リフレッシュトークンで更新
		newToken, err := refreshGoogleAccessToken(calendarToken.RefreshToken)
		if err != nil {
			log.Printf("ERROR: Failed to refresh Google access token for UID %s: %v", token.UID, err)
			return map[string]interface{}{"error": "Googleアクセストークンの更新に失敗しました"}, http.StatusInternalServerError
		}

		// 新しいトークン情報を保存
		calendarToken.AccessToken = newToken.AccessToken
		calendarToken.TokenType = newToken.TokenType
		calendarToken.ExpiresAt = now.Add(time.Duration(newToken.ExpiresIn) * time.Second)
		calendarToken.UpdatedAt = now

		if newToken.RefreshToken != "" {
			calendarToken.RefreshToken = newToken.RefreshToken
		}

		if err := saveGoogleCalendarToken(ctx, token.UID, calendarToken); err != nil {
			log.Printf("WARN: Failed to save refreshed Google Calendar token: %v", err)
		}
	}

	return map[string]interface{}{
		"accessToken": calendarToken.AccessToken,
		"tokenType":   calendarToken.TokenType,
		"expiresIn":   int(calendarToken.ExpiresAt.Sub(now).Seconds()),
	}, http.StatusOK
}

// getGoogleCalendarToken はFirestoreからGoogle Calendarトークン情報を取得します
func getGoogleCalendarToken(ctx context.Context, userUID string) (*GoogleCalendarToken, error) {
	doc, err := firestoreClient.Collection("google_calendar_tokens").Doc(userUID).Get(ctx)
	if err != nil {
		if err.Error() == "rpc error: code = NotFound desc = Document not found" {
			return nil, nil // トークンが存在しない
		}
		return nil, fmt.Errorf("Firestore取得エラー: %v", err)
	}

	var token GoogleCalendarToken
	if err := doc.DataTo(&token); err != nil {
		return nil, fmt.Errorf("トークンデータ解析エラー: %v", err)
	}

	return &token, nil
}

// saveGoogleCalendarToken はFirestoreにGoogle Calendarトークン情報を保存します
func saveGoogleCalendarToken(ctx context.Context, userUID string, token *GoogleCalendarToken) error {
	_, err := firestoreClient.Collection("google_calendar_tokens").Doc(userUID).Set(ctx, token)
	if err != nil {
		return fmt.Errorf("Firestore保存エラー: %v", err)
	}
	return nil
}

// refreshGoogleAccessToken はリフレッシュトークンを使用してアクセストークンを更新します
func refreshGoogleAccessToken(refreshToken string) (*GoogleTokenResponse, error) {
	clientID := getGoogleClientID()
	clientSecret := getGoogleClientSecret()

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Google OAuth2.0設定が不完全です")
	}

	// トークンエンドポイントにリクエスト
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("トークン更新リクエストエラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("トークン更新失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("トークンレスポンス解析エラー: %v", err)
	}

	if tokenResponse.Error != "" {
		return nil, fmt.Errorf("Google API エラー: %s - %s", tokenResponse.Error, tokenResponse.ErrorDesc)
	}

	return &tokenResponse, nil
}

// saveGoogleCalendarTokenFromAuth は認証時にGoogle Calendarトークンを保存します
func saveGoogleCalendarTokenFromAuth(ctx context.Context, userUID string, accessToken, refreshToken, tokenType string, expiresIn int) error {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second)

	token := &GoogleCalendarToken{
		UserUID:      userUID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tokenType,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return saveGoogleCalendarToken(ctx, userUID, token)
}

// lambdaGoogleAccessTokenHandler はLambda用のGoogleアクセストークン取得ハンドラーです
func lambdaGoogleAccessTokenHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	log.Printf("DEBUG: lambdaGoogleAccessTokenHandler called")
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
			log.Printf("DEBUG: Calling processGoogleAccessTokenRequest for UID: %s", token.UID)
			result, statusCode = processGoogleAccessTokenRequest(ctx, request, token)
			log.Printf("DEBUG: processGoogleAccessTokenRequest returned result: %+v", result)
		}
	}

	body, _ := json.Marshal(result)
	log.Printf("DEBUG: lambdaGoogleAccessTokenHandler returning body: %s", string(body))
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}
