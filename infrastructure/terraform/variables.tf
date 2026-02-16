# Nephoran Intent Operator - Terraform Variables

# ============================================================================
# Global Configuration
# ============================================================================

variable "environment" {
  description = "Environment name (development, staging, production)"
  type        = string
  default     = "development"

  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be development, staging, or production."
  }
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default = {
    Project   = "nephoran-intent-operator"
    ManagedBy = "terraform"
  }
}

# ============================================================================
# AWS Configuration
# ============================================================================

variable "aws_region" {
  description = "AWS region for EKS cluster"
  type        = string
  default     = "us-west-2"
}

variable "aws_cluster_name" {
  description = "Name of the AWS EKS cluster"
  type        = string
  default     = "nephoran-eks"
}

variable "aws_cluster_version" {
  description = "Kubernetes version for EKS cluster"
  type        = string
  default     = "1.29"
}

variable "aws_vpc_cidr" {
  description = "CIDR block for AWS VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "aws_availability_zones" {
  description = "Availability zones for AWS infrastructure"
  type        = list(string)
  default     = ["us-west-2a", "us-west-2b", "us-west-2c"]
}

variable "aws_node_instance_type" {
  description = "EC2 instance type for EKS worker nodes"
  type        = string
  default     = "t3.large"
}

variable "aws_node_desired_size" {
  description = "Desired number of worker nodes"
  type        = number
  default     = 3
}

variable "aws_node_min_size" {
  description = "Minimum number of worker nodes"
  type        = number
  default     = 1
}

variable "aws_node_max_size" {
  description = "Maximum number of worker nodes"
  type        = number
  default     = 5
}

# ============================================================================
# Azure Configuration
# ============================================================================

variable "azure_location" {
  description = "Azure region for AKS cluster"
  type        = string
  default     = "eastus"
}

variable "azure_cluster_name" {
  description = "Name of the Azure AKS cluster"
  type        = string
  default     = "nephoran-aks"
}

variable "azure_cluster_version" {
  description = "Kubernetes version for AKS cluster"
  type        = string
  default     = "1.29"
}

variable "azure_resource_group_name" {
  description = "Name of the Azure resource group"
  type        = string
  default     = "nephoran-rg"
}

variable "azure_node_vm_size" {
  description = "VM size for AKS worker nodes"
  type        = string
  default     = "Standard_D4s_v3"
}

variable "azure_node_count" {
  description = "Initial number of worker nodes"
  type        = number
  default     = 3
}

variable "azure_node_min_count" {
  description = "Minimum number of worker nodes (autoscaling)"
  type        = number
  default     = 1
}

variable "azure_node_max_count" {
  description = "Maximum number of worker nodes (autoscaling)"
  type        = number
  default     = 5
}
