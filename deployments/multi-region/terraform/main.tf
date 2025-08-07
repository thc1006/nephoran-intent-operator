terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.47"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.47"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.24"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
  }

  backend "gcs" {
    bucket = "nephoran-terraform-state"
    prefix = "multi-region/state"
  }
}

# Provider configurations
provider "google" {
  project = var.project_id
}

provider "google-beta" {
  project = var.project_id
}

# Local variables
locals {
  regions = {
    primary = {
      name     = "us-central1"
      zones    = ["us-central1-a", "us-central1-b", "us-central1-c"]
      priority = 100
      type     = "primary"
    }
    secondary = {
      name     = "europe-west1"
      zones    = ["europe-west1-b", "europe-west1-c", "europe-west1-d"]
      priority = 90
      type     = "secondary"
    }
    tertiary = {
      name     = "asia-southeast1"
      zones    = ["asia-southeast1-a", "asia-southeast1-b", "asia-southeast1-c"]
      priority = 80
      type     = "tertiary"
    }
  }

  edge_regions = {
    us_east = {
      name  = "us-east1"
      zones = ["us-east1-b", "us-east1-c"]
      type  = "edge"
    }
    us_west = {
      name  = "us-west1"
      zones = ["us-west1-a", "us-west1-b"]
      type  = "edge"
    }
    eu_west = {
      name  = "europe-west4"
      zones = ["europe-west4-a", "europe-west4-b"]
      type  = "edge"
    }
  }

  all_regions = merge(local.regions, local.edge_regions)

  common_labels = {
    project     = "nephoran"
    environment = var.environment
    managed_by  = "terraform"
    component   = "multi-region-ha"
  }
}

# Enable required APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "servicenetworking.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "servicemesh.googleapis.com",
    "anthosconfigmanagement.googleapis.com",
    "gkehub.googleapis.com",
    "multiclusteringress.googleapis.com",
    "dns.googleapis.com",
    "certificatemanager.googleapis.com",
    "networkservices.googleapis.com",
    "trafficdirector.googleapis.com",
    "iap.googleapis.com",
    "secretmanager.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "cloudtrace.googleapis.com",
    "cloudprofiler.googleapis.com",
    "clouddebugger.googleapis.com"
  ])

  service                    = each.value
  disable_on_destroy        = false
  disable_dependent_services = false
}

# Create Artifact Registry for container images
resource "google_artifact_registry_repository" "main" {
  for_each = local.regions

  location      = each.value.name
  repository_id = "nephoran-${each.key}"
  description   = "Nephoran container registry for ${each.value.name}"
  format        = "DOCKER"

  labels = merge(local.common_labels, {
    region = each.value.name
    type   = each.value.type
  })
}

# Cloud Storage buckets for backups and state
resource "google_storage_bucket" "backup" {
  for_each = local.regions

  name          = "${var.project_id}-nephoran-backup-${each.value.name}"
  location      = upper(each.value.name)
  storage_class = "STANDARD"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type          = "SetStorageClass"
      storage_class = "COLDLINE"
    }
  }

  labels = merge(local.common_labels, {
    region  = each.value.name
    purpose = "backup"
  })
}

# Cross-region bucket replication
resource "google_storage_bucket" "global_backup" {
  name          = "${var.project_id}-nephoran-global-backup"
  location      = "US"
  storage_class = "STANDARD"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  labels = merge(local.common_labels, {
    purpose = "global-backup"
  })
}

# Cloud KMS for encryption
resource "google_kms_key_ring" "regional" {
  for_each = local.regions

  name     = "nephoran-${each.key}-keyring"
  location = each.value.name
}

resource "google_kms_crypto_key" "regional" {
  for_each = local.regions

  name     = "nephoran-${each.key}-key"
  key_ring = google_kms_key_ring.regional[each.key].id

  rotation_period = "7776000s" # 90 days

  lifecycle {
    prevent_destroy = true
  }
}

# Global resources
module "global_resources" {
  source = "./modules/global"

  project_id     = var.project_id
  environment    = var.environment
  regions        = local.regions
  edge_regions   = local.edge_regions
  common_labels  = local.common_labels
  domain_name    = var.domain_name
  dns_zone_name  = var.dns_zone_name
}

# VPC and networking
module "networking" {
  source = "./modules/networking"

  project_id    = var.project_id
  environment   = var.environment
  regions       = local.all_regions
  common_labels = local.common_labels

  depends_on = [google_project_service.required_apis]
}

# GKE clusters
module "gke_clusters" {
  source = "./modules/gke"

  for_each = local.regions

  project_id         = var.project_id
  environment        = var.environment
  region            = each.value.name
  zones             = each.value.zones
  cluster_name      = "nephoran-${each.key}"
  vpc_id            = module.networking.vpc_id
  subnet_id         = module.networking.subnet_ids[each.value.name]
  pods_range_name   = module.networking.pods_range_names[each.value.name]
  svc_range_name    = module.networking.services_range_names[each.value.name]
  common_labels     = local.common_labels
  kms_key_id        = google_kms_crypto_key.regional[each.key].id
  
  # Node pool configurations
  node_pools = var.node_pools[each.key]
  
  # Feature flags
  enable_autopilot           = false
  enable_workload_identity  = true
  enable_binary_authorization = true
  enable_shielded_nodes     = true
  enable_network_policy     = true
  enable_istio              = true
  enable_config_sync        = true
  
  depends_on = [
    module.networking,
    google_kms_crypto_key.regional
  ]
}

# Edge GKE clusters
module "edge_gke_clusters" {
  source = "./modules/gke"

  for_each = local.edge_regions

  project_id         = var.project_id
  environment        = var.environment
  region            = each.value.name
  zones             = each.value.zones
  cluster_name      = "nephoran-edge-${each.key}"
  vpc_id            = module.networking.vpc_id
  subnet_id         = module.networking.subnet_ids[each.value.name]
  pods_range_name   = module.networking.pods_range_names[each.value.name]
  svc_range_name    = module.networking.services_range_names[each.value.name]
  common_labels     = local.common_labels
  
  # Edge-specific configuration
  node_pools = var.edge_node_pools
  
  # Feature flags for edge clusters
  enable_autopilot           = false
  enable_workload_identity  = true
  enable_binary_authorization = true
  enable_shielded_nodes     = true
  enable_network_policy     = true
  enable_istio              = false  # Lighter weight for edge
  enable_config_sync        = true
  
  depends_on = [
    module.networking
  ]
}

# Multi-cluster ingress
module "multi_cluster_ingress" {
  source = "./modules/mci"

  project_id     = var.project_id
  environment    = var.environment
  config_cluster = module.gke_clusters["primary"].cluster_id
  
  member_clusters = {
    for k, v in module.gke_clusters : k => {
      cluster_id   = v.cluster_id
      cluster_name = v.cluster_name
      region       = local.regions[k].name
      priority     = local.regions[k].priority
    }
  }

  common_labels = local.common_labels
  domain_name   = var.domain_name

  depends_on = [module.gke_clusters]
}

# Weaviate clusters
module "weaviate" {
  source = "./modules/weaviate"

  project_id    = var.project_id
  environment   = var.environment
  regions       = local.regions
  vpc_id        = module.networking.vpc_id
  subnet_ids    = module.networking.subnet_ids
  common_labels = local.common_labels

  # Weaviate configuration
  weaviate_config = {
    version              = "1.24.1"
    replicas            = 3
    storage_size        = "100Gi"
    memory_limit        = "8Gi"
    cpu_limit           = "4"
    enable_persistence  = true
    enable_backup       = true
    backup_schedule     = "0 2 * * *"  # Daily at 2 AM
  }

  depends_on = [module.gke_clusters]
}

# Cloud SQL for metadata
module "cloud_sql" {
  source = "./modules/cloud-sql"

  project_id     = var.project_id
  environment    = var.environment
  primary_region = local.regions.primary.name
  replica_regions = [
    local.regions.secondary.name,
    local.regions.tertiary.name
  ]
  
  vpc_id                = module.networking.vpc_id
  private_ip_address    = module.networking.sql_private_ip
  common_labels         = local.common_labels

  # Database configuration
  database_version      = "POSTGRES_15"
  tier                 = "db-custom-4-16384"
  disk_size            = 100
  disk_type            = "PD_SSD"
  backup_configuration = {
    enabled                        = true
    start_time                    = "03:00"
    point_in_time_recovery_enabled = true
    transaction_log_retention_days = 7
    retained_backups              = 30
  }

  depends_on = [module.networking]
}

# Redis clusters for caching
module "redis" {
  source = "./modules/redis"

  for_each = local.regions

  project_id    = var.project_id
  environment   = var.environment
  region        = each.value.name
  vpc_id        = module.networking.vpc_id
  common_labels = local.common_labels

  # Redis configuration
  tier           = "STANDARD_HA"
  memory_size_gb = 5
  version        = "REDIS_7_0"
  
  # Enable auth and transit encryption
  auth_enabled            = true
  transit_encryption_mode = "SERVER_AUTHENTICATION"

  depends_on = [module.networking]
}

# Monitoring and observability
module "monitoring" {
  source = "./modules/monitoring"

  project_id    = var.project_id
  environment   = var.environment
  regions       = local.regions
  clusters      = module.gke_clusters
  common_labels = local.common_labels

  # Monitoring configuration
  enable_prometheus    = true
  enable_grafana      = true
  enable_alerting     = true
  enable_tracing      = true
  
  # Thanos configuration for global metrics
  thanos_config = {
    retention_raw      = "7d"
    retention_5m       = "30d"
    retention_1h       = "90d"
    storage_bucket     = google_storage_bucket.global_backup.name
  }

  # Alerting configuration
  alert_config = {
    slack_webhook_url = var.slack_webhook_url
    pagerduty_key    = var.pagerduty_integration_key
    email_addresses  = var.alert_email_addresses
  }

  depends_on = [module.gke_clusters]
}

# Outputs
output "cluster_endpoints" {
  description = "GKE cluster endpoints by region"
  value = {
    for k, v in module.gke_clusters : k => v.endpoint
  }
}

output "global_load_balancer_ip" {
  description = "Global load balancer IP address"
  value       = module.global_resources.global_load_balancer_ip
}

output "weaviate_endpoints" {
  description = "Weaviate cluster endpoints"
  value       = module.weaviate.endpoints
}

output "cloud_sql_connection" {
  description = "Cloud SQL connection details"
  value       = module.cloud_sql.connection_name
  sensitive   = true
}

output "redis_endpoints" {
  description = "Redis cluster endpoints"
  value = {
    for k, v in module.redis : k => v.host
  }
}

output "monitoring_urls" {
  description = "Monitoring dashboard URLs"
  value       = module.monitoring.dashboard_urls
}