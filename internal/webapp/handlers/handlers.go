package handlers

import (
	"net/http"

	"github.com/rmagalhaes85/codesurveys/internal/webapp/models"
	"github.com/rmagalhaes85/codesurveys/internal/webapp/render"
)

type Handlers struct {
	store models.Store
	tpl *render.Renderer
}

func New(store models.Store, tpl *render.Renderer) *Handlers {
	return &Handlers{store: store, tpl: tpl}
}

func (h *Handlers) Home() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.tpl.Render(w, r, "home.tmpl", render.M{})
	})
}
