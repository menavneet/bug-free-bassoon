IMAGE_NAME_GCR=gcr.io/<project-id>/go-api
IMAGE_NAME_DOCKER_HUB=<username>/go-api

build:
	docker build -t $(IMAGE_NAME_GCR) .

push: build
	gcloud auth configure-docker
	docker push $(IMAGE_NAME_GCR)

deploy: push
	kubectl apply -f deployment.yaml


docker-hub-build:
	docker build -t $(IMAGE_NAME_DOCKER_HUB) .

docker-hub-push: build
	echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin
	docker push $(IMAGE_NAME_DOCKER_HUB)
