#!/usr/bin/env bash

set -u
set -o pipefail

TEST_REPORT_DIR="${TEST_REPORT_DIR:-test}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"

mkdir -p "$TEST_REPORT_DIR"

COMBINED_PROFILE="$TEST_REPORT_DIR/coverage.out"
COMBINED_COVERAGE_TXT="$TEST_REPORT_DIR/coverage.txt"
COMBINED_COVERAGE_HTML="$TEST_REPORT_DIR/coverage.html"
SUMMARY_FILE="$TEST_REPORT_DIR/summary.txt"

rm -rf "$TEST_REPORT_DIR"
mkdir -p "$TEST_REPORT_DIR"

SERVICES=(
  "api-gateway:./services/api-gateway/internal/..."
  "user-service:./services/user-service/internal/..."
  "content-service:./services/content-service/internal/..."
  "feed-service:./services/feed-service/internal/..."
  "notification-service:./services/notification-service/internal/..."
  "storage-service:./services/storage-service/internal/..."
)

TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0
FAILED_SERVICES=0

PROFILES=()

printf "\n%-24s %10s %10s %10s %10s %12s\n" "SERVICE" "TESTS" "PASSED" "FAILED" "SKIPPED" "COVERAGE" > "$SUMMARY_FILE"
printf "%-24s %10s %10s %10s %10s %12s\n" "------------------------" "----------" "----------" "----------" "----------" "------------" >> "$SUMMARY_FILE"

for entry in "${SERVICES[@]}"; do
  SERVICE_NAME="${entry%%:*}"
  PACKAGE_PATTERN="${entry#*:}"
  SERVICE_DIR="$TEST_REPORT_DIR/$SERVICE_NAME"

  mkdir -p "$SERVICE_DIR"

  PACKAGES="$(go list "$PACKAGE_PATTERN" 2>/dev/null \
    | grep -v '/cmd/' \
    | grep -v '/docs' \
    | grep -v '/proto/' \
    | grep -v '/provider$' || true)"

  JSON_LOG="$SERVICE_DIR/test.json"
  SERVICE_PROFILE="$SERVICE_DIR/coverage.out"
  SERVICE_COVERAGE_TXT="$SERVICE_DIR/coverage.txt"
  SERVICE_COVERAGE_HTML="$SERVICE_DIR/coverage.html"

  if [[ -z "$PACKAGES" ]]; then
    printf "%-24s %10d %10d %10d %10d %12s\n" \
      "$SERVICE_NAME" 0 0 0 0 "0.0%" >> "$SUMMARY_FILE"
    continue
  fi

  echo ""
  echo "============================================================"
  echo "Running tests for $SERVICE_NAME"
  echo "Output: $SERVICE_DIR"
  echo "============================================================"

  go test $PACKAGES \
    -json \
    -covermode=atomic \
    -coverprofile="$SERVICE_PROFILE" 2>&1 | tee "$JSON_LOG"

  TEST_EXIT=${PIPESTATUS[0]}

  if [[ "$TEST_EXIT" -ne 0 ]]; then
    FAILED_SERVICES=$((FAILED_SERVICES + 1))
  fi

  COUNTS="$(python3 - "$JSON_LOG" <<'PY'
import json
import sys

path = sys.argv[1]
tests = {}

with open(path, "r", encoding="utf-8", errors="ignore") as f:
    for line in f:
        line = line.strip()

        if not line.startswith("{"):
            continue

        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue

        test_name = event.get("Test")
        action = event.get("Action")

        if not test_name:
            continue

        if action in ("pass", "fail", "skip"):
            tests[test_name] = action

passed = sum(1 for status in tests.values() if status == "pass")
failed = sum(1 for status in tests.values() if status == "fail")
skipped = sum(1 for status in tests.values() if status == "skip")
total = passed + failed + skipped

print(total, passed, failed, skipped)
PY
)"

  read -r SERVICE_TOTAL SERVICE_PASSED SERVICE_FAILED SERVICE_SKIPPED <<< "$COUNTS"

  TOTAL_TESTS=$((TOTAL_TESTS + SERVICE_TOTAL))
  TOTAL_PASSED=$((TOTAL_PASSED + SERVICE_PASSED))
  TOTAL_FAILED=$((TOTAL_FAILED + SERVICE_FAILED))
  TOTAL_SKIPPED=$((TOTAL_SKIPPED + SERVICE_SKIPPED))

  if [[ -s "$SERVICE_PROFILE" ]]; then
    go tool cover -func="$SERVICE_PROFILE" | tee "$SERVICE_COVERAGE_TXT" >/dev/null
    go tool cover -html="$SERVICE_PROFILE" -o "$SERVICE_COVERAGE_HTML"

    SERVICE_COVERAGE="$(awk '/total:/ { print $3 }' "$SERVICE_COVERAGE_TXT")"
    PROFILES+=("$SERVICE_PROFILE")
  else
    SERVICE_COVERAGE="0.0%"
    echo "No coverage profile generated." > "$SERVICE_COVERAGE_TXT"
  fi

  printf "%-24s %10d %10d %10d %10d %12s\n" \
    "$SERVICE_NAME" \
    "$SERVICE_TOTAL" \
    "$SERVICE_PASSED" \
    "$SERVICE_FAILED" \
    "$SERVICE_SKIPPED" \
    "$SERVICE_COVERAGE" >> "$SUMMARY_FILE"
done

if [[ "${#PROFILES[@]}" -gt 0 ]]; then
  echo "mode: atomic" > "$COMBINED_PROFILE"

  for profile in "${PROFILES[@]}"; do
    tail -n +2 "$profile" >> "$COMBINED_PROFILE"
  done

  go tool cover -func="$COMBINED_PROFILE" | tee "$COMBINED_COVERAGE_TXT" >/dev/null
  go tool cover -html="$COMBINED_PROFILE" -o "$COMBINED_COVERAGE_HTML"

  TOTAL_COVERAGE="$(awk '/total:/ { print $3 }' "$COMBINED_COVERAGE_TXT")"
else
  TOTAL_COVERAGE="0.0%"
  echo "No combined coverage profile generated." > "$COMBINED_COVERAGE_TXT"
fi

printf "%-24s %10s %10s %10s %10s %12s\n" "------------------------" "----------" "----------" "----------" "----------" "------------" >> "$SUMMARY_FILE"

printf "%-24s %10d %10d %10d %10d %12s\n" \
  "TOTAL" \
  "$TOTAL_TESTS" \
  "$TOTAL_PASSED" \
  "$TOTAL_FAILED" \
  "$TOTAL_SKIPPED" \
  "$TOTAL_COVERAGE" >> "$SUMMARY_FILE"

echo ""
echo "============================================================"
echo "TEST SUMMARY"
echo "============================================================"
cat "$SUMMARY_FILE"

echo ""
echo "Reports generated:"
echo "  Total summary:        $SUMMARY_FILE"
echo "  Total coverage text:  $COMBINED_COVERAGE_TXT"
echo "  Total coverage HTML:  $COMBINED_COVERAGE_HTML"
echo ""

for entry in "${SERVICES[@]}"; do
  SERVICE_NAME="${entry%%:*}"
  SERVICE_DIR="$TEST_REPORT_DIR/$SERVICE_NAME"

  if [[ -d "$SERVICE_DIR" ]]; then
    echo "  $SERVICE_NAME:"
    echo "    Test JSON:      $SERVICE_DIR/test.json"
    echo "    Coverage text: $SERVICE_DIR/coverage.txt"
    echo "    Coverage HTML: $SERVICE_DIR/coverage.html"
  fi
done

COVERAGE_NUMBER="${TOTAL_COVERAGE%\%}"

python3 - "$COVERAGE_NUMBER" "$COVERAGE_THRESHOLD" <<'PY'
import sys

coverage = float(sys.argv[1])
threshold = float(sys.argv[2])

print()

if coverage < threshold:
    print(f"Coverage check failed: {coverage:.1f}% < {threshold:.1f}%")
    sys.exit(1)

print(f"Coverage check passed: {coverage:.1f}% >= {threshold:.1f}%")
PY

COVERAGE_EXIT=$?

if [[ "$FAILED_SERVICES" -ne 0 ]]; then
  echo ""
  echo "Test run failed: $FAILED_SERVICES service test group(s) failed."
  exit 1
fi

exit "$COVERAGE_EXIT"