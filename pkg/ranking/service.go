package ranking

import (
	"errors"
	"fmt"
	"log/slog"
	"math"

	ore "github.com/Open-Ranking/go-sdk"
	"github.com/nbd-wtf/go-nostr"
	"github.com/redis/go-redis/v9"
	"github.com/vertex-lab/crawler_v2/pkg/leaks"
	"github.com/vertex-lab/crawler_v2/pkg/regraph"
	"github.com/vertex-lab/crawler_v2/pkg/store"
	sqlite "github.com/vertex-lab/nostr-sqlite"
)

// Errors returned by the ranking service.
var (
	ErrUnsupportedAlgo   = errors.New("unsupported algorithm")
	ErrBadlyFormattedKey = errors.New("badly formatted key")
	ErrUnknownPubkey     = errors.New("pubkey is unknown")
)

// Supported open ranking algorithms.
var (
	GlobalPagerank       ore.AlgorithmID = "global-pagerank"
	FollowersCount       ore.AlgorithmID = "followers-count"
	PersonalizedPagerank ore.AlgorithmID = "personalized-pagerank"
	SignatureProof       ore.AlgorithmID = "signature-proof"

	GlobalPagerankAlgo       = ore.Algorithm{ID: GlobalPagerank, Name: "Influence", Description: "Discover trusted, high-reputation voices across the entire network."}
	FollowersCountAlgo       = ore.Algorithm{ID: FollowersCount, Name: "Most Followed", Description: "See who has the largest audience on the platform."}
	PersonalizedPagerankAlgo = ore.Algorithm{ID: PersonalizedPagerank, Name: "Most Relevant", Description: "Find profiles that are closest to you and your social circle.", POV: true}
	SignatureProofAlgo       = ore.Algorithm{ID: SignatureProof, Name: "Security Check", Description: "Identify compromised profiles by detecting leaked keys"}
)

// The capability document representing what this package actually supports.
// Change every time the capabilities change.
var Capabilities = ore.CapabilityDoc{
	StatsPubkey:        []ore.Algorithm{GlobalPagerankAlgo, FollowersCountAlgo, PersonalizedPagerankAlgo},
	RankPubkeys:        []ore.Algorithm{GlobalPagerankAlgo, FollowersCountAlgo, PersonalizedPagerankAlgo},
	RecommendPubkeys:   []ore.Algorithm{GlobalPagerankAlgo, FollowersCountAlgo, PersonalizedPagerankAlgo},
	SearchPubkeys:      []ore.Algorithm{GlobalPagerankAlgo, FollowersCountAlgo, PersonalizedPagerankAlgo},
	Followers:          []ore.Algorithm{GlobalPagerankAlgo, FollowersCountAlgo, PersonalizedPagerankAlgo},
	CompromisedPubkeys: []ore.Algorithm{SignatureProofAlgo},
	// Muters: nil — not supported, endpoint returns 501.
}

// Service encapsulates the business logic of the Vertex services.
type Service struct {
	Sqlite *sqlite.Store
	Graph  regraph.DB
	Leaks  leaks.DB
}

// New creates a [Service] initialized with the specified [Config].
func NewService(c Config) (*Service, error) {
	sqlite, err := store.New(c.SqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service: %w", err)
	}
	slog.Info("sqlite connected", "address", c.SqlitePath)

	redis := redis.NewClient(&redis.Options{Addr: c.RedisAddress})
	leaks := leaks.NewDB(redis)
	graph, err := regraph.New(redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service: %w", err)
	}
	slog.Info("redis connected", "address", c.RedisAddress)

	return &Service{
		Sqlite: sqlite,
		Graph:  graph,
		Leaks:  leaks,
	}, nil
}

// Close closes the service database connections, releasing resources.
func (s *Service) Close() error {
	err1 := s.Sqlite.Close()
	err2 := s.Graph.Close()

	if err1 == nil && err2 == nil {
		return nil
	}
	return fmt.Errorf("service failed to close: sqlite: %w; redis: %w", err1, err2)
}

const (
	minute   = 60
	hour     = 3600        // number of seconds in an hour
	infinite = math.MaxInt // for when the TTL is infinite
)

// TTL returns the time-to-live (TTL) for a given algorithm, in seconds.
func TTL(algo ore.AlgorithmID) int {
	switch algo {
	case GlobalPagerank:
		return 24 * hour
	case FollowersCount:
		return 12 * hour
	case PersonalizedPagerank:
		return 6 * hour
	default:
		return 0
	}
}

// Request represent a request made to the service.
type Request interface {
	// Normalize the request in place. It returns an error if invalid.
	Normalize() error

	// Cost returns the cost (measured in credits) of a service call with the provided request.
	Cost() int
}

// TODO: eventually move away from go-nostr in favor of a minimalistic nostr library
// where only types, crypto and JSON serialization are defined.

// validatePubkey validates a public key string.
func validatePubkey(s string) error {
	if s == "" {
		return errors.New("empty string")
	}
	if !nostr.IsValidPublicKey(s) {
		return errors.New("invalid pubkey")
	}
	return nil
}
