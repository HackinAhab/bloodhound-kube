# Bloodhound-Kube

Collect Kubernetes resources and parse json suitable for BloodHound-style graph analysis.

## Overview
This tool was built to help visualize attack paths and relationships in Kubernetes environments using BloodHound for the purposes penetration testing engagements. As such some assumptions and design choices may be more focused on that use case, but it should be generally useful for security teams to visualize potential attack paths. 

I'm not a software engineer, so this is a first pass at a tool to solve this problem, and there are very **many** improvements that could be made, but it works for now and I'm open to contributions. 

As a disclaimer, I have used AI to help write/refactor chunks of the code and documentation, but I have reviewed and tested it to ensure it works as intended for most common use-cases.

## Example Graphs

HTTPRoutes to Nodes:
![](./docs/img/paths-httproute-node.png)


## Quickstart

Requirements:
- Golang 1.25+

### Clone and build:
```bash
git clone git@github.com:HackinAhab/bloodhound-kube.git
cd bloodhound-kube
go build -o bloodhound-kube
./bloodhound-kube collect --help
```

### Standard Build (Development)
```bash
go build -o bloodhound-kube
```

With `just`:
```bash
just build
```

### Embedded Build (Production/Distribution)
Build with embedded config files for standalone distribution:
```bash
go build -tags embedded -o bloodhound-kube
```

With `just`:
```bash
just build-embedded          # Single binary
just build-all-embedded      # Cross-platform binaries
```

The embedded build includes `config/custom_queries.json` and `config/schema.json` directly in the binary, allowing standalone distribution without external config files.

## Usage

### Collection command
```bash
./bloodhound-kube collect [flags]
```

Auth precedence:
1. `--server` + `--token`
2. `--kubeconfig`
3. `KUBECONFIG` environment variable
4. `~/.kube/config` static path
5. In-cluster config

Common examples:
```bash
# Default kubeconfig + current context namespace
./bloodhound-kube collect

# Default scope is core (built-in allowlist)
./bloodhound-kube collect --scope core

# Collect all discovered resources
./bloodhound-kube collect --scope all --accept-crds

# Collect built-in core set plus allowlisted resources
./bloodhound-kube collect --scope allowlist --discovery-allowlist ./allowlist.txt

# Custom kubeconfig
./bloodhound-kube collect --kubeconfig /path/to/config

# Namespace selection
./bloodhound-kube collect --namespace production
./bloodhound-kube collect --namespace prod,staging,dev

# Direct API access
./bloodhound-kube collect --server https://k8s-api.example.com --token <token>

# Discovery options
./bloodhound-kube collect --discovery-list

# Output control
./bloodhound-kube collect --output /tmp/my-collection.jsonl

# Zip the BloodHound JSON output (JSONL stays on disk uncompressed)
./bloodhound-kube collect --zip
./bloodhound-kube collect --zip --output /tmp/my-collection.jsonl

# Logging
./bloodhound-kube collect --log debug --no-color

# Redact secret values
./bloodhound-kube collect --redacted

# Multi-cluster collection from a YAML config file
./bloodhound-kube collect --clusters-config clusters.yaml
./bloodhound-kube collect -C clusters.yaml --no-parse
```

### Multi-cluster collection

Pass `--clusters-config` (or `-C`) to collect from multiple clusters in a single run. Each cluster's output lands in a separate JSONL file (and `.json` if parse is enabled). On partial failure the tool continues, reports per-cluster status, and exits non-zero.

```bash
./bloodhound-kube collect -C clusters.yaml
```

The config file format:

```yaml
# clusters.yaml
defaults:
  scope: core
  concurrency: 10
  paginateLimit: 100
  outputDir: ./output
  clusterConcurrency: 4  # run up to 4 cluster pipelines in parallel
  acceptCRDs: true        # required for multi-cluster mode (see below)

clusters:
  - name: prod-us-east
    kubeconfig: ~/.kube/prod
    clusterType: kubernetes

  - name: staging
    server: https://k8s-staging.example.com
    token: ${STAGING_TOKEN}       # ${VAR} references are expanded from env
    redacted: true
    outputFile: staging.jsonl     # explicit output path overrides outputDir
```

See [`clusters.example.yaml`](./clusters.example.yaml) for a fully annotated reference.

**Per-cluster fields** override the corresponding `defaults` value. Boolean fields (`allNamespaces`, `redacted`, `acceptCRDs`) use three-state semantics: explicit `true`/`false` overrides the default; omitted inherits it.

`--no-parse`, `--resume`, `--checkpoint-file`, `--fetch-mode-full`, and `--zip` apply globally to all clusters.

**Concurrent collection**: by default clusters are collected sequentially. Use `defaults.clusterConcurrency: N` in the YAML or `--cluster-concurrency N` on the CLI (CLI takes precedence) to run up to N cluster pipelines in parallel. The CLI flag defaults to `0` (defers to YAML; falls back to `1`).

**CRD discovery requirement**: interactive `[y/N]` prompts cannot be used with more than one cluster because multiple goroutines would race on stdin. Ensure every cluster is non-interactive by setting `acceptCRDs: true` (or per-cluster), providing a `discoveryAllowlist`, or using `scope: core`/`scope: allowlist`. The tool rejects multi-cluster configs that would require a prompt.

### Discovery and allowlists
- `--scope` controls collection behavior:
  - `core` (default): collect the curated default set
  - `all`: collect all discovered resources
  - `allowlist`: collect curated default set plus entries from `--discovery-allowlist`
- Migration note: legacy `--discovery-auto` was removed; use `--scope all`.
- `--accept-crds` skips confirmation prompts for large numbers of CRDs.
- `--discovery-allowlist` is used by `--scope allowlist`. Each line can be:
  - `group/version/resource`
  - `group/version`
  - `group/resource`
  - `group/*`
  - `v1/resource`
- `--redacted` omits Secret `data` values during collection while preserving key names.
- `--zip` compresses the BloodHound JSON output (`.json`) into a zip archive after parsing. The raw JSONL file is left on disk and not included in the archive. Has no effect when `--no-parse` is set.

### Parsing
Parsing runs as part of `collect` by default — no separate command. The collect step writes JSONL, and the parse step immediately turns it into BloodHound-compatible JSON in the same run. Use `--no-parse` to skip the parse step and keep JSONL only.

Parse-related flags on `collect`:

```bash
# Set the cluster name embedded in node IDs and added as the `cluster` node property
./bloodhound-kube collect --cluster prod-us-east-1

# Override the BloodHound JSON output path (defaults to the JSONL filename with a .json extension)
./bloodhound-kube collect --parsed-output /tmp/my-graph.json

# Skip the parse step entirely (JSONL only)
./bloodhound-kube collect --no-parse

# Emit generic nodes for resource kinds that don't have a dedicated builder
./bloodhound-kube collect --parse-undefined-nodes
```

When uploading data from multiple clusters to the same BloodHound instance, set a unique `--cluster` value per cluster so you can filter results per cluster in BloodHound.

#### Nodes and Edges
Relationships are implemented in Go and compiled into the binary. Rules live in `internal/edges`, while node parsing lives in `internal/nodes`.
See [docs/nodes.md](./docs/nodes.md) for details on existing node types and how to add new ones.
See [docs/edges.md](./docs/edges.md) for details on existing edge rules and how to add new ones.

#### Parsing Undefined Nodes
Resources that are undefined in the node builders are by default skipped to avoid excess noise. To enable generic node creation for undefined kinds, use `--parse-undefined-nodes` on the `collect` command. This creates generic nodes for any resource with a `kind` field, using the format `kind:group` (e.g. `MyResource:mygroup.example.com`). Useful for discovery, but it generates **a lot** of noise.

### Upload command
The `upload` command is used to upload the icons, colors, and custom cypher queries in BloodHound instance for Kubernetes data. This only needs to be done once per BloodHound instance, and the same types can be used for multiple collections.

BloodHound API access uses HMAC credentials via token ID + token key created in the BloodHound UI.

#### Standard Usage (External Files)
```bash
bloodhound-kube upload --schema-file config/schema.json --queries-file config/custom_queries.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY
```

#### Embedded Build Usage
When using a binary built with `-tags embedded`:

```bash
# Upload data only (no config changes)
bloodhound-kube upload --upload-file data.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

# Upload embedded configs only
bloodhound-kube upload --queries-file='' --schema-file='' --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

# Upload custom configs (merges with embedded if available)
bloodhound-kube upload --queries-file=my-queries.json --schema-file=my-types.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

# Upload both configs and data
bloodhound-kube upload --queries-file='' --schema-file='' --upload-file data.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY
```

**Config File Merging**: When both embedded and user-provided config files exist:
- **Queries**: Merged by `name` field. User queries override embedded queries with the same name.
- **Custom Types**: Merged by node type name. User types override embedded types with the same name.
- User-provided queries/types appear first in the merged output.

Additionally, parsed collections can be uploaded directly to BloodHound with the `--upload-file` flag.
In the latest versions of BloodHound, the custom types and queries can be uploaded directly through the UI, so this command is optional if you prefer to do it that way.

### Report command
The `report` command is used to generate a quick summary report of the collected data with common misconfigurations and potential attack paths. This is not meant to be comprehensive, but can be a useful starting point for analysis or to quickly identify common issues. This has received minimal testing and attention, so expect bugs and edge cases where it may not work as intended. It was included to bake in some existing scripts, but may recieve more development in the future.
 To generate report files from a collection file:

```bash
bloodhound-kube report -i /tmp/my-collection.jsonl
```

## Design choices
Relationships are implemented as Go edge rules for clarity, performance, and easier extension. Rules are registered explicitly in `internal/edges/edge_registry.go` and grouped by domain under `internal/edges/*`. Node definitions remain in Go for the same reasons.

Architecture reference: see `docs/architecture-v2.md` for collect/parse/edge execution flow and extension points.

The tool uses the Kubernetes API to discover resources present in the cluster. Collection scope is explicit: `core` (default curated set), `all` (all discovered resources), or `allowlist` (curated set plus file entries). CRD-backed resources still require confirmation unless `--accept-crds` is used.

Data collected is output as a raw kubernetes resource dump in JSONL format, which *should* be easy to parse with `jq --slurp`.

## Contributor notes
- Put command wiring and flags in `cmd/*` only.
- Put orchestration and request policy in `internal/cli/*`.
- Put discovery/planning/execution logic in `internal/collector/*`.
- Put node transforms in `internal/nodes/*` and relationships in `internal/edges/*`.

## Future improvements
- Add more commonly deployed CRDs to the edge rules to improve visibility in clusters using popular tools like ArgoCD, cert-manager, etc.
- Add more queries to the BloodHound UI for common attack paths and misconfigurations.

## Acknowledgements
### Projects
These projects were instrumental in inspiring and informing the development of BloodHound-Kube, and I highly recommend checking them out:
- [KubeHound](https://kubehound.io/) for their attack reference library and as a major inspiration for this project. 
- [Trivy](https://trivy.dev/) which inspired some of the edge checks.
- [BloodHound OpenGraph](https://bloodhound.specterops.io/opengraph/overview), which enabled the use of BloodHound for Kubernetes. 
### People
Credit to the people who listened to my rambling about this project and providing exceptional feedback, suggestions, and beta testing my first janky golang tool.
- [Josh Neimann]()
- [Lukas Harris]()
- [Michael Mitchell]() 
