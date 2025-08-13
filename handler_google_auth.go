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
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// GoogleAuthRequest はGoogle OAuth2.0認証リクエストの構造体です
type GoogleAuthRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	LinkUID     string `json:"linkUID,omitempty"` // アカウントリンク時のUID
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
func createOrGetFirebaseUser(ctx context.Context, googleUser *GoogleUserInfo, linkUID string) (string, error) {
	// アカウントリンク時は、指定されたUIDを使用
	if linkUID != "" {
		log.Printf("INFO: Account linking mode for UID: %s", linkUID)
		
		// 指定されたUIDのユーザーが存在するか確認
		_, err := authClient.GetUser(ctx, linkUID)
		if err != nil {
			return "", fmt.Errorf("リンク先のユーザーが見つかりません: %v", err)
		}
		
		// 既存のユーザーデータを取得
		userData, err := getUserDataByUID(ctx, linkUID)
		if err != nil {
			log.Printf("INFO: No existing user data found for UID %s, creating new user data", linkUID)
			// 既存のユーザーデータが見つからない場合は新しいユーザーデータを作成
			userData = &UserData{
				UserName:  "", // 新規ユーザーの場合は空
				UserColor: "#3b82f6", // 新規ユーザーの場合はデフォルトカラー
				UID:       linkUID,
				Email:     []EmailProviderInfo{},
				Google:    []OAuthProviderInfo{},
				GitHub:    []OAuthProviderInfo{},
				Twitter:   []OAuthProviderInfo{},
			}
		} else {
			log.Printf("INFO: Found existing user data for UID %s (UserName: %s, UserColor: %s)", 
				linkUID, userData.UserName, userData.UserColor)
		}
		
		// Googleプロバイダー情報を追加（既存のユーザーデータは保持）
		addOAuthProvider(userData, "google", googleUser.Email, linkUID)
		
		// ユーザーデータを保存（既存のユーザー名、カラーは保持される）
		log.Printf("INFO: About to save user data for UID: %s (UserName: %s, UserColor: %s)", 
			linkUID, userData.UserName, userData.UserColor)
		if err := saveUserDataToFirestore(ctx, linkUID, userData); err != nil {
			log.Printf("WARN: Failed to save user data: %v", err)
		}
		
		return linkUID, nil
	}
	
	// 新規ログイン時は、メールアドレスで既存のユーザーを確認
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
		
		// 新しいユーザーデータを作成してFirestoreに保存
		userData := &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       uid,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
		addOAuthProvider(userData, "google", googleUser.Email, uid)
		
		if err := saveUserDataToFirestore(ctx, uid, userData); err != nil {
			log.Printf("WARN: Failed to save user data for new Google user: %v", err)
		}
	} else {
		// 既存のユーザーが見つかった場合
		log.Printf("INFO: Found existing Firebase user for email: %s", googleUser.Email)
		
		// 既存のユーザーデータを取得（UIDベースで検索）
		userData, err := getUserDataByUID(ctx, userRecord.UID)
		if err != nil {
			log.Printf("INFO: No existing user data found for UID %s, creating new user data", userRecord.UID)
			// 既存のユーザーデータが見つからない場合は新しいユーザーデータを作成
			userData = &UserData{
				UserName:  "", // 新規ユーザーの場合は空
				UserColor: "#3b82f6", // 新規ユーザーの場合はデフォルトカラー
				UID:       userRecord.UID,
				Email:     []EmailProviderInfo{},
				Google:    []OAuthProviderInfo{},
				GitHub:    []OAuthProviderInfo{},
				Twitter:   []OAuthProviderInfo{},
			}
		} else {
			log.Printf("INFO: Found existing user data for UID %s (UserName: %s, UserColor: %s)", 
				userRecord.UID, userData.UserName, userData.UserColor)
		}
		
		// UIDを既存のユーザーレコードのUIDに統一
		userData.UID = userRecord.UID
		
		// Googleプロバイダー情報を追加（既存のユーザーデータは保持）
		addOAuthProvider(userData, "google", googleUser.Email, userRecord.UID)
		
		// ユーザーデータを保存（既存のユーザー名、カラーは保持される）
		log.Printf("INFO: About to save user data for UID: %s (UserName: %s, UserColor: %s)", 
			userRecord.UID, userData.UserName, userData.UserColor)
		if err := saveUserDataToFirestore(ctx, userRecord.UID, userData); err != nil {
			log.Printf("WARN: Failed to save user data: %v", err)
		}
		
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

	log.Printf("INFO: Google OAuth2.0 request received with code length: %d, linkUID: %s", len(authData.Code), authData.LinkUID)

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
	uid, err := createOrGetFirebaseUser(ctx, userInfo, authData.LinkUID)
	if err != nil {
		log.Printf("ERROR: Failed to create/get Firebase user: %v\n", err)
		// アカウントリンクが必要な場合のエラーメッセージ
		if strings.Contains(err.Error(), "already in use") {
			return map[string]interface{}{"error": "このGoogleアカウントは既に他のアカウントで使用されています"}, http.StatusConflict
		}
		return map[string]interface{}{"error": "ユーザーアカウントの作成に失敗しました"}, http.StatusInternalServerError
	}

	// セッショントークンを生成
	sessionToken, err := generateSessionToken(uid, userInfo.Email)
	if err != nil {
		log.Printf("ERROR: Failed to generate session token for UID %s: %v\n", uid, err)
		return map[string]interface{}{"error": "セッショントークンの生成に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: Google OAuth2.0 authentication successful for UID: %s", uid)

	return map[string]interface{}{
		"message":      "Googleアカウントでのログインが成功しました",
		"uid":          uid,
		"email":        userInfo.Email,
		"sessionToken": sessionToken,
	}, http.StatusOK
} 