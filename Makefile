docker-build:
	docker build -t bumbacea/k8s-sidecar .
docker-push: docker-build
	docker push bumbacea/k8s-sidecar