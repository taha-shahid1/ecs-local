package docker

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	if client.cli == nil {
		t.Error("expected client to be initialized")
	}
}

func TestClient_Ping(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	err = client.Ping(ctx)
	if err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestClient_Version(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	version, err := client.Version(ctx)
	if err != nil {
		t.Errorf("failed to get version: %v", err)
	}

	if version == "" {
		t.Error("expected non-empty version string")
	}

	t.Logf("Docker version: %s", version)
}

func TestClient_Info(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	info, err := client.Info(ctx)
	if err != nil {
		t.Errorf("failed to get info: %v", err)
	}

	if info == "" {
		t.Error("expected non-empty info string")
	}

	t.Logf("Docker info: %s", info)
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}

	err = client.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Error("closing already closed client should not error")
	}
}
