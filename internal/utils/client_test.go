package utils

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestClientPinsKubeconfigContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"gitVersion":"v1.30.0"}`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config")
	cfg := clientcmdapi.Config{
		Clusters:       map[string]*clientcmdapi.Cluster{"original": {Server: server.URL}, "other": {Server: "http://127.0.0.1:1"}},
		Contexts:       map[string]*clientcmdapi.Context{"original": {Cluster: "original", Namespace: "original-ns"}, "other": {Cluster: "other", Namespace: "other-ns"}},
		CurrentContext: "original",
	}
	if err := clientcmd.WriteToFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	first, err := NewClient(ClientConfig{Kubeconfig: path, ClusterType: ClusterTypeKubernetes})
	if err != nil {
		t.Fatal(err)
	}
	cfg.CurrentContext = "other"
	if err := clientcmd.WriteToFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	resumed, err := NewClient(ClientConfig{Kubeconfig: first.Kubeconfig, Context: first.Context, ClusterType: ClusterTypeKubernetes})
	if err != nil {
		t.Fatal(err)
	}
	ns, err := GetContextNamespace(resumed.Kubeconfig, resumed.Context)
	if err != nil || ns != "original-ns" || resumed.APIServer != first.APIServer {
		t.Fatalf("current context replaced pinned connection: namespace=%s server=%s error=%v", ns, resumed.APIServer, err)
	}
}

func TestTokenRestConfigTLS(t *testing.T) {
	tests := []struct {
		name         string
		cfg          ClientConfig
		wantInsecure bool
		wantCA       string
	}{
		{"legacy default stays insecure", ClientConfig{Server: "https://x", Token: "t"}, true, ""},
		{"ca verifies", ClientConfig{Server: "https://x", Token: "t", CAFile: "/ca.pem"}, false, "/ca.pem"},
		{"explicit insecure", ClientConfig{Server: "https://x", Token: "t", InsecureSkipTLSVerify: true}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenRestConfig(tt.cfg)
			if got.Insecure != tt.wantInsecure || got.CAFile != tt.wantCA || got.BearerToken != "t" {
				t.Errorf("got insecure=%v ca=%q", got.Insecure, got.CAFile)
			}
		})
	}
}
