.PHONY: all build build-proxy build-mock build-loadtest run clean docker-compose-up docker-compose-down

PROXY_IMAGE = my-proxy
MOCK_IMAGE = my-mock
LOADTEST_IMAGE = my-loadtest

DOCKER_COMPOSE_FILE = docker-compose.yml

all: build

build: build-proxy build-mock build-loadtest

build-proxy:
	docker build -f Dockerfile -t $(PROXY_IMAGE) .

build-mock:
	docker build -f Dockerfile_mock -t $(MOCK_IMAGE) .

build-loadtest:
	docker build -f Dockerfile_loadtest -t $(LOADTEST_IMAGE) .

docker-compose-up:
	docker-compose -f $(DOCKER_COMPOSE_FILE) up --build

docker-compose-down:
	docker-compose -f $(DOCKER_COMPOSE_FILE) down --rmi all

run: docker-compose-up

clean: docker-compose-down