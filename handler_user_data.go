package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// UserData はユーザーデータの構造体です
type UserData struct {
	UserName  string                    `json:"userName" firestore:"userName"`
	UserColor string                    `json:"userColor" firestore:"userColor"`
	UID       string                    `json:"uid" firestore:"uid"`
	Email     []EmailProviderInfo       `json:"email,omitempty" firestore:"email,omitempty"`
	Google    []OAuthProviderInfo       `json:"google,omitempty" firestore:"google,omitempty"`
	GitHub    []OAuthProviderInfo       `json:"github,omitempty" firestore:"github,omitempty"`
	Twitter   []OAuthProviderInfo       `json:"twitter,omitempty" firestore:"twitter,omitempty"`
}

// EmailProviderInfo はメールアドレス認証の情報です
type EmailProviderInfo struct {
	EmailAddress string `json:"emailAddress" firestore:"emailAddress"`
	UserUID      string `json:"userUID" firestore:"userUID"`
}

// OAuthProviderInfo はOAuth認証の情報です
type OAuthProviderInfo struct {
	UserUID      string `json:"userUID" firestore:"userUID"`
	EmailAddress string `json:"emailAddress" firestore:"emailAddress"`
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

// UserProvidersResponse はユーザープロバイダー情報のレスポンス構造体です
type UserProvidersResponse struct {
	Providers []string `json:"providers,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// UserProfileResponse は統合されたユーザープロフィール情報のレスポンス構造体です
type UserProfileResponse struct {
	UserName  string           `json:"userName,omitempty"`
	UserColor string           `json:"userColor,omitempty"`
	Providers []string         `json:"providers,omitempty"`
	ProviderDetails []ProviderDetail `json:"providerDetails,omitempty"`
	Message   string           `json:"message,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// LinkAccountRequest はアカウントリンクリクエストの構造体です
type LinkAccountRequest struct {
	Provider   string `json:"provider"`
	Credential string `json:"credential"`
}

// LinkAccountResponse はアカウントリンクレスポンスの構造体です
type LinkAccountResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UnlinkAccountRequest はアカウント解除リクエストの構造体です
type UnlinkAccountRequest struct {
	Provider string `json:"provider"`
}

// ProviderDetail はプロバイダーの詳細情報を表します
type ProviderDetail struct {
	Provider    string `json:"provider"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	IsLinked    bool   `json:"isLinked"`
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

	// 既存のユーザーデータを取得
	existingUserData, err := getUserDataByUID(ctx, token.UID)
	if err != nil {
		log.Printf("WARN: Failed to get existing user data: %v", err)
		// 既存データが見つからない場合は新規作成
		existingUserData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       token.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}

	// ユーザー名とカラーのみを更新（プロバイダー情報は保持）
	existingUserData.UserName = userData.UserName
	existingUserData.UserColor = userData.UserColor
	existingUserData.UID = token.UID

	// Firestoreに保存
	if err := saveUserDataToFirestore(ctx, token.UID, existingUserData); err != nil {
		log.Printf("ERROR: Failed to save user data to Firestore: %v", err)
		return map[string]interface{}{"error": "ユーザーデータの保存に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: User data saved successfully for UID: %s (UserName: %s, UserColor: %s)", 
		token.UID, userData.UserName, userData.UserColor)
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

func lambdaUserProvidersHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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
			result, statusCode = processUserProvidersRequest(ctx, token)
		}
	}

	body, _ := json.Marshal(result)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

func lambdaUserProvidersDetailHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	log.Printf("DEBUG: lambdaUserProvidersDetailHandler called")
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
			log.Printf("DEBUG: Calling processUserProvidersDetailRequest for UID: %s", token.UID)
			result, statusCode = processUserProvidersDetailRequest(ctx, token)
			log.Printf("DEBUG: processUserProvidersDetailRequest returned result: %+v", result)
		}
	}

	body, _ := json.Marshal(result)
	log.Printf("DEBUG: lambdaUserProvidersDetailHandler returning body: %s", string(body))
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
	sessionToken := parts[1]

	log.Printf("DEBUG: validateAuthHeader called with token length: %d", len(sessionToken))

	// セッショントークンを検証
	userSession, err := validateSessionToken(sessionToken)
	if err != nil {
		log.Printf("ERROR: Session token validation failed: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: Session token validated successfully for UID: %s, Email: %s", userSession.UID, userSession.Email)

	// Firebase auth.Token形式にマッピング（既存のコードとの互換性のため）
	token := &auth.Token{
		UID: userSession.UID,
	}

	return token, nil
}

// processUserProvidersRequest はユーザープロバイダー情報取得リクエストを処理します
func processUserProvidersRequest(ctx context.Context, token *auth.Token) (map[string]interface{}, int) {
	// データベースからユーザーデータを取得
	userData, err := getUserDataByUID(ctx, token.UID)
	if err != nil {
		log.Printf("ERROR: Failed to get user data for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}
	
	// userDataがnilの場合はデフォルト値を設定
	if userData == nil {
		log.Printf("INFO: User data is nil for UID %s, using default values", token.UID)
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       token.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}

	log.Printf("DEBUG: User data retrieved from database")
	log.Printf("DEBUG: Email providers count: %d", len(userData.Email))
	log.Printf("DEBUG: Google providers count: %d", len(userData.Google))
	log.Printf("DEBUG: GitHub providers count: %d", len(userData.GitHub))
	log.Printf("DEBUG: Twitter providers count: %d", len(userData.Twitter))

	// プロバイダー情報を抽出（データベースから）
	var providers []string
	
	// メールアドレスプロバイダー
	if len(userData.Email) > 0 {
		providers = append(providers, "password")
		log.Printf("DEBUG: Added password provider")
	}
	
	// OAuthプロバイダー
	if len(userData.Google) > 0 {
		providers = append(providers, "google.com")
		log.Printf("DEBUG: Added Google provider")
	}
	if len(userData.GitHub) > 0 {
		providers = append(providers, "github.com")
		log.Printf("DEBUG: Added GitHub provider")
	}
	if len(userData.Twitter) > 0 {
		providers = append(providers, "twitter.com")
		log.Printf("DEBUG: Added Twitter provider")
	}

	log.Printf("INFO: User providers retrieved for UID: %s, providers: %v", token.UID, providers)
	return map[string]interface{}{
		"providers": providers,
	}, http.StatusOK
}

// processUserProvidersDetailRequest はユーザープロバイダー詳細情報取得リクエストを処理します
func processUserProvidersDetailRequest(ctx context.Context, token *auth.Token) (map[string]interface{}, int) {
	log.Printf("DEBUG: processUserProvidersDetailRequest called for UID: %s", token.UID)
	
	// データベースからユーザーデータを取得
	userData, err := getUserDataByUID(ctx, token.UID)
	if err != nil {
		log.Printf("ERROR: Failed to get user data for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}
	
	// userDataがnilの場合はデフォルト値を設定
	if userData == nil {
		log.Printf("INFO: User data is nil for UID %s, using default values", token.UID)
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       token.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}
	
	log.Printf("DEBUG: User data retrieved from database")
	log.Printf("DEBUG: Email providers count: %d", len(userData.Email))
	log.Printf("DEBUG: Google providers count: %d", len(userData.Google))
	log.Printf("DEBUG: GitHub providers count: %d", len(userData.GitHub))
	log.Printf("DEBUG: Twitter providers count: %d", len(userData.Twitter))

	// プロバイダー詳細情報を抽出（データベースから）
	var providerDetails []ProviderDetail
	
	// メールアドレスプロバイダー
	if len(userData.Email) > 0 {
		for _, emailInfo := range userData.Email {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "password",
				Email:       emailInfo.EmailAddress,
				DisplayName: "メールアドレス",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added email provider - Email: %s", emailInfo.EmailAddress)
		}
	}
	
	// Googleプロバイダー
	if len(userData.Google) > 0 {
		for _, googleInfo := range userData.Google {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "google.com",
				Email:       googleInfo.EmailAddress,
				DisplayName: "Google",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added Google provider - Email: %s", googleInfo.EmailAddress)
		}
	}
	
	// GitHubプロバイダー
	if len(userData.GitHub) > 0 {
		for _, githubInfo := range userData.GitHub {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "github.com",
				Email:       githubInfo.EmailAddress,
				DisplayName: "GitHub",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added GitHub provider - Email: %s", githubInfo.EmailAddress)
		}
	}
	
	// Twitterプロバイダー
	if len(userData.Twitter) > 0 {
		for _, twitterInfo := range userData.Twitter {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "twitter.com",
				Email:       twitterInfo.EmailAddress,
				DisplayName: "Twitter",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added Twitter provider - Email: %s", twitterInfo.EmailAddress)
		}
	}

	log.Printf("INFO: User provider details retrieved for UID: %s, providers: %d", token.UID, len(providerDetails))
	
	result := map[string]interface{}{
		"providers": providerDetails,
	}
	
	log.Printf("DEBUG: Returning result: %+v", result)
	return result, http.StatusOK
}

// processUserProfileRequest は統合されたユーザープロフィール情報取得リクエストを処理します
func processUserProfileRequest(ctx context.Context, token *auth.Token) (map[string]interface{}, int) {
	log.Printf("DEBUG: processUserProfileRequest called for UID: %s", token.UID)
	
	// 1. データベースからユーザーデータを取得
	userData, err := getUserDataByUID(ctx, token.UID)
	if err != nil {
		log.Printf("WARN: Failed to get user data for UID %s: %v", token.UID, err)
		// ユーザーデータが存在しない場合はデフォルト値を設定
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       token.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}
	
	log.Printf("DEBUG: User data retrieved from database")
	log.Printf("DEBUG: UserName: %s, UserColor: %s", userData.UserName, userData.UserColor)
	log.Printf("DEBUG: Email providers count: %d", len(userData.Email))
	log.Printf("DEBUG: Google providers count: %d", len(userData.Google))
	log.Printf("DEBUG: GitHub providers count: %d", len(userData.GitHub))
	log.Printf("DEBUG: Twitter providers count: %d", len(userData.Twitter))

	// 2. プロバイダー一覧を抽出（データベースから）
	var providers []string
	var providerDetails []ProviderDetail
	
	// メールアドレスプロバイダー
	if len(userData.Email) > 0 {
		providers = append(providers, "password")
		for _, emailInfo := range userData.Email {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "password",
				Email:       emailInfo.EmailAddress,
				DisplayName: "メールアドレス",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added email provider - Email: %s", emailInfo.EmailAddress)
		}
	}
	
	// Googleプロバイダー
	if len(userData.Google) > 0 {
		providers = append(providers, "google.com")
		for _, googleInfo := range userData.Google {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "google.com",
				Email:       googleInfo.EmailAddress,
				DisplayName: "Google",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added Google provider - Email: %s", googleInfo.EmailAddress)
		}
	}
	
	// GitHubプロバイダー
	if len(userData.GitHub) > 0 {
		providers = append(providers, "github.com")
		for _, githubInfo := range userData.GitHub {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "github.com",
				Email:       githubInfo.EmailAddress,
				DisplayName: "GitHub",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added GitHub provider - Email: %s", githubInfo.EmailAddress)
		}
	}
	
	// Twitterプロバイダー
	if len(userData.Twitter) > 0 {
		providers = append(providers, "twitter.com")
		for _, twitterInfo := range userData.Twitter {
			providerDetails = append(providerDetails, ProviderDetail{
				Provider:    "twitter.com",
				Email:       twitterInfo.EmailAddress,
				DisplayName: "Twitter",
				IsLinked:    true,
			})
			log.Printf("DEBUG: Added Twitter provider - Email: %s", twitterInfo.EmailAddress)
		}
	}

	log.Printf("INFO: User profile retrieved for UID: %s, providers: %d", token.UID, len(providers))
	
	result := map[string]interface{}{
		"userName":        userData.UserName,
		"userColor":       userData.UserColor,
		"providers":       providers,
		"providerDetails": providerDetails,
	}
	
	log.Printf("DEBUG: Returning integrated result: %+v", result)
	return result, http.StatusOK
}

// handleUserProfileRequest は統合されたユーザープロフィール情報取得リクエストのエントリーポイントです
func handleUserProfileRequest(w http.ResponseWriter, r *http.Request) {
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
			log.Printf("DEBUG: handleUserProfileRequest - Successfully extracted UID: %s", token.UID)
			
			if r.Method == http.MethodGet {
				result, statusCode = processUserProfileRequest(r.Context(), token)
			} else {
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

// processLinkAccountRequest はアカウントリンクリクエストを処理します
func processLinkAccountRequest(ctx context.Context, req interface{}, token *auth.Token) (map[string]interface{}, int) {
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
	var linkData LinkAccountRequest
	if err := json.Unmarshal(bodyBytes, &linkData); err != nil {
		log.Printf("ERROR: Failed to decode JSON: %v\n", err)
		return map[string]interface{}{"error": "JSONの解析に失敗しました"}, http.StatusBadRequest
	}

	// バリデーション
	if linkData.Provider == "" {
		return map[string]interface{}{"error": "プロバイダーが指定されていません"}, http.StatusBadRequest
	}

	// 現在のユーザーを取得
	userRecord, err := authClient.GetUser(ctx, token.UID)
	if err != nil {
		log.Printf("ERROR: Failed to get user record for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}

	// 既にリンクされているかチェック
	for _, provider := range userRecord.ProviderUserInfo {
		if provider.ProviderID == linkData.Provider {
			return map[string]interface{}{"error": "このプロバイダーは既にリンクされています"}, http.StatusBadRequest
		}
	}

	// プロバイダーに応じたリンク処理
	switch linkData.Provider {
	case "google":
		// Googleアカウントのリンク処理
		err = linkGoogleAccount(ctx, userRecord, linkData.Credential)
	case "github":
		// GitHubアカウントのリンク処理
		err = linkGitHubAccount(ctx, userRecord, linkData.Credential)
	case "twitter":
		// Twitterアカウントのリンク処理
		err = linkTwitterAccount(ctx, userRecord, linkData.Credential)
	default:
		return map[string]interface{}{"error": "サポートされていないプロバイダーです"}, http.StatusBadRequest
	}

	if err != nil {
		log.Printf("ERROR: Failed to link account for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "アカウントのリンクに失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: Account linked successfully for UID: %s, provider: %s", token.UID, linkData.Provider)
	return map[string]interface{}{
		"success": true,
	}, http.StatusOK
}

// processUnlinkAccountRequest はアカウント解除リクエストを処理します
func processUnlinkAccountRequest(ctx context.Context, req interface{}, token *auth.Token) (map[string]interface{}, int) {
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
	var unlinkData UnlinkAccountRequest
	if err := json.Unmarshal(bodyBytes, &unlinkData); err != nil {
		log.Printf("ERROR: Failed to decode JSON: %v\n", err)
		return map[string]interface{}{"error": "JSONの解析に失敗しました"}, http.StatusBadRequest
	}

	// バリデーション
	if unlinkData.Provider == "" {
		return map[string]interface{}{"error": "プロバイダーが指定されていません"}, http.StatusBadRequest
	}

	// 現在のユーザーを取得
	userRecord, err := authClient.GetUser(ctx, token.UID)
	if err != nil {
		log.Printf("ERROR: Failed to get user record for UID %s: %v", token.UID, err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}

	// プロバイダーがリンクされているかチェック
	var foundProvider bool
	for _, provider := range userRecord.ProviderUserInfo {
		if provider.ProviderID == unlinkData.Provider {
			foundProvider = true
			break
		}
	}

	if !foundProvider {
		return map[string]interface{}{"error": "このプロバイダーはリンクされていません"}, http.StatusBadRequest
	}

	// 最後のプロバイダーを解除しようとしている場合はエラー
	if len(userRecord.ProviderUserInfo) <= 1 {
		return map[string]interface{}{"error": "最後の認証方法は解除できません"}, http.StatusBadRequest
	}

	// プロバイダーを解除（Firebase Admin SDKでは直接的なUnlinkProviderメソッドがないため、
	// ユーザー情報を更新してプロバイダーを削除する方法を使用）
	// 注意：実際の実装では、Firebase Admin SDKの適切なメソッドを使用する必要があります
	log.Printf("WARN: UnlinkProvider not implemented in current Firebase Admin SDK version")
	return map[string]interface{}{"error": "アカウント解除機能は現在実装中です"}, http.StatusNotImplemented
}

// linkGoogleAccount はGoogleアカウントをリンクします
func linkGoogleAccount(ctx context.Context, userRecord *auth.UserRecord, credential string) error {
	log.Printf("INFO: Linking Google account for user: %s", userRecord.UID)
	
	// 既存のユーザーデータを取得
	userData, err := getUserDataByUID(ctx, userRecord.UID)
	if err != nil {
		log.Printf("WARN: Failed to get user data for UID %s: %v", userRecord.UID, err)
		// ユーザーデータが存在しない場合は新規作成
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       userRecord.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}
	
	// Googleプロバイダー情報を追加（既存のユーザーデータは保持）
	addOAuthProvider(userData, "google", userRecord.Email, userRecord.UID)
	
	// ユーザーデータを保存
	if err := saveUserDataToFirestore(ctx, userRecord.UID, userData); err != nil {
		return fmt.Errorf("failed to save user data: %v", err)
	}
	
	log.Printf("INFO: Google account linked successfully for UID: %s", userRecord.UID)
	return nil
}

// linkGitHubAccount はGitHubアカウントをリンクします
func linkGitHubAccount(ctx context.Context, userRecord *auth.UserRecord, credential string) error {
	log.Printf("INFO: Linking GitHub account for user: %s", userRecord.UID)
	
	// 既存のユーザーデータを取得
	userData, err := getUserDataByUID(ctx, userRecord.UID)
	if err != nil {
		log.Printf("WARN: Failed to get user data for UID %s: %v", userRecord.UID, err)
		// ユーザーデータが存在しない場合は新規作成
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       userRecord.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}
	
	// GitHubプロバイダー情報を追加（既存のユーザーデータは保持）
	addOAuthProvider(userData, "github", userRecord.Email, userRecord.UID)
	
	// ユーザーデータを保存
	if err := saveUserDataToFirestore(ctx, userRecord.UID, userData); err != nil {
		return fmt.Errorf("failed to save user data: %v", err)
	}
	
	log.Printf("INFO: GitHub account linked successfully for UID: %s", userRecord.UID)
	return nil
}

// linkTwitterAccount はTwitterアカウントをリンクします
func linkTwitterAccount(ctx context.Context, userRecord *auth.UserRecord, credential string) error {
	log.Printf("INFO: Linking Twitter account for user: %s", userRecord.UID)
	
	// 既存のユーザーデータを取得
	userData, err := getUserDataByUID(ctx, userRecord.UID)
	if err != nil {
		log.Printf("WARN: Failed to get user data for UID %s: %v", userRecord.UID, err)
		// ユーザーデータが存在しない場合は新規作成
		userData = &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       userRecord.UID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
	}
	
	// Twitterプロバイダー情報を追加（既存のユーザーデータは保持）
	addOAuthProvider(userData, "twitter", userRecord.Email, userRecord.UID)
	
	// ユーザーデータを保存
	if err := saveUserDataToFirestore(ctx, userRecord.UID, userData); err != nil {
		return fmt.Errorf("failed to save user data: %v", err)
	}
	
	log.Printf("INFO: Twitter account linked successfully for UID: %s", userRecord.UID)
	return nil
}

// findOrCreateUserDataByEmail はメールアドレスでユーザーデータを検索または作成します
func findOrCreateUserDataByEmail(ctx context.Context, email string) (*UserData, error) {
	// Firestoreからメールアドレスでユーザーデータを検索
	usersRef := firestoreClient.Collection("users")
	
	// メールアドレスプロバイダーで検索
	query := usersRef.Where("email.emailAddress", "==", email).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found existing user data by email provider for: %s (UserName: %s, UserColor: %s)", 
			email, userData.UserName, userData.UserColor)
		return &userData, nil
	}
	
	// Googleプロバイダーで検索
	query = usersRef.Where("google.emailAddress", "==", email).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found existing user data by Google provider for: %s (UserName: %s, UserColor: %s)", 
			email, userData.UserName, userData.UserColor)
		return &userData, nil
	}
	
	// GitHubプロバイダーで検索
	query = usersRef.Where("github.emailAddress", "==", email).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found existing user data by GitHub provider for: %s (UserName: %s, UserColor: %s)", 
			email, userData.UserName, userData.UserColor)
		return &userData, nil
	}
	
	// Twitterプロバイダーで検索
	query = usersRef.Where("twitter.emailAddress", "==", email).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found existing user data by Twitter provider for: %s (UserName: %s, UserColor: %s)", 
			email, userData.UserName, userData.UserColor)
		return &userData, nil
	}

	// 既存のユーザーデータが見つからない場合はnilを返す（新しいユーザーデータは呼び出し元で作成）
	log.Printf("INFO: No existing user data found for email: %s", email)
	return nil, fmt.Errorf("no existing user data found for email: %s", email)
}

// addEmailProvider はメールアドレスプロバイダーをユーザーデータに追加します
func addEmailProvider(userData *UserData, email, uid string) {
	// 既に存在するかチェック（メールアドレスとUIDの両方でチェック）
	for _, emailInfo := range userData.Email {
		if emailInfo.EmailAddress == email || emailInfo.UserUID == uid {
			log.Printf("INFO: Email provider already exists for email: %s or UID: %s", email, uid)
			return // 既に存在する場合は何もしない
		}
	}

	// 新しいメールアドレスプロバイダーを追加
	userData.Email = append(userData.Email, EmailProviderInfo{
		EmailAddress: email,
		UserUID:      uid,
	})
	
	log.Printf("INFO: Added email provider for email: %s, UID: %s (existing UserName: %s, UserColor: %s)", 
		email, uid, userData.UserName, userData.UserColor)
}

// addOAuthProvider はOAuthプロバイダーをユーザーデータに追加します
func addOAuthProvider(userData *UserData, provider string, email, uid string) {
	var providerSlice *[]OAuthProviderInfo

	switch provider {
	case "google":
		providerSlice = &userData.Google
	case "github":
		providerSlice = &userData.GitHub
	case "twitter":
		providerSlice = &userData.Twitter
	default:
		return
	}

	// 既に存在するかチェック（メールアドレスとUIDの両方でチェック）
	for _, oauthInfo := range *providerSlice {
		if oauthInfo.EmailAddress == email || oauthInfo.UserUID == uid {
			log.Printf("INFO: OAuth provider %s already exists for email: %s or UID: %s", provider, email, uid)
			return // 既に存在する場合は何もしない
		}
	}

	// 新しいOAuthプロバイダーを追加
	*providerSlice = append(*providerSlice, OAuthProviderInfo{
		UserUID:      uid,
		EmailAddress: email,
	})
	
	log.Printf("INFO: Added OAuth provider %s for email: %s, UID: %s (existing UserName: %s, UserColor: %s)", 
		provider, email, uid, userData.UserName, userData.UserColor)
}

// getUserDataByUID はUIDでユーザーデータを取得します
func getUserDataByUID(ctx context.Context, uid string) (*UserData, error) {
	log.Printf("DEBUG: getUserDataByUID called for UID: %s", uid)
	
	// まず、UIDで直接検索
	usersRef := firestoreClient.Collection("users")
	doc, err := usersRef.Doc(uid).Get(ctx)
	if err == nil {
		var userData UserData
		if err := doc.DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found user data by UID for: %s (UserName: %s, UserColor: %s)", 
			uid, userData.UserName, userData.UserColor)
		log.Printf("DEBUG: GitHub providers count: %d", len(userData.GitHub))
		if len(userData.GitHub) > 0 {
			log.Printf("DEBUG: First GitHub provider: %+v", userData.GitHub[0])
		}
		return &userData, nil
	}
	
	log.Printf("INFO: User data not found by UID: %s, searching by provider UID", uid)

	// UIDで見つからない場合、各プロバイダーで検索
	// Googleプロバイダーで検索
	query := usersRef.Where("google.userUID", "==", uid).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found user data by Google provider UID for: %s (UserName: %s, UserColor: %s)", 
			uid, userData.UserName, userData.UserColor)
		return &userData, nil
	}

	// GitHubプロバイダーで検索
	query = usersRef.Where("github.userUID", "==", uid).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found user data by GitHub provider UID for: %s (UserName: %s, UserColor: %s)", 
			uid, userData.UserName, userData.UserColor)
		return &userData, nil
	}

	// Twitterプロバイダーで検索
	query = usersRef.Where("twitter.userUID", "==", uid).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found user data by Twitter provider UID for: %s (UserName: %s, UserColor: %s)", 
			uid, userData.UserName, userData.UserColor)
		return &userData, nil
	}

	// メールアドレスプロバイダーで検索
	query = usersRef.Where("email.userUID", "==", uid).Limit(1)
	docs, err = query.Documents(ctx).GetAll()
	if err == nil && len(docs) > 0 {
		var userData UserData
		if err := docs[0].DataTo(&userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %v", err)
		}
		log.Printf("INFO: Found user data by email provider UID for: %s (UserName: %s, UserColor: %s)", 
			uid, userData.UserName, userData.UserColor)
		return &userData, nil
	}

	// ユーザーデータが見つからない場合はデフォルト値を返す
	log.Printf("INFO: User data not found for UID: %s, returning default values", uid)
	defaultUserData := &UserData{
		UserName:  "",
		UserColor: "#3b82f6",
		UID:       uid,
		Email:     []EmailProviderInfo{},
		Google:    []OAuthProviderInfo{},
		GitHub:    []OAuthProviderInfo{},
		Twitter:   []OAuthProviderInfo{},
	}
	return defaultUserData, nil
}

// lambdaUserProfileHandler はLambda環境での統合されたユーザープロフィール情報取得リクエストを処理します
func lambdaUserProfileHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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
			if request.RequestContext.HTTP.Method == http.MethodGet {
				result, statusCode = processUserProfileRequest(ctx, token)
			} else {
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


