package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// クエリパラメータを解析して、年・月・移動方向を取得する関数
func parseQueryParams(r *http.Request)(int, int, string, error){
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	moveStr := r.URL.Query().Get("move")

	// 年と月の基準値を格納する変数
	var baseYear, baseMonth int

		// クエリパラメータがあればそれを基準に、なければ現在日時を基準に年と月を設定
	if yearStr != "" && monthStr != "" {
		var err error
		baseYear, err = strconv.Atoi(yearStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid year parameter")
		}

		baseMonth, err = strconv.Atoi(monthStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid month parameter")
		}

		if baseMonth < 1 || baseMonth > 12 {
			return 0, 0, "", fmt.Errorf("month must be between 1 and 12")
		}

	} else {
		now := time.Now()
		baseYear = now.Year()
		baseMonth = int(now.Month())
	}

	// moveStr の値チェック
	if moveStr != "" && moveStr != "next" && moveStr != "prev" {
		return 0, 0, "", fmt.Errorf("invalid move parameter")
	}
	return baseYear, baseMonth, moveStr, nil
}