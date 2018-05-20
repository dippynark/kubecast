REGISTRY ?= dippynark
APP_NAME ?= kubepf-ui

docker: docker_build docker_push

docker_build:
	docker build -t dippynark/kubepf-ui .

docker_push:
	docker push dippynark/kubepf-ui