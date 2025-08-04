# GKE Cluster
resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.region

  # Use regional cluster for HA
  node_locations = var.zones

  # Network configuration
  network    = var.vpc_id
  subnetwork = var.subnet_id

  # IP allocation policy for VPC-native cluster
  ip_allocation_policy {
    cluster_secondary_range_name  = var.pods_range_name
    services_secondary_range_name = var.svc_range_name
  }

  # Enable private cluster
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block = cidrsubnet("172.16.0.0/12", 12, 
      index(["us-central1", "europe-west1", "asia-southeast1", "us-east1", "us-west1", "europe-west4"], var.region)
    )
  }

  # Master authorized networks
  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "0.0.0.0/0"  # Adjust for production
      display_name = "All networks"
    }
  }

  # Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Release channel
  release_channel {
    channel = "RAPID"  # Options: RAPID, REGULAR, STABLE
  }

  # Cluster autoscaling
  cluster_autoscaling {
    enabled = true
    
    resource_limits {
      resource_type = "cpu"
      minimum       = 10
      maximum       = 1000
    }
    
    resource_limits {
      resource_type = "memory"
      minimum       = 40
      maximum       = 4000
    }

    auto_provisioning_defaults {
      min_cpu_platform = "Intel Cascade Lake"
      oauth_scopes = [
        "https://www.googleapis.com/auth/cloud-platform"
      ]
      
      management {
        auto_repair  = true
        auto_upgrade = true
      }

      shielded_instance_config {
        enable_secure_boot          = true
        enable_integrity_monitoring = true
      }
    }
  }

  # Addons
  addons_config {
    http_load_balancing {
      disabled = false
    }
    
    horizontal_pod_autoscaling {
      disabled = false
    }
    
    network_policy_config {
      disabled = !var.enable_network_policy
    }
    
    cloudrun_config {
      disabled = true
    }
    
    dns_cache_config {
      enabled = true
    }
    
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
    
    gcp_filestore_csi_driver_config {
      enabled = true
    }
    
    gcs_fuse_csi_driver_config {
      enabled = true
    }
    
    config_connector_config {
      enabled = true
    }
    
    gke_backup_agent_config {
      enabled = true
    }
    
    stateful_ha_config {
      enabled = true
    }
  }

  # Enable features
  enable_autopilot             = var.enable_autopilot
  enable_binary_authorization = var.enable_binary_authorization
  enable_shielded_nodes       = var.enable_shielded_nodes
  enable_intranode_visibility = true
  enable_l4_ilb_subsetting    = true

  # Maintenance policy
  maintenance_policy {
    daily_maintenance_window {
      start_time = "03:00"
    }
    
    recurring_window {
      start_time = "2023-01-01T00:00:00Z"
      end_time   = "2023-01-01T04:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=SU"
    }
  }

  # Security
  binary_authorization {
    evaluation_mode = var.enable_binary_authorization ? "PROJECT_SINGLETON_POLICY_ENFORCE" : "DISABLED"
  }

  # Database encryption
  database_encryption {
    state    = "ENCRYPTED"
    key_name = var.kms_key_id
  }

  # Network policy
  network_policy {
    enabled  = var.enable_network_policy
    provider = var.enable_network_policy ? "CALICO" : "PROVIDER_UNSPECIFIED"
  }

  # Pod security policy
  pod_security_policy_config {
    enabled = false  # Deprecated, use Pod Security Standards
  }

  # Resource usage export
  resource_usage_export_config {
    enable_network_egress_metering = true
    enable_resource_consumption_metering = true
    
    bigquery_destination {
      dataset_id = "${var.project_id}_gke_usage"
    }
  }

  # Logging and monitoring
  logging_config {
    enable_components = [
      "SYSTEM_COMPONENTS",
      "WORKLOADS",
      "APISERVER",
      "CONTROLLER_MANAGER",
      "SCHEDULER"
    ]
  }

  monitoring_config {
    enable_components = [
      "SYSTEM_COMPONENTS",
      "WORKLOADS",
      "APISERVER",
      "CONTROLLER_MANAGER",
      "SCHEDULER",
      "STORAGE",
      "POD",
      "DAEMONSET",
      "DEPLOYMENT",
      "STATEFULSET"
    ]

    managed_prometheus {
      enabled = true
    }
  }

  # Service mesh
  mesh_config {
    mode = var.enable_istio ? "FLEET" : "DISABLED"
  }

  # Cost management
  cost_management_config {
    enabled = true
  }

  # Gateway API
  gateway_api_config {
    channel = "CHANNEL_STANDARD"
  }

  # Notification config
  notification_config {
    pubsub {
      enabled = true
      topic   = "projects/${var.project_id}/topics/gke-cluster-notifications"
    }
  }

  # Labels
  resource_labels = merge(var.common_labels, {
    cluster_name = var.cluster_name
    region       = var.region
  })

  # Default node pool - minimal, will be removed
  initial_node_count = 1
  
  node_config {
    machine_type = "e2-micro"
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  lifecycle {
    ignore_changes = [
      initial_node_count,
      node_config,
    ]
  }

  # Remove default node pool immediately
  remove_default_node_pool = true

  depends_on = [
    var.vpc_id,
    var.subnet_id
  ]
}

# Node pools
resource "google_container_node_pool" "pools" {
  for_each = { for idx, pool in var.node_pools : pool.name => pool }

  name       = each.value.name
  location   = var.region
  cluster    = google_container_cluster.primary.name
  
  # Node locations - spread across zones
  node_locations = var.zones

  # Autoscaling
  autoscaling {
    min_node_count  = each.value.min_count
    max_node_count  = each.value.max_count
    location_policy = "BALANCED"
  }

  initial_node_count = each.value.initial_count

  # Management
  management {
    auto_repair  = each.value.auto_repair
    auto_upgrade = each.value.auto_upgrade
  }

  # Node configuration
  node_config {
    machine_type = each.value.machine_type
    disk_size_gb = each.value.disk_size_gb
    disk_type    = each.value.disk_type

    # Use spot/preemptible instances
    preemptible = each.value.preemptible
    spot        = each.value.spot

    # OAuth scopes
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    # Labels
    labels = merge(var.common_labels, each.value.node_labels, {
      node_pool = each.value.name
    })

    # Taints
    dynamic "taint" {
      for_each = each.value.node_taints
      content {
        key    = taint.value.key
        value  = taint.value.value
        effect = taint.value.effect
      }
    }

    # Metadata
    metadata = {
      disable-legacy-endpoints = "true"
    }

    # Service account
    service_account = google_service_account.gke_node.email

    # Workload Identity
    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    # Shielded instance
    shielded_instance_config {
      enable_secure_boot          = var.enable_shielded_nodes
      enable_integrity_monitoring = var.enable_shielded_nodes
    }

    # Sandbox (gVisor)
    dynamic "sandbox_config" {
      for_each = each.value.name == "sandbox" ? [1] : []
      content {
        sandbox_type = "gvisor"
      }
    }

    # Guest accelerator (GPU)
    dynamic "guest_accelerator" {
      for_each = each.value.name == "gpu" ? [1] : []
      content {
        type  = "nvidia-tesla-t4"
        count = 1
        
        gpu_driver_installation_config {
          gpu_driver_version = "DEFAULT"
        }
      }
    }

    # Reservation affinity
    reservation_affinity {
      consume_reservation_type = "NO_RESERVATION"
    }

    # Linux node config
    linux_node_config {
      sysctls = {
        "net.core.netdev_max_backlog" = "30000"
        "net.core.rmem_max"          = "134217728"
        "net.core.wmem_max"          = "134217728"
        "net.ipv4.tcp_rmem"          = "4096 87380 134217728"
        "net.ipv4.tcp_wmem"          = "4096 65536 134217728"
      }
    }

    # GcfsConfig for image streaming
    gcfs_config {
      enabled = true
    }

    # Advanced machine features
    advanced_machine_features {
      threads_per_core = 2
    }
  }

  # Upgrade settings
  upgrade_settings {
    max_surge       = 1
    max_unavailable = 0
    strategy        = "SURGE"
  }

  # Network config
  network_config {
    create_pod_range     = false
    enable_private_nodes = true
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [initial_node_count]
  }
}

# Service account for GKE nodes
resource "google_service_account" "gke_node" {
  account_id   = "${var.cluster_name}-node-sa"
  display_name = "GKE node service account for ${var.cluster_name}"
  description  = "Service account for GKE nodes in ${var.cluster_name}"
}

# IAM roles for node service account
resource "google_project_iam_member" "gke_node_roles" {
  for_each = toset([
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/monitoring.viewer",
    "roles/stackdriver.resourceMetadata.writer",
    "roles/storage.objectViewer",
    "roles/artifactregistry.reader"
  ])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.gke_node.email}"
}

# Fleet membership for multi-cluster features
resource "google_gke_hub_membership" "membership" {
  membership_id = var.cluster_name
  
  endpoint {
    gke_cluster {
      resource_link = google_container_cluster.primary.id
    }
  }

  authority {
    issuer = "https://container.googleapis.com/v1/${google_container_cluster.primary.id}"
  }
}

# Config Sync configuration
resource "google_gke_hub_feature_membership" "configsync" {
  count = var.enable_config_sync ? 1 : 0

  location   = "global"
  feature    = "configmanagement"
  membership = google_gke_hub_membership.membership.membership_id

  configmanagement {
    config_sync {
      git {
        sync_repo   = "https://github.com/${var.project_id}/config-sync"
        sync_branch = "main"
        policy_dir  = "clusters/${var.cluster_name}"
        sync_wait_secs = "20"
      }
    }
    
    policy_controller {
      enabled = true
      template_library_installed = true
      referential_rules_enabled = true
    }
  }
}

# Outputs
output "cluster_id" {
  description = "GKE cluster ID"
  value       = google_container_cluster.primary.id
}

output "cluster_name" {
  description = "GKE cluster name"
  value       = google_container_cluster.primary.name
}

output "endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.primary.endpoint
  sensitive   = true
}

output "master_version" {
  description = "Master version"
  value       = google_container_cluster.primary.master_version
}

output "ca_certificate" {
  description = "Cluster CA certificate"
  value       = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "node_pools" {
  description = "Node pool details"
  value = {
    for k, v in google_container_node_pool.pools : k => {
      name         = v.name
      node_count   = v.node_count
      node_version = v.version
      status       = v.status
    }
  }
}

output "service_account" {
  description = "GKE node service account"
  value       = google_service_account.gke_node.email
}

output "workload_identity_pool" {
  description = "Workload Identity pool"
  value       = "${var.project_id}.svc.id.goog"
}

output "fleet_membership_id" {
  description = "Fleet membership ID"
  value       = google_gke_hub_membership.membership.membership_id
}