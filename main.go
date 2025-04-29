package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// 各日のデータ構造
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Day  int    `json:"day"`  // 数値の「日」
}

// レスポンス全体の構造
type CalendarResponse struct {
	Year  int   `json:"year"`  // 対象の年
	Month int   `json:"month"` // 対象の月
	Days  []Day `json:"days"`  // 各日の配列
}

// APIのエンドポイントのハンドラー関数
// w：サーバーからブラウザへ返事（レスポンス）を書くためのもの
// r：ブラウザからサーバーへ送られてきたリクエスト情報を持っているもの
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	// CORS：TypeScript（ブラウザ）からリクエストを送れるようにする許可、"*"はall許可
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// レスポンスのデータ形式の宣言 → JSONで送るよ
	w.Header().Set("Content-Type", "application/json")

	// クエリパラメータから年・月・移動方向を取得
	// r.URL → リクエストされたURLを取り出して
	// .Query() → URLにくっついているクエリパラメータを取り出して
	// .Get("year") → 「year」という名前のデータを取り出している！
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	moveStr := r.URL.Query().Get("move")

	// カレンダーの基準になる年・月
	var baseYear int
	var baseMonth int

	// そのうち使うかも
	// URLに年・月が指定されている場合 → その年・月をカレンダーに表示
	// 指定されていない → 今現在の年・月をカレンダーに表示する
	if yearStr != "" && monthStr != "" {
		baseYear, _ = strconv.Atoi(yearStr)
		baseMonth, _ = strconv.Atoi(monthStr)
	} else {
		now := time.Now()
		baseYear = now.Year()
		baseMonth = int(now.Month())
	}

	// "next" または "prev" の指示があれば月を移動
	switch moveStr {
	case "next":
		if baseMonth == 12 {
			baseMonth = 1
			baseYear++
		} else {
			baseMonth++
		}
	case "prev":
		if baseMonth == 1 {
			baseMonth = 12
			baseYear--
		} else {
			baseMonth--
		}
	}

	// 指定された年月の月末日が何日かを調べる
	// 次の月の「0日目」を指定する → 前の月の月末日になる
	// time.Date(~~)：指定した年月日・時間を表す日時オブジェクトの作成
	// その日時オブジェクトに対して、その日の日付だけ取り出せる
	daysInMonth := time.Date(baseYear, time.Month(baseMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	// Dayという構造体のからのスライスの作成：一日分の情報を持つデータを入れる箱
	var days []Day
	// 1日から月末までループし、1日ずつの日付データを作成する
	for i := 1; i <= daysInMonth; i++ {
		date := time.Date(baseYear, time.Month(baseMonth), i, 0, 0, 0, 0, time.UTC)
		// 作った日付(date)を、Day型のデータに変換してスライス(days)に追加する
		days = append(days, Day{
			Date: date.Format("2006-01-02"), // Go独特の日時フォーマット
			Day:  i,
		})
	}

	// レスポンス用の構造体を生成
	resp := CalendarResponse{
		Year:  baseYear,
		Month: baseMonth,
		Days:  days,
	}

	// resp(データ) → Encode(変換) → レスポンス出力
	// JSONにエンコードし、TypeScriptへレスポンスする
	w.Header().Set("Content-Type", "application/json")
	// resp（送るデータ）をJSONに変換して、そのままwに書き込む
	json.NewEncoder(w).Encode(resp)
}

// サーバーのエントリーポイント
func main() {
	// api/calendar というURLに来たリクエストは、calendarHandler という関数で処理する設定
	http.HandleFunc("/api/calendar", calendarHandler) // エンドポイントを登録
	fmt.Println("Listening on :8080")                 // 起動ログ
	// ポート8080番でリクエストを待つサーバーを動かす（リクエストが来たらHandleFuncが動く）
	http.ListenAndServe(":8080", nil)                 // ポート8080でリクエストを待機
}
