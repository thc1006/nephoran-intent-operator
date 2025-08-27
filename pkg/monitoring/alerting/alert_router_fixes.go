package alerting

// Quick fixes for critical errors

// Fix for isChannelAvailable signature
func (ar *AlertRouter) isChannelAvailableForEnriched(channelName string, alert *EnrichedAlert) bool {
	return ar.isChannelAvailable(channelName, alert.SLAAlert)
}

// Fix for checkRateLimit signature
func (ar *AlertRouter) checkRateLimitPtr(channelName string, rateLimit *RateLimit) bool {
	if rateLimit == nil {
		return true
	}
	return ar.checkRateLimit(channelName, *rateLimit)
}

// Fix for getFallbackRouting to return int priority
func (ar *AlertRouter) getFallbackRoutingFixed() *RoutingRule {
	return &RoutingRule{Name: "fallback", Priority: 50}
}

// Fix for ImpactAnalysis to use string severity
func (ia *ImpactAnalyzer) AnalyzeImpactFixed(alert *SLAAlert) (*ImpactAnalysisFixed, error) {
	return &ImpactAnalysisFixed{Severity: string(alert.Severity), AffectedServices: []string{}}, nil
}

type ImpactAnalysisFixed struct {
	Severity         string   `json:"severity"`
	AffectedServices []string `json:"affected_services"`
}
