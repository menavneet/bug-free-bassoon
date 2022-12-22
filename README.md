# bug-free-bassoon
A simple go server created with chat CPT


# Deployment 
To deploy a Go-based REST API to a Kubernetes cluster using k9s, you can create a deployment script like the following:
```
# Build the image
docker build -t go-api .

# Push the image to a registry (e.g. Docker Hub or Google Container Registry)
docker push go-api

# Deploy the image to the cluster
kubectl apply -f deployment.yaml
```


This script builds the Docker image for the API, pushes it to a registry, and deploys it to the cluster using kubectl.

The deployment.yaml file should contain a Kubernetes Deployment resource that specifies the image to deploy and any other desired configurations, such as the number of replicas and resource limits.

Here is an example deployment.yaml file:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-api
  labels:
    app: go-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: go-api
  template:
    metadata:
      labels:
        app: go-api
    spec:
      containers:
      - name: go-api
        image: go-api
        ports:
        - containerPort: 8080
      resources:
        limits:
          memory: "128Mi"
          cpu: "500m"
```
This deployment creates two replicas of the go-api container and exposes it on port 8080. It also sets resource limits for memory and CPU.

To deploy the API using this script, run the following command:

    $ ./deploy.sh

You can then use k9s to view and manage the deployment:

    $ k9s

Press d to view the deployments, and use the arrow keys to navigate and select the go-api deployment. Press Enter to view the details, and use the q key to exit.