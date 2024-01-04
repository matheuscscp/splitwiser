.PHONY: bootstrap
bootstrap:
	cd .github/workflows/ && terraform init && terraform apply
