variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "zones" {
  description = "GCP zones for the cluster"
  type        = list(string)
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
}

variable "vpc_id" {
  description = "VPC network ID"
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID"
  type        = string
}

variable "pods_range_name" {
  description = "Name of the secondary range for pods"
  type        = string
}

variable "svc_range_name" {
  description = "Name of the secondary range for services"
  type        = string
}

variable "common_labels" {
  description = "Common labels to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "kms_key_id" {
  description = "KMS key ID for database encryption"
  type        = string
  default     = ""
}

variable "node_pools" {
  description = "Node pool configurations"
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
}

variable "enable_autopilot" {
  description = "Enable GKE Autopilot mode"
  type        = bool
  default     = false
}

variable "enable_workload_identity" {
  description = "Enable Workload Identity"
  type        = bool
  default     = true
}

variable "enable_binary_authorization" {
  description = "Enable Binary Authorization"
  type        = bool
  default     = true
}

variable "enable_shielded_nodes" {
  description = "Enable Shielded GKE nodes"
  type        = bool
  default     = true
}

variable "enable_network_policy" {
  description = "Enable Network Policy"
  type        = bool
  default     = true
}

variable "enable_istio" {
  description = "Enable Istio service mesh"
  type        = bool
  default     = true
}

variable "enable_config_sync" {
  description = "Enable Config Sync"
  type        = bool
  default     = true
}