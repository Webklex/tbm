package response

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type JsonResponse struct {
	DefaultResponse
}

func NewJsonResponse(w http.ResponseWriter, r *http.Request, p httprouter.Params) Response {
	return &JsonResponse{
		NewDefaultResponse(w, r, p),
	}
}

func (jr *JsonResponse) Render() {
	b, err := json.Marshal(jr.DefaultResponse)
	if err != nil {
		jr.AddError(NewErrorFromString("500 invalid data", http.StatusInternalServerError))
		jr.renderWithError()
		return
	}

	_, _ = jr.writer.Write(b)
}

func (jr *JsonResponse) renderWithError() {
	if len(jr.Errors) > 0 {
		jr.Data = map[string]interface{}{}
		if jr.Status == http.StatusOK {
			for _, err := range jr.Errors {
				if err.Status > jr.Status {
					jr.Status = err.Status
				}
			}
		}
	}
	jr.Render()
}
