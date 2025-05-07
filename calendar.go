package main

import "time"

// 各日のデータ構造
type Day struct {
	Date string `json:"date"` // "YYYY-MM-DD" 形式の日付
	Week string `json:"week"`  // "Mon", "Tue", など
}

func adjustDate(baseYear int, baseMonth int, moveStr string)(int, int){
   // 現在の年月を「1日」で作る（AddDateでズレないように）
	baseDate := time.Date(baseYear, time.Month(baseMonth), 1, 0, 0, 0, 0, time.UTC)

	// 月移動処理
	switch moveStr {
	case "next":
      baseDate = baseDate.AddDate(0, 1, 0) // 1ヶ月進める
	case "prev":
  	  baseDate = baseDate.AddDate(0, -1, 0) // 1ヶ月戻す
	}

	return baseDate.Year(), int(baseDate.Month())
}

// 指定された月の月末までの日数を計算
func getEndOfMonth(year int, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// 日付をDay型で生成する関数
func generateDay(baseYear int, baseMonth int, day int) Day {
	date := time.Date(baseYear, time.Month(baseMonth), day, 0, 0, 0, 0, time.UTC)
	week := date.Weekday().String()[:3] //曜日を3文字に

	return Day{
		Date: date.Format("2006-01-02"),  // Go特有の日時フォーマット
		Week: week,
	}
}

// 日付データの生成
func generateDays(baseYear int, baseMonth int, endOfMonth int) []Day {
	// 1日～月末までループし、１日ずつDay型でデータを作成（スライスに格納）
	var days []Day

	// 前月分を追加する処理（1日の曜日が日曜でない場合）
	firstDay := time.Date(baseYear, time.Month(baseMonth), 1, 0, 0, 0, 0, time.UTC)
	weekday := int(firstDay.Weekday()) // Sunday=0, Monday=1,...

	if weekday != 0 {
		// 1月の時の12月の処理
		prevYear, prevMonth := baseYear, int(time.Month(baseMonth)-1)
		if prevMonth < 1 {
			prevYear--
			prevMonth = 12
		}
        
		// 前月の末日から、必要な日数分だけ生成して前に追加
		prevEnd := getEndOfMonth(prevYear, prevMonth)
		// 0(日曜)でなかったら、1日の曜日から0になるまで引く
		for i := weekday - 1; i >= 0; i-- {
			days = append(days, generateDay(prevYear, prevMonth, prevEnd - i)) // prevEnd - iにより、古い日付から生成
		}
	}

	// 該当月の日付データ生成
	for i := 1; i <= endOfMonth; i++ {
		days = append(days, generateDay(baseYear, baseMonth, i))
	}
	return days
}