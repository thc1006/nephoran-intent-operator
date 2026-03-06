# Repo Structure (pre-generated)
```
.
├─ CLAUDE.md
├─ README.md
├─ Makefile
├─ pyproject.toml
├─ schemas/
│  ├─ intent-plan.schema.json
│  ├─ intent-plan.example.json
│  └─ policy.schema.json
├─ docs/
│  ├─ adr/
│  │  ├─ ADR-0001-northbound-tmf921.md
│  │  ├─ ADR-0002-porch-gitops-core.md
│  │  ├─ ADR-0003-inplace-hydration.md
│  │  ├─ ADR-0004-no-direct-kubectl.md
│  │  ├─ ADR-0005-ric-xapps-upstream.md
│  │  ├─ ADR-0006-e2sim-mvp.md
│  │  ├─ ADR-0007-naming-and-identity.md
│  │  └─ README.md
│  ├─ api/
│  │  └─ openapi.yaml
│  ├─ runbooks/
│  │  └─ closed-loop-demo.md
│  └─ sdd/
│     └─ system-design.md
├─ llm_nephio_oran/
│  ├─ __init__.py
│  ├─ intentd/
│  │  ├─ __init__.py
│  │  └─ app.py
│  ├─ intentctl.py
│  ├─ models.py
│  ├─ planner/
│  │  ├─ __init__.py
│  │  └─ stub_planner.py
│  ├─ generator/
│  │  ├─ __init__.py
│  │  └─ kpt_generator.py
│  ├─ gitops/
│  │  ├─ __init__.py
│  │  └─ pr_stub.py
│  ├─ observability/
│  │  ├─ __init__.py
│  │  └─ metrics_stub.py
│  └─ validators/
│     ├─ __init__.py
│     └─ schema_validate.py
├─ packages/
│  ├─ base/
│  │  ├─ README.md
│  │  └─ components/
│  │     ├─ ric-kpimon/README.md
│  │     ├─ ric-ts/README.md
│  │     ├─ sim-e2/README.md
│  │     └─ trafficgen/README.md
│  └─ instances/
│     └─ README.md
├─ hack/
│  ├─ bootstrap-kind.sh
│  └─ e2e.sh
└─ tests/
   └─ test_schema.py
```
