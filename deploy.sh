# Build the image
docker build -t go-api .

# Push the image to a registry (e.g. Docker Hub or Google Container Registry)
docker push go-api

# Deploy the image to the cluster
kubectl apply -f deployment.yaml
