package httpin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"demo_service/internal/core/domain"
	"demo_service/internal/ports/inbound"
	"demo_service/internal/web"

	"github.com/starfederation/datastar-go/datastar"
)

type UI struct {
	uc inbound.OrderUseCase
}

func NewUI(uc inbound.OrderUseCase) *UI {
	return &UI{uc: uc}
}

type uiSignals struct {
	OrderID string `json:"order_id"`
}

func (u *UI) Index(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(web.MustFS())).ServeHTTP(w, r)
}

func (u *UI) FetchOrderSSE(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	signals := &uiSignals{}
	if err := datastar.ReadSignals(r, signals); err != nil {
		sse.PatchElements(`<p id="status">Bad request: invalid signals</p>`)
		return
	}

	if signals.OrderID == "" {
		sse.PatchElements(`<p id="status">Please enter an order_id</p>`)
		return
	}

	sse.PatchElements(`<p id="status">Fetching order...</p>`)
	time.Sleep(150 * time.Millisecond)

	order, err := u.uc.GetByID(r.Context(), signals.OrderID)
	if err != nil {
		if err == domain.ErrNotFound {
			sse.PatchElements(`<p id="status">Not found</p>`)
			sse.PatchElements(`<pre id="result">{}</pre>`)
			return
		}
		sse.PatchElements(`<p id="status">Internal error</p>`)
		return
	}

	b, _ := json.MarshalIndent(order, "", "  ")
	sse.PatchElements(`<p id="status">OK</p>`)
	sse.PatchElements(`<pre id="result">` + htmlEscape(string(b)) + `</pre>`)
}

func htmlEscape(s string) string {
	repl := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return repl.Replace(s)
}
