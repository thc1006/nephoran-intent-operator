# Nephoran Intent Operator - Terraform Outputs

# ============================================================================
# AWS Outputs
# ============================================================================

output "aws_cluster_id" {
  description = "AWS EKS cluster ID"
  value       = module.aws_infrastructure.cluster_id
}

output "aws_cluster_endpoint" {
  description = "Endpoint for AWS EKS cluster API server"
  value       = module.aws_infrastructure.cluster_endpoint
  sensitive   = true
}

output "aws_cluster_certificate_authority" {
  description = "Certificate authority data for AWS EKS cluster"
  value       = module.aws_infrastructure.cluster_certificate_authority
  sensitive   = true
}

output "aws_cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.aws_infrastructure.cluster_security_group_id
}

output "aws_node_group_id" {
  description = "EKS node group ID"
  value       = module.aws_infrastructure.node_group_id
}

output "aws_vpc_id" {
  description = "ID of the AWS VPC"
  value       = module.aws_infrastructure.vpc_id
}

output "aws_kubeconfig_command" {
  description = "Command to configure kubectl for AWS EKS cluster"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${var.aws_cluster_name}"
}

# ============================================================================
# Azure Outputs
# ============================================================================

output "azure_cluster_id" {
  description = "Azure AKS cluster ID"
  value       = module.azure_infrastructure.cluster_id
}

output "azure_cluster_name" {
  description = "Azure AKS cluster name"
  value       = module.azure_infrastructure.cluster_name
}

output "azure_cluster_fqdn" {
  description = "FQDN of the Azure AKS cluster"
  value       = module.azure_infrastructure.cluster_fqdn
  sensitive   = true
}

output "azure_kube_config" {
  description = "Kubernetes configuration for Azure AKS cluster"
  value       = module.azure_infrastructure.kube_config
  sensitive   = true
}

output "azure_resource_group_name" {
  description = "Name of the Azure resource group"
  value       = module.azure_infrastructure.resource_group_name
}

output "azure_kubeconfig_command" {
  description = "Command to configure kubectl for Azure AKS cluster"
  value       = "az aks get-credentials --resource-group ${var.azure_resource_group_name} --name ${var.azure_cluster_name}"
}

# ============================================================================
# Combined Outputs
# ============================================================================

output "deployment_summary" {
  description = "Summary of deployed infrastructure"
  value = {
    aws = {
      cluster_name = var.aws_cluster_name
      region       = var.aws_region
      endpoint     = module.aws_infrastructure.cluster_endpoint
    }
    azure = {
      cluster_name = var.azure_cluster_name
      location     = var.azure_location
      fqdn         = module.azure_infrastructure.cluster_fqdn
    }
    environment = var.environment
  }
}
