#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ALLOWLIST_FILE="${ROOT_DIR}/ci/root-allowlist.txt"
CANDIDATES_FILE="${ROOT_DIR}/ci/root-cleanup-candidates.txt"

if [[ ! -f "${ALLOWLIST_FILE}" ]]; then
  echo "ERROR: allowlist file not found: ${ALLOWLIST_FILE}" >&2
  exit 1
fi

current_entries="$(git -C "${ROOT_DIR}" ls-files | awk -F'/' '{print $1}' | sort -u)"
allowed_entries="$(grep -vE '^\s*#|^\s*$' "${ALLOWLIST_FILE}" | sort)"

extra_entries="$(comm -23 <(printf '%s\n' "${current_entries}") <(printf '%s\n' "${allowed_entries}"))"
missing_entries="$(comm -13 <(printf '%s\n' "${current_entries}") <(printf '%s\n' "${allowed_entries}"))"

echo "Root allowlist validation"
echo "  current entries: $(printf '%s\n' "${current_entries}" | sed '/^$/d' | wc -l | tr -d ' ')"
echo "  allowlist entries: $(printf '%s\n' "${allowed_entries}" | sed '/^$/d' | wc -l | tr -d ' ')"

status=0
if [[ -n "${extra_entries}" ]]; then
  echo ""
  echo "Unexpected root entries (not in allowlist):"
  printf '  - %s\n' ${extra_entries}
  status=1
fi

if [[ -n "${missing_entries}" ]]; then
  echo ""
  echo "Allowlist entries missing from root:"
  printf '  - %s\n' ${missing_entries}
  status=1
fi

if [[ ${status} -eq 0 ]]; then
  echo ""
  echo "PASS: Root entries match allowlist."
fi

if [[ -f "${CANDIDATES_FILE}" ]]; then
  echo ""
  echo "Phase 3 candidates:"
  awk -F'|' 'BEGIN { OFS=" | " } !/^\s*#/ && NF>=4 && $3 !~ /^intentional-root-exception$/ { print "  - " $1, $2, $3, $4 }' "${CANDIDATES_FILE}"

  echo ""
  echo "Intentional root exceptions (approved):"
  awk -F'|' 'BEGIN { OFS=" | " } !/^\s*#/ && NF>=4 && $3 ~ /^intentional-root-exception$/ { print "  - " $1, "[" $4 "]" }' "${CANDIDATES_FILE}"
fi

exit ${status}
