#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-coverage.out}"
THRESHOLD="${2:-95}"

if [[ ! -f "$PROFILE" ]]; then
  echo "coverage profile not found: $PROFILE" >&2
  exit 1
fi

GATED=(
  "github.com/ashisharyan/ghostwriter-prompt-engine/db"
  "github.com/ashisharyan/ghostwriter-prompt-engine/handlers"
  "github.com/ashisharyan/ghostwriter-prompt-engine/services"
  "github.com/ashisharyan/ghostwriter-prompt-engine/router"
  "github.com/ashisharyan/ghostwriter-prompt-engine/utils"
)

fail=0
for g in "${GATED[@]}"; do
  avg=$(go tool cover -func="$PROFILE" | awk -v pkg="$g/" '
    $0 ~ pkg { sum += $(NF); n++ }
    END {
      if (n == 0) { print -1; exit }
      print sum / n
    }')
  if awk -v a="$avg" 'BEGIN { exit !(a < 0) }'; then
    echo "WARN no coverage data for $g" >&2
    continue
  fi
  if awk -v a="$avg" -v t="$THRESHOLD" 'BEGIN { exit !(a+0 < t+0) }'; then
    echo "FAIL package below ${THRESHOLD}%: $g (${avg}%)" >&2
    fail=1
  else
    printf "OK %s %.1f%%\n" "$g" "$avg"
  fi
done

while IFS= read -r line; do
  [[ "$line" == total:* ]] && continue
  pct=$(echo "$line" | awk '{print $NF}' | tr -d '%')
  fn=$(echo "$line" | awk '{print $1}')
  pkg=$(echo "$fn" | sed 's|/[^/]*\.go:.*||')
  skip=1
  for g in "${GATED[@]}"; do
    if [[ "$pkg" == "$g" ]]; then
      skip=0
      break
    fi
  done
  [[ "$skip" -eq 1 ]] && continue
  awk -v p="$pct" -v t="$THRESHOLD" -v l="$line" 'BEGIN {
    if (p+0 < t+0) { print "FAIL function below " t "%: " l > "/dev/stderr"; exit 1 }
  }' || fail=1
done < <(go tool cover -func="$PROFILE" | grep -v "^total:")

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "Coverage gate passed at ${THRESHOLD}% (package + function) for gated packages."
