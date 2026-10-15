# One-command deploy for the geoconnex knowledge-and-insights demo.
#
#   make deploy SAL_PROJECT_REPO=https://github.com/you/your-sal-project
#
# runs `terraform apply` (creates/updates the Artifact Registry repo,
# service accounts, and the Cloud Run service shell), then
# `gcloud builds submit` (harvest via geoconnex-salmodule -> bake the serve
# image -> deploy it), feeding the build its substitutions straight from
# that apply's outputs instead of you having to copy them by hand.
#
# Prerequisites: terraform and gcloud on PATH, gcloud already
# authenticated (`gcloud auth login` + `gcloud config set project ...`,
# or equivalent), and demo/terraform.tfvars filled in (see
# terraform.tfvars.example).
#
# Run the two halves separately with `make apply` / `make build` if you
# only want one of them — e.g. `make build` again on its own to re-harvest
# and redeploy without touching the infrastructure.

SHELL := /bin/bash
TF_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ROOT_DIR := $(abspath $(TF_DIR)/..)
TF := terraform -chdir=$(TF_DIR)

# Passed through to `terraform apply` / `terraform destroy`. Left blank by
# default so those still ask for confirmation before touching real GCP
# resources; set to -auto-approve for non-interactive use (e.g. CI).
TF_APPLY_FLAGS ?=

# Required for build/deploy: the existing SAL project repo that declares
# the JsonLdHarvester task instance with your ~10 sitemap ids (see
# example-task.ttl). Not managed by Terraform, since it's your repo.
SAL_PROJECT_REPO ?=
SAL_BUILD_PATH ?= data/

.PHONY: help init plan apply build deploy url destroy check-tools check-tfvars check-repo

help:
	@echo "make deploy SAL_PROJECT_REPO=<git url>   # terraform apply, then harvest/build/deploy"
	@echo "make apply                               # terraform apply only"
	@echo "make build SAL_PROJECT_REPO=<git url>     # harvest/build/deploy only (needs a prior apply)"
	@echo "make url                                 # print the deployed Cloud Run URL"
	@echo "make plan                                # terraform plan"
	@echo "make destroy                             # terraform destroy"

check-tools:
	@command -v terraform >/dev/null || { echo "error: terraform not on PATH"; exit 1; }
	@command -v gcloud >/dev/null || { echo "error: gcloud not on PATH"; exit 1; }

check-tfvars: check-tools
	@test -f "$(TF_DIR)terraform.tfvars" || { \
		echo "error: $(TF_DIR)terraform.tfvars not found."; \
		echo "       cp demo/terraform.tfvars.example demo/terraform.tfvars, fill it in, and retry."; \
		exit 1; \
	}

check-repo:
	@test -n "$(SAL_PROJECT_REPO)" || { \
		echo "error: SAL_PROJECT_REPO is required, e.g.:"; \
		echo "       make build SAL_PROJECT_REPO=https://github.com/you/your-sal-project"; \
		exit 1; \
	}

init: check-tfvars
	$(TF) init

plan: init
	$(TF) plan

apply: init
	$(TF) apply $(TF_APPLY_FLAGS)

# Substitutions are read back from the apply this depends on, so there is
# nothing to keep in sync by hand between Terraform and the build.
build: check-repo init
	gcloud builds submit "$(ROOT_DIR)" \
		--config=demo/cloudbuild.yaml \
		--service-account="projects/$$($(TF) output -raw project_id)/serviceAccounts/$$($(TF) output -raw sal_builder_service_account_email)" \
		--substitutions="_SAL_PROJECT_REPO=$(SAL_PROJECT_REPO),_SAL_BUILD_PATH=$(SAL_BUILD_PATH),_AR_IMAGE=$$($(TF) output -raw ar_image),_SERVICE_NAME=$$($(TF) output -raw service_name),_REGION=$$($(TF) output -raw region)"

deploy: apply build url

url:
	@echo "$$($(TF) output -raw cloud_run_url)"

destroy: init
	$(TF) destroy $(TF_APPLY_FLAGS)
