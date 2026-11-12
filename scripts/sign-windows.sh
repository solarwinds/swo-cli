#!/usr/bin/env bash
# Authenticode-signs a Windows binary via Azure Trusted Signing using jsign, then verifies the signature chains to ATS_ROOT_CA.
# Usage: sign-windows.sh <path-to-exe>
# Requires ATS_ACCESS_TOKEN, ATS_ENDPOINT, ATS_ACCOUNT, ATS_CERT_PROFILE, JSIGN_JAR and ATS_ROOT_CA (PEM) plus osslsigncode on PATH.
# Skips when ATS_ACCESS_TOKEN is unset unless REQUIRE_SIGNING=1.
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

: "${ATS_ENDPOINT:?ATS_ENDPOINT must be set}"
: "${ATS_ACCOUNT:?ATS_ACCOUNT must be set}"
: "${ATS_CERT_PROFILE:?ATS_CERT_PROFILE must be set}"
: "${JSIGN_JAR:?JSIGN_JAR must be set}"
: "${ATS_ROOT_CA:?ATS_ROOT_CA must be set}"

# Token is read from the environment so it never appears in the process list.
java -jar "${JSIGN_JAR}" \
  --storetype TRUSTEDSIGNING \
  --keystore "${ATS_ENDPOINT}" \
  --storepass env:ATS_ACCESS_TOKEN \
  --alias "${ATS_ACCOUNT}/${ATS_CERT_PROFILE}" \
  --tsaurl http://timestamp.acs.microsoft.com \
  --tsmode RFC3161 \
  "${file}"

# Signing cert and timestamp both chain to the Microsoft Identity Verification Root CA 2020.
osslsigncode verify \
  -CAfile "${ATS_ROOT_CA}" \
  -TSA-CAfile "${ATS_ROOT_CA}" \
  -in "${file}"

echo "Signed and verified ${file}"
