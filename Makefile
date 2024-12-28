build_macos:
	DOCKER_BUILDKIT=0 docker build -t yvv4docker/speedtest .

build_linux:
	docker buildx build --platform linux/amd64 -t yvv4docker/speedtest .

compose_up_local:
	docker compose up -d

compose_down_local:
	docker compose down

compose_up_server:
	docker compose up -d  --scale client=0

compose_up_client:
	docker compose up -d --scale server=0 --scale prometheus=0 --scale grafana=0