package main

// SchedulePostRequest は、POSTリクエストのJSONボディの構造を定義します。
// これにより、型安全なデコードが可能になります。
type SchedulePostRequest struct {
	AllowOtherEdit bool                   `json:"allowOtherEdit"`
	StartDate      *string                `json:"startDate,omitempty"`
	EndDate        *string                `json:"endDate,omitempty"`
	Events         map[string][]TimeEntry `json:"events"`
}

// Firestoreに保存する際のキー名を小文字にするため、`firestore`タグを追加
type TimeEntry struct {
	Start     string `json:"start"     firestore:"start"`
	End       string `json:"end"       firestore:"end"`
	Order     *int   `json:"order,omitempty" firestore:"order,omitempty"`
	Username  string `json:"username"  firestore:"username"`
	UserColor string `json:"userColor" firestore:"userColor"`
}

// StartDateとEndDateをポインタ型(*string)にし、omitemptyタグを追加
type ScheduleDocument struct {
	OwnerUID       string                 `json:"ownerUid,omitempty" firestore:"ownerUid,omitempty"`
	AllowOtherEdit bool                   `firestore:"allowOtherEdit"`
	StartDate      *string                `firestore:"startDate,omitempty"`
	EndDate        *string                `firestore:"endDate,omitempty"`
	Events         map[string][]TimeEntry `firestore:"events"`
}