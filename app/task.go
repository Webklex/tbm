package app

type Task struct {
	Command string                 `json:"command"`
	Payload map[string]interface{} `json:"payload"`
}
