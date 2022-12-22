IMAGE_NAME=gcr.io/<project-id>/go-api

build:
	docker build -t $(IMAGE_NAME) .

push: build
	gcloud auth configure-docker
	docker push $(IMAGE_NAME)

deploy: push
	kubectl apply -f deployment.yaml
