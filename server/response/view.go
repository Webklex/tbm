package response

import (
	"github.com/julienschmidt/httprouter"
	"html/template"
	"net/http"
	"tbm/utils/log"
)

type ViewResponse struct {
	DefaultResponse
	template *template.Template
}

func NewViewResponse(temp *template.Template, w http.ResponseWriter, r *http.Request, p httprouter.Params) Response {
	return &ViewResponse{
		DefaultResponse: NewDefaultResponse(w, r, p),
		template:        temp,
	}
}

func (vr *ViewResponse) Render() {
	if err := vr.template.Execute(vr.Writer(), vr.Data); err != nil {
		log.Error("Failed to execute template: %s", err.Error())
	}
}
