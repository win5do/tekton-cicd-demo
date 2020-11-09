.PHONY: desc

sync:
	cd src/sync && go run . --sync=true

IMG := registry.cn-huhehaote.aliyuncs.com/feng-566/git-poll:v0.0.1

build-pull:
	cd src/pull && \
    docker build -t $(IMG) . && \
	docker push $(IMG)
