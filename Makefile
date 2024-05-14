# import deploy config
# You can change the default deploy config with `make cnf="deploy_special.env" release`
dpl ?= deploy.env
include $(dpl)
export $(shell sed 's/=.*//' $(dpl))

.PHONY: build

build:
	go mod tidy
	go mod vendor
	cd cmd && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ../artifacts/${BINARY_NAME}

image: build
	docker build -f build/Dockerfile -t ${DOCKER_REPO_PATH}/${DOCKER_REPO_NAME}:${DOCKER_TAG} .

auth:
	aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $(DOCKER_REPO_PATH)

push: image auth
	docker push ${DOCKER_REPO_PATH}/${DOCKER_REPO_NAME}:${DOCKER_TAG}
