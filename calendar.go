package main

import "time"

// 各日のデータ構造
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Week string `json:"week"`  // "Mon", "Tue", など
}

// 月を1ヶ月進めたり戻したりする
func adjustDate(baseYear int, baseMonth int, moveStr string)(int, int){
   // 現在の年月を「1日」で作る（AddDateでズレないように）
	baseDate := time.Date(baseYear, time.Month(baseMonth), 1, 0, 0, 0, 0, time.UTC)
	switch moveStr {
	case "next":
      baseDate = baseDate.AddDate(0, 1, 0) // 1ヶ月進める
	case "prev":
  	  baseDate = baseDate.AddDate(0, -1, 0) // 1ヶ月戻す
	}
    return baseDate.Year(), int(baseDate.Month())
}

// 月末日を取得
func getEndOfMonth(year int, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// 1日分のデータを生成
func generateDay(baseYear int, baseMonth int, day int) Day {
	date := time.Date(baseYear, time.Month(baseMonth), day, 0, 0, 0, 0, time.UTC)
	week := date.Weekday().String()[:3] //曜日を3文字に

	return Day{
		Date: date.Format("2006-01-02"),  // Go特有の日時フォーマット
		Week: week,
	}
}

// 日付データを全体として生成（前月・当月・翌月も含む）
func generateDays(year int, month int, endOfMonth int) []Day {
	var days []Day

	days = append(days, getPrevMonthDays(year, month)...)   // 前月の追加（調整用）
	days = append(days, getMonthDays(year, month, endOfMonth)...) // 当月の日付
	days = append(days, getNextMonthDays(year, month, endOfMonth)...) // 翌月の追加（調整用）

	return days
}

// 前月の末日から、必要な日数分だけ取得（日曜スタート調整）
func getPrevMonthDays(year int, month int) []Day {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	weekday := int(firstDay.Weekday())
	if weekday == 0 {
		return nil  // すでに日曜始まりなら不要
	}

	prevYear, prevMonth := adjustDate(year, month, "prev")
	prevEnd := getEndOfMonth(prevYear, prevMonth)

	var result []Day
	for i := weekday - 1; i >= 0; i-- {
		day := prevEnd - i
		result = append(result, generateDay(prevYear, prevMonth, day))
	}
	return result
}

// 当月の1日〜月末までの日付を生成
func getMonthDays(year int, month int, endOfMonth int) []Day {
	var result []Day
	for i := 1; i <= endOfMonth; i++ {
		result = append(result, generateDay(year, month, i))
	}
	return result
}

// 月末が土曜でない場合、翌月の先頭日から土曜まで追加（土曜終わり調整）
func getNextMonthDays(year int, month int, endOfMonth int) []Day {
	lastDay := time.Date(year, time.Month(month), endOfMonth, 0, 0, 0, 0, time.UTC)
	weekday := int(lastDay.Weekday())
	if weekday == 6 {
		return nil
	}

	nextYear, nextMonth := adjustDate(year, month, "next")
	var result []Day
	for i := 1; weekday < 6; i++ {
		weekday++
		result = append(result, generateDay(nextYear, nextMonth, i))
	}
	return result
}