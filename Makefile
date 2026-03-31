.PHONY: validate test ci

validate:
	docker buildx build --target validate -f Dockerfile .

test:
	docker buildx build --target test-artifacts --output type=local,dest=. -f Dockerfile .

ci:
	docker buildx build --target ci-artifacts --output type=local,dest=. -f Dockerfile .

.DEFAULT_GOAL := ci
