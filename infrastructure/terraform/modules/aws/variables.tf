# AWS Infrastructure Module Variables for Nephoran Intent Operator

# General Configuration
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "nephoran-intent-operator"
}

variable "environment" {
  description = "Environment name (e.g., dev, staging, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "staging", "prod", "test"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod, test."
  }
}

variable "owner" {
  description = "Owner of the resources"
  type        = string
  default     = "platform-team"
}

variable "cost_center" {
  description = "Cost center for billing"
  type        = string
  default     = "engineering"
}

variable "compliance_level" {
  description = "Compliance level required (standard, high, carrier-grade)"
  type        = string
  default     = "standard"
  validation {
    condition     = contains(["standard", "high", "carrier-grade"], var.compliance_level)
    error_message = "Compliance level must be one of: standard, high, carrier-grade."
  }
}

variable "additional_tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

# VPC Configuration
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
  validation {
    condition     = can(cidrhost(var.vpc_cidr, 0))
    error_message = "VPC CIDR must be a valid IPv4 CIDR block."
  }
}

variable "availability_zones_count" {
  description = "Number of availability zones to use"
  type        = number
  default     = 3
  validation {
    condition     = var.availability_zones_count >= 2 && var.availability_zones_count <= 6
    error_message = "Availability zones count must be between 2 and 6."
  }
}

variable "single_nat_gateway" {
  description = "Use a single NAT gateway instead of one per AZ"
  type        = bool
  default     = false
}

# EKS Configuration
variable "kubernetes_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.28"
}

variable "cluster_endpoint_private_access" {
  description = "Enable private API server endpoint"
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access" {
  description = "Enable public API server endpoint"
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access_cidrs" {
  description = "List of CIDR blocks that can access the public API server endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "cluster_enabled_log_types" {
  description = "List of enabled EKS cluster log types"
  type        = list(string)
  default     = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
}

variable "node_group_bootstrap_arguments" {
  description = "Additional arguments for EKS node group bootstrap"
  type        = string
  default     = ""
}

variable "node_group_disk_size" {
  description = "Disk size for EKS node group instances"
  type        = number
  default     = 100
  validation {
    condition     = var.node_group_disk_size >= 20
    error_message = "Node group disk size must be at least 20 GB."
  }
}

variable "node_groups" {
  description = "Map of EKS node group configurations"
  type = map(object({
    ami_type                    = string
    capacity_type              = string
    instance_types             = list(string)
    desired_capacity           = number
    max_capacity              = number
    min_capacity              = number
    max_unavailable_percentage = number
    key_name                   = optional(string)
    source_security_group_ids  = optional(list(string))
    k8s_labels                = map(string)
    k8s_taints = list(object({
      key    = string
      value  = string
      effect = string
    }))
  }))
  default = {
    control_plane = {
      ami_type                    = "AL2_x86_64"
      capacity_type              = "ON_DEMAND"
      instance_types             = ["m5.large", "m5.xlarge"]
      desired_capacity           = 3
      max_capacity              = 6
      min_capacity              = 3
      max_unavailable_percentage = 25
      key_name                   = null
      source_security_group_ids  = null
      k8s_labels = {
        "nephoran.com/node-type" = "control-plane"
        "nephoran.com/workload"  = "system"
      }
      k8s_taints = [{
        key    = "nephoran.com/control-plane"
        value  = "true"
        effect = "NO_SCHEDULE"
      }]
    }
    ai_processing = {
      ami_type                    = "AL2_x86_64"
      capacity_type              = "ON_DEMAND"
      instance_types             = ["c5.2xlarge", "c5.4xlarge"]
      desired_capacity           = 2
      max_capacity              = 10
      min_capacity              = 2
      max_unavailable_percentage = 25
      key_name                   = null
      source_security_group_ids  = null
      k8s_labels = {
        "nephoran.com/node-type" = "ai-processing"
        "nephoran.com/workload"  = "ml"
      }
      k8s_taints = [{
        key    = "nephoran.com/ai-workload"
        value  = "true"
        effect = "NO_SCHEDULE"
      }]
    }
    database = {
      ami_type                    = "AL2_x86_64"
      capacity_type              = "ON_DEMAND"
      instance_types             = ["r5.xlarge", "r5.2xlarge"]
      desired_capacity           = 3
      max_capacity              = 6
      min_capacity              = 3
      max_unavailable_percentage = 25
      key_name                   = null
      source_security_group_ids  = null
      k8s_labels = {
        "nephoran.com/node-type" = "database"
        "nephoran.com/workload"  = "database"
      }
      k8s_taints = [{
        key    = "nephoran.com/database-workload"
        value  = "true"
        effect = "NO_SCHEDULE"
      }]
    }
  }
}

# RDS Configuration
variable "enable_rds" {
  description = "Enable RDS PostgreSQL instance"
  type        = bool
  default     = true
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15.4"
}

variable "postgres_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r5.large"
}

variable "postgres_allocated_storage" {
  description = "Initial allocated storage for RDS"
  type        = number
  default     = 100
}

variable "postgres_max_allocated_storage" {
  description = "Maximum allocated storage for RDS auto scaling"
  type        = number
  default     = 1000
}

variable "postgres_database_name" {
  description = "Name of the PostgreSQL database"
  type        = string
  default     = "nephoran"
}

variable "postgres_username" {
  description = "Username for PostgreSQL"
  type        = string
  default     = "nephoran_admin"
  sensitive   = true
}

variable "postgres_password" {
  description = "Password for PostgreSQL"
  type        = string
  sensitive   = true
}

variable "postgres_backup_retention_period" {
  description = "Backup retention period in days"
  type        = number
  default     = 30
  validation {
    condition     = var.postgres_backup_retention_period >= 1 && var.postgres_backup_retention_period <= 35
    error_message = "Backup retention period must be between 1 and 35 days."
  }
}

variable "postgres_backup_window" {
  description = "Preferred backup window"
  type        = string
  default     = "07:00-08:00"
}

variable "postgres_maintenance_window" {
  description = "Preferred maintenance window"
  type        = string
  default     = "sun:08:00-sun:09:00"
}

variable "postgres_skip_final_snapshot" {
  description = "Skip final snapshot when destroying RDS"
  type        = bool
  default     = false
}

variable "postgres_performance_insights_enabled" {
  description = "Enable Performance Insights"
  type        = bool
  default     = true
}

variable "postgres_monitoring_interval" {
  description = "Enhanced monitoring interval in seconds"
  type        = number
  default     = 60
  validation {
    condition     = contains([0, 1, 5, 10, 15, 30, 60], var.postgres_monitoring_interval)
    error_message = "Monitoring interval must be one of: 0, 1, 5, 10, 15, 30, 60."
  }
}

# ElastiCache Configuration
variable "enable_elasticache" {
  description = "Enable ElastiCache Redis cluster"
  type        = bool
  default     = true
}

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.r6g.large"
}

variable "redis_parameter_group_name" {
  description = "Redis parameter group name"
  type        = string
  default     = "default.redis7"
}

variable "redis_num_cache_clusters" {
  description = "Number of cache clusters (replicas) in replication group"
  type        = number
  default     = 3
  validation {
    condition     = var.redis_num_cache_clusters >= 2 && var.redis_num_cache_clusters <= 6
    error_message = "Number of cache clusters must be between 2 and 6."
  }
}

variable "redis_automatic_failover_enabled" {
  description = "Enable automatic failover"
  type        = bool
  default     = true
}

variable "redis_multi_az_enabled" {
  description = "Enable Multi-AZ"
  type        = bool
  default     = true
}

variable "redis_auth_token" {
  description = "Auth token for Redis"
  type        = string
  sensitive   = true
  default     = null
}

variable "redis_snapshot_retention_limit" {
  description = "Number of days to retain snapshots"
  type        = number
  default     = 30
}

variable "redis_snapshot_window" {
  description = "Daily time range for snapshots"
  type        = string
  default     = "06:30-07:30"
}

variable "redis_maintenance_window" {
  description = "Weekly maintenance window"
  type        = string
  default     = "sun:07:30-sun:08:30"
}

# S3 Configuration
variable "enable_s3_backup" {
  description = "Enable S3 bucket for backups"
  type        = bool
  default     = true
}

variable "s3_backup_retention_days" {
  description = "S3 backup retention in days"
  type        = number
  default     = 2555  # ~7 years
}

# ECR Configuration
variable "ecr_repositories" {
  description = "ECR repositories to create"
  type = map(object({
    image_tag_mutability = string
    scan_on_push        = bool
    lifecycle_policy = object({
      untagged_images = number
      tagged_images   = number
    })
  }))
  default = {
    nephoran-operator = {
      image_tag_mutability = "MUTABLE"
      scan_on_push        = true
      lifecycle_policy = {
        untagged_images = 5
        tagged_images   = 50
      }
    }
    llm-processor = {
      image_tag_mutability = "MUTABLE"
      scan_on_push        = true
      lifecycle_policy = {
        untagged_images = 5
        tagged_images   = 30
      }
    }
    rag-api = {
      image_tag_mutability = "MUTABLE"
      scan_on_push        = true
      lifecycle_policy = {
        untagged_images = 5
        tagged_images   = 30
      }
    }
    nephio-bridge = {
      image_tag_mutability = "MUTABLE"
      scan_on_push        = true
      lifecycle_policy = {
        untagged_images = 5
        tagged_images   = 30
      }
    }
    oran-adaptor = {
      image_tag_mutability = "MUTABLE"
      scan_on_push        = true
      lifecycle_policy = {
        untagged_images = 5
        tagged_images   = 30
      }
    }
  }
}

# Load Balancer Configuration
variable "enable_load_balancer" {
  description = "Enable Application Load Balancer"
  type        = bool
  default     = true
}

variable "load_balancer_internal" {
  description = "Make load balancer internal"
  type        = bool
  default     = false
}

variable "load_balancer_enable_deletion_protection" {
  description = "Enable deletion protection for load balancer"
  type        = bool
  default     = true
}

variable "load_balancer_access_logs_bucket" {
  description = "S3 bucket for load balancer access logs"
  type        = string
  default     = null
}

variable "load_balancer_ingress_cidrs" {
  description = "CIDR blocks allowed to access the load balancer"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# KMS Configuration
variable "kms_deletion_window" {
  description = "KMS key deletion window in days"
  type        = number
  default     = 7
  validation {
    condition     = var.kms_deletion_window >= 7 && var.kms_deletion_window <= 30
    error_message = "KMS deletion window must be between 7 and 30 days."
  }
}

# CloudWatch Configuration
variable "cloudwatch_log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 30
  validation {
    condition = contains([
      1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653
    ], var.cloudwatch_log_retention_days)
    error_message = "CloudWatch log retention must be a valid retention period."
  }
}

# Monitoring and Observability
variable "enable_monitoring" {
  description = "Enable comprehensive monitoring stack"
  type        = bool
  default     = true
}

variable "enable_prometheus" {
  description = "Enable Prometheus monitoring"
  type        = bool
  default     = true
}

variable "enable_grafana" {
  description = "Enable Grafana dashboards"
  type        = bool
  default     = true
}

variable "enable_alertmanager" {
  description = "Enable AlertManager for alerting"
  type        = bool
  default     = true
}

variable "enable_jaeger" {
  description = "Enable Jaeger for distributed tracing"
  type        = bool
  default     = true
}

variable "enable_fluentd" {
  description = "Enable Fluentd for log collection"
  type        = bool
  default     = true
}

# Security Configuration
variable "enable_pod_security_policy" {
  description = "Enable Pod Security Policy"
  type        = bool
  default     = true
}

variable "enable_network_policy" {
  description = "Enable Network Policy enforcement"
  type        = bool
  default     = true
}

variable "enable_falco" {
  description = "Enable Falco for runtime security"
  type        = bool
  default     = true
}

variable "enable_opa_gatekeeper" {
  description = "Enable OPA Gatekeeper for policy enforcement"
  type        = bool
  default     = true
}

# Backup and Disaster Recovery
variable "enable_velero" {
  description = "Enable Velero for backup and disaster recovery"
  type        = bool
  default     = true
}

variable "velero_backup_retention" {
  description = "Velero backup retention period"
  type        = string
  default     = "720h"  # 30 days
}

variable "enable_cross_region_backup" {
  description = "Enable cross-region backup replication"
  type        = bool
  default     = false
}

variable "backup_regions" {
  description = "List of regions for backup replication"
  type        = list(string)
  default     = []
}

# Auto Scaling Configuration
variable "enable_cluster_autoscaler" {
  description = "Enable Cluster Autoscaler"
  type        = bool
  default     = true
}

variable "enable_vertical_pod_autoscaler" {
  description = "Enable Vertical Pod Autoscaler"
  type        = bool
  default     = true
}

variable "enable_keda" {
  description = "Enable KEDA for advanced autoscaling"
  type        = bool
  default     = true
}

# Service Mesh Configuration
variable "enable_istio" {
  description = "Enable Istio service mesh"
  type        = bool
  default     = false
}

variable "enable_linkerd" {
  description = "Enable Linkerd service mesh"
  type        = bool
  default     = false
}

# Certificate Management
variable "enable_cert_manager" {
  description = "Enable cert-manager for certificate management"
  type        = bool
  default     = true
}

variable "enable_external_dns" {
  description = "Enable external-dns for DNS management"
  type        = bool
  default     = true
}

# Secrets Management
variable "enable_external_secrets" {
  description = "Enable External Secrets Operator"
  type        = bool
  default     = true
}

variable "secrets_backend" {
  description = "Secrets backend (aws-secrets-manager, vault, etc.)"
  type        = string
  default     = "aws-secrets-manager"
  validation {
    condition = contains([
      "aws-secrets-manager", "vault", "azure-keyvault", "gcp-secret-manager"
    ], var.secrets_backend)
    error_message = "Secrets backend must be supported by External Secrets Operator."
  }
}

# GitOps Configuration
variable "enable_argocd" {
  description = "Enable ArgoCD for GitOps"
  type        = bool
  default     = true
}

variable "enable_flux" {
  description = "Enable Flux for GitOps (alternative to ArgoCD)"
  type        = bool
  default     = false
}

# Development and Testing
variable "enable_development_tools" {
  description = "Enable development and testing tools"
  type        = bool
  default     = false
}

variable "enable_chaos_engineering" {
  description = "Enable chaos engineering tools (Chaos Monkey, Litmus)"
  type        = bool
  default     = false
}

# Cost Optimization
variable "enable_kube_cost" {
  description = "Enable Kubecost for cost monitoring"
  type        = bool
  default     = true
}

variable "enable_node_termination_handler" {
  description = "Enable AWS Node Termination Handler for Spot instances"
  type        = bool
  default     = true
}

# Compliance and Governance
variable "enable_policy_engine" {
  description = "Enable policy engine for compliance"
  type        = bool
  default     = true
}

variable "policy_violations_action" {
  description = "Action to take on policy violations (warn, block)"
  type        = string
  default     = "warn"
  validation {
    condition     = contains(["warn", "block"], var.policy_violations_action)
    error_message = "Policy violations action must be either 'warn' or 'block'."
  }
}

# Performance and Optimization
variable "enable_node_local_dns" {
  description = "Enable NodeLocal DNS Cache for better DNS performance"
  type        = bool
  default     = true
}

variable "enable_bandwidth_plugin" {
  description = "Enable bandwidth CNI plugin for network QoS"
  type        = bool
  default     = false
}

# Edge Computing Support
variable "enable_edge_deployment" {
  description = "Enable edge computing deployment features"
  type        = bool
  default     = false
}

variable "edge_regions" {
  description = "List of edge regions for deployment"
  type        = list(string)
  default     = []
}

# Carrier-Grade Features
variable "enable_carrier_grade_features" {
  description = "Enable carrier-grade reliability features"
  type        = bool
  default     = false
}

variable "sla_tier" {
  description = "SLA tier for the deployment (standard, high, carrier-grade)"
  type        = string
  default     = "standard"
  validation {
    condition     = contains(["standard", "high", "carrier-grade"], var.sla_tier)
    error_message = "SLA tier must be one of: standard, high, carrier-grade."
  }
}

variable "enable_multi_region_ha" {
  description = "Enable multi-region high availability"
  type        = bool
  default     = false
}

# Telecom-Specific Configuration
variable "enable_oran_interfaces" {
  description = "Enable O-RAN interface support"
  type        = bool
  default     = false
}

variable "enable_network_slicing" {
  description = "Enable 5G network slicing capabilities"
  type        = bool
  default     = false
}

variable "enable_edge_orchestration" {
  description = "Enable edge orchestration for telecom workloads"
  type        = bool
  default     = false
}