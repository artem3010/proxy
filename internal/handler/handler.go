package handler

import "net/http"

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(writer http.ResponseWriter, request *http.Request) {

}
