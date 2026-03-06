# ADR-004: O-RAN CNF Source Selection

**Status:** Accepted | **Date:** 2025-03-06

## Context
Need containerized O-RAN components (O-DU, O-CU-CP, O-CU-UP) with E2 agent support for Near-RT RIC integration. Must run as Kubernetes CNFs.

## Decision
- **O-DU / O-CU-CP / O-CU-UP**: OpenAirInterface (OAI) `openairinterface5g` — mature, container-ready, E2AP built into gNB
- **5G Core**: OAI CN5G (AMF, SMF, UPF, NRF, NSSF) — consistent ecosystem
- Container images from OAI official registry or built from source

## Alternatives Considered
- srsRAN Project: less mature container/E2 support at time of decision
- VIAVI / Radisys commercial: license cost, not open-source

## Consequences
- Dependent on OAI release cycle for bug fixes and E2 enhancements
- Track OAI `develop` branch for latest container support
- OAI gNB supports E2AP with E2SM-KPM and E2SM-RC service models
