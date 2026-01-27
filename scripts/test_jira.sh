#!/bin/bash
# test_jira.sh - Jira API 연결 테스트

set -e

ISSUE_KEY=${1:-""}

if [ -z "$ISSUE_KEY" ]; then
    echo "Usage: ./scripts/test_jira.sh <ISSUE_KEY>"
    echo "Example: ./scripts/test_jira.sh ITSM-5239"
    exit 1
fi

# config.ini에서 설정 읽기
JIRA_URL=$(grep "^url" config.ini | cut -d'=' -f2 | tr -d ' ')
JIRA_EMAIL=$(grep "^email" config.ini | cut -d'=' -f2 | tr -d ' ')
JIRA_API_KEY=$(grep "^api_key" config.ini | cut -d'=' -f2 | tr -d ' ')

echo "🔍 Jira 이슈 조회: ${ISSUE_KEY}"
echo "📡 URL: ${JIRA_URL}/rest/api/3/issue/${ISSUE_KEY}"

# API 호출
curl -s -u "${JIRA_EMAIL}:${JIRA_API_KEY}" \
    -H "Accept: application/json" \
    "${JIRA_URL}/rest/api/3/issue/${ISSUE_KEY}" | jq '.key, .fields.summary'
