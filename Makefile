# Existing Makefile content... (keep all previous content)

.PHONY: validate-crds
validate-crds: manifests
	@echo "Validating generated CRDs..."
	@for crd in config/crd/bases/*.yaml; do \
		echo "Validating $$crd..."; \
		kubeval --strict $$crd || exit 1; \
	done
	@echo "[SUCCESS] All CRDs are valid"