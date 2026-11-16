package relay

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/vertex-lab/relay/pkg/rate"
)

func TestRelayInformation(t *testing.T) {
	operator, err := nostr.GetPublicKey(nostr.GeneratePrivateKey())
	if err != nil {
		t.Fatal(err)
	}
	for _, custom := range []bool{false, true} {
		name := "defaults"
		if custom {
			name = "custom"
		}
		t.Run(name, func(t *testing.T) {
			c := NewConfig()
			values := map[string]string{
				"RELAY_SECRET_KEY": nostr.GeneratePrivateKey(),
				"RELAY_DOMAIN":     "wss://example.com",
			}
			if custom {
				values["RELAY_NAME"] = "Community index"
				values["RELAY_DESCRIPTION"] = "Community metadata"
				values["RELAY_ICON"] = "https://example.com/icon.png"
				values["RELAY_CONTACT"] = "mailto:operator@example.com"
				values["RELAY_OPERATOR_PUBKEY"] = operator
			}
			if err := env.ParseWithOptions(&c, env.Options{Environment: values}); err != nil {
				t.Fatal(err)
			}
			if err := c.Init(); err != nil {
				t.Fatal(err)
			}
			if err := c.Validate(); err != nil {
				t.Fatal(err)
			}
			signingKey, _ := nostr.GetPublicKey(c.SecretKey)
			if c.PublicKey != signingKey {
				t.Fatal("metadata changed signing identity")
			}

			r := Setup(c, nil, rate.Limiter{})
			w := httptest.NewRecorder()
			r.ServeNIP11(w)
			var info nip11.RelayInformationDocument
			if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
				t.Fatal(err)
			}
			want := nip11.RelayInformationDocument{
				Name:        "Vertex Relay",
				Description: "Web of Trust Relay powered by Vertex",
				Icon:        "https://image.nostr.build/7afc9d727d6486851cc2fe09865e7cc383449f8bad1700a9508db4d2815b6f1a.png",
				PubKey:      signingKey,
			}
			if custom {
				want.Name = values["RELAY_NAME"]
				want.Description = values["RELAY_DESCRIPTION"]
				want.Icon = values["RELAY_ICON"]
				want.Contact = values["RELAY_CONTACT"]
				want.PubKey = operator
			}
			if info.Name != want.Name || info.Description != want.Description ||
				info.Icon != want.Icon || info.Contact != want.Contact || info.PubKey != want.PubKey {
				t.Fatalf("unexpected relay information: %+v", info)
			}
		})
	}
}

func TestInvalidOperatorPublicKey(t *testing.T) {
	c := NewConfig()
	c.SecretKey = nostr.GeneratePrivateKey()
	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"not-a-key", "npub1invalid", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"} {
		c.OperatorPublicKey = key
		if err := c.Validate(); err == nil {
			t.Errorf("accepted invalid operator pubkey %q", key)
		}
	}
}
