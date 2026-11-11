#!/usr/bin/env bash
# Authenticode-signs a Windows binary via Azure Trusted Signing using jsign.
# Usage: sign-windows.sh <path-to-exe>
# Requires ATS_ACCESS_TOKEN, ATS_CERT_PROFILE and JSIGN_JAR; skips when ATS_ACCESS_TOKEN is unset unless REQUIRE_SIGNING=1.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <path-to-exe>" >&2
  exit 2
fi

file="$1"

if [[ -z "${ATS_ACCESS_TOKEN:-}" ]]; then
  if [[ "${REQUIRE_SIGNING:-}" == "1" ]]; then
    echo "ATS_ACCESS_TOKEN not set but REQUIRE_SIGNING=1, refusing to ship unsigned ${file}" >&2
    exit 1
  fi
  echo "ATS_ACCESS_TOKEN not set, skipping Authenticode signing of ${file}"
  exit 0
fi

: "${ATS_CERT_PROFILE:?ATS_CERT_PROFILE must be set}"
: "${JSIGN_JAR:?JSIGN_JAR must be set}"

ats_endpoint="eus.codesigning.azure.net"
ats_account="ts-github-prod-01"

# Token is read from the environment so it never appears in the process list.
java -jar "${JSIGN_JAR}" \
  --storetype TRUSTEDSIGNING \
  --keystore "${ats_endpoint}" \
  --storepass env:ATS_ACCESS_TOKEN \
  --alias "${ats_account}/${ATS_CERT_PROFILE}" \
  --tsaurl http://timestamp.acs.microsoft.com \
  --tsmode RFC3161 \
  "${file}"

echo "Signed ${file}"
