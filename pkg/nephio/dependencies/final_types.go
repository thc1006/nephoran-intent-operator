package dependencies

import "time"

// Final missing types for updater.go

type UpdateValidationError struct {
	Code     string            `json:"code"`
	Type     string            `json:"type"`
	Message  string            `json:"message"`
	Package  *PackageReference `json:"package,omitempty"`
	Severity string            `json:"severity"`
	Rule     string            `json:"rule,omitempty"`
}

type UpdateValidationWarning struct {
	Code       string            `json:"code"`
	Type       string            `json:"type"`
	Message    string            `json:"message"`
	Package    *PackageReference `json:"package,omitempty"`
	Severity   string            `json:"severity"`
	Rule       string            `json:"rule,omitempty"`
	Suggestion string            `json:"suggestion,omitempty"`
}

type UpdateHistoryFilter struct {
	PackageName string     `json:"packageName,omitempty"`
	Environment string     `json:"environment,omitempty"`
	DateFrom    time.Time  `json:"dateFrom,omitempty"`
	DateTo      time.Time  `json:"dateTo,omitempty"`
	Status      string     `json:"status,omitempty"`
	UpdateType  UpdateType `json:"updateType,omitempty"`
	Limit       int        `json:"limit,omitempty"`
}

type UpdateRecord struct {
	RecordID     string            `json:"recordId"`
	Package      *PackageReference `json:"package"`
	OldVersion   string            `json:"oldVersion"`
	NewVersion   string            `json:"newVersion"`
	UpdateType   UpdateType        `json:"updateType"`
	UpdateReason UpdateReason      `json:"updateReason"`
	Environment  string            `json:"environment,omitempty"`
	Status       string            `json:"status"`
	StartedAt    time.Time         `json:"startedAt"`
	CompletedAt  time.Time         `json:"completedAt,omitempty"`
	Duration     time.Duration     `json:"duration"`
	UpdatedBy    string            `json:"updatedBy"`
	Notes        string            `json:"notes,omitempty"`
}

type ChangeTracker struct {
	TrackerID     string    `json:"trackerId"`
	Enabled       bool      `json:"enabled"`
	TrackingRules []string  `json:"trackingRules"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ChangeReport struct {
	ReportID    string           `json:"reportId"`
	GeneratedAt time.Time        `json:"generatedAt"`
	TimeRange   *TimeRange       `json:"timeRange"`
	Changes     []*PackageChange `json:"changes"`
	Summary     *ChangeSummary   `json:"summary"`
}

type PackageChange struct {
	Package     *PackageReference `json:"package"`
	ChangeType  string            `json:"changeType"`
	OldVersion  string            `json:"oldVersion,omitempty"`
	NewVersion  string            `json:"newVersion,omitempty"`
	ChangedAt   time.Time         `json:"changedAt"`
	ChangedBy   string            `json:"changedBy"`
	Description string            `json:"description,omitempty"`
}

type ChangeSummary struct {
	TotalChanges    int `json:"totalChanges"`
	AddedPackages   int `json:"addedPackages"`
	UpdatedPackages int `json:"updatedPackages"`
	RemovedPackages int `json:"removedPackages"`
}

type ApprovalFilter struct {
	Status    string         `json:"status,omitempty"`
	Requester string         `json:"requester,omitempty"`
	Approver  string         `json:"approver,omitempty"`
	DateFrom  time.Time      `json:"dateFrom,omitempty"`
	DateTo    time.Time      `json:"dateTo,omitempty"`
	Priority  UpdatePriority `json:"priority,omitempty"`
	Limit     int            `json:"limit,omitempty"`
}

type UpdateNotification struct {
	NotificationID string            `json:"notificationId"`
	Type           string            `json:"type"`
	Title          string            `json:"title"`
	Message        string            `json:"message"`
	Package        *PackageReference `json:"package,omitempty"`
	Recipients     []string          `json:"recipients"`
	Channel        string            `json:"channel"`
	Priority       string            `json:"priority"`
	SentAt         time.Time         `json:"sentAt"`
	Status         string            `json:"status"`
}

type UpdaterHealth struct {
	Status          string            `json:"status"`
	LastHealthCheck time.Time         `json:"lastHealthCheck"`
	ComponentHealth map[string]string `json:"componentHealth"`
	ActiveUpdates   int               `json:"activeUpdates"`
	QueuedUpdates   int               `json:"queuedUpdates"`
	FailedUpdates   int               `json:"failedUpdates"`
	Uptime          time.Duration     `json:"uptime"`
}

type UpdaterMetrics struct {
	TotalUpdates      int64         `json:"totalUpdates"`
	SuccessfulUpdates int64         `json:"successfulUpdates"`
	FailedUpdates     int64         `json:"failedUpdates"`
	SkippedUpdates    int64         `json:"skippedUpdates"`
	AverageUpdateTime time.Duration `json:"averageUpdateTime"`
	LastUpdated       time.Time     `json:"lastUpdated"`
	UpdateRate        float64       `json:"updateRate"` // updates per hour
	ErrorRate         float64       `json:"errorRate"`  // errors per hour
}
