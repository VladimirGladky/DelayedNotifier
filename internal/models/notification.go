package models

type Notification struct {
	Id      string `json:"id"`
	Message string `json:"message"`
	Time    string `json:"time"`
	Status  string `json:"status"`
}
