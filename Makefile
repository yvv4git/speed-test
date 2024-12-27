docker_image_build_macos:
	DOCKER_BUILDKIT=0 docker build -t yvv4docker/speedtest .

docker_image_build_linux_x64:
	docker buildx build --platform linux/amd64 -t yvv4docker/speedtest .

compose_up_local:
	docker compose up -d

compose_down_local:
	docker compose down

compose_up_server_srv:
	docker compose up -d  --scale client=0

compose_up_server_cli:
	docker compose up -d server --scale server=0 --scale prometheus=0 --scale grafana=0