export PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on
LDFLAGS := -s -w

os-archs=windows:386 windows:amd64 darwin:amd64 darwin:arm64 freebsd:386 freebsd:amd64 freebsd:arm64 linux:386 linux:amd64 linux:arm linux:arm64 linux:s390x linux:loong64 linux:riscv64 linux:ppc64le

all: package

build: app

app:
	@mkdir -p release
	@$(foreach n, $(os-archs),\
		os=$(shell echo "$(n)" | cut -d : -f 1);\
		arch=$(shell echo "$(n)" | cut -d : -f 2);\
		gomips=$(shell echo "$(n)" | cut -d : -f 3);\
		target_suffix=$${os}_$${arch};\
		mkdir -p ./release/$${target_suffix}/smartping/bin;\
		echo "Build $${os}-$${arch}...";\
		env CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} GOMIPS=$${gomips} go build -ldflags "$(LDFLAGS)" -o ./release/$${target_suffix}/smartping/bin/smartping ./src/smartping.go;\
		mkdir -p ./release/$${target_suffix}/smartping/conf;\
		cp conf/config-base.json ./release/$${target_suffix}/smartping/conf/config-base.json;\
		cp conf/seelog.xml ./release/$${target_suffix}/smartping/conf/seelog.xml;\
		mkdir -p ./release/$${target_suffix}/smartping/db;\
		cp db/database-base.db ./release/$${target_suffix}/smartping/db/database-base.db;\
		cp -rf html ./release/$${target_suffix}/smartping;\
		cp -rf src ./release/$${target_suffix}/smartping;\
		if [ "$${os}" = "windows" ]; then \
			mv ./release/$${target_suffix}/smartping/bin/smartping ./release/$${target_suffix}/smartping/bin/smartping.exe;\
			cp control.cmd ./release/$${target_suffix}/smartping;\
			cp ./bin/nssm.exe ./release/$${target_suffix}/smartping/bin/nssm.exe;\
		else \
			cp -f control ./release/$${target_suffix}/smartping;\
		fi;\
		echo "Build $${os}-$${arch} done";\
	)

package: build
	@$(foreach n, $(os-archs),\
		os=$(shell echo "$(n)" | cut -d : -f 1);\
		arch=$(shell echo "$(n)" | cut -d : -f 2);\
		target_suffix=$${os}_$${arch};\
		echo "Packaging $${target_suffix}...";\
		tar -czf ./release/smartping_$${target_suffix}.tar.gz -C ./release/$${target_suffix} smartping && \
		rm -rf ./release/$${target_suffix};\
		echo "Packaging $${target_suffix} done";\
	)

.PHONY: all build app package
