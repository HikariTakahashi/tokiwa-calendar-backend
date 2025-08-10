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

// TwitterAuthRequest はTwitter OAuth2.0認証リクエストの構造体です
type TwitterAuthRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	LinkUID     string `json:"linkUID,omitempty"` // アカウントリンク時のUID
}

// TwitterAuthResponse はTwitter OAuth2.0認証レスポンスの構造体です
type TwitterAuthResponse struct {
	Message     string `json:"message"`
	UID         string `json:"uid,omitempty"`
	Email       string `json:"email,omitempty"`
	CustomToken string `json:"customToken,omitempty"`
	Error       string `json:"error,omitempty"`
}

// TwitterTokenResponse はTwitter OAuth2.0トークンレスポンスの構造体です
type TwitterTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// TwitterUserInfo はTwitterユーザー情報の構造体です
type TwitterUserInfo struct {
	ID              string `json:"id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	ProfileImageURL string `json:"profile_image_url"`
	Verified        bool   `json:"verified"`
	Protected       bool   `json:"protected"`
	CreatedAt       string `json:"created_at"`
}

// getTwitterClientID は環境変数からTwitter Client IDを取得します
func getTwitterClientID() string {
	return os.Getenv("TWITTER_CLIENT_ID")
}

// getTwitterClientSecret は環境変数からTwitter Client Secretを取得します
func getTwitterClientSecret() string {
	return os.Getenv("TWITTER_CLIENT_SECRET")
}

// exchangeTwitterCodeForToken は認証コードをアクセストークンと交換します
func exchangeTwitterCodeForToken(code, redirectURI string) (*TwitterTokenResponse, error) {
	clientID := getTwitterClientID()
	clientSecret := getTwitterClientSecret()

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Twitter Client ID or Secret not configured")
	}

	// Twitter OAuth2.0トークンエンドポイント
	tokenURL := "https://api.twitter.com/2/oauth2/token"

	// リクエストボディの準備
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code_verifier", "challenge") // PKCEの実装が必要

	// HTTPリクエストの作成
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Basic認証ヘッダーの設定
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// リクエストの実行
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Twitter token exchange failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	// JSONレスポンスの解析
	var tokenResponse TwitterTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v", err)
	}

	return &tokenResponse, nil
}

// getUserInfoFromTwitter はTwitterからユーザー情報を取得します
func getUserInfoFromTwitter(accessToken string) (*TwitterUserInfo, error) {
	// Twitter API v2のユーザー情報エンドポイント
	userInfoURL := "https://api.twitter.com/2/users/me?user.fields=id,username,name,profile_image_url,verified,protected,created_at"

	// HTTPリクエストの作成
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 認証ヘッダーの設定
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	// リクエストの実行
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Twitter user info request failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	// Twitter API v2のレスポンス形式
	var twitterResponse struct {
		Data TwitterUserInfo `json:"data"`
	}

	if err := json.Unmarshal(body, &twitterResponse); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %v", err)
	}

	// メールアドレスを取得するための追加リクエスト
	emailURL := "https://api.twitter.com/2/users/me?user.fields=id,username,email"
	emailReq, err := http.NewRequest("GET", emailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create email request: %v", err)
	}

	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailReq.Header.Set("Accept", "application/json")

	emailResp, err := client.Do(emailReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute email request: %v", err)
	}
	defer emailResp.Body.Close()

	if emailResp.StatusCode == http.StatusOK {
		emailBody, err := io.ReadAll(emailResp.Body)
		if err == nil {
			var emailResponse struct {
				Data struct {
					Email string `json:"email"`
				} `json:"data"`
			}
			if json.Unmarshal(emailBody, &emailResponse) == nil {
				twitterResponse.Data.Email = emailResponse.Data.Email
			}
		}
	}

	return &twitterResponse.Data, nil
}

// createOrGetFirebaseUserForTwitter はTwitterユーザー情報に基づいてFirebaseユーザーを作成または取得します
func createOrGetFirebaseUserForTwitter(ctx context.Context, userInfo *TwitterUserInfo, linkUID string) (string, error) {
	// メールアドレスが取得できない場合はエラー
	if userInfo.Email == "" {
		return "", fmt.Errorf("Twitterアカウントからメールアドレスを取得できませんでした")
	}

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
		
		// Twitterプロバイダー情報を追加（既存のユーザーデータは保持）
		addOAuthProvider(userData, "twitter", userInfo.Email, linkUID)
		
		// ユーザーデータを保存（既存のユーザー名、カラーは保持される）
		log.Printf("INFO: About to save user data for UID: %s (UserName: %s, UserColor: %s)", 
			linkUID, userData.UserName, userData.UserColor)
		if err := saveUserDataToFirestore(ctx, linkUID, userData); err != nil {
			log.Printf("WARN: Failed to save user data: %v", err)
		}
		
		return linkUID, nil
	}
	
	// 新規ログイン時は、メールアドレスで既存のユーザーを確認
	userRecord, err := authClient.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		// ユーザーが存在しない場合は新しいTwitterユーザーを作成
		twitterUID := "twitter:" + userInfo.ID
		params := (&auth.UserToCreate{}).
			UID(twitterUID).
			DisplayName(userInfo.Name).
			PhotoURL(userInfo.ProfileImageURL).
			Email(userInfo.Email).
			EmailVerified(true)

		userRecord, err = authClient.CreateUser(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to create Firebase user: %v", err)
		}

		log.Printf("INFO: New Twitter user created: %s", userRecord.UID)
		
		// 新しいユーザーデータを作成してFirestoreに保存
		userData := &UserData{
			UserName:  "",
			UserColor: "#3b82f6",
			UID:       twitterUID,
			Email:     []EmailProviderInfo{},
			Google:    []OAuthProviderInfo{},
			GitHub:    []OAuthProviderInfo{},
			Twitter:   []OAuthProviderInfo{},
		}
		addOAuthProvider(userData, "twitter", userInfo.Email, twitterUID)
		
		if err := saveUserDataToFirestore(ctx, twitterUID, userData); err != nil {
			log.Printf("WARN: Failed to save user data for new Twitter user: %v", err)
		}
	} else {
		// 既存のユーザーが見つかった場合
		log.Printf("INFO: Found existing Firebase user for email: %s", userInfo.Email)
		
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
		
		// Twitterプロバイダー情報を追加（既存のユーザーデータは保持）
		addOAuthProvider(userData, "twitter", userInfo.Email, userRecord.UID)
		
		// ユーザーデータを保存（既存のユーザー名、カラーは保持される）
		log.Printf("INFO: About to save user data for UID: %s (UserName: %s, UserColor: %s)", 
			userRecord.UID, userData.UserName, userData.UserColor)
		if err := saveUserDataToFirestore(ctx, userRecord.UID, userData); err != nil {
			log.Printf("WARN: Failed to save user data: %v", err)
		}
	}

	return userRecord.UID, nil
}

func processTwitterAuthRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
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
	var authData TwitterAuthRequest
	if err := json.Unmarshal(bodyBytes, &authData); err != nil {
		log.Printf("WARN: Failed to parse Twitter auth JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// バリデーション
	if authData.Code == "" {
		return map[string]interface{}{"error": "認証コードが提供されていません"}, http.StatusBadRequest
	}
	if authData.RedirectURI == "" {
		return map[string]interface{}{"error": "リダイレクトURIが提供されていません"}, http.StatusBadRequest
	}

	log.Printf("INFO: Twitter OAuth2.0 request received with code length: %d, linkUID: %s", len(authData.Code), authData.LinkUID)

	// 認証コードをアクセストークンと交換
	tokenResponse, err := exchangeTwitterCodeForToken(authData.Code, authData.RedirectURI)
	if err != nil {
		log.Printf("ERROR: Failed to exchange code for token: %v\n", err)
		return map[string]interface{}{"error": "認証コードの交換に失敗しました"}, http.StatusBadRequest
	}

	// Twitterからユーザー情報を取得
	userInfo, err := getUserInfoFromTwitter(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("ERROR: Failed to get user info from Twitter: %v\n", err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}

	// メールアドレスの検証
	if userInfo.Email == "" {
		return map[string]interface{}{"error": "メールアドレスが取得できませんでした"}, http.StatusBadRequest
	}

	log.Printf("INFO: Twitter user info retrieved for email: %s", userInfo.Email)

	// Firebaseユーザーを作成または取得
	uid, err := createOrGetFirebaseUserForTwitter(ctx, userInfo, authData.LinkUID)
	if err != nil {
		log.Printf("ERROR: Failed to create/get Firebase user: %v\n", err)
		// アカウントリンクが必要な場合のエラーメッセージ
		if strings.Contains(err.Error(), "already in use") {
			return map[string]interface{}{"error": "このTwitterアカウントは既に他のアカウントで使用されています"}, http.StatusConflict
		}
		return map[string]interface{}{"error": "ユーザーアカウントの作成に失敗しました"}, http.StatusInternalServerError
	}

	// セッショントークンを生成
	sessionToken, err := generateSessionToken(uid, userInfo.Email)
	if err != nil {
		log.Printf("ERROR: Failed to generate session token for UID %s: %v\n", uid, err)
		return map[string]interface{}{"error": "セッショントークンの生成に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: Twitter OAuth2.0 authentication successful for UID: %s", uid)

	return map[string]interface{}{
		"message":      "Twitterアカウントでのログインが成功しました",
		"uid":          uid,
		"email":        userInfo.Email,
		"sessionToken": sessionToken,
	}, http.StatusOK
} 