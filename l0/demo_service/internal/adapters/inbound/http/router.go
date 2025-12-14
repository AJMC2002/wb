package httpin

import (
	"net/http"

	"demo_service/internal/ports/inbound"
)

func NewMux(h *Handlers, uc inbound.OrderUseCase) *http.ServeMux {
	mux := http.NewServeMux()

	h.Register(mux)

	ui := NewUI(uc)
	mux.HandleFunc("/", ui.Index)
	mux.HandleFunc("/ui/order", ui.FetchOrderSSE)

	return mux
}
