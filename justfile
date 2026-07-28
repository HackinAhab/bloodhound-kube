bin_dir := "~/.local/bin"
binary := "bloodhound-kube"
token_key := "token-key-here"
token_id := "token-id-here"
queries := "./config/custom_queries.json"
schema := "./config/schema.json"

# Addons (calico, cilium, external_secrets, cert_manager, istio) are compiled
# in by default. Pass addons=no_addons to exclude all of them, or a comma list
# like addons=no_calico,no_cilium to exclude specific families.
build addons="":
    mkdir -p {{bin_dir}}
    go build {{ if addons != "" { "-tags " + addons } else { "" } }} -trimpath -ldflags "-s -w" -o {{bin_dir}}/{{binary}} .

test:
    go test ./...

# Runs the suite with all addon families excluded (//go:build !no_addons gates
# their tests) to verify the no-addons build path still passes.
test-no-addons:
    go test -tags no_addons ./...

build-all:
    mkdir -p {{bin_dir}}
    for os in linux darwin windows; do \
      for arch in amd64 arm64; do \
        name="{{binary}}-${os}-${arch}"; \
        if [ "$${os}" = "windows" ]; then name="$${name}.exe"; fi; \
        CGO_ENABLED=0 GOOS="$${os}" GOARCH="$${arch}" \
          go build -trimpath -ldflags "-s -w" -o "{{bin_dir}}/$${name}" .; \
      done; \
    done

build-docker:
    docker build -t {{image}} .

build-embedded:
    mkdir -p {{bin_dir}}
    go build -tags embedded -trimpath -ldflags "-s -w" -o {{bin_dir}}/{{binary}} .

build-all-embedded:
    mkdir -p {{bin_dir}}
    for os in linux darwin windows; do \
      for arch in amd64 arm64; do \
        name="{{binary}}-${os}-${arch}"; \
        if [ "$${os}" = "windows" ]; then name="$${name}.exe"; fi; \
        CGO_ENABLED=0 GOOS="$${os}" GOARCH="$${arch}" \
          go build -trimpath -ldflags "-s -w" -o "{{bin_dir}}/$${name}" .; \
      done; \
    done

reset:
  bloodhound-kube upload --reset  --reset-db --configs --url http://localhost:8080 --token-id {{token_id}} --token-key {{token_key}}

upload: 
  sleep 60
  bloodhound-kube upload --reset --upload-file tmp/test.json --schema-file {{schema}} --queries-file {{queries}} --url http://localhost:8282 --token-id {{token_id}} --token-key {{token_key}} --cluster test

collect:
  bloodhound-kube collect --scope all -A -o tmp/test --cluster test --kubeconfig ~/.kube/config --accept-crds
