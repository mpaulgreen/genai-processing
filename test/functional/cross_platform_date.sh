#!/bin/bash
# Cross-platform date utilities for OpenShift Audit Queries
# Source: PRD Appendices Usage Notes section

get_yesterday() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-1d '+%Y-%m-%d'
    else
        date -d yesterday '+%Y-%m-%d'
    fi
}

get_today() {
    date '+%Y-%m-%d'
}

get_hours_ago() {
    local hours=$1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-${hours}H -u +%Y-%m-%dT%H:%M:%SZ
    else
        date -d "${hours} hours ago" --iso-8601
    fi
}

get_days_ago_iso() {
    local days=$1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-${days}d -u +%Y-%m-%dT%H:%M:%SZ
    else
        date -d "${days} days ago" --iso-8601
    fi
}

# Additional utility functions for comprehensive time handling
get_minutes_ago() {
    local minutes=$1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-${minutes}M -u +%Y-%m-%dT%H:%M:%SZ
    else
        date -d "${minutes} minutes ago" --iso-8601
    fi
}

get_weeks_ago() {
    local weeks=$1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-${weeks}w -u +%Y-%m-%dT%H:%M:%SZ
    else
        date -d "${weeks} weeks ago" --iso-8601
    fi
}

# Advanced time functions for intermediate queries
get_time_range_start() {
    local hours_back=$1
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -v-${hours_back}H -u +%Y-%m-%dT%H:%M:%SZ
    else
        date -d "${hours_back} hours ago" --iso-8601=seconds
    fi
}

get_time_range_end() {
    local hours_back=${1:-0}
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if [ "$hours_back" -eq 0 ]; then
            date -u +%Y-%m-%dT%H:%M:%SZ
        else
            date -v-${hours_back}H -u +%Y-%m-%dT%H:%M:%SZ
        fi
    else
        if [ "$hours_back" -eq 0 ]; then
            date --iso-8601=seconds
        else
            date -d "${hours_back} hours ago" --iso-8601=seconds
        fi
    fi
}

get_business_hours_start() {
    local date_arg="${1:-today}"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if [ "$date_arg" = "today" ]; then
            date '+%Y-%m-%dT09:00:00Z'
        else
            date -v-1d '+%Y-%m-%dT09:00:00Z'
        fi
    else
        if [ "$date_arg" = "today" ]; then
            date '+%Y-%m-%dT09:00:00Z'
        else
            date -d yesterday '+%Y-%m-%dT09:00:00Z'
        fi
    fi
}

get_business_hours_end() {
    local date_arg="${1:-today}"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if [ "$date_arg" = "today" ]; then
            date '+%Y-%m-%dT17:00:00Z'
        else
            date -v-1d '+%Y-%m-%dT17:00:00Z'
        fi
    else
        if [ "$date_arg" = "today" ]; then
            date '+%Y-%m-%dT17:00:00Z'
        else
            date -d yesterday '+%Y-%m-%dT17:00:00Z'
        fi
    fi
}

get_hour_window() {
    local target_hour=$1
    local date_offset=${2:-0}
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if [ "$date_offset" -eq 0 ]; then
            printf "%s-%02d:00:00Z\n" "$(date '+%Y-%m-%dT')" "$target_hour"
        else
            printf "%s-%02d:00:00Z\n" "$(date -v-${date_offset}d '+%Y-%m-%dT')" "$target_hour"
        fi
    else
        if [ "$date_offset" -eq 0 ]; then
            printf "%s-%02d:00:00Z\n" "$(date '+%Y-%m-%dT')" "$target_hour"
        else
            printf "%s-%02d:00:00Z\n" "$(date -d "${date_offset} days ago" '+%Y-%m-%dT')" "$target_hour"
        fi
    fi
}