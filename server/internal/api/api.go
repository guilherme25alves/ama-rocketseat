package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/guilherme25alves/ama-rocketseat-server/internal/store/pgstore"
)

type apiHandler struct {
	q *pgstore.Queries // O ideal aqui era uso de interface (se fosse necessário mudar o banco, por exemplo, ia mudar implementação, mas poderíamos manter a mesma interface)
	r *chi.Mux
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q: q,
	}

	r := chi.NewRouter()

	a.r = r
	return a
}
