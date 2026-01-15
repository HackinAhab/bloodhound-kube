# Description

## Install

```
git clone git@github.com:HackinAhab/bloodhound-kube.git
```
```
cd bloodhound-kube && git checkout dev
```
```
go build -o bh-kube
```

## Run
```
chmod +x bh-kube
```
```
./bh-kube collect --help
```

```
./bh-kube collect --help   
Collect Kubernetes resources from the cluster and stream as JSONL

Authentication methods (in order of precedence):
1. --server and --token flags for direct API access
2. --kubeconfig flag to specify custom kubeconfig file  
3. KUBECONFIG environment variable
4. ~/.kube/config (default kubeconfig location)
5. In-cluster configuration (when running inside a pod)

Examples:
  # Use default kubeconfig and current context namespace
  bloodhound-kube collect

  # Use custom kubeconfig file
  bloodhound-kube collect --kubeconfig /path/to/config

  # Use custom config directory
  bloodhound-kube collect --config-dir ./custom-configs

  # Specify single namespace
  bloodhound-kube collect --namespace production

  # Specify multiple namespaces
  bloodhound-kube collect --namespace prod,staging,dev

  # Direct API access with token
  bloodhound-kube collect --server https://k8s-api.example.com --token eyJhbGciOi...

  # Specify cluster type (auto-detects by default)  
  bloodhound-kube collect --cluster-type openshift

  # Resume interrupted collection
  bloodhound-kube collect --resume

  # Resume with custom checkpoint file
  bloodhound-kube collect --resume --checkpoint-file /path/to/checkpoint.json

  # Specify custom output filename
  bloodhound-kube collect --output my-collection.jsonl

  # Specify output to a different directory
  bloodhound-kube collect --output /tmp/

  # Specify full path with directory and filename
  bloodhound-kube collect --output /tmp/my-collection.jsonl

Usage:
  kube-bloodhound collect [flags]

Flags:
  -A, --all-namespaces           Collect from all namespaces (cannot be used with -n)
      --checkpoint-file string   Path to checkpoint file (auto-generated if not specified)
  -T, --cluster-type string      Cluster type: kubernetes, openshift, or auto (auto-detect) (default "auto")
  -c, --concurrency int          Number of concurrent workers for streaming collection (default 10)
      --config-dir string        Directory containing configuration files (collections.yaml, parsers.yaml) (default "config")
  -h, --help                     help for collect
      --kubeconfig string        Path to kubeconfig file (overrides KUBECONFIG and ~/.kube/config)
  -l, --log string               Log level (debug, info, warn, error) (default "info")
  -n, --namespace string         Kubernetes namespace(s) - comma-delimited for multiple (defaults to current context namespace)
  -o, --output string            Output file path (can be directory, filename, or full path). Defaults to bloodhound-kube-YYYY-MM-DD-HHMMSS.jsonl in current directory
      --redacted                 Redact secrets and sensitive data during collection
      --resume                   Resume from previous interrupted collection
  -s, --server string            Kubernetes API server address (requires --token)
      --timeout int              Timeout in seconds for the entire collection (default 300)
      --token string             Bearer token for authentication (requires --server)
  -t, --type strings             Resource types to collect (see config/collections.yaml for available types). Default: all enabled types
```

> Collection types can be found in config/collections.yaml, along with some default values that are in various phases of implementation and may not be working. This will likely change in the future