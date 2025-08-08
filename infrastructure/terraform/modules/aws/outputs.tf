# AWS Infrastructure Module Outputs for Nephoran Intent Operator

# VPC Outputs
output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.main.id
}

output "vpc_cidr_block" {
  description = "CIDR block of the VPC"
  value       = aws_vpc.main.cidr_block
}

output "private_subnets" {
  description = "List of IDs of private subnets"
  value       = aws_subnet.private[*].id
}

output "public_subnets" {
  description = "List of IDs of public subnets"
  value       = aws_subnet.public[*].id
}

output "database_subnets" {
  description = "List of IDs of database subnets"
  value       = aws_subnet.database[*].id
}

output "private_subnet_cidrs" {
  description = "List of CIDR blocks of private subnets"
  value       = aws_subnet.private[*].cidr_block
}

output "public_subnet_cidrs" {
  description = "List of CIDR blocks of public subnets"
  value       = aws_subnet.public[*].cidr_block
}

output "database_subnet_cidrs" {
  description = "List of CIDR blocks of database subnets"
  value       = aws_subnet.database[*].cidr_block
}

output "availability_zones" {
  description = "List of availability zones used"
  value       = local.azs
}

output "nat_gateway_ids" {
  description = "List of IDs of NAT Gateways"
  value       = aws_nat_gateway.main[*].id
}

output "internet_gateway_id" {
  description = "ID of the Internet Gateway"
  value       = aws_internet_gateway.main.id
}

# Security Group Outputs
output "eks_cluster_security_group_id" {
  description = "ID of the EKS cluster security group"
  value       = aws_security_group.eks_cluster.id
}

output "eks_node_group_security_group_id" {
  description = "ID of the EKS node group security group"
  value       = aws_security_group.eks_node_group.id
}

output "rds_security_group_id" {
  description = "ID of the RDS security group"
  value       = aws_security_group.rds.id
}

output "elasticache_security_group_id" {
  description = "ID of the ElastiCache security group"
  value       = aws_security_group.elasticache.id
}

output "alb_security_group_id" {
  description = "ID of the ALB security group"
  value       = var.enable_load_balancer ? aws_security_group.alb[0].id : null
}

# EKS Outputs
output "cluster_id" {
  description = "Name of the EKS cluster"
  value       = aws_eks_cluster.main.name
}

output "cluster_name" {
  description = "Name of the EKS cluster"
  value       = aws_eks_cluster.main.name
}

output "cluster_arn" {
  description = "ARN of the EKS cluster"
  value       = aws_eks_cluster.main.arn
}

output "cluster_endpoint" {
  description = "Endpoint for the EKS cluster"
  value       = aws_eks_cluster.main.endpoint
}

output "cluster_version" {
  description = "Version of the EKS cluster"
  value       = aws_eks_cluster.main.version
}

output "cluster_platform_version" {
  description = "Platform version of the EKS cluster"
  value       = aws_eks_cluster.main.platform_version
}

output "cluster_status" {
  description = "Status of the EKS cluster"
  value       = aws_eks_cluster.main.status
}

output "cluster_certificate_authority_data" {
  description = "Base64 encoded certificate data required to communicate with the cluster"
  value       = aws_eks_cluster.main.certificate_authority[0].data
}

output "cluster_oidc_issuer_url" {
  description = "The URL on the EKS cluster for the OpenID Connect identity provider"
  value       = aws_eks_cluster.main.identity[0].oidc[0].issuer
}

output "cluster_primary_security_group_id" {
  description = "The cluster primary security group ID created by the EKS service"
  value       = aws_eks_cluster.main.vpc_config[0].cluster_security_group_id
}

output "cloudwatch_log_group_name" {
  description = "Name of cloudwatch log group for EKS cluster"
  value       = aws_cloudwatch_log_group.cluster.name
}

output "cloudwatch_log_group_arn" {
  description = "ARN of cloudwatch log group for EKS cluster"
  value       = aws_cloudwatch_log_group.cluster.arn
}

# Node Group Outputs
output "node_groups" {
  description = "EKS node groups information"
  value = {
    for k, v in aws_eks_node_group.main : k => {
      arn           = v.arn
      status        = v.status
      capacity_type = v.capacity_type
      instance_types = v.instance_types
      ami_type      = v.ami_type
      node_role_arn = v.node_role_arn
      scaling_config = v.scaling_config
      update_config = v.update_config
    }
  }
}

output "node_group_arns" {
  description = "List of node group ARNs"
  value       = [for ng in aws_eks_node_group.main : ng.arn]
}

output "node_group_statuses" {
  description = "Map of node group statuses"
  value       = { for k, v in aws_eks_node_group.main : k => v.status }
}

# IAM Outputs
output "cluster_iam_role_name" {
  description = "IAM role name associated with EKS cluster"
  value       = aws_iam_role.eks_cluster.name
}

output "cluster_iam_role_arn" {
  description = "IAM role ARN associated with EKS cluster"
  value       = aws_iam_role.eks_cluster.arn
}

output "node_group_iam_role_name" {
  description = "IAM role name associated with EKS node group"
  value       = aws_iam_role.eks_node_group.name
}

output "node_group_iam_role_arn" {
  description = "IAM role ARN associated with EKS node group"
  value       = aws_iam_role.eks_node_group.arn
}

# KMS Outputs
output "kms_key_id" {
  description = "ID of the KMS key used for encryption"
  value       = aws_kms_key.main.id
}

output "kms_key_arn" {
  description = "ARN of the KMS key used for encryption"
  value       = aws_kms_key.main.arn
}

output "kms_alias_name" {
  description = "Name of the KMS key alias"
  value       = aws_kms_alias.main.name
}

output "kms_alias_arn" {
  description = "ARN of the KMS key alias"
  value       = aws_kms_alias.main.arn
}

# RDS Outputs
output "rds_endpoint" {
  description = "RDS instance endpoint"
  value       = var.enable_rds ? aws_db_instance.main[0].endpoint : null
  sensitive   = true
}

output "rds_port" {
  description = "RDS instance port"
  value       = var.enable_rds ? aws_db_instance.main[0].port : null
}

output "rds_db_name" {
  description = "RDS database name"
  value       = var.enable_rds ? aws_db_instance.main[0].db_name : null
}

output "rds_username" {
  description = "RDS database username"
  value       = var.enable_rds ? aws_db_instance.main[0].username : null
  sensitive   = true
}

output "rds_identifier" {
  description = "RDS instance identifier"
  value       = var.enable_rds ? aws_db_instance.main[0].identifier : null
}

output "rds_arn" {
  description = "RDS instance ARN"
  value       = var.enable_rds ? aws_db_instance.main[0].arn : null
}

output "rds_resource_id" {
  description = "RDS instance resource ID"
  value       = var.enable_rds ? aws_db_instance.main[0].resource_id : null
}

output "db_subnet_group_name" {
  description = "Name of the DB subnet group"
  value       = aws_db_subnet_group.main.name
}

output "db_subnet_group_id" {
  description = "ID of the DB subnet group"
  value       = aws_db_subnet_group.main.id
}

# ElastiCache Outputs
output "elasticache_primary_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = var.enable_elasticache ? aws_elasticache_replication_group.main[0].primary_endpoint_address : null
  sensitive   = true
}

output "elasticache_reader_endpoint" {
  description = "ElastiCache Redis reader endpoint"
  value       = var.enable_elasticache ? aws_elasticache_replication_group.main[0].reader_endpoint_address : null
  sensitive   = true
}

output "elasticache_port" {
  description = "ElastiCache Redis port"
  value       = var.enable_elasticache ? aws_elasticache_replication_group.main[0].port : null
}

output "elasticache_replication_group_id" {
  description = "ElastiCache replication group ID"
  value       = var.enable_elasticache ? aws_elasticache_replication_group.main[0].replication_group_id : null
}

output "elasticache_configuration_endpoint" {
  description = "ElastiCache configuration endpoint"
  value       = var.enable_elasticache ? aws_elasticache_replication_group.main[0].configuration_endpoint_address : null
  sensitive   = true
}

# S3 Outputs
output "s3_backup_bucket_id" {
  description = "ID of the S3 backup bucket"
  value       = var.enable_s3_backup ? aws_s3_bucket.backups[0].id : null
}

output "s3_backup_bucket_arn" {
  description = "ARN of the S3 backup bucket"
  value       = var.enable_s3_backup ? aws_s3_bucket.backups[0].arn : null
}

output "s3_backup_bucket_domain_name" {
  description = "Domain name of the S3 backup bucket"
  value       = var.enable_s3_backup ? aws_s3_bucket.backups[0].bucket_domain_name : null
}

output "s3_backup_bucket_regional_domain_name" {
  description = "Regional domain name of the S3 backup bucket"
  value       = var.enable_s3_backup ? aws_s3_bucket.backups[0].bucket_regional_domain_name : null
}

# ECR Outputs
output "ecr_repository_urls" {
  description = "Map of ECR repository URLs"
  value = {
    for k, v in aws_ecr_repository.main : k => v.repository_url
  }
}

output "ecr_repository_arns" {
  description = "Map of ECR repository ARNs"
  value = {
    for k, v in aws_ecr_repository.main : k => v.arn
  }
}

output "ecr_registry_id" {
  description = "ECR registry ID"
  value = length(aws_ecr_repository.main) > 0 ? values(aws_ecr_repository.main)[0].registry_id : null
}

# Load Balancer Outputs
output "load_balancer_id" {
  description = "ID of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.main[0].id : null
}

output "load_balancer_arn" {
  description = "ARN of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.main[0].arn : null
}

output "load_balancer_dns_name" {
  description = "DNS name of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.main[0].dns_name : null
}

output "load_balancer_hosted_zone_id" {
  description = "Hosted zone ID of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.main[0].zone_id : null
}

output "load_balancer_type" {
  description = "Type of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.main[0].load_balancer_type : null
}

# Regional Information
output "region" {
  description = "AWS region"
  value       = data.aws_region.current.name
}

output "region_description" {
  description = "AWS region description"
  value       = data.aws_region.current.description
}

output "account_id" {
  description = "AWS account ID"
  value       = data.aws_caller_identity.current.account_id
}

# Resource Naming
output "name_prefix" {
  description = "Name prefix used for resources"
  value       = local.name_prefix
}

output "common_tags" {
  description = "Common tags applied to all resources"
  value       = local.common_tags
}

# Monitoring and Logging
output "cloudwatch_log_groups" {
  description = "Map of CloudWatch log groups"
  value = {
    cluster_logs = aws_cloudwatch_log_group.cluster.name
    elasticache_logs = var.enable_elasticache ? aws_cloudwatch_log_group.elasticache_slow[0].name : null
  }
}

# Connection Information for Helm Values
output "cluster_connection_info" {
  description = "EKS cluster connection information for Helm deployment"
  value = {
    cluster_name     = aws_eks_cluster.main.name
    cluster_endpoint = aws_eks_cluster.main.endpoint
    cluster_ca_certificate = aws_eks_cluster.main.certificate_authority[0].data
    cluster_arn      = aws_eks_cluster.main.arn
    oidc_issuer_url  = aws_eks_cluster.main.identity[0].oidc[0].issuer
  }
  sensitive = true
}

output "database_connection_info" {
  description = "Database connection information for Helm deployment"
  value = var.enable_rds ? {
    host     = aws_db_instance.main[0].endpoint
    port     = aws_db_instance.main[0].port
    database = aws_db_instance.main[0].db_name
    username = aws_db_instance.main[0].username
  } : null
  sensitive = true
}

output "cache_connection_info" {
  description = "Cache connection information for Helm deployment"
  value = var.enable_elasticache ? {
    primary_endpoint = aws_elasticache_replication_group.main[0].primary_endpoint_address
    reader_endpoint  = aws_elasticache_replication_group.main[0].reader_endpoint_address
    port            = aws_elasticache_replication_group.main[0].port
  } : null
  sensitive = true
}

# Storage Information
output "storage_info" {
  description = "Storage information for Helm deployment"
  value = {
    s3_backup_bucket = var.enable_s3_backup ? aws_s3_bucket.backups[0].id : null
    kms_key_id       = aws_kms_key.main.id
    kms_key_arn      = aws_kms_key.main.arn
  }
}

# Network Information
output "network_info" {
  description = "Network information for Helm deployment"
  value = {
    vpc_id              = aws_vpc.main.id
    vpc_cidr            = aws_vpc.main.cidr_block
    private_subnets     = aws_subnet.private[*].id
    public_subnets      = aws_subnet.public[*].id
    database_subnets    = aws_subnet.database[*].id
    availability_zones  = local.azs
    nat_gateway_ips     = aws_eip.nat[*].public_ip
    security_groups = {
      cluster   = aws_security_group.eks_cluster.id
      nodes     = aws_security_group.eks_node_group.id
      rds       = aws_security_group.rds.id
      cache     = aws_security_group.elasticache.id
      alb       = var.enable_load_balancer ? aws_security_group.alb[0].id : null
    }
  }
}

# IAM Information
output "iam_info" {
  description = "IAM information for Helm deployment"
  value = {
    cluster_role_arn    = aws_iam_role.eks_cluster.arn
    node_group_role_arn = aws_iam_role.eks_node_group.arn
    oidc_provider_arn   = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${replace(aws_eks_cluster.main.identity[0].oidc[0].issuer, "https://", "")}"
  }
}

# Container Registry Information
output "container_registry_info" {
  description = "Container registry information for Helm deployment"
  value = {
    registry_url = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com"
    repositories = {
      for k, v in aws_ecr_repository.main : k => v.repository_url
    }
  }
}

# DNS and Load Balancing Information
output "dns_info" {
  description = "DNS and load balancing information for Helm deployment"
  value = var.enable_load_balancer ? {
    load_balancer_dns_name    = aws_lb.main[0].dns_name
    load_balancer_hosted_zone = aws_lb.main[0].zone_id
    load_balancer_arn         = aws_lb.main[0].arn
  } : null
}

# Comprehensive deployment information for automation scripts
output "deployment_info" {
  description = "Comprehensive deployment information for automation scripts"
  value = {
    infrastructure = {
      provider    = "aws"
      region      = data.aws_region.current.name
      account_id  = data.aws_caller_identity.current.account_id
      environment = var.environment
      project     = var.project_name
    }
    cluster = {
      name         = aws_eks_cluster.main.name
      endpoint     = aws_eks_cluster.main.endpoint
      version      = aws_eks_cluster.main.version
      arn          = aws_eks_cluster.main.arn
      oidc_issuer  = aws_eks_cluster.main.identity[0].oidc[0].issuer
    }
    networking = {
      vpc_id           = aws_vpc.main.id
      private_subnets  = aws_subnet.private[*].id
      public_subnets   = aws_subnet.public[*].id
      security_groups  = {
        cluster = aws_security_group.eks_cluster.id
        nodes   = aws_security_group.eks_node_group.id
      }
    }
    storage = {
      kms_key_arn      = aws_kms_key.main.arn
      backup_bucket    = var.enable_s3_backup ? aws_s3_bucket.backups[0].id : null
    }
    databases = var.enable_rds ? {
      postgres = {
        endpoint = aws_db_instance.main[0].endpoint
        port     = aws_db_instance.main[0].port
        database = aws_db_instance.main[0].db_name
      }
    } : {}
    cache = var.enable_elasticache ? {
      redis = {
        primary_endpoint = aws_elasticache_replication_group.main[0].primary_endpoint_address
        port            = aws_elasticache_replication_group.main[0].port
      }
    } : {}
  }
  sensitive = true
}