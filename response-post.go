package main

import (
	"context" // Firestore操作に必要
	"encoding/json"
	"fmt"
	"net/http"
	// "time" // 日付や時刻のバリデーションを行う場合に必要
)

// Firestoreクライアントのグローバル変数 (mainなどで初期化される想定)
// var client *firestore.Client // この行はmain.goで定義されているため、ここでは不要です

// 各日付の開始時刻と終了時刻を格納する構造体
type TimeEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// /api/time にPOSTされた時の処理
func postHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POSTリクエストを受信しました")

	// OPTIONSリクエストの場合はCORSプリフライトとして処理
	if r.Method == http.MethodOptions {
		setCORS(w) // utils.go の setCORS 関数が呼び出される
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTメソッド以外はエラー
	if r.Method != http.MethodPost {
		http.Error(w, "許可されていないメソッドです (POSTのみ許可)", http.StatusMethodNotAllowed)
		return
	}

	// フロントエンドからのアクセスを許可（CORS対応）
	setCORS(w) // utils.go の setCORS 関数が呼び出される

	// リクエストボディを汎用的なマップとしてデコード
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "JSONの解析に失敗しました: "+err.Error(), http.StatusBadRequest)
		fmt.Println("デコードエラー:", err)
		return
	}
	defer r.Body.Close()

	fmt.Printf("受信した生データ: %+v\n", requestData)

	// "spaceId" を抽出
	spaceIdInterface, ok := requestData["spaceId"]
	if !ok {
		http.Error(w, "'spaceId' がリクエストデータに含まれていません", http.StatusBadRequest)
		fmt.Println("エラー: 'spaceId' が見つかりません")
		return
	}
	spaceId, ok := spaceIdInterface.(string)
	if !ok || spaceId == "" {
		http.Error(w, "'spaceId' は空でない文字列である必要があります", http.StatusBadRequest)
		fmt.Println("エラー: 'spaceId' が文字列でないか、空です")
		return
	}

	// 日付データ（TimeEntry）を格納するマップ作成
	eventsToStore := make(map[string]TimeEntry)

	// requestDataから情報を取り出す
	for key, value := range requestData {
		// "spaceId" は日付データではないのでスキップする
		if key == "spaceId" {
			continue
		}

		// value がマップ形式（{"start": "...", "end": "..."}のような形）であることを確認
		eventMap, ok := value.(map[string]interface{})
		if !ok {
			// 想定外のデータ形式の場合、エラーレスポンスを返し、処理を中断
			errMsg := fmt.Sprintf("キー '%s' のデータ形式が無効です。オブジェクト形式である必要があります。", key)
			http.Error(w, errMsg, http.StatusBadRequest)
			fmt.Printf("エラー: キー '%s' の値が予期したマップ形式ではありません: %+v\n", key, value)
			return
		}

		// eventMapから "start" と "end" を文字列として抽出
		startStrInterface, startExists := eventMap["start"]
		endStrInterface, endExists := eventMap["end"]

		if !startExists || !endExists {
			errMsg := fmt.Sprintf("キー '%s' に 'start' または 'end' が含まれていません。", key)
			http.Error(w, errMsg, http.StatusBadRequest)
			fmt.Printf("エラー: キー '%s' のデータに 'start' または 'end' がありません: %+v\n", key, eventMap)
			return
		}

		startStr, startOk := startStrInterface.(string)
		endStr, endOk := endStrInterface.(string)

		if !startOk || !endOk {
			// "start" または "end" が文字列でない場合、エラーレスポンスを返し、処理を中断
			errMsg := fmt.Sprintf("キー '%s' の 'start' または 'end' の形式が無効です。文字列である必要があります。", key)
			http.Error(w, errMsg, http.StatusBadRequest)
			fmt.Printf("エラー: キー '%s' の 'start' または 'end' が文字列ではありません: %+v\n", key, eventMap)
			return
		}

		// TODO: 必要であれば日付キー(key)や時刻文字列(startStr, endStr)のバリデーションを追加
		// 例: _, err := time.Parse("2006-01-02", key); if err != nil { ... }
		// 例: _, err := time.Parse("15:04", startStr); if err != nil { ... }

		// 取り出した情報を整理し、eventsToStoreマップに格納
		eventsToStore[key] = TimeEntry{Start: startStr, End: endStr}
	}

	fmt.Printf("抽出した spaceId: %s\n", spaceId)
	fmt.Printf("Firestoreに保存するイベントデータ: %+v\n", eventsToStore)

	// Firestoreクライアントの確認 (main.goなどで初期化済みのはず)
	if client == nil { // client はグローバル変数または適切に渡された Firestore クライアントインスタンス
		http.Error(w, "Firestoreクライアントが初期化されていません", http.StatusInternalServerError)
		fmt.Println("重大なエラー: Firestoreクライアント (global 'client') がnilです")
		return
	}

	// もし、eventsToStore が空の場合（つまり、spaceId以外の有効な日付データがなかった場合）の処理
	if len(eventsToStore) == 0 {
		fmt.Printf("spaceId '%s' に関連する有効なカレンダーイベントが見つかりませんでした。Firestoreへの保存は行いません。\n", spaceId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // または http.StatusBadRequest など状況に応じて
		json.NewEncoder(w).Encode(map[string]string{
			"message": "有効なカレンダーイベントデータが見つかりませんでした。",
			"spaceId": spaceId,
		})
		return
	}

	docRef := client.Collection("schedules").Doc(spaceId) // "schedules" は任意のコレクション名
	_, err := docRef.Set(context.Background(), eventsToStore)
	if err != nil {
		http.Error(w, "Firestoreへのデータ保存に失敗しました: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("Firestore Setエラー (spaceId: %s, Collection: schedules): %v\n", spaceId, err)
		return
	}

	fmt.Printf("データがFirestoreに正常に保存されました。Collection: schedules, Document ID: %s\n", spaceId)

	// 成功レスポンス
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":     "データは正常に受信され、Firestoreに保存されました。",
		"spaceId":     spaceId,
		"savedEvents": eventsToStore, // 保存したイベントデータをレスポンスに含める
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("レスポンスのJSONエンコードに失敗しました:", err)
	}
}