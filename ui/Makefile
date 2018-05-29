REGISTRY ?= dippynark
APP_NAME ?= kubecast-ui

docker: docker_build docker_push

docker_build:
	docker build -t dippynark/kubecast-ui .

docker_push:
	docker push dippynark/kubecast-ui