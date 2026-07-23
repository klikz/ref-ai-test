package models

type Responce struct {
	Result string      `json:"result"`
	Error  interface{} `json:"error"`
	Data   interface{} `json:"data"`
}
