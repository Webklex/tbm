package response

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"tbm/utils/log"
)

type Response interface {
	Render()
	AddError(err *Error)
	SetData(data map[string]interface{})
	AddData(key string, value interface{})

	Writer() http.ResponseWriter
	Request() *http.Request
	Parameter() httprouter.Params
}

type DefaultResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []*Error               `json:"errors,omitempty"`
	Status int                    `json:"status"`

	writer  http.ResponseWriter
	request *http.Request
	params  httprouter.Params
}

func NewDefaultResponse(w http.ResponseWriter, r *http.Request, p httprouter.Params) DefaultResponse {
	return DefaultResponse{
		Data:   map[string]interface{}{},
		Errors: nil,
		Status: http.StatusOK,

		writer:  w,
		request: r,
		params:  p,
	}
}

func (dr *DefaultResponse) AddError(err *Error) {
	if dr.Errors == nil {
		dr.Errors = []*Error{}
	}
	log.Error("An error occurred: %s", err.Error.Error())
	dr.Errors = append(dr.Errors, err)
}

func (dr *DefaultResponse) SetData(data map[string]interface{}) {
	dr.Data = data
}

func (dr *DefaultResponse) AddData(key string, value interface{}) {
	if dr.Data == nil {
		dr.Data = map[string]interface{}{}
	}
	dr.Data[key] = value
}

func (dr *DefaultResponse) Writer() http.ResponseWriter {
	return dr.writer
}

func (dr *DefaultResponse) Request() *http.Request {
	return dr.request
}

func (dr *DefaultResponse) Parameter() httprouter.Params {
	return dr.params
}
