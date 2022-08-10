package app

import "encoding/json"

type Response struct {
	Errors []string               `json:"errors"`
	Data   map[string]interface{} `json:"data"`
}

func NewResponse() *Response {
	return &Response{
		Errors: make([]string, 0),
		Data:   map[string]interface{}{},
	}
}

func (r *Response) Encode() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Response) SetError(err error) {
	r.Errors = append(r.Errors, err.Error())
}

func (r *Response) SetErrorStr(err string) {
	r.Errors = append(r.Errors, err)
}
