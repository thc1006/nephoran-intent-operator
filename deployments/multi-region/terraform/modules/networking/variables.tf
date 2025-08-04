variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "regions" {
  description = "Map of regions with their configuration"
  type = map(object({
    name  = string
    zones = list(string)
    type  = string
  }))
}

variable "common_labels" {
  description = "Common labels to apply to all resources"
  type        = map(string)
  default     = {}
}