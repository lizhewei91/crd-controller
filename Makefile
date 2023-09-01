TARGET_PLATFORMS ?= linux/amd64,linux/arm64
BASE_IMAGE ?= golang
BASE_IMAGE_VERSION ?= alpine3.17
IMAGE_REPO ?= ${DOCKER_HUB_REPO}/crd-controller
IMAGE_VERSION ?= v0.0.1
DOCKER_HUB_REPO ?= hub.xxx.cn
DOCKER_HUB_USERNAME ?= xxx
DOCKER_HUB_PASSWORD ?= xxx


# debug模式下，使用的kubeconfig文件
KUBECONFIG ?= hack/debug/91-cke-068nrd4t2h5mh5fq8tjs-kubeconfig
# 创建debug-all-in-one
DEBUG_DEPLOY ?= hack/debug/debug-all-in-one.yaml

.PHONY: all
all: docker-hub-login build images

docker-hub-login:
	docker logout
	docker login ${DOCKER_HUB_REPO} -u ${DOCKER_HUB_USERNAME} -p ${DOCKER_HUB_PASSWORD}

build:
	go build -o /go/src/crd-controller/bin/crd-controller main.go

images:
	docker buildx build \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg BASE_IMAGE_VERSION=$(BASE_IMAGE_VERSION) \
		--platform $(TARGET_PLATFORMS) \
		-t $(IMAGE_REPO):$(IMAGE_VERSION) \
		-f ./Dockfile.buildx --push .