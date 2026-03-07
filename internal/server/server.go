package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/VAGRAMCHIC/wg-agent/internal/manager"
	"github.com/VAGRAMCHIC/wg-agent/pkg/logger"
)

type Server struct {
	mgr *manager.Manager
	log *logger.Logger
}

func New(m *manager.Manager, log *logger.Logger) *Server {
	return &Server{
		mgr: m,
		log: log,
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func withRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, logger.RequestIDKey, uuid.New().String())
}

type AddPeerReq struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
}

type RemovePeerReq struct {
	PublicKey string `json:"public_key"`
}

func (s *Server) AddPeer(w http.ResponseWriter, r *http.Request) {

	ctx := withRequestID(r.Context())

	var req AddPeerReq

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.log.Error(ctx, "invalid_request", nil)
		http.Error(w, err.Error(), 400)
		return
	}

	err = s.mgr.AddPeer(ctx, req.PublicKey, req.IP)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.log.Info(ctx, "peer_added_request", map[string]interface{}{
		"ip": req.IP,
	})

	writeJSON(w, 200, map[string]string{"public_key": req.PublicKey, "ip": req.IP})
}

func (s *Server) RemovePeer(w http.ResponseWriter, r *http.Request) {
	ctx := withRequestID(r.Context())

	var req RemovePeerReq

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.log.Error(ctx, "invalid_request", nil)
		http.Error(w, err.Error(), 400)
		return
	}

	err = s.mgr.RemovePeer(ctx, req.PublicKey)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info(ctx, "peer_added_request", map[string]interface{}{
		"public_key": req.PublicKey,
	})
	writeJSON(w, 200, map[string]string{"public_key": req.PublicKey})
}

func (s *Server) ListPeers(w http.ResponseWriter, r *http.Request) {
	ctx := withRequestID(r.Context())

	peers, err := s.mgr.ListPeers(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.log.Info(ctx, "peer_list_request", map[string]interface{}{})

	writeJSON(w, 200, peers)

}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {

	writeJSON(w, 200, map[string]string{
		"status": "ok",
	})
}

func (s *Server) Start(addr string) error {

	http.HandleFunc("/peer/add", s.AddPeer)
	http.HandleFunc("/peer/remove", s.RemovePeer)
	http.HandleFunc("/peers", s.ListPeers)
	http.HandleFunc("/health", s.health)

	return http.ListenAndServe(addr, nil)
}
