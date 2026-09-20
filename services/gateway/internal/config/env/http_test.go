package env

import "testing"

func TestHTTPConfig(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("GATEWAY_HTTP_ADDR", "127.0.0.1:8080")
		cfg, err := NewHTTPConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Address() != "127.0.0.1:8080" {
			t.Fatalf("Address() = %q", cfg.Address())
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Setenv("GATEWAY_HTTP_ADDR", ":9090")
		cfg, err := NewHTTPConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Address() != ":9090" {
			t.Fatalf("Address() = %q", cfg.Address())
		}
	})
}
