MACHINE := longhorn

.PHONY: validate test ci

buildx-machine:
	@docker buildx create --name=$(MACHINE) --buildkitd-flags '--allow-insecure-entitlement security.insecure' 2>/dev/null || true

validate:
	docker buildx build --target validate -f Dockerfile .

test: buildx-machine
	docker buildx build --builder=$(MACHINE) --allow security.insecure --target test-artifacts --output type=local,dest=. -f Dockerfile .

ci: buildx-machine
	docker buildx build --builder=$(MACHINE) --allow security.insecure --target ci-artifacts --output type=local,dest=. -f Dockerfile .

.DEFAULT_GOAL := ci
