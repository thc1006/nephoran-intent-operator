# ITERATION PROTOCOL - MANDATORY ORDER

## CRITICAL RULES - DO NOT DEVIATE

### 1. ITERATION CYCLE (MANDATORY)
```
LOOP UNTIL NO ERRORS:
1. Search internet for latest fixes (search-specialist agent)
2. Deploy ULTRA SPEED multiple agents 
3. Apply fixes with maximum timeout
4. Run comprehensive scan with maximum timeout
5. IF ERRORS EXIST → COMMIT + REPEAT FROM STEP 1
6. IF NO ERRORS → PUSH
```

### 2. TIMEOUT REQUIREMENTS
- **MINIMUM**: 900 seconds (15 minutes) per job
- **PREFERRED**: 1800+ seconds (30+ minutes) per job
- **CRITICAL**: Always use maximum available timeout

### 3. COMMIT STRATEGY
- **EVERY ITERATION**: Must commit after each fix attempt
- **MESSAGE FORMAT**: "fix(iteration-N): [description of fixes applied]"
- **NO PUSH**: Until zero errors confirmed

### 4. AGENT DEPLOYMENT
- **MANDATORY**: Use search-specialist FIRST each iteration
- **ULTRA SPEED**: Deploy 5+ specialized agents simultaneously
- **SPECIALIZATION**: Each agent focuses on specific error types

### 5. ERROR TOLERANCE
- **ZERO TOLERANCE**: No errors allowed in final state
- **COMPREHENSIVE SCAN**: Must test entire project (`go build ./...`)
- **CI SIMULATION**: Must verify GitHub Actions will pass

### 6. DOCUMENTATION
- **TRACK ITERATIONS**: Log each cycle in PROGRESS.md
- **TIMESTAMP**: ISO format for each iteration
- **AGENT COORDINATION**: Log which agents deployed

### 7. EMERGENCY PROTOCOL
IF STUCK IN LOOP >10 ITERATIONS:
1. Deploy debug-specialist agent
2. Search for alternative approaches
3. Consider architectural changes
4. Escalate to multi-agent-coordinator

## CURRENT STATUS
- **ITERATION COUNT**: 0
- **LAST COMMIT**: [PENDING]
- **ERROR STATUS**: [TO BE DETERMINED]
- **NEXT ACTION**: Begin iteration cycle

---
**THIS IS A MANDATORY ORDER - FOLLOW WITHOUT DEVIATION**