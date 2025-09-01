package main

import (
	_ "github.com/thc1006/nephoran-intent-operator/internal/intent"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/templates"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/validation/yang"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/monitoring/reporting"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/common"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/ims"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/o1/security"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/ims/modeladapter"
)

func main() {}
