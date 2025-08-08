# Azure Infrastructure Module for Nephoran Intent Operator
# Production-grade, carrier-grade, and enterprise-ready Azure infrastructure

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.80"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.45"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.24"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

# Configure the Azure Provider
provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy    = true
      recover_soft_deleted_key_vaults = true
    }
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# Data sources
data "azurerm_client_config" "current" {}
data "azuread_client_config" "current" {}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  
  common_tags = merge(var.additional_tags, {
    Project      = var.project_name
    Environment  = var.environment
    ManagedBy    = "terraform"
    Component    = "nephoran-intent-operator"
    Owner        = var.owner
    CostCenter   = var.cost_center
    Compliance   = var.compliance_level
  })
  
  # Location and availability zones
  primary_location = var.location
  availability_zones = var.availability_zones
  
  # AKS cluster configuration
  cluster_name = "${local.name_prefix}-aks"
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "${local.name_prefix}-rg"
  location = local.primary_location
  
  tags = local.common_tags
}

# Virtual Network
resource "azurerm_virtual_network" "main" {
  name                = "${local.name_prefix}-vnet"
  address_space       = [var.vnet_address_space]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  tags = local.common_tags
}

# Subnets
resource "azurerm_subnet" "aks_system" {
  name                 = "${local.name_prefix}-aks-system-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [var.aks_system_subnet_address_prefix]
  
  # Delegate subnet to AKS
  delegation {
    name = "aks-delegation"
    service_delegation {
      name    = "Microsoft.ContainerService/managedClusters"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "aks_user" {
  name                 = "${local.name_prefix}-aks-user-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [var.aks_user_subnet_address_prefix]
  
  # Delegate subnet to AKS
  delegation {
    name = "aks-delegation"
    service_delegation {
      name    = "Microsoft.ContainerService/managedClusters"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "database" {
  name                 = "${local.name_prefix}-database-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [var.database_subnet_address_prefix]
  
  # Delegate subnet to PostgreSQL
  delegation {
    name = "postgresql-delegation"
    service_delegation {
      name = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = [
        "Microsoft.Network/virtualNetworks/subnets/join/action"
      ]
    }
  }
}

resource "azurerm_subnet" "application_gateway" {
  count = var.enable_application_gateway ? 1 : 0
  
  name                 = "${local.name_prefix}-appgw-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [var.application_gateway_subnet_address_prefix]
}

# Network Security Groups
resource "azurerm_network_security_group" "aks_system" {
  name                = "${local.name_prefix}-aks-system-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  security_rule {
    name                       = "AllowKubernetesAPI"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = var.vnet_address_space
    destination_address_prefix = "*"
  }
  
  security_rule {
    name                       = "AllowSSH"
    priority                   = 1010
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefixes    = var.management_cidrs
    destination_address_prefix = "*"
  }
  
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 4000
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  
  tags = local.common_tags
}

resource "azurerm_network_security_group" "aks_user" {
  name                = "${local.name_prefix}-aks-user-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  security_rule {
    name                       = "AllowIntraSubnet"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = var.aks_user_subnet_address_prefix
    destination_address_prefix = var.aks_user_subnet_address_prefix
  }
  
  security_rule {
    name                       = "AllowFromSystemSubnet"
    priority                   = 1010
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = var.aks_system_subnet_address_prefix
    destination_address_prefix = "*"
  }
  
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 4000
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  
  tags = local.common_tags
}

resource "azurerm_network_security_group" "database" {
  name                = "${local.name_prefix}-database-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  security_rule {
    name                       = "AllowPostgreSQL"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "5432"
    source_address_prefixes    = [var.aks_system_subnet_address_prefix, var.aks_user_subnet_address_prefix]
    destination_address_prefix = "*"
  }
  
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 4000
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  
  tags = local.common_tags
}

# Associate NSGs with Subnets
resource "azurerm_subnet_network_security_group_association" "aks_system" {
  subnet_id                 = azurerm_subnet.aks_system.id
  network_security_group_id = azurerm_network_security_group.aks_system.id
}

resource "azurerm_subnet_network_security_group_association" "aks_user" {
  subnet_id                 = azurerm_subnet.aks_user.id
  network_security_group_id = azurerm_network_security_group.aks_user.id
}

resource "azurerm_subnet_network_security_group_association" "database" {
  subnet_id                 = azurerm_subnet.database.id
  network_security_group_id = azurerm_network_security_group.database.id
}

# Private DNS Zone for PostgreSQL
resource "azurerm_private_dns_zone" "postgresql" {
  count = var.enable_postgresql ? 1 : 0
  
  name                = "${local.name_prefix}.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
  
  tags = local.common_tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgresql" {
  count = var.enable_postgresql ? 1 : 0
  
  name                  = "${local.name_prefix}-postgresql-dns-link"
  private_dns_zone_name = azurerm_private_dns_zone.postgresql[0].name
  virtual_network_id    = azurerm_virtual_network.main.id
  resource_group_name   = azurerm_resource_group.main.name
  
  tags = local.common_tags
}

# Log Analytics Workspace
resource "azurerm_log_analytics_workspace" "main" {
  name                = "${local.name_prefix}-logs"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = var.log_analytics_sku
  retention_in_days   = var.log_analytics_retention_days
  
  tags = local.common_tags
}

# Application Insights
resource "azurerm_application_insights" "main" {
  name                = "${local.name_prefix}-appinsights"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  workspace_id        = azurerm_log_analytics_workspace.main.id
  application_type    = "web"
  
  tags = local.common_tags
}

# Key Vault
resource "azurerm_key_vault" "main" {
  name                = "${replace(local.name_prefix, "-", "")}kv${random_string.key_vault_suffix.result}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  
  sku_name = var.key_vault_sku
  
  enable_rbac_authorization       = true
  enabled_for_disk_encryption     = true
  enabled_for_deployment          = true
  enabled_for_template_deployment = true
  purge_protection_enabled        = var.enable_key_vault_purge_protection
  soft_delete_retention_days      = var.key_vault_soft_delete_retention_days
  
  network_acls {
    default_action             = "Deny"
    bypass                     = "AzureServices"
    virtual_network_subnet_ids = [azurerm_subnet.aks_system.id, azurerm_subnet.aks_user.id]
    ip_rules                   = var.key_vault_allowed_ips
  }
  
  tags = local.common_tags
}

resource "random_string" "key_vault_suffix" {
  length  = 6
  special = false
  upper   = false
}

# User-Assigned Managed Identity for AKS
resource "azurerm_user_assigned_identity" "aks" {
  name                = "${local.name_prefix}-aks-identity"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  tags = local.common_tags
}

# Role assignments for AKS identity
resource "azurerm_role_assignment" "aks_network_contributor" {
  scope                = azurerm_virtual_network.main.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.aks.principal_id
}

resource "azurerm_role_assignment" "aks_monitoring_metrics_publisher" {
  scope                = azurerm_resource_group.main.id
  role_definition_name = "Monitoring Metrics Publisher"
  principal_id         = azurerm_user_assigned_identity.aks.principal_id
}

# Container Registry
resource "azurerm_container_registry" "main" {
  name                = "${replace(local.name_prefix, "-", "")}acr${random_string.acr_suffix.result}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = var.container_registry_sku
  admin_enabled       = false
  
  # Enable system-assigned managed identity
  identity {
    type = "SystemAssigned"
  }
  
  # Network access rules
  public_network_access_enabled = var.container_registry_public_access
  network_rule_bypass_option    = "AzureServices"
  
  dynamic "network_rule_set" {
    for_each = var.container_registry_public_access ? [] : [1]
    content {
      default_action = "Deny"
      
      virtual_network {
        action    = "Allow"
        subnet_id = azurerm_subnet.aks_system.id
      }
      
      virtual_network {
        action    = "Allow"
        subnet_id = azurerm_subnet.aks_user.id
      }
      
      dynamic "ip_rule" {
        for_each = var.container_registry_allowed_ips
        content {
          action   = "Allow"
          ip_range = ip_rule.value
        }
      }
    }
  }
  
  # Encryption
  dynamic "encryption" {
    for_each = var.enable_container_registry_encryption ? [1] : []
    content {
      enabled            = true
      key_vault_key_id   = azurerm_key_vault_key.acr[0].id
      identity_client_id = azurerm_user_assigned_identity.acr[0].client_id
    }
  }
  
  tags = local.common_tags
}

resource "random_string" "acr_suffix" {
  length  = 6
  special = false
  upper   = false
}

# User-Assigned Identity for ACR encryption
resource "azurerm_user_assigned_identity" "acr" {
  count = var.enable_container_registry_encryption ? 1 : 0
  
  name                = "${local.name_prefix}-acr-encryption-identity"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  tags = local.common_tags
}

# Key Vault key for ACR encryption
resource "azurerm_key_vault_key" "acr" {
  count = var.enable_container_registry_encryption ? 1 : 0
  
  name         = "${local.name_prefix}-acr-encryption-key"
  key_vault_id = azurerm_key_vault.main.id
  key_type     = "RSA"
  key_size     = 2048
  
  key_opts = [
    "decrypt",
    "encrypt",
    "sign",
    "unwrapKey",
    "verify",
    "wrapKey",
  ]
  
  tags = local.common_tags
  
  depends_on = [
    azurerm_role_assignment.current_user_key_vault_admin
  ]
}

# Role assignment for current user to manage Key Vault
resource "azurerm_role_assignment" "current_user_key_vault_admin" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Administrator"
  principal_id         = data.azuread_client_config.current.object_id
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "main" {
  name                = local.cluster_name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = "${local.name_prefix}-k8s"
  
  kubernetes_version        = var.kubernetes_version
  automatic_channel_upgrade = var.aks_automatic_channel_upgrade
  sku_tier                 = var.aks_sku_tier
  
  # System node pool
  default_node_pool {
    name                         = "system"
    vm_size                     = var.system_node_pool_vm_size
    node_count                  = var.system_node_pool_node_count
    min_count                   = var.system_node_pool_min_count
    max_count                   = var.system_node_pool_max_count
    enable_auto_scaling         = var.system_node_pool_enable_auto_scaling
    availability_zones          = local.availability_zones
    enable_node_public_ip       = false
    vnet_subnet_id             = azurerm_subnet.aks_system.id
    type                       = "VirtualMachineScaleSets"
    orchestrator_version       = var.kubernetes_version
    
    # Security and performance
    enable_host_encryption     = var.enable_host_encryption
    fips_enabled              = var.enable_fips
    
    # Taints for system workloads
    node_taints = var.system_node_pool_taints
    
    # Node labels
    node_labels = merge(var.system_node_pool_labels, {
      "nephoran.com/node-type" = "system"
      "kubernetes.io/role"     = "system"
    })
    
    # OS and storage configuration
    os_disk_type         = "Managed"
    os_disk_size_gb     = var.system_node_pool_os_disk_size
    os_sku              = var.node_pool_os_sku
    
    upgrade_settings {
      max_surge = "10%"
    }
    
    tags = merge(local.common_tags, {
      NodePool = "system"
      Role     = "system"
    })
  }
  
  # Identity configuration
  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks.id]
  }
  
  # Network profile
  network_profile {
    network_plugin      = var.network_plugin
    network_policy      = var.network_policy
    dns_service_ip      = var.dns_service_ip
    service_cidr        = var.service_cidr
    load_balancer_sku   = "standard"
    outbound_type       = var.outbound_type
    
    # Load balancer profile for advanced configurations
    dynamic "load_balancer_profile" {
      for_each = var.outbound_type == "loadBalancer" ? [1] : []
      content {
        outbound_ports_allocated  = var.load_balancer_outbound_ports_allocated
        idle_timeout_in_minutes  = var.load_balancer_idle_timeout_in_minutes
        managed_outbound_ip_count = var.load_balancer_managed_outbound_ip_count
      }
    }
  }
  
  # Azure AD integration
  azure_active_directory_role_based_access_control {
    managed                = true
    admin_group_object_ids = var.aks_admin_group_object_ids
    azure_rbac_enabled     = true
  }
  
  # OMS Agent (Azure Monitor)
  oms_agent {
    log_analytics_workspace_id      = azurerm_log_analytics_workspace.main.id
    msi_auth_for_monitoring_enabled = true
  }
  
  # Microsoft Defender
  microsoft_defender {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  }
  
  # Key Vault Secrets Provider
  key_vault_secrets_provider {
    secret_rotation_enabled  = true
    secret_rotation_interval = "2m"
  }
  
  # Workload Identity
  workload_identity_enabled = true
  oidc_issuer_enabled      = true
  
  # API Server configuration
  api_server_access_profile {
    authorized_ip_ranges     = var.api_server_authorized_ip_ranges
    vnet_integration_enabled = var.api_server_vnet_integration_enabled
    subnet_id               = var.api_server_vnet_integration_enabled ? azurerm_subnet.aks_system.id : null
  }
  
  # Storage profile
  storage_profile {
    blob_driver_enabled         = true
    disk_driver_enabled         = true
    disk_driver_version         = "v1"
    file_driver_enabled         = true
    snapshot_controller_enabled = true
  }
  
  # Image Cleaner
  image_cleaner_enabled        = true
  image_cleaner_interval_hours = var.image_cleaner_interval_hours
  
  # HTTP Application Routing (disabled for production)
  http_application_routing_enabled = false
  
  # Maintenance window
  dynamic "maintenance_window" {
    for_each = var.maintenance_window != null ? [var.maintenance_window] : []
    content {
      allowed {
        day   = maintenance_window.value.day
        hours = maintenance_window.value.hours
      }
    }
  }
  
  # Auto-scaler profile
  auto_scaler_profile {
    balance_similar_node_groups      = var.auto_scaler_profile.balance_similar_node_groups
    expander                        = var.auto_scaler_profile.expander
    max_graceful_termination_sec    = var.auto_scaler_profile.max_graceful_termination_sec
    max_node_provisioning_time      = var.auto_scaler_profile.max_node_provisioning_time
    max_unready_nodes              = var.auto_scaler_profile.max_unready_nodes
    max_unready_percentage         = var.auto_scaler_profile.max_unready_percentage
    new_pod_scale_up_delay         = var.auto_scaler_profile.new_pod_scale_up_delay
    scale_down_delay_after_add     = var.auto_scaler_profile.scale_down_delay_after_add
    scale_down_delay_after_delete  = var.auto_scaler_profile.scale_down_delay_after_delete
    scale_down_delay_after_failure = var.auto_scaler_profile.scale_down_delay_after_failure
    scale_down_unneeded           = var.auto_scaler_profile.scale_down_unneeded
    scale_down_unready            = var.auto_scaler_profile.scale_down_unready
    scale_down_utilization_threshold = var.auto_scaler_profile.scale_down_utilization_threshold
    scan_interval                 = var.auto_scaler_profile.scan_interval
    skip_nodes_with_local_storage = var.auto_scaler_profile.skip_nodes_with_local_storage
    skip_nodes_with_system_pods   = var.auto_scaler_profile.skip_nodes_with_system_pods
  }
  
  tags = merge(local.common_tags, {
    Type = "kubernetes-cluster"
  })
  
  depends_on = [
    azurerm_role_assignment.aks_network_contributor,
    azurerm_role_assignment.aks_monitoring_metrics_publisher,
  ]
}

# Additional Node Pools
resource "azurerm_kubernetes_cluster_node_pool" "additional" {
  for_each = var.additional_node_pools
  
  name                  = each.key
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size              = each.value.vm_size
  node_count           = each.value.node_count
  min_count            = each.value.min_count
  max_count            = each.value.max_count
  enable_auto_scaling  = each.value.enable_auto_scaling
  availability_zones   = each.value.availability_zones
  enable_node_public_ip = false
  vnet_subnet_id       = azurerm_subnet.aks_user.id
  
  # Performance and security
  enable_host_encryption = var.enable_host_encryption
  fips_enabled          = var.enable_fips
  
  # Node configuration
  node_taints = each.value.node_taints
  node_labels = merge(each.value.node_labels, {
    "nephoran.com/node-pool" = each.key
  })
  
  # OS and storage
  os_type         = each.value.os_type
  os_disk_type    = "Managed"
  os_disk_size_gb = each.value.os_disk_size_gb
  os_sku          = var.node_pool_os_sku
  
  # Spot instances configuration
  priority        = each.value.priority
  eviction_policy = each.value.priority == "Spot" ? each.value.eviction_policy : null
  spot_max_price  = each.value.priority == "Spot" ? each.value.spot_max_price : null
  
  # Kubernetes version
  orchestrator_version = var.kubernetes_version
  
  upgrade_settings {
    max_surge = each.value.max_surge
  }
  
  tags = merge(local.common_tags, {
    NodePool = each.key
    Role     = each.value.role
  })
}

# PostgreSQL Flexible Server
resource "azurerm_postgresql_flexible_server" "main" {
  count = var.enable_postgresql ? 1 : 0
  
  name                   = "${local.name_prefix}-postgres"
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = var.postgresql_version
  delegated_subnet_id    = azurerm_subnet.database.id
  private_dns_zone_id    = azurerm_private_dns_zone.postgresql[0].id
  
  administrator_login    = var.postgresql_admin_login
  administrator_password = var.postgresql_admin_password
  
  zone                   = var.postgresql_availability_zone
  
  storage_mb   = var.postgresql_storage_mb
  storage_tier = var.postgresql_storage_tier
  
  sku_name = var.postgresql_sku_name
  
  backup_retention_days        = var.postgresql_backup_retention_days
  geo_redundant_backup_enabled = var.postgresql_geo_redundant_backup_enabled
  
  auto_grow_enabled = true
  
  # High availability
  dynamic "high_availability" {
    for_each = var.postgresql_high_availability_enabled ? [1] : []
    content {
      mode                      = "ZoneRedundant"
      standby_availability_zone = var.postgresql_standby_availability_zone
    }
  }
  
  # Maintenance window
  dynamic "maintenance_window" {
    for_each = var.postgresql_maintenance_window != null ? [var.postgresql_maintenance_window] : []
    content {
      day_of_week  = maintenance_window.value.day_of_week
      start_hour   = maintenance_window.value.start_hour
      start_minute = maintenance_window.value.start_minute
    }
  }
  
  tags = merge(local.common_tags, {
    Type = "postgresql-database"
  })
  
  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgresql]
}

# PostgreSQL Configuration
resource "azurerm_postgresql_flexible_server_configuration" "extensions" {
  count = var.enable_postgresql ? length(var.postgresql_extensions) : 0
  
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.main[0].id
  value     = join(",", var.postgresql_extensions)
}

# PostgreSQL Database
resource "azurerm_postgresql_flexible_server_database" "main" {
  count = var.enable_postgresql ? 1 : 0
  
  name      = var.postgresql_database_name
  server_id = azurerm_postgresql_flexible_server.main[0].id
  collation = "en_US.utf8"
  charset   = "utf8"
}

# Redis Cache
resource "azurerm_redis_cache" "main" {
  count = var.enable_redis ? 1 : 0
  
  name                = "${local.name_prefix}-redis"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  capacity            = var.redis_capacity
  family              = var.redis_family
  sku_name            = var.redis_sku_name
  
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"
  
  # Redis configuration
  redis_configuration {
    enable_authentication           = true
    maxmemory_reserved             = var.redis_maxmemory_reserved
    maxmemory_delta                = var.redis_maxmemory_delta
    maxmemory_policy               = var.redis_maxmemory_policy
    notify_keyspace_events         = var.redis_notify_keyspace_events
    rdb_backup_enabled             = var.redis_rdb_backup_enabled
    rdb_backup_frequency           = var.redis_rdb_backup_enabled ? var.redis_rdb_backup_frequency : null
    rdb_backup_max_snapshot_count  = var.redis_rdb_backup_enabled ? var.redis_rdb_backup_max_snapshot_count : null
    rdb_storage_connection_string  = var.redis_rdb_backup_enabled ? var.redis_rdb_storage_connection_string : null
  }
  
  # Patch schedule
  dynamic "patch_schedule" {
    for_each = var.redis_patch_schedule != null ? [var.redis_patch_schedule] : []
    content {
      day_of_week        = patch_schedule.value.day_of_week
      start_hour_utc     = patch_schedule.value.start_hour_utc
      maintenance_window = patch_schedule.value.maintenance_window
    }
  }
  
  tags = merge(local.common_tags, {
    Type = "redis-cache"
  })
}

# Storage Account for backups
resource "azurerm_storage_account" "backups" {
  count = var.enable_storage_backup ? 1 : 0
  
  name                = "${replace(local.name_prefix, "-", "")}backups${random_string.storage_suffix.result}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  
  account_tier                     = "Standard"
  account_replication_type         = var.backup_storage_replication_type
  account_kind                     = "StorageV2"
  access_tier                      = "Hot"
  enable_https_traffic_only        = true
  min_tls_version                 = "TLS1_2"
  allow_nested_items_to_be_public = false
  shared_access_key_enabled       = false
  
  # Network access rules
  public_network_access_enabled = var.backup_storage_public_access
  
  dynamic "network_rules" {
    for_each = var.backup_storage_public_access ? [] : [1]
    content {
      default_action                 = "Deny"
      bypass                        = ["AzureServices"]
      virtual_network_subnet_ids    = [azurerm_subnet.aks_system.id, azurerm_subnet.aks_user.id]
      ip_rules                      = var.backup_storage_allowed_ips
    }
  }
  
  # Blob properties
  blob_properties {
    delete_retention_policy {
      days = var.backup_blob_retention_days
    }
    
    container_delete_retention_policy {
      days = var.backup_container_retention_days
    }
    
    versioning_enabled       = true
    last_access_time_enabled = true
    change_feed_enabled     = true
  }
  
  # Queue properties  
  queue_properties {
    logging {
      delete                = true
      read                  = true
      write                 = true
      version               = "1.0"
      retention_policy_days = 10
    }
    
    minute_metrics {
      enabled               = true
      version               = "1.0"
      include_apis          = true
      retention_policy_days = 10
    }
  }
  
  tags = merge(local.common_tags, {
    Type = "backup-storage"
  })
}

resource "random_string" "storage_suffix" {
  length  = 6
  special = false
  upper   = false
}

# Application Gateway (if enabled)
resource "azurerm_public_ip" "application_gateway" {
  count = var.enable_application_gateway ? 1 : 0
  
  name                = "${local.name_prefix}-appgw-pip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                = "Standard"
  zones              = local.availability_zones
  
  tags = local.common_tags
}

resource "azurerm_application_gateway" "main" {
  count = var.enable_application_gateway ? 1 : 0
  
  name                = "${local.name_prefix}-appgw"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  zones              = local.availability_zones
  
  sku {
    name     = var.application_gateway_sku_name
    tier     = var.application_gateway_sku_tier
    capacity = var.application_gateway_capacity
  }
  
  # WAF configuration (if WAF SKU is selected)
  dynamic "waf_configuration" {
    for_each = var.application_gateway_sku_tier == "WAF_v2" ? [1] : []
    content {
      enabled          = true
      firewall_mode    = var.application_gateway_waf_mode
      rule_set_type    = "OWASP"
      rule_set_version = "3.2"
      
      dynamic "disabled_rule_group" {
        for_each = var.application_gateway_waf_disabled_rule_groups
        content {
          rule_group_name = disabled_rule_group.value.rule_group_name
          rules          = disabled_rule_group.value.rules
        }
      }
    }
  }
  
  gateway_ip_configuration {
    name      = "gateway-ip-configuration"
    subnet_id = azurerm_subnet.application_gateway[0].id
  }
  
  frontend_ip_configuration {
    name                 = "frontend-ip-configuration"
    public_ip_address_id = azurerm_public_ip.application_gateway[0].id
  }
  
  frontend_port {
    name = "frontend-port-80"
    port = 80
  }
  
  frontend_port {
    name = "frontend-port-443"
    port = 443
  }
  
  backend_address_pool {
    name = "backend-address-pool"
  }
  
  backend_http_settings {
    name                  = "backend-http-settings"
    cookie_based_affinity = "Disabled"
    path                 = "/"
    port                 = 80
    protocol             = "Http"
    request_timeout      = 60
  }
  
  http_listener {
    name                           = "http-listener"
    frontend_ip_configuration_name = "frontend-ip-configuration"
    frontend_port_name            = "frontend-port-80"
    protocol                      = "Http"
  }
  
  request_routing_rule {
    name                       = "request-routing-rule"
    rule_type                 = "Basic"
    http_listener_name        = "http-listener"
    backend_address_pool_name = "backend-address-pool"
    backend_http_settings_name = "backend-http-settings"
    priority                  = 1
  }
  
  # Enable autoscaling
  dynamic "autoscale_configuration" {
    for_each = var.application_gateway_enable_autoscaling ? [1] : []
    content {
      min_capacity = var.application_gateway_autoscale_min_capacity
      max_capacity = var.application_gateway_autoscale_max_capacity
    }
  }
  
  tags = merge(local.common_tags, {
    Type = "application-gateway"
  })
}

# Diagnostic Settings
resource "azurerm_monitor_diagnostic_setting" "aks" {
  name               = "${local.name_prefix}-aks-diagnostics"
  target_resource_id = azurerm_kubernetes_cluster.main.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  
  dynamic "enabled_log" {
    for_each = var.aks_diagnostic_logs
    content {
      category = enabled_log.value
    }
  }
  
  dynamic "metric" {
    for_each = var.aks_diagnostic_metrics
    content {
      category = metric.value
      enabled  = true
    }
  }
}