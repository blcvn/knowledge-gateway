package observability

import (
	"net/http"

	"kg-service/internal/access"
	"kg-service/internal/httpapi/respond"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

func (h Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	_, _ = access.IdentityFromContext(r.Context())
	respond.OK(w, h.service.Snapshot())
}
