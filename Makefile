.PHONY: desc

sync:
	cd src/sync && go run . --sync=true

IMG_PULL := registry.cn-huhehaote.aliyuncs.com/feng-566/tekton-git-polling:v0.0.1
IMG_CLEANUP := registry.cn-huhehaote.aliyuncs.com/feng-566/tekton-cleanup:v0.0.1

build-pull:
	cd src && \
    docker build -t $(IMG_PULL) -f pull.Dockerfile . && \
	docker push $(IMG_PULL)

build-cleanup:
	cd src && \
    docker build -t $(IMG_CLEANUP) -f cleanup.Dockerfile . && \
	docker push $(IMG_CLEANUP)
