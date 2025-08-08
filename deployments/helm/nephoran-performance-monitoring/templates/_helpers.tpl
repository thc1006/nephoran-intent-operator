{{/*
Expand the name of the chart.
*/}}
{{- define "nephoran-performance-monitoring.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "nephoran-performance-monitoring.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nephoran-performance-monitoring.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nephoran-performance-monitoring.labels" -}}
helm.sh/chart: {{ include "nephoran-performance-monitoring.chart" . }}
{{ include "nephoran-performance-monitoring.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: performance-monitoring
app.kubernetes.io/part-of: nephoran-intent-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nephoran-performance-monitoring.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nephoran-performance-monitoring.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nephoran-performance-monitoring.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nephoran-performance-monitoring.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate Prometheus server name
*/}}
{{- define "nephoran-performance-monitoring.prometheus.server.fullname" -}}
{{- printf "%s-prometheus-server" (include "nephoran-performance-monitoring.fullname" .) }}
{{- end }}

{{/*
Generate Grafana fullname
*/}}
{{- define "nephoran-performance-monitoring.grafana.fullname" -}}
{{- printf "%s-grafana" (include "nephoran-performance-monitoring.fullname" .) }}
{{- end }}

{{/*
Generate AlertManager fullname
*/}}
{{- define "nephoran-performance-monitoring.alertmanager.fullname" -}}
{{- printf "%s-alertmanager" (include "nephoran-performance-monitoring.fullname" .) }}
{{- end }}

{{/*
Common performance monitoring annotations
*/}}
{{- define "nephoran-performance-monitoring.annotations" -}}
nephoran.com/monitoring-stack: "true"
nephoran.com/performance-monitoring: "enabled"
nephoran.com/claims-validation: "enabled"
prometheus.io/scrape: "true"
{{- end }}

{{/*
Security context for performance monitoring pods
*/}}
{{- define "nephoran-performance-monitoring.securityContext" -}}
{{- if .Values.global.security.enabled }}
securityContext:
  runAsNonRoot: {{ .Values.global.security.runAsNonRoot }}
  runAsUser: {{ .Values.global.security.runAsUser }}
  readOnlyRootFilesystem: {{ .Values.global.security.readOnlyRootFilesystem }}
  allowPrivilegeEscalation: {{ .Values.global.security.allowPrivilegeEscalation }}
  capabilities:
    drop:
      - ALL
{{- end }}
{{- end }}

{{/*
Resource limits and requests
*/}}
{{- define "nephoran-performance-monitoring.resources" -}}
resources:
  limits:
    cpu: {{ .Values.global.resources.limits.cpu | quote }}
    memory: {{ .Values.global.resources.limits.memory | quote }}
  requests:
    cpu: {{ .Values.global.resources.requests.cpu | quote }}
    memory: {{ .Values.global.resources.requests.memory | quote }}
{{- end }}

{{/*
Generate storage class name
*/}}
{{- define "nephoran-performance-monitoring.storageClass" -}}
{{- if .Values.global.storageClass }}
{{- .Values.global.storageClass }}
{{- else }}
{{- "fast-ssd" }}
{{- end }}
{{- end }}

{{/*
Generate namespace
*/}}
{{- define "nephoran-performance-monitoring.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace }}
{{- end }}

{{/*
Performance claims validation labels
*/}}
{{- define "nephoran-performance-monitoring.claimsLabels" -}}
nephoran.com/claim-1: "intent-processing-latency"
nephoran.com/claim-2: "concurrent-user-capacity"
nephoran.com/claim-3: "throughput-capacity"
nephoran.com/claim-4: "service-availability"
nephoran.com/claim-5: "rag-retrieval-latency"
nephoran.com/claim-6: "cache-hit-rate"
{{- end }}

{{/*
Service monitor selector
*/}}
{{- define "nephoran-performance-monitoring.serviceMonitorSelector" -}}
matchLabels:
  {{ include "nephoran-performance-monitoring.selectorLabels" . }}
  nephoran.com/monitoring: "enabled"
{{- end }}

{{/*
Alert routing configuration
*/}}
{{- define "nephoran-performance-monitoring.alertRouting" -}}
{{- if .Values.alertmanager.config.route }}
{{- .Values.alertmanager.config.route | toYaml }}
{{- else }}
group_by: ['alertname', 'cluster', 'service']
group_wait: 10s
group_interval: 10s
repeat_interval: 12h
receiver: 'performance-team'
{{- end }}
{{- end }}

{{/*
Performance monitoring probe configuration
*/}}
{{- define "nephoran-performance-monitoring.probes" -}}
livenessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /ready
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
{{- end }}

{{/*
High availability configuration
*/}}
{{- define "nephoran-performance-monitoring.highAvailability" -}}
{{- if .Values.extraConfig.highAvailability.enabled }}
replicas: {{ .Values.extraConfig.highAvailability.replicaCount | default 2 }}
{{- if .Values.extraConfig.highAvailability.affinity }}
affinity:
  {{- .Values.extraConfig.highAvailability.affinity | toYaml | nindent 2 }}
{{- end }}
{{- end }}
{{- end }}

{{/*
TLS configuration
*/}}
{{- define "nephoran-performance-monitoring.tls" -}}
{{- if .Values.extraConfig.security.enableTLS }}
tls:
  secretName: {{ .Values.extraConfig.security.tlsSecretName | default "monitoring-tls" }}
{{- end }}
{{- end }}

{{/*
Performance optimization configuration
*/}}
{{- define "nephoran-performance-monitoring.performanceConfig" -}}
{{- if .Values.extraConfig.performance }}
performance:
  caching:
    enabled: {{ .Values.extraConfig.performance.enableCaching | default true }}
    size: {{ .Values.extraConfig.performance.cacheSize | default "1Gi" }}
  query:
    parallelism: {{ .Values.extraConfig.performance.queryParallelism | default 10 }}
{{- end }}
{{- end }}