IMAGE_NAME=gcr.io/<project-id>/go-api

build:
	docker build -t $(IMAGE_NAME) .

push: build
	gcloud auth configure-docker
	docker push $(IMAGE_NAME)

deploy: push
	kubectl apply -f deployment.yaml

IMAGE_NAME=<username>/go-api

docker-hub-build:
	docker build -t $(IMAGE_NAME) .

docker-hub-push: build
	echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin
	docker push $(IMAGE_NAME)
