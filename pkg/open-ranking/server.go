// Package openranking implements the Open Ranking protocol server.
// Learn more: https://github.com/Open-Ranking/protocol
package openranking

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	ore "github.com/Open-Ranking/go-sdk"
	"github.com/pippellia-btc/rely/v2"
	"github.com/vertex-lab/relay/pkg/ranking"
	"github.com/vertex-lab/relay/pkg/rate"
)

// Server implements the Open Ranking protocol.
type Server struct {
	service *ranking.Service
	limiter rate.Limiter

	caps   ore.CapabilityDoc
	config Config
}

// NewServer creates a new Open Ranking server.
func NewServer(c Config, s *ranking.Service, limiter rate.Limiter) Server {
	return Server{
		service: s,
		limiter: limiter,
		caps:    ranking.Capabilities,
		config:  c,
	}
}

// StartAndServe starts the Open Ranking server and serves requests.
// It's a blocking operation that returns only when the context is cancelled.
func (s Server) StartAndServe(ctx context.Context, address string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+ore.EndpointCapabilities, s.Capabilities)
	mux.HandleFunc("POST "+ore.EndpointStatsPubkey, s.StatsPubkey)
	mux.HandleFunc("POST "+ore.EndpointRankPubkeys, s.RankPubkeys)
	mux.HandleFunc("POST "+ore.EndpointRecommendPubkeys, s.RecommendPubkeys)
	mux.HandleFunc("POST "+ore.EndpointSearchPubkeys, s.SearchPubkeys)
	mux.HandleFunc("POST "+ore.EndpointFollowers, s.Followers)
	mux.HandleFunc("POST "+ore.EndpointCompromisedPubkeys, s.CompromisedPubkeys)

	server := &http.Server{
		Addr:              address,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		IdleTimeout:       s.config.IdleTimeout,
	}

	slog.Info("openranking: server start", "address", address)
	defer slog.Info("openranking: server shutdown", "address", address)

	exit := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			exit <- err
		}
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		return server.Shutdown(ctx)

	case err := <-exit:
		return err
	}
}

// withCORS adds the CORS headers required by the Open Ranking protocol on all responses
// and handles OPTIONS preflight requests.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Capabilities serves GET /.well-known/open-ranking.json (ORE-01).
func (s Server) Capabilities(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 1) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.caps); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// StatsPubkey serves POST /stats/pubkey (ORE-02).
func (s Server) StatsPubkey(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.StatsPubkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.StatsPubkey(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: stats pubkey error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// RankPubkeys serves POST /rank/pubkeys (ORE-03).
func (s Server) RankPubkeys(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.RankPubkeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.RankPubkeys(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: rank pubkeys error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// RecommendPubkeys serves POST /recommend/pubkeys (ORE-04).
func (s Server) RecommendPubkeys(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.RecommendPubkeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.RecommendPubkeys(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: recommend pubkeys error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// SearchPubkeys serves POST /search/pubkeys (ORE-05).
func (s Server) SearchPubkeys(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.SearchPubkeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.SearchPubkeys(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: search pubkeys error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// Followers serves POST /followers (ORE-06).
func (s Server) Followers(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.FollowersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.Followers(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: followers error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}

// CompromisedPubkeys serves POST /compromised/pubkeys (ORE-08).
func (s Server) CompromisedPubkeys(w http.ResponseWriter, r *http.Request) {
	ip := rely.GetIP(r).Group()
	if !s.limiter.Allow(ip, 5) {
		ore.WriteError(w, ore.ErrTooMany("Rate limit exceeded, try again later"))
		return
	}

	var req ranking.CompromisedPubkeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ore.WriteError(w, ore.ErrBadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := req.Normalize(); err != nil {
		ore.WriteError(w, ore.ErrBadRequest(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.config.RequestTimeout)
	defer cancel()

	res, err := s.service.CompromisedPubkeys(ctx, req)
	if err != nil {
		ore.WriteError(w, ore.ErrInternal("internal error"))
		slog.Error("openranking: compromised pubkeys error", "request", req, "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("openranking: failed to encode response", "error", err)
	}
}
