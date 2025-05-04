package main

import "time"

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
	return Day{
		Date: date.Format("2006-01-02"),  // Go特有の日時フォーマット
		Day:  day,
	}
}

// 日付データの生成
func generateDays(baseYear int, baseMonth int, endOfMonth int) []Day {
	// 1日～月末までループし、１日ずつDay型でデータを作成（スライスに格納）
	var days []Day
	for i := 1; i <= endOfMonth; i++ {
		days = append(days, generateDay(baseYear, baseMonth, i))
	}
	return days
}