package models

type Notification struct {
	Id      int    `json:"id"`
	Message string `json:"message"`
	Time    string `json:"time"`
}
