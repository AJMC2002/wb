package runtime

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HTTPServer struct {
	srv *http.Server
}

func NewHTTPServer(addr string, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *HTTPServer) Start() {
	go func() {
		log.Printf("[http] listening on %s", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[http] server error: %v", err)
		}
	}()
}

func (s *HTTPServer) Shutdown(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
