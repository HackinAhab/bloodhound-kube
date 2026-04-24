# Bloodhound-Kube

Collect Kubernetes resources and parse json suitable for BloodHound-style graph analysis.

## Overview
This tool was built to help visualize attack paths and relationships in Kubernetes environments using BloodHound for the purposes penetration testing engagements. As such some assumptions and design choices may be more focused on that use case, but it should be generally useful for security teams to visualize potential attack paths. 

I'm not a software engineer, so this is a first pass at a tool to solve this problem, and there are very **many** improvements that could be made, but it works for now and I'm open to contributions. 

As a disclaimer, I have used AI to help write/refactor chunks of the code and documentation, but I have reviewed and tested it to ensure it works as intended for most common use-cases.

Example Graphs:

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
```bash
go build -o bloodhound-kube
```

With `just`:
```bash
just build
```

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

# Logging
./bloodhound-kube collect --log debug --no-color

# Redact secret values
./bloodhound-kube collect --redacted
```

### Discovery and allowlists
- Default behavior runs discovery and collects resources from the built-in allowlist.
- `--discovery-auto` collects all discovered resources; `--discovery-auto-accept` skips confirmation prompts for large numbers of CRDs.
- `--discovery-allowlist` appends entries to the built-in allowlist. Each line can be:
  - `group/version/resource`
  - `group/version`
  - `group/resource`
  - `group/*`
  - `v1/resource`
- `--redacted` omits Secret `data` values during collection while preserving key names.

### Parse command
```bash
./bloodhound-kube parse [flags]
```
Example parse
```
./bloodhound-kube parse -i /tmp/my-collection.jsonl -o /tmp/my-graph.json --cluster prod-us-east-1
```

When uploading data from multiple clusters to the same BloodHound instance, set a unique `--cluster` value per cluster. This value is embedded in node IDs and added as a `cluster` node property so you can filter results per cluster.

#### Nodes and Edges
Relationships are implemented in Go and compiled into the binary. Rules live in `internal/edges`, while node parsing lives in `internal/nodes`.
See [docs/nodes.md](./docs/nodes.md) for details on existing node types and how to add new ones.
See [docs/edges.md](./docs/edges.md) for details on existing edge rules and how to add new ones.

#### Parsing Undefined Nodes
Resources that are undefined in the node builders are byy default skipped to avoid excess noise. To enable generic node creation for undefined kinds, use:

```bash
bloodhound-kube parse --parse-undefined-nodes
```
This will create generic nodes for any resource with a `kind` field, using the format `kind:group` (e.g. `MyResource:mygroup.example.com`). This can be useful for discovery, but will create **a lot** of noise.

### Upload command
The `upload` command is used to upload the icons, colors, and custom cypher queries in BloodHound instance for Kubernetes data. This only needs to be done once per BloodHound instance, and the same types can be used for multiple collections.

BloodHound API access uses HMAC credentials via token ID + token key created in the BloodHound UI.

```bash
bloodhound-kube upload --model-file config/custom_types.json --queries config/custom_queries.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY
```
Additionally, parsed collections can be uploaded directly to BloodHound with the `--upload-file` flag.
In the latest versions of BloodHound, the custom types and queries can be uploaded directly through the UI, so this command is optional if you prefer to do it that way.
```bash
bloodhound-kube upload --queries-file config/custom_queries.json --model-file config/custom_types.json --url http://localhost:9000 --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY --upload-file tmp/data.json
```

### Report command
The `report` command is used to generate a quick summary report of the collected data with common misconfigurations and potential attack paths. This is not meant to be comprehensive, but can be a useful starting point for analysis or to quickly identify common issues. This has received minimal testing and attention, so expect bugs and edge cases where it may not work as intended. It was included to bake in some existing scripts, but may recieve more development in the future.
 To generate report files from a collection file:

```bash
bloodhound-kube report -i /tmp/my-collection.jsonl
```

## Design choices
Relationships are implemented as Go edge rules for clarity, performance, and easier extension. Rules are registered explicitly in `internal/edges/edge_registry.go` and grouped by domain under `internal/edges/rules/*`. Node definitions remain in Go for the same reasons.

The tool uses the Kubernetes API to automatically find and collect resources based on what is present in the cluster, which can be useful for large or unfamiliar environments. It can be set to collect all resources, many of which will not be relevant for your use case. By default a set of core components is defined, and CRDs are discovered and require confirmation before collection to avoid collecting large numbers of irrelevant resources. To selectively collect releveant CRDs, an allow-list can be provided to automatically collect matching resources in **addition** to the default set without confirmation. This area could likely use some improvement to make it more intuitive, and allow overrding the default allow-list.

Data collected is output as a raw kubernetes resource dump in JSONL format, which *should* be easy to parse with `jq --slurp`.

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
