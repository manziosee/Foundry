package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"imgstore/internal/api/handlers"
	"imgstore/internal/api/middleware"
	"imgstore/internal/types"
)

type Server struct {
	db     *sql.DB
	svc    ServiceInterface
	server *http.Server
}

type ServiceInterface interface {
	EnqueueImage(ctx context.Context, name, url, checksum string) error
	GetImageStatus(name string) (string, error)
	GetAllImages() ([]types.ImageInfo, error)
	RemoveImage(name string) error
	Cleanup() error
}



func NewServer(db *sql.DB, svc ServiceInterface, addr string) *Server {
	mux := http.NewServeMux()
	
	h := handlers.New(db, svc)
	
	// API routes
	mux.HandleFunc("/api/v1/images", middleware.CORS(h.HandleImages))
	mux.HandleFunc("/api/v1/images/", middleware.CORS(h.HandleImageByName))
	mux.HandleFunc("/api/v1/status", middleware.CORS(h.HandleStatus))
	mux.HandleFunc("/api/v1/cleanup", middleware.CORS(h.HandleCleanup))
	
	// Static files (future web UI)
	mux.HandleFunc("/", h.HandleRoot)
	
	return &Server{
		db:  db,
		svc: svc,
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down API server...")
	return s.server.Shutdown(ctx)
}