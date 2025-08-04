variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name (prod, staging, dev)"
  type        = string
  default     = "prod"

  validation {
    condition     = contains(["prod", "staging", "dev"], var.environment)
    error_message = "Environment must be prod, staging, or dev."
  }
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "nephoran.example.com"
}

variable "dns_zone_name" {
  description = "Cloud DNS zone name"
  type        = string
  default     = "nephoran-zone"
}

variable "node_pools" {
  description = "Node pool configurations for primary regions"
  type = map(list(object({
    name               = string
    machine_type      = string
    min_count         = number
    max_count         = number
    initial_count     = number
    disk_size_gb      = number
    disk_type         = string
    preemptible       = bool
    spot              = bool
    auto_repair       = bool
    auto_upgrade      = bool
    node_labels       = map(string)
    node_taints = list(object({
      key    = string
      value  = string
      effect = string
    }))
  })))
  
  default = {
    primary = [
      {
        name          = "system"
        machine_type  = "n2-standard-4"
        min_count     = 2
        max_count     = 5
        initial_count = 3
        disk_size_gb  = 100
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "system"
        }
        node_taints = []
      },
      {
        name          = "ai-processing"
        machine_type  = "n2-highmem-8"
        min_count     = 2
        max_count     = 10
        initial_count = 3
        disk_size_gb  = 200
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "ai-processing"
        }
        node_taints = [
          {
            key    = "nephoran.com/ai-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      },
      {
        name          = "data"
        machine_type  = "n2-highmem-4"
        min_count     = 3
        max_count     = 6
        initial_count = 3
        disk_size_gb  = 500
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "data"
        }
        node_taints = [
          {
            key    = "nephoran.com/data-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      },
      {
        name          = "spot"
        machine_type  = "n2-standard-4"
        min_count     = 0
        max_count     = 20
        initial_count = 2
        disk_size_gb  = 100
        disk_type     = "pd-standard"
        preemptible   = false
        spot          = true
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "spot"
        }
        node_taints = [
          {
            key    = "cloud.google.com/gke-spot"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      }
    ]
    
    secondary = [
      {
        name          = "system"
        machine_type  = "n2-standard-4"
        min_count     = 2
        max_count     = 4
        initial_count = 2
        disk_size_gb  = 100
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "system"
        }
        node_taints = []
      },
      {
        name          = "ai-processing"
        machine_type  = "n2-highmem-8"
        min_count     = 1
        max_count     = 8
        initial_count = 2
        disk_size_gb  = 200
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "ai-processing"
        }
        node_taints = [
          {
            key    = "nephoran.com/ai-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      },
      {
        name          = "data"
        machine_type  = "n2-highmem-4"
        min_count     = 2
        max_count     = 5
        initial_count = 2
        disk_size_gb  = 500
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "data"
        }
        node_taints = [
          {
            key    = "nephoran.com/data-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      }
    ]
    
    tertiary = [
      {
        name          = "system"
        machine_type  = "n2-standard-4"
        min_count     = 2
        max_count     = 4
        initial_count = 2
        disk_size_gb  = 100
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "system"
        }
        node_taints = []
      },
      {
        name          = "ai-processing"
        machine_type  = "n2-highmem-8"
        min_count     = 1
        max_count     = 8
        initial_count = 2
        disk_size_gb  = 200
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "ai-processing"
        }
        node_taints = [
          {
            key    = "nephoran.com/ai-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      },
      {
        name          = "data"
        machine_type  = "n2-highmem-4"
        min_count     = 2
        max_count     = 5
        initial_count = 2
        disk_size_gb  = 500
        disk_type     = "pd-ssd"
        preemptible   = false
        spot          = false
        auto_repair   = true
        auto_upgrade  = true
        node_labels = {
          "nephoran.com/node-type" = "data"
        }
        node_taints = [
          {
            key    = "nephoran.com/data-workload"
            value  = "true"
            effect = "NoSchedule"
          }
        ]
      }
    ]
  }
}

variable "edge_node_pools" {
  description = "Node pool configuration for edge regions"
  type = list(object({
    name               = string
    machine_type      = string
    min_count         = number
    max_count         = number
    initial_count     = number
    disk_size_gb      = number
    disk_type         = string
    preemptible       = bool
    spot              = bool
    auto_repair       = bool
    auto_upgrade      = bool
    node_labels       = map(string)
    node_taints = list(object({
      key    = string
      value  = string
      effect = string
    }))
  }))
  
  default = [
    {
      name          = "edge-compute"
      machine_type  = "n2-standard-2"
      min_count     = 1
      max_count     = 3
      initial_count = 2
      disk_size_gb  = 50
      disk_type     = "pd-standard"
      preemptible   = false
      spot          = false
      auto_repair   = true
      auto_upgrade  = true
      node_labels = {
        "nephoran.com/node-type" = "edge"
      }
      node_taints = []
    }
  ]
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for alerts"
  type        = string
  sensitive   = true
  default     = ""
}

variable "pagerduty_integration_key" {
  description = "PagerDuty integration key for critical alerts"
  type        = string
  sensitive   = true
  default     = ""
}

variable "alert_email_addresses" {
  description = "Email addresses for alert notifications"
  type        = list(string)
  default     = []
}

variable "enable_istio" {
  description = "Enable Istio service mesh"
  type        = bool
  default     = true
}

variable "enable_config_sync" {
  description = "Enable Config Sync for GitOps"
  type        = bool
  default     = true
}

variable "enable_binary_authorization" {
  description = "Enable Binary Authorization"
  type        = bool
  default     = true
}

variable "enable_workload_identity" {
  description = "Enable Workload Identity"
  type        = bool
  default     = true
}

variable "enable_network_policy" {
  description = "Enable Network Policy"
  type        = bool
  default     = true
}

variable "enable_pod_security_policy" {
  description = "Enable Pod Security Policy"
  type        = bool
  default     = false
}

variable "maintenance_window" {
  description = "Maintenance window for GKE clusters"
  type = object({
    start_time = string
    end_time   = string
    recurrence = string
  })
  default = {
    start_time = "2023-01-01T00:00:00Z"
    end_time   = "2023-01-01T04:00:00Z"
    recurrence = "FREQ=WEEKLY;BYDAY=SU"
  }
}

variable "budget_amount" {
  description = "Monthly budget amount in USD"
  type        = number
  default     = 10000
}

variable "budget_alert_thresholds" {
  description = "Budget alert threshold percentages"
  type        = list(number)
  default     = [50, 80, 90, 100]
}