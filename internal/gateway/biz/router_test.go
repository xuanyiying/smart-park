package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestStaticDiscovery(t *testing.T) {
	routes := []*RouteConfig{
		{Path: "/api/v1/device", Target: "vehicle-svc:8001"},
		{Path: "/api/v1/billing", Target: "billing-svc:8002"},
	}

	discovery := NewStaticDiscovery(routes)

	// Test Discover
	instances, err := discovery.Discover(context.Background(), "vehicle-svc")
	if err != nil {
		t.Fatalf("failed to discover: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].Name != "vehicle-svc" {
		t.Fatalf("expected name vehicle-svc, got %s", instances[0].Name)
	}
}

func TestRouterUseCase(t *testing.T) {
	logger := log.NewStdLogger(nil)
	routes := []*RouteConfig{
		{Path: "/api/v1/device", Target: "vehicle-svc:8001"},
		{Path: "/api/v1/billing", Target: "billing-svc:8002"},
		{Path: "/api/v1/pay", Target: "payment-svc:8003"},
		{Path: "/api/v1/admin", Target: "admin-svc:8004"},
	}

	discovery := NewStaticDiscovery(routes)
	uc := NewRouterUseCase(discovery, routes, logger)

	tests := []struct {
		path     string
		expected string
		wantErr  bool
	}{
		{"/api/v1/device/entry", "vehicle-svc:8001", false},
		{"/api/v1/billing/calculate", "billing-svc:8002", false},
		{"/api/v1/pay/create", "payment-svc:8003", false},
		{"/api/v1/admin/lots", "admin-svc:8004", false},
		{"/api/v1/unknown", "", true},
	}

	for _, tt := range tests {
		target, err := uc.GetServiceTarget(context.Background(), tt.path)
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected error for path %s, got nil", tt.path)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for path %s: %v", tt.path, err)
			}
			if target != tt.expected {
				t.Errorf("expected target %s, got %s for path %s", tt.expected, target, tt.path)
			}
		}
	}
}

func TestMatchRoute(t *testing.T) {
	logger := log.NewStdLogger(nil)
	routes := []*RouteConfig{
		{Path: "/api/v1/device", Target: "vehicle-svc:8001"},
		{Path: "/api/v1/billing", Target: "billing-svc:8002"},
	}

	discovery := NewStaticDiscovery(routes)
	uc := NewRouterUseCase(discovery, routes, logger)

	// Test matching route
	route := uc.MatchRoute("/api/v1/device/entry")
	if route == nil {
		t.Fatal("expected route to match")
	}
	if route.Target != "vehicle-svc:8001" {
		t.Errorf("expected target vehicle-svc:8001, got %s", route.Target)
	}

	// Test non-matching route
	route = uc.MatchRoute("/api/v1/unknown")
	if route != nil {
		t.Fatal("expected route to not match")
	}
}
