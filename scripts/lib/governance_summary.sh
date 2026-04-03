#!/usr/bin/env bash

governance_summary_trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s\n' "$value"
}

governance_summary_extract_status() {
  local target_file="$1"
  local line
  line="$(awk '/^- Status: `/ { print; exit }' "$target_file")"
  line="${line#- Status: \`}"
  line="${line%\`}"
  governance_summary_trim "$line"
}

governance_summary_extract_table_value() {
  local target_file="$1"
  local field="$2"
  awk -F'|' -v target="$field" '
    {
      left=$2
      right=$3
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", left)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", right)
      if (left == target) {
        print right
        exit
      }
    }
  ' "$target_file"
}

governance_summary_append_delta_highlights() {
  local delta_file="$1"
  local summary_output="$2"

  if [[ ! -f "$delta_file" ]]; then
    return 0
  fi

  {
    echo "### Delta Highlights"
    echo
    echo "- Regression Status: \`$(governance_summary_extract_status "$delta_file")\`"
    echo
  } >> "$summary_output"

  awk '
    /^## Regression Signals$/ { in_block=1; next }
    /^## / && in_block { exit }
    /^- Status: `/ && in_block { next }
    in_block { print }
  ' "$delta_file" >> "$summary_output"
}

governance_summary_append_failure_section() {
  local report_file="$1"
  local summary_output="$2"
  local heading="${3:-### Failure Summary}"
  local empty_message="${4:-- No failed scenarios.}"

  if awk '/^## Failure Summary$/ { found=1; exit } END { exit(found ? 0 : 1) }' "$report_file"; then
    awk -v heading="$heading" '
      /^## Failure Summary$/ { in_block=1; print heading; next }
      /^## / && in_block { exit }
      in_block { print }
    ' "$report_file" >> "$summary_output"
  else
    {
      echo "$heading"
      echo
      echo "$empty_message"
    } >> "$summary_output"
  fi
}

governance_summary_assert_inputs() {
  local log_prefix="$1"
  local report_file="$2"
  local summary_output="$3"

  if [[ ! -f "$report_file" ]]; then
    echo "$log_prefix missing report file: $report_file"
    exit 1
  fi

  if [[ -z "$summary_output" ]]; then
    echo "$log_prefix missing summary output target"
    exit 1
  fi
}
