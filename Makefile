docker_image_build_macos:
	DOCKER_BUILDKIT=0 docker build -t yvv4docker/speedtest-macos .

docker_image_build_linux_x64:
	docker buildx build --platform linux/amd64 -t yvv4docker/speedtest-linux .

compose_up_local:
	docker compose -f docker-compose.local.yml up -d

compose_down_local:
	docker compose -f docker-compose.local.yml down