# Nephoran Intent Operator - Multi-Cloud Infrastructure
# Main Terraform configuration for AWS and Azure deployments

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }

  # Backend configuration for state management
  # Uncomment and configure for production use
  # backend "s3" {
  #   bucket         = "nephoran-terraform-state"
  #   key            = "infrastructure/terraform.tfstate"
  #   region         = "us-west-2"
  #   dynamodb_table = "nephoran-terraform-locks"
  #   encrypt        = true
  # }
}

# AWS Provider Configuration
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "nephoran-intent-operator"
      ManagedBy   = "terraform"
      Environment = var.environment
    }
  }
}

# Azure Provider Configuration
provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# AWS Infrastructure Module
module "aws_infrastructure" {
  source = "./modules/aws"

  # Cluster configuration
  cluster_name    = var.aws_cluster_name
  cluster_version = var.aws_cluster_version
  region          = var.aws_region

  # Networking
  vpc_cidr            = var.aws_vpc_cidr
  availability_zones  = var.aws_availability_zones

  # Node configuration
  node_instance_type = var.aws_node_instance_type
  node_desired_size  = var.aws_node_desired_size
  node_min_size      = var.aws_node_min_size
  node_max_size      = var.aws_node_max_size

  # Tags
  environment = var.environment
  tags        = var.common_tags
}

# Azure Infrastructure Module
module "azure_infrastructure" {
  source = "./modules/azure"

  # Cluster configuration
  cluster_name    = var.azure_cluster_name
  cluster_version = var.azure_cluster_version
  location        = var.azure_location

  # Resource Group
  resource_group_name = var.azure_resource_group_name

  # Node configuration
  node_vm_size   = var.azure_node_vm_size
  node_count     = var.azure_node_count
  node_min_count = var.azure_node_min_count
  node_max_count = var.azure_node_max_count

  # Tags
  environment = var.environment
  tags        = var.common_tags
}
