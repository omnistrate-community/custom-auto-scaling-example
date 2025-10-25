SERVICE_NAME=service-test
SERVICE_PLAN=service-test
MAIN_RESOURCE_NAME=web
ENVIRONMENT=Dev
AWS_CLOUD_PROVIDER=aws
AWS_REGION=ap-south-1
GCP_CLOUD_PROVIDER=gcp
GCP_REGION=us-central1
AZURE_CLOUD_PROVIDER=azure
AZURE_REGION=eastus2

# Load variables from .env if it exists
ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

.PHONY: install-ctl
install-ctl:
	@brew tap omnistrate/tap
	@brew install omnistrate/tap/omnistrate-ctl

.PHONY: upgrade-ctl
upgrade-ctl:
	@brew upgrade omnistrate/tap/omnistrate-ctl
	
.PHONY: login
login:
	@cat ./.omnistrate.password | omnistrate-ctl login --email $(OMNISTRATE_EMAIL) --password-stdin

.PHONY: release
release:
	@omnistrate-ctl build -f compose.yaml --product-name ${SERVICE_NAME}  --environment ${ENVIRONMENT} --environment-type ${ENVIRONMENT} --release-as-preferred

.PHONY: create-aws
create-aws:
	@omnistrate-ctl instance create --environment ${ENVIRONMENT} --cloud-provider ${AWS_CLOUD_PROVIDER} --region ${AWS_REGION} --plan ${SERVICE_PLAN} --service ${SERVICE_NAME} --resource ${MAIN_RESOURCE_NAME} 

.PHONY: create-gcp
create-gcp:
	@omnistrate-ctl instance create --environment ${ENVIRONMENT} --cloud-provider ${GCP_CLOUD_PROVIDER} --region ${GCP_REGION} --plan ${SERVICE_PLAN} --service ${SERVICE_NAME} --resource ${MAIN_RESOURCE_NAME}

.PHONY: create-azure
create-azure:
	@omnistrate-ctl instance create --environment ${ENVIRONMENT} --cloud-provider ${AZURE_CLOUD_PROVIDER} --region ${AZURE_REGION} --plan ${SERVICE_PLAN} --service ${SERVICE_NAME} --resource ${MAIN_RESOURCE_NAME}

.PHONY: list
list:
	@omnistrate-ctl instance list --filter=service:${SERVICE_NAME},plan:${SERVICE_PLAN} --output json

.PHONY: delete-all
delete-all:
	@echo "Deleting all instances..."
	@for id in $$(omnistrate-ctl instance list --filter=service:${SERVICE_NAME},plan:${SERVICE_PLAN} --output json | jq -r '.[].instance_id'); do \
		echo "Deleting instance: $$id"; \
		omnistrate-ctl instance delete $$id; \
	done

.PHONY: destroy
destroy: delete-all-wait
	@echo "Destroying service: ${SERVICE_NAME}..."
	@omnistrate-ctl service delete ${SERVICE_NAME}

.PHONY: delete-all-wait
delete-all-wait:
	@echo "Deleting all instances and waiting for completion..."
	@instances_to_delete=$$(omnistrate-ctl instance list --filter=service:${SERVICE_NAME},plan:${SERVICE_PLAN} --output json | jq -r '.[].instance_id'); \
	if [ -n "$$instances_to_delete" ]; then \
		for id in $$instances_to_delete; do \
			echo "Deleting instance: $$id"; \
			omnistrate-ctl instance delete $$id; \
		done; \
		echo "Waiting for instances to be deleted..."; \
		while true; do \
			remaining=$$(omnistrate-ctl instance list --filter=service:${SERVICE_NAME},plan:${SERVICE_PLAN} --output json | jq -r '.[].instance_id'); \
			if [ -z "$$remaining" ]; then \
				echo "All instances deleted successfully"; \
				break; \
			fi; \
			echo "Still waiting for deletion to complete..."; \
			sleep 10; \
		done; \
	else \
		echo "No instances found to delete"; \
	fi