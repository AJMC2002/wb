package httpin

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"demo_service/internal/core/domain"
	"demo_service/internal/ports/inbound"
	"demo_service/internal/web"
)

type Handlers struct {
	uc        inbound.OrderUseCase
	adminTmpl *template.Template
}

func NewHandlers(uc inbound.OrderUseCase) *Handlers {
	t := template.Must(template.ParseFS(web.MustFS(), "admin.html"))
	return &Handlers{
		uc:        uc,
		adminTmpl: t,
	}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/order/", h.getOrderByID)
	mux.HandleFunc("/admin", h.admin)
}

func (h *Handlers) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handlers) getOrderByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/order/")
	id = strings.TrimSpace(id)
	if id == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	order, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, order, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

type adminVM struct {
	Page     int
	PageSize int
	Total    int
	Pages    int
	HasPrev  bool
	HasNext  bool
	PrevPage int
	NextPage int
	Orders   []adminOrderRow
}

type adminOrderRow struct {
	OrderUID    string
	TrackNumber string
	CustomerID  string
	DateCreated string
	ItemsCount  int
	Amount      int
	Currency    string
}

func (h *Handlers) admin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := intQuery(r, "page", 1)
	size := intQuery(r, "size", 20)

	orders, total, err := h.uc.ListPage(r.Context(), page, size)
	if err != nil {
		http.Error(w, "admin error", http.StatusInternalServerError)
		return
	}

	if size <= 0 {
		size = 20
	}
	pages := (total + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}

	vm := adminVM{
		Page:     page,
		PageSize: size,
		Total:    total,
		Pages:    pages,
		HasPrev:  page > 1,
		HasNext:  page < pages,
		PrevPage: page - 1,
		NextPage: page + 1,
	}

	for _, o := range orders {
		vm.Orders = append(vm.Orders, adminOrderRow{
			OrderUID:    o.OrderUID,
			TrackNumber: o.TrackNumber,
			CustomerID:  o.CustomerID,
			DateCreated: o.DateCreated.Format("2006-01-02 15:04:05"),
			ItemsCount:  len(o.Items),
			Amount:      o.Payment.Amount,
			Currency:    o.Payment.Currency,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.adminTmpl.Execute(w, vm); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

func intQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
