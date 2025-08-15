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