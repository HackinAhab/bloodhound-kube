bin_dir := "bin"
binary := "bloodhound-kube"
image := "bloodhound-kube:local"

build:
    mkdir -p {{bin_dir}}
    go build -trimpath -ldflags "-s -w" -o {{bin_dir}}/{{binary}} .

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
          go build -tags embedded -trimpath -ldflags "-s -w" -o "{{bin_dir}}/$${name}" .; \
      done; \
    done
