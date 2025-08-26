package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Server struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	logger    *zap.Logger
	srv       *http.Server
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Server {
	return &Server{
		cfg:       cfg,
		blockChan: blockChan,
		mem:       mem,
		logger:    logger,
	}
}

func (s *Server) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.auth(s.statusHandler))
	mux.HandleFunc("/latest", s.auth(s.latestHandler))
	mux.HandleFunc("/metrics", s.auth(s.metricsHandler))
	mux.HandleFunc("/stream", s.auth(s.streamHandler))

	addr := s.cfg.APIHost + ":" + strconv.Itoa(s.cfg.APIPort)
	s.srv = &http.Server{Addr: addr, Handler: mux}
	s.logger.Info("API started", zap.String("addr", addr))
	s.srv.ListenAndServe()
}

func (s *Server) Stop() {
	if s.srv != nil {
		s.srv.Close()
	}
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != s.cfg.APIKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status": "ok",
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) latestHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case blk := <-s.blockChan:
		json.NewEncoder(w).Encode(blk)
	default:
		json.NewEncoder(w).Encode(map[string]string{"msg": "no block yet"})
	}
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("sprint_active_peers 1\nsprint_blocks_detected 100\n"))
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	for blk := range s.blockChan {
		ws.WriteJSON(blk)
	}
}
