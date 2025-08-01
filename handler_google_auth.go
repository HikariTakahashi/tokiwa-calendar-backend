package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// GoogleAuthRequest はGoogle OAuth2.0認証リクエストの構造体です
type GoogleAuthRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// GoogleAuthResponse はGoogle OAuth2.0認証レスポンスの構造体です
type GoogleAuthResponse struct {
	Message     string `json:"message"`
	UID         string `json:"uid,omitempty"`
	Email       string `json:"email,omitempty"`
	CustomToken string `json:"customToken,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GoogleTokenResponse はGoogle OAuth2.0トークンレスポンスの構造体です
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
}

// GoogleUserInfo はGoogleユーザー情報の構造体です
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// getGoogleClientID は環境変数からGoogle Client IDを取得します
func getGoogleClientID() string {
	return os.Getenv("GOOGLE_CLIENT_ID")
}

// getGoogleClientSecret は環境変数からGoogle Client Secretを取得します
func getGoogleClientSecret() string {
	return os.Getenv("GOOGLE_CLIENT_SECRET")
}

// exchangeCodeForToken は認証コードをアクセストークンと交換します
func exchangeCodeForToken(code, redirectURI string) (*GoogleTokenResponse, error) {
	clientID := getGoogleClientID()
	clientSecret := getGoogleClientSecret()

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Google OAuth2.0設定が不完全です")
	}

	// トークンエンドポイントにリクエスト
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("トークン交換リクエストエラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("トークン交換失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("トークンレスポンス解析エラー: %v", err)
	}

	return &tokenResponse, nil
}

// getUserInfoFromGoogle はGoogleからユーザー情報を取得します
func getUserInfoFromGoogle(accessToken string) (*GoogleUserInfo, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報リクエスト作成エラー: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報取得エラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ユーザー情報取得失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("ユーザー情報解析エラー: %v", err)
	}

	return &userInfo, nil
}

// createOrGetFirebaseUser はGoogleアカウントでFirebaseユーザーを作成または取得します
func createOrGetFirebaseUser(ctx context.Context, googleUser *GoogleUserInfo) (string, error) {
	// 既存のユーザーを確認
	userRecord, err := authClient.GetUserByEmail(ctx, googleUser.Email)
	if err != nil {
		// ユーザーが存在しない場合は新しいGoogleユーザーを作成
		uid := "google_" + googleUser.ID
		params := (&auth.UserToCreate{}).
			Email(googleUser.Email).
			DisplayName(googleUser.Name).
			PhotoURL(googleUser.Picture).
			EmailVerified(googleUser.VerifiedEmail).
			UID(uid)

		userRecord, err = authClient.CreateUser(ctx, params)
		if err != nil {
			return "", fmt.Errorf("Firebaseユーザー作成エラー: %v", err)
		}
		log.Printf("INFO: Created new Firebase user for Google account: %s", googleUser.Email)
	} else {
		// 既存のユーザーが見つかった場合
		log.Printf("INFO: Found existing Firebase user for email: %s", googleUser.Email)
		
		// 既存のユーザーがGoogleプロバイダーで作成されているかチェック
		providers := userRecord.ProviderUserInfo
		hasGoogleProvider := false
		
		for _, provider := range providers {
			if provider.ProviderID == "google.com" {
				hasGoogleProvider = true
				break
			}
		}
		
		if !hasGoogleProvider {
			// 既存のユーザーがメールアドレスログインで作成されている場合
			// この場合、既存のユーザーアカウントを使用する
			// （Googleアカウントとのリンクは後でクライアントサイドで行う）
			log.Printf("INFO: Using existing email user for Google login: %s", googleUser.Email)
			
			// 既存ユーザーの情報を更新（表示名やプロフィール画像など）
			updateParams := (&auth.UserToUpdate{}).
				DisplayName(googleUser.Name).
				PhotoURL(googleUser.Picture).
				EmailVerified(true)
			
			_, err = authClient.UpdateUser(ctx, userRecord.UID, updateParams)
			if err != nil {
				log.Printf("WARN: Failed to update user profile: %v", err)
				// 更新に失敗してもログインは続行
			}
		}
	}

	return userRecord.UID, nil
}

func processGoogleAuthRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
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
	var authData GoogleAuthRequest
	if err := json.Unmarshal(bodyBytes, &authData); err != nil {
		log.Printf("WARN: Failed to parse Google auth JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// バリデーション
	if authData.Code == "" {
		return map[string]interface{}{"error": "認証コードが提供されていません"}, http.StatusBadRequest
	}
	if authData.RedirectURI == "" {
		return map[string]interface{}{"error": "リダイレクトURIが提供されていません"}, http.StatusBadRequest
	}

	log.Printf("INFO: Google OAuth2.0 request received with code length: %d", len(authData.Code))

	// 認証コードをアクセストークンと交換
	tokenResponse, err := exchangeCodeForToken(authData.Code, authData.RedirectURI)
	if err != nil {
		log.Printf("ERROR: Failed to exchange code for token: %v\n", err)
		return map[string]interface{}{"error": "認証コードの交換に失敗しました"}, http.StatusBadRequest
	}

	// Googleからユーザー情報を取得
	userInfo, err := getUserInfoFromGoogle(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("ERROR: Failed to get user info from Google: %v\n", err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}

	// メールアドレスの検証
	if !userInfo.VerifiedEmail {
		return map[string]interface{}{"error": "メールアドレスが認証されていません"}, http.StatusBadRequest
	}

	log.Printf("INFO: Google user info retrieved for email: %s", userInfo.Email)

	// Firebaseユーザーを作成または取得
	uid, err := createOrGetFirebaseUser(ctx, userInfo)
	if err != nil {
		log.Printf("ERROR: Failed to create/get Firebase user: %v\n", err)
		return map[string]interface{}{"error": "ユーザーアカウントの作成に失敗しました"}, http.StatusInternalServerError
	}

	// カスタムトークンを生成
	customToken, err := authClient.CustomToken(ctx, uid)
	if err != nil {
		log.Printf("ERROR: Failed to create custom token for UID %s: %v\n", uid, err)
		return map[string]interface{}{"error": "認証トークンの生成に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: Google OAuth2.0 authentication successful for UID: %s", uid)

	return map[string]interface{}{
		"message":     "Googleアカウントでのログインが成功しました",
		"uid":         uid,
		"email":       userInfo.Email,
		"customToken": customToken,
	}, http.StatusOK
} 