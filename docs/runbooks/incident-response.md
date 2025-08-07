# Nephoran Intent Operator - Incident Response Procedures

## Table of Contents
1. [Overview](#overview)
2. [Incident Severity Levels](#incident-severity-levels)
3. [Initial Triage Steps](#initial-triage-steps)
4. [Data Collection and Evidence Gathering](#data-collection-and-evidence-gathering)
5. [Root Cause Analysis](#root-cause-analysis)
6. [Communication Templates](#communication-templates)
7. [Post-Incident Procedures](#post-incident-procedures)
8. [Lessons Learned Integration](#lessons-learned-integration)
9. [Incident Response Tools](#incident-response-tools)

## Overview

This runbook defines the structured approach for responding to incidents affecting the Nephoran Intent Operator. The goal is to minimize service disruption, maintain stakeholder communication, and capture learnings for continuous improvement.

**Incident Response Objectives:**
- Restore service within SLA targets
- Minimize customer impact
- Preserve evidence for analysis
- Maintain clear communication
- Learn and improve from incidents

**Key Principles:**
- Safety first - avoid making the situation worse
- Speed with accuracy - balance quick resolution with proper analysis
- Communication - keep stakeholders informed
- Documentation - record all actions and findings

## Incident Severity Levels

### P1 - Critical (Complete Service Outage)

**Definition**: Service is completely unavailable or severely degraded affecting all users

**Examples**:
- All NetworkIntent processing stopped
- Controller completely down
- Data corruption or loss
- Security breach detected
- Multiple component failures

**SLA Targets**:
- Detection: < 2 minutes (automated)
- Response: < 5 minutes
- Communication: < 10 minutes
- Resolution: < 30 minutes

**Escalation**: Immediate management notification

### P2 - High (Significant Service Degradation)

**Definition**: Significant functionality impacted affecting majority of users

**Examples**:
- High error rate (>25%)
- Performance degradation (>5x normal)
- Key feature unavailable
- Single critical component failure

**SLA Targets**:
- Detection: < 5 minutes
- Response: < 15 minutes
- Communication: < 30 minutes
- Resolution: < 2 hours

**Escalation**: Team lead notification within 30 minutes

### P3 - Medium (Partial Service Impact)

**Definition**: Limited functionality affected impacting some users

**Examples**:
- Intermittent failures
- Non-critical feature issues
- Performance degradation (2-5x normal)
- Single component failures with redundancy

**SLA Targets**:
- Detection: < 15 minutes
- Response: < 1 hour
- Communication: < 2 hours
- Resolution: < 8 hours

### P4 - Low (Minor Issues)

**Definition**: Minor issues with minimal user impact

**Examples**:
- Non-user-facing issues
- Documentation problems
- Minor performance issues
- Cosmetic bugs

**SLA Targets**:
- Detection: < 1 hour
- Response: < 4 hours
- Communication: Next business day
- Resolution: < 24 hours

## Initial Triage Steps

### Step 1: Incident Declaration (2 minutes)

```bash
# Immediate actions upon incident detection
echo "=== INCIDENT DECLARED ==="
echo "Time: $(date -u)"
echo "Severity: [P1/P2/P3/P4]"
echo "Description: [Brief description]"
echo "Reporter: [Name/System]"

# Create incident tracking
INCIDENT_ID="INC-$(date +%Y%m%d-%H%M%S)"
echo "Incident ID: $INCIDENT_ID"

# Set up incident workspace
mkdir -p "/tmp/incidents/$INCIDENT_ID"
cd "/tmp/incidents/$INCIDENT_ID"
```

### Step 2: Initial Assessment (3 minutes)

```bash
# High-level system status check
echo "=== SYSTEM STATUS OVERVIEW ===" | tee system-status.log
kubectl get pods -n nephoran-system --sort-by=.status.startTime | tee -a system-status.log
kubectl get networkintents -A -o wide | tee -a system-status.log

# Check recent events
echo -e "\n=== RECENT EVENTS ===" | tee -a system-status.log
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20 | tee -a system-status.log

# Service health check
echo -e "\n=== SERVICE HEALTH ===" | tee -a system-status.log
for service in nephoran-controller llm-processor rag-api weaviate nephio-bridge; do
  echo "Checking $service..."
  kubectl get pods -n nephoran-system -l app=$service -o wide | tee -a system-status.log
done
```

### Step 3: Impact Assessment (2 minutes)

```bash
# Determine scope of impact
echo "=== IMPACT ASSESSMENT ===" | tee impact-assessment.log

# Check affected NetworkIntents
echo "Failed NetworkIntents:" | tee -a impact-assessment.log
kubectl get networkintents -A -o json | \
  jq -r '.items[] | select(.status.phase == "Failed" or (.status.phase == "Processing" and (.metadata.creationTimestamp | fromdateiso8601) < (now - 300))) | "\(.metadata.namespace)/\(.metadata.name) - \(.status.phase)"' | \
  tee -a impact-assessment.log

# Check error rates
echo -e "\nError Metrics:" | tee -a impact-assessment.log
kubectl port-forward -n nephoran-system svc/nephoran-controller 8081:8081 >/dev/null 2>&1 &
PF_PID=$!
sleep 2
curl -s http://localhost:8081/metrics 2>/dev/null | grep -E "(error_total|processing_duration)" | tee -a impact-assessment.log
kill $PF_PID 2>/dev/null
```

### Step 4: Initial Communication (5 minutes for P1/P2)

```bash
# Send initial incident notification
cat > initial-notification.md << EOF
## INCIDENT ALERT - $INCIDENT_ID

**Severity**: [P1/P2/P3/P4]
**Status**: Investigating
**Start Time**: $(date -u)
**Summary**: [Brief description of the issue]

### Impact
- [Describe user impact]
- [Affected components]
- [Current workarounds if any]

### Actions Taken
- Incident declared at $(date -u)
- Initial triage completed
- Investigation in progress

### Next Update
- Next update in [15/30/60] minutes

**Incident Commander**: [Name]
EOF

# Send notification (adjust channels based on severity)
echo "Send initial-notification.md to:"
echo "- Slack: #incidents (P1/P2)"
echo "- Email: on-call@company.com (P1)"
echo "- Status Page: Update for P1/P2"
```

## Data Collection and Evidence Gathering

### Automated Data Collection Script

```bash
# Create comprehensive data collection script
cat > collect-incident-data.sh << 'EOF'
#!/bin/bash
set -e

INCIDENT_ID=${1:-"INC-$(date +%Y%m%d-%H%M%S)"}
NAMESPACE="nephoran-system"
DATA_DIR="incident-data-$INCIDENT_ID"

echo "Collecting incident data for: $INCIDENT_ID"
mkdir -p "$DATA_DIR"
cd "$DATA_DIR"

# Timestamp all collection
echo "Data collection started: $(date -u)" > collection-timestamp.txt

# 1. System Overview
echo "=== Collecting System Overview ==="
kubectl cluster-info > cluster-info.txt
kubectl get nodes -o wide > nodes.txt
kubectl get namespaces > namespaces.txt

# 2. Nephoran Components
echo "=== Collecting Component Status ==="
kubectl get all -n "$NAMESPACE" -o wide > components-status.txt
kubectl describe pods -n "$NAMESPACE" > pods-detailed.txt
kubectl get configmaps -n "$NAMESPACE" -o yaml > configmaps.yaml
kubectl get secrets -n "$NAMESPACE" -o yaml > secrets-metadata.yaml 2>/dev/null || echo "Cannot access secrets"

# 3. Custom Resources
echo "=== Collecting Custom Resources ==="
kubectl get networkintents -A -o yaml > networkintents.yaml
kubectl get e2nodesets -A -o yaml > e2nodesets.yaml 2>/dev/null || echo "No E2NodeSets"

# 4. Events (last 2 hours)
echo "=== Collecting Events ==="
kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' > events.txt
kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp' > warnings-all-namespaces.txt

# 5. Logs (last hour)
echo "=== Collecting Logs ==="
mkdir -p logs
for component in nephoran-controller llm-processor rag-api weaviate nephio-bridge; do
  echo "Collecting logs for: $component"
  kubectl logs -n "$NAMESPACE" -l app="$component" --since=1h --prefix=true > "logs/${component}-logs.txt" 2>/dev/null || echo "No logs for $component"
  
  # Get previous logs if pod restarted
  kubectl logs -n "$NAMESPACE" -l app="$component" --since=1h --previous --prefix=true > "logs/${component}-previous-logs.txt" 2>/dev/null || echo "No previous logs for $component"
done

# 6. Metrics (if accessible)
echo "=== Collecting Metrics ==="
mkdir -p metrics
kubectl port-forward -n "$NAMESPACE" svc/nephoran-controller 8081:8081 >/dev/null 2>&1 &
CONTROLLER_PF=$!
sleep 2
curl -s http://localhost:8081/metrics > metrics/controller-metrics.txt 2>/dev/null || echo "Cannot collect controller metrics"
kill $CONTROLLER_PF 2>/dev/null

kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
LLM_PF=$!
sleep 2
curl -s http://localhost:8080/metrics > metrics/llm-processor-metrics.txt 2>/dev/null || echo "Cannot collect LLM processor metrics"
curl -s http://localhost:8080/admin/circuit-breaker/status > metrics/circuit-breaker-status.txt 2>/dev/null || echo "Cannot collect circuit breaker status"
kill $LLM_PF 2>/dev/null

# 7. Network Information
echo "=== Collecting Network Information ==="
kubectl get services -n "$NAMESPACE" -o wide > network-services.txt
kubectl get endpoints -n "$NAMESPACE" -o wide > network-endpoints.txt
kubectl get networkpolicies -n "$NAMESPACE" -o yaml > network-policies.yaml
kubectl get ingress -n "$NAMESPACE" -o yaml > ingress.yaml 2>/dev/null || echo "No ingress resources"

# 8. Resource Usage
echo "=== Collecting Resource Usage ==="
kubectl top nodes > resource-nodes.txt 2>/dev/null || echo "Metrics server not available"
kubectl top pods -n "$NAMESPACE" > resource-pods.txt 2>/dev/null || echo "Metrics server not available"

# 9. External Dependencies Status
echo "=== Checking External Dependencies ==="
mkdir -p external-checks
kubectl run external-check-temp --rm -i --tty --image=curlimages/curl --restart=Never -- \
  sh -c "curl -s -w 'Time: %{time_total}s\nHTTP: %{http_code}\n' https://api.openai.com/v1/models -o /dev/null" > external-checks/openai-api.txt 2>/dev/null &

# 10. System Resources
echo "=== Collecting System Resource Information ==="
kubectl describe nodes > system-resources.txt

echo "Data collection completed: $(date -u)" >> collection-timestamp.txt
echo "Data collected in: $DATA_DIR"
EOF

chmod +x collect-incident-data.sh

# Execute data collection
./collect-incident-data.sh "$INCIDENT_ID"
```

### Manual Evidence Collection

```bash
# For complex incidents requiring deeper investigation
cat > manual-evidence.sh << 'EOF'
#!/bin/bash

INCIDENT_ID=$1
echo "Manual evidence collection for: $INCIDENT_ID"

# 1. Memory dumps (if process is still running)
echo "=== Collecting Memory Information ==="
for pod in $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o name); do
  echo "Memory info for $pod:"
  kubectl exec -n nephoran-system "$pod" -- cat /proc/meminfo | head -10
  kubectl exec -n nephoran-system "$pod" -- ps aux --sort=-%mem | head -10
done

# 2. Network connectivity tests
echo "=== Network Connectivity Tests ==="
kubectl run network-test-$INCIDENT_ID --rm -i --image=nicolaka/netshoot -- bash << 'NETTEST'
echo "Testing internal connectivity..."
nslookup nephoran-controller.nephoran-system.svc.cluster.local
nslookup llm-processor.nephoran-system.svc.cluster.local
nslookup rag-api.nephoran-system.svc.cluster.local
nslookup weaviate.nephoran-system.svc.cluster.local

echo "Testing external connectivity..."
nslookup api.openai.com
curl -I https://api.openai.com/v1/models --max-time 10
NETTEST

# 3. Database integrity checks
echo "=== Database Integrity Checks ==="
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -s "http://localhost:8080/v1/meta" | jq '.version, .modules'

kubectl exec -n nephoran-system weaviate-0 -- \
  curl -s "http://localhost:8080/v1/schema" | jq '.classes[] | {class: .class, properties: (.properties | length)}'

# 4. Configuration drift detection
echo "=== Configuration Drift Check ==="
kubectl diff -f deployments/helm/nephoran-operator/templates/ 2>/dev/null || echo "No configuration drift detected"

# 5. Persistent volume checks
echo "=== Storage Health Checks ==="
kubectl get pv,pvc -n nephoran-system -o wide
kubectl describe pvc -n nephoran-system | grep -E "(Status|Events)" -A 5

EOF

chmod +x manual-evidence.sh
```

## Root Cause Analysis

### RCA Framework

```bash
# Root Cause Analysis template
cat > rca-template.md << 'EOF'
# Root Cause Analysis - {INCIDENT_ID}

## Incident Summary
- **Incident ID**: {INCIDENT_ID}
- **Start Time**: {START_TIME}
- **End Time**: {END_TIME}
- **Duration**: {DURATION}
- **Severity**: {SEVERITY}
- **Impact**: {IMPACT_DESCRIPTION}

## Timeline of Events
| Time | Event | Action Taken | Result |
|------|-------|--------------|--------|
| {TIME} | {EVENT} | {ACTION} | {RESULT} |
| ... | ... | ... | ... |

## Root Cause
### Primary Cause
{DETAILED_DESCRIPTION_OF_ROOT_CAUSE}

### Contributing Factors
1. {FACTOR_1}
2. {FACTOR_2}
3. {FACTOR_3}

### Evidence
- {EVIDENCE_1}
- {EVIDENCE_2}
- {EVIDENCE_3}

## 5 Whys Analysis
1. **Why did the incident occur?** {ANSWER_1}
2. **Why {ANSWER_1}?** {ANSWER_2}
3. **Why {ANSWER_2}?** {ANSWER_3}
4. **Why {ANSWER_3}?** {ANSWER_4}
5. **Why {ANSWER_4}?** {ROOT_CAUSE}

## Impact Analysis
### Users Affected
- {NUMBER} NetworkIntents failed
- {DURATION} of service disruption
- {PERCENTAGE}% error rate during incident

### Business Impact
- {BUSINESS_IMPACT_DESCRIPTION}
- {ESTIMATED_COST}

### Technical Impact
- {TECHNICAL_CONSEQUENCES}

## Actions Taken
### Immediate Actions (During Incident)
1. {ACTION_1} - {RESULT_1}
2. {ACTION_2} - {RESULT_2}
3. {ACTION_3} - {RESULT_3}

### Resolution Actions
1. {RESOLUTION_ACTION_1}
2. {RESOLUTION_ACTION_2}

### Verification Steps
1. {VERIFICATION_1}
2. {VERIFICATION_2}

## Lessons Learned
### What Went Well
- {POSITIVE_1}
- {POSITIVE_2}

### What Could Be Improved
- {IMPROVEMENT_1}
- {IMPROVEMENT_2}

### What We Learned
- {LEARNING_1}
- {LEARNING_2}

## Action Items
| Action | Owner | Due Date | Priority | Status |
|--------|-------|----------|----------|--------|
| {ACTION_1} | {OWNER_1} | {DATE_1} | {PRIORITY_1} | {STATUS_1} |
| {ACTION_2} | {OWNER_2} | {DATE_2} | {PRIORITY_2} | {STATUS_2} |

## Prevention Measures
### Immediate (< 1 week)
- {IMMEDIATE_1}
- {IMMEDIATE_2}

### Short-term (< 1 month)
- {SHORT_TERM_1}
- {SHORT_TERM_2}

### Long-term (< 3 months)
- {LONG_TERM_1}
- {LONG_TERM_2}
EOF
```

### RCA Investigation Script

```bash
# Automated RCA data analysis
cat > analyze-rca.sh << 'EOF'
#!/bin/bash

INCIDENT_DATA_DIR=$1
echo "Analyzing incident data from: $INCIDENT_DATA_DIR"

cd "$INCIDENT_DATA_DIR"

echo "=== RCA ANALYSIS REPORT ===" > rca-analysis.txt
echo "Generated: $(date -u)" >> rca-analysis.txt
echo >> rca-analysis.txt

# 1. Error pattern analysis
echo "=== ERROR PATTERNS ===" >> rca-analysis.txt
find logs/ -name "*.txt" -exec grep -l "ERROR\|FATAL\|panic" {} \; | while read logfile; do
  echo "File: $logfile" >> rca-analysis.txt
  grep -E "ERROR|FATAL|panic" "$logfile" | sort | uniq -c | sort -nr | head -5 >> rca-analysis.txt
  echo >> rca-analysis.txt
done

# 2. Timeline reconstruction
echo "=== TIMELINE RECONSTRUCTION ===" >> rca-analysis.txt
if [ -f events.txt ]; then
  echo "Events during incident:" >> rca-analysis.txt
  grep -E "Warning|Failed|Error" events.txt | head -20 >> rca-analysis.txt
  echo >> rca-analysis.txt
fi

# 3. Resource consumption analysis
echo "=== RESOURCE ANALYSIS ===" >> rca-analysis.txt
if [ -f resource-pods.txt ]; then
  echo "Resource usage during incident:" >> rca-analysis.txt
  cat resource-pods.txt >> rca-analysis.txt
  echo >> rca-analysis.txt
fi

# 4. Component status correlation
echo "=== COMPONENT STATUS CORRELATION ===" >> rca-analysis.txt
grep -E "NotReady|Failed|Error|Pending" components-status.txt >> rca-analysis.txt 2>/dev/null
echo >> rca-analysis.txt

# 5. Configuration changes
echo "=== CONFIGURATION ANALYSIS ===" >> rca-analysis.txt
echo "Recent ConfigMap changes:" >> rca-analysis.txt
if [ -f configmaps.yaml ]; then
  grep -A 2 "creationTimestamp\|lastUpdateTime" configmaps.yaml >> rca-analysis.txt
fi
echo >> rca-analysis.txt

# 6. External dependencies
echo "=== EXTERNAL DEPENDENCIES ===" >> rca-analysis.txt
if [ -d external-checks ]; then
  find external-checks/ -name "*.txt" -exec echo "File: {}" \; -exec cat {} \; >> rca-analysis.txt
fi

echo "RCA analysis completed. Review rca-analysis.txt for patterns."
EOF

chmod +x analyze-rca.sh
```

## Communication Templates

### Initial Incident Notification

```markdown
## 🚨 INCIDENT ALERT - {INCIDENT_ID}

**Severity**: {P1/P2/P3/P4}
**Status**: Investigating
**Start Time**: {UTC_TIME}
**Incident Commander**: {NAME}

### Summary
{BRIEF_DESCRIPTION_OF_ISSUE}

### Current Impact
- **Users Affected**: {NUMBER/PERCENTAGE}
- **Services Affected**: {SERVICE_LIST}
- **Expected Duration**: {ESTIMATE}

### Actions Being Taken
- ✅ Incident declared and team notified
- 🔄 Initial triage completed
- 🔄 Investigation in progress
- ⏳ Root cause analysis underway

### Workarounds
{LIST_WORKAROUNDS_IF_AVAILABLE}

### Next Update
{TIME} - or sooner if status changes significantly

**Stay tuned to this channel for updates**
```

### Progress Update Template

```markdown
## 📊 UPDATE - {INCIDENT_ID}

**Status**: {Investigating/Mitigating/Resolved}
**Time**: {UTC_TIME}
**Duration**: {ELAPSED_TIME}

### Progress Summary
{WHAT_HAS_BEEN_DISCOVERED_OR_FIXED}

### Current Status
- **Root Cause**: {IDENTIFIED/SUSPECTED/UNKNOWN}
- **Fix Status**: {IN_PROGRESS/DEPLOYED/TESTING}
- **Service Status**: {DEGRADED/IMPROVING/RESTORED}

### Actions Completed
- ✅ {COMPLETED_ACTION_1}
- ✅ {COMPLETED_ACTION_2}

### Actions In Progress
- 🔄 {ONGOING_ACTION_1} (ETA: {TIME})
- 🔄 {ONGOING_ACTION_2} (ETA: {TIME})

### Next Steps
- ⏳ {NEXT_ACTION_1}
- ⏳ {NEXT_ACTION_2}

**Next Update**: {TIME}
```

### Resolution Notification

```markdown
## ✅ RESOLVED - {INCIDENT_ID}

**Status**: Resolved
**Resolution Time**: {UTC_TIME}
**Total Duration**: {TOTAL_DURATION}

### Resolution Summary
{BRIEF_DESCRIPTION_OF_HOW_ISSUE_WAS_RESOLVED}

### Root Cause
{ROOT_CAUSE_SUMMARY}

### Fix Applied
{DESCRIPTION_OF_FIX}

### Verification
- ✅ {VERIFICATION_1}
- ✅ {VERIFICATION_2}
- ✅ {VERIFICATION_3}

### Current Status
- **Service Status**: Fully Operational
- **Performance**: Normal
- **Monitoring**: Enhanced monitoring in place for 24 hours

### Follow-up Actions
- 📋 Post-incident review scheduled for {DATE}
- 🔍 Root cause analysis document will be published by {DATE}
- 🛠️ Prevention measures being implemented

**Thank you for your patience during this incident.**
```

### Stakeholder-Specific Communications

```bash
# Generate stakeholder-specific messages
cat > generate-comms.sh << 'EOF'
#!/bin/bash

INCIDENT_ID=$1
SEVERITY=$2
IMPACT=$3
STATUS=$4

# Executive Summary (for leadership)
cat > executive-summary.md << EXEC
# Executive Summary - $INCIDENT_ID

**Business Impact**: $IMPACT
**Current Status**: $STATUS
**Estimated Resolution**: {ETA}

## Key Points
- Service disruption affecting {X}% of operations
- Technical team actively working on resolution
- Customer communication plan activated
- No data loss or security concerns identified

## Next Steps
- Continued focus on service restoration
- Post-incident review to prevent recurrence
- Customer retention measures if needed

**Point of Contact**: {INCIDENT_COMMANDER}
EXEC

# Customer-facing message
cat > customer-message.md << CUSTOMER
# Service Status Update

We are currently experiencing {DESCRIPTION} that may affect your ability to process network intents.

## What we're doing
Our team is actively working to resolve this issue and restore full service.

## What you can do
- Check our status page for real-time updates
- Contact support if you need immediate assistance
- Consider using {WORKAROUND} if applicable

We apologize for any inconvenience and appreciate your patience.

**Updates**: Every {FREQUENCY} on our status page
CUSTOMER

# Internal team update
cat > team-update.md << TEAM
# Team Update - $INCIDENT_ID

## Situation
$STATUS - $IMPACT

## Actions Needed
- [ ] Continue monitoring resolution
- [ ] Prepare rollback plan if needed
- [ ] Document all actions taken
- [ ] Coordinate with dependent teams

## Resources
- War room: {LINK}
- Incident data: {LOCATION}
- Monitoring: {DASHBOARD_LINK}

**All hands on deck until resolved**
TEAM

echo "Communication templates generated:"
echo "- executive-summary.md"
echo "- customer-message.md"  
echo "- team-update.md"
EOF

chmod +x generate-comms.sh
```

## Post-Incident Procedures

### Immediate Post-Resolution Actions (Within 1 Hour)

```bash
# Post-resolution checklist
cat > post-resolution-checklist.sh << 'EOF'
#!/bin/bash

INCIDENT_ID=$1
echo "Post-resolution actions for: $INCIDENT_ID"

echo "=== IMMEDIATE ACTIONS (within 1 hour) ==="

# 1. Service verification
echo "1. Verifying service restoration..."
kubectl get pods -n nephoran-system -o wide
kubectl get networkintents -A -o wide | grep -E "Processing|Failed" | wc -l

# 2. Update status page
echo "2. Update status page to 'Resolved'"
echo "   - Set incident status to resolved"
echo "   - Update service status to operational"
echo "   - Post final update message"

# 3. Stand down response team
echo "3. Stand down incident response team"
echo "   - Thank responders"
echo "   - Dismiss war room"
echo "   - Return to normal operations"

# 4. Preserve evidence
echo "4. Preserving incident evidence..."
tar -czf "$INCIDENT_ID-evidence.tar.gz" incident-data-"$INCIDENT_ID"/ 
echo "Evidence preserved in: $INCIDENT_ID-evidence.tar.gz"

# 5. Schedule post-incident review
echo "5. Schedule post-incident review"
echo "   - Book meeting within 48 hours"
echo "   - Invite all key stakeholders"
echo "   - Prepare preliminary timeline"

echo -e "\n=== VERIFICATION TESTS ==="

# Test NetworkIntent processing
cat <<EOF | kubectl apply -f - | echo "Test intent created"
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: post-incident-test-$(date +%s)
  namespace: default
spec:
  intent: "Deploy test AMF for incident verification"
  priority: low
  scope: cluster
EOF

# Monitor test intent
echo "Monitoring test intent processing..."
sleep 10
kubectl get networkintents -l "nephoran.com/test=post-incident" -o wide

echo -e "\n=== ACTION ITEMS ==="
echo "- [ ] Complete RCA document"
echo "- [ ] Schedule post-incident review"
echo "- [ ] Update runbooks if needed"
echo "- [ ] Implement prevention measures"
echo "- [ ] Customer follow-up if required"
EOF

chmod +x post-resolution-checklist.sh
```

### Post-Incident Review (PIR) Template

```markdown
# Post-Incident Review - {INCIDENT_ID}

**Date**: {DATE}
**Duration**: {START_TIME} - {END_TIME}
**Attendees**: {LIST_OF_ATTENDEES}
**Facilitator**: {NAME}

## Incident Overview
- **Incident ID**: {INCIDENT_ID}
- **Severity**: {P1/P2/P3/P4}
- **Duration**: {TOTAL_DURATION}
- **Impact**: {USER_IMPACT}
- **Root Cause**: {ROOT_CAUSE_SUMMARY}

## Timeline Review
{DETAILED_TIMELINE_OF_EVENTS}

## What Went Well ✅
1. {POSITIVE_ASPECT_1}
2. {POSITIVE_ASPECT_2}
3. {POSITIVE_ASPECT_3}

## What Didn't Go Well ❌
1. {ISSUE_1}
2. {ISSUE_2}
3. {ISSUE_3}

## Action Items
| Action | Owner | Due Date | Priority | Tracking |
|--------|-------|----------|----------|----------|
| {ACTION_1} | {OWNER_1} | {DATE_1} | High | {LINK_1} |
| {ACTION_2} | {OWNER_2} | {DATE_2} | Medium | {LINK_2} |

## Process Improvements
### Monitoring and Alerting
- {IMPROVEMENT_1}
- {IMPROVEMENT_2}

### Response Procedures
- {IMPROVEMENT_1}
- {IMPROVEMENT_2}

### Technical Changes
- {IMPROVEMENT_1}
- {IMPROVEMENT_2}

## Lessons Learned
1. {LESSON_1}
2. {LESSON_2}
3. {LESSON_3}

## Follow-up
- **Next Review**: {DATE}
- **Action Item Check**: {DATE}
- **Documentation Updates**: {DATE}
```

## Lessons Learned Integration

### Knowledge Base Update Process

```bash
# Script to integrate lessons learned
cat > integrate-lessons.sh << 'EOF'
#!/bin/bash

INCIDENT_ID=$1
LESSONS_FILE=$2

echo "Integrating lessons learned from: $INCIDENT_ID"

# 1. Update troubleshooting guides
echo "Updating troubleshooting guides..."
if [ -f "$LESSONS_FILE" ]; then
  # Extract new troubleshooting steps
  grep -A 10 "## New Troubleshooting Steps" "$LESSONS_FILE" > new-troubleshooting.txt
  
  # Append to existing troubleshooting guide
  echo -e "\n## Lessons from $INCIDENT_ID" >> docs/runbooks/troubleshooting-guide.md
  cat new-troubleshooting.txt >> docs/runbooks/troubleshooting-guide.md
fi

# 2. Update monitoring alerts
echo "Reviewing monitoring alerts..."
if grep -q "monitoring" "$LESSONS_FILE"; then
  echo "New monitoring requirements identified. Update monitoring/prometheus/alerts/"
fi

# 3. Update runbooks
echo "Updating runbooks..."
if grep -q "procedure" "$LESSONS_FILE"; then
  echo "New procedures identified. Review and update relevant runbooks."
fi

# 4. Update FAQ
echo "Updating FAQ..."
cat >> docs/FAQ.md << FAQ

## Incident $INCIDENT_ID Learnings
$(grep -A 5 "## FAQ Updates" "$LESSONS_FILE" 2>/dev/null || echo "No FAQ updates")
FAQ

# 5. Create prevention checklist
cat > "$INCIDENT_ID-prevention-checklist.md" << CHECKLIST
# Prevention Checklist - $INCIDENT_ID

Based on the lessons learned from incident $INCIDENT_ID, implement these preventive measures:

## Immediate Actions (< 1 week)
- [ ] Implement enhanced monitoring for {SPECIFIC_AREA}
- [ ] Update alert thresholds based on incident data
- [ ] Add new health checks for {COMPONENT}

## Short-term Actions (< 1 month)
- [ ] Enhance error handling in {COMPONENT}
- [ ] Implement circuit breaker improvements
- [ ] Add automated recovery for {SCENARIO}

## Long-term Actions (< 3 months)
- [ ] Architectural improvements to prevent {ROOT_CAUSE}
- [ ] Enhanced testing for {FAILURE_MODE}
- [ ] Documentation and training updates

CHECKLIST

echo "Prevention checklist created: $INCIDENT_ID-prevention-checklist.md"

# 6. Update metrics and KPIs
echo "Consider updating SLA/SLO based on incident learnings"
cat > sla-review.txt << SLA
# SLA Review after $INCIDENT_ID

Current SLAs:
- Detection: < 2 minutes
- Response: < 5 minutes (P1)
- Resolution: < 30 minutes (P1)

Actual Performance:
- Detection: {ACTUAL_DETECTION}
- Response: {ACTUAL_RESPONSE}  
- Resolution: {ACTUAL_RESOLUTION}

Recommendations:
- {RECOMMENDATION_1}
- {RECOMMENDATION_2}
SLA

echo "Integration complete. Review generated files and implement recommendations."
EOF

chmod +x integrate-lessons.sh
```

### Continuous Improvement Process

```bash
# Monthly incident review process
cat > monthly-incident-review.sh << 'EOF'
#!/bin/bash

MONTH=$1
YEAR=$2

echo "=== MONTHLY INCIDENT REVIEW: $MONTH $YEAR ==="

# Collect all incidents for the month
find /tmp/incidents/ -name "INC-$YEAR$(printf "%02d" $MONTH)*" -type d > incidents-list.txt

echo "Incidents this month: $(wc -l < incidents-list.txt)"

# Generate statistics
echo -e "\n=== INCIDENT STATISTICS ==="
echo "P1 Incidents: $(grep -l "P1" /tmp/incidents/INC-$YEAR$(printf "%02d" $MONTH)*/severity.txt | wc -l)"
echo "P2 Incidents: $(grep -l "P2" /tmp/incidents/INC-$YEAR$(printf "%02d" $MONTH)*/severity.txt | wc -l)"
echo "P3 Incidents: $(grep -l "P3" /tmp/incidents/INC-$YEAR$(printf "%02d" $MONTH)*/severity.txt | wc -l)"
echo "P4 Incidents: $(grep -l "P4" /tmp/incidents/INC-$YEAR$(printf "%02d" $MONTH)*/severity.txt | wc -l)"

# Analyze trends
echo -e "\n=== TREND ANALYSIS ==="
echo "Common failure modes:"
find /tmp/incidents/INC-$YEAR$(printf "%02d" $MONTH)* -name "rca-analysis.txt" -exec grep -h "Root Cause:" {} \; | sort | uniq -c | sort -nr | head -5

echo -e "\nAverage resolution time by severity:"
# This would require parsing actual incident data

# Generate improvement recommendations
echo -e "\n=== IMPROVEMENT RECOMMENDATIONS ==="
cat > monthly-improvements.md << IMPROVEMENTS
# Monthly Incident Review - $MONTH $YEAR

## Summary
- Total Incidents: {COUNT}
- Average Resolution Time: {TIME}
- Top Contributing Factors: {FACTORS}

## Trends
{TREND_ANALYSIS}

## Recommendations
1. {RECOMMENDATION_1}
2. {RECOMMENDATION_2}
3. {RECOMMENDATION_3}

## Action Items for Next Month
- [ ] Implement monitoring improvement X
- [ ] Update procedure Y
- [ ] Training on topic Z

IMPROVEMENTS

echo "Monthly review complete. See monthly-improvements.md for recommendations."
EOF

chmod +x monthly-incident-review.sh
```

## Incident Response Tools

### Quick Reference Commands

```bash
# Emergency command reference
cat > emergency-commands.sh << 'EOF'
#!/bin/bash

echo "=== EMERGENCY COMMAND REFERENCE ==="

echo "== System Status =="
echo "kubectl get pods -n nephoran-system -o wide --sort-by=.status.startTime"
echo "kubectl get networkintents -A -o wide"
echo "kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20"

echo -e "\n== Quick Fixes =="
echo "# Restart all services:"
echo "kubectl rollout restart deployment -n nephoran-system"
echo ""
echo "# Clear stuck NetworkIntents:"
echo "kubectl patch networkintent INTENT_NAME -n NAMESPACE --type='merge' -p='{\"metadata\":{\"annotations\":{\"nephoran.com/requeue\":\"true\"}}}'"
echo ""
echo "# Reset circuit breakers:"
echo "kubectl exec -n nephoran-system \$(kubectl get pods -n nephoran-system -l app=llm-processor -o jsonpath='{.items[0].metadata.name}') -- curl -X POST http://localhost:8080/admin/circuit-breaker/reset"

echo -e "\n== Data Collection =="
echo "# Collect all logs:"
echo "kubectl logs -n nephoran-system -l app=COMPONENT --since=1h"
echo ""
echo "# Get detailed pod info:"
echo "kubectl describe pod POD_NAME -n nephoran-system"
echo ""
echo "# Export configuration:"
echo "kubectl get configmap CONFIG_NAME -n nephoran-system -o yaml"

echo -e "\n== Escalation =="
echo "# Emergency contacts:"
echo "# - Slack: #nephoran-ops-alerts"
echo "# - On-call: +1-XXX-XXX-XXXX"
echo "# - Team Lead: leader@company.com"
EOF

chmod +x emergency-commands.sh
```

### Incident Response Dashboard

```bash
# Create incident response dashboard
cat > create-incident-dashboard.sh << 'EOF'
#!/bin/bash

echo "Creating incident response dashboard..."

# This would typically integrate with Grafana or similar
cat > incident-dashboard.json << DASHBOARD
{
  "dashboard": {
    "title": "Nephoran Incident Response",
    "panels": [
      {
        "title": "Active Incidents",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(nephoran_active_incidents)",
            "legendFormat": "Active Incidents"
          }
        ]
      },
      {
        "title": "System Health",
        "type": "graph",
        "targets": [
          {
            "expr": "up{job=\"nephoran-controller\"}",
            "legendFormat": "Controller"
          },
          {
            "expr": "up{job=\"llm-processor\"}",
            "legendFormat": "LLM Processor"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(nephoran_processing_errors_total[5m])",
            "legendFormat": "Error Rate"
          }
        ]
      }
    ]
  }
}
DASHBOARD

echo "Incident dashboard configuration created: incident-dashboard.json"
echo "Import this into your Grafana instance for incident response visibility"
EOF

chmod +x create-incident-dashboard.sh
```

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-07  
**Next Review**: 2025-02-07  
**Owner**: Nephoran Incident Response Team  
**Approvers**: Operations Manager, Engineering Manager