# bloodhound-kube

Collect Kubernetes resources and emit JSONL suitable for BloodHound-style graph analysis. Policies are implemented in Rego and can be loaded from disk or embedded into the binary.

## Quickstart
```bash
git clone git@github.com:HackinAhab/bloodhound-kube.git
cd bloodhound-kube
go build -o bloodhound-kube
./bloodhound-kube collect --help
```

## Build
```bash
go build -o bloodhound-kube
```

Optional embedded policies:
```bash
go build -tags embedded -o bloodhound-kube
```

With `just`:
```bash
just build
just build-embedded
just build-docker
```

## Usage
```bash
./bloodhound-kube collect [flags]
```

Auth precedence:
1. `--server` + `--token`
2. `--kubeconfig`
3. `KUBECONFIG`
4. `~/.kube/config`
5. In-cluster config

Common examples:
```bash
# Default kubeconfig + current context namespace
./bloodhound-kube collect

# Custom kubeconfig
./bloodhound-kube collect --kubeconfig /path/to/config

# Namespace selection
./bloodhound-kube collect --namespace production
./bloodhound-kube collect --namespace prod,staging,dev

# Direct API access
./bloodhound-kube collect --server https://k8s-api.example.com --token <token>

# Discovery options
./bloodhound-kube collect --discovery-list
./bloodhound-kube collect --discovery-auto --discovery-auto-accept

# Output control
./bloodhound-kube collect --output /tmp/my-collection.jsonl
```

## Discovery and allowlists
- Default behavior runs discovery and collects resources from the built-in allowlist.
- `--discovery-auto` collects all discovered resources; `--discovery-auto-accept` skips confirmation prompts for large numbers of CRDs.
- `--discovery-allowlist` appends entries to the built-in allowlist. Each line can be:
  - `group/version/resource`
  - `group/version`
  - `group/resource`
  - `group/*`
  - `v1/resource`

## Docker
Local image build:
```bash
docker build -t bloodhound-kube:local .
```

## Releases
- GitHub Releases are created from `vX.Y.Z` tags and include multi-arch binaries.
- GHCR images are built on `main` pushes; tag builds can be added if you want semver image tags.
