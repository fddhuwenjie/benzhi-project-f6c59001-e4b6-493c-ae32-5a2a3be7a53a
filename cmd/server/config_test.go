package main

import "testing"

func TestAddressPrecedence(t *testing.T) {
	env := func(string) string { return "23123" }
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "127.0.0.1:23123" {
		t.Fatalf("PORT address = %s", cfg.address)
	}
	cfg, err = parseConfig([]string{"-addr=127.0.0.1:24123"}, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "127.0.0.1:24123" {
		t.Fatalf("-addr did not take precedence: %s", cfg.address)
	}
}

func TestDefaultAndInvalidPort(t *testing.T) {
	empty := func(string) string { return "" }
	cfg, err := parseConfig(nil, empty)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != defaultAddress {
		t.Fatalf("default address = %s", cfg.address)
	}
	invalid := func(string) string { return "not-a-port" }
	if _, err := parseConfig(nil, invalid); err == nil {
		t.Fatal("invalid PORT accepted")
	}
}
