# VPC Network
resource "google_compute_network" "vpc" {
  name                            = "${var.project_id}-nephoran-vpc"
  auto_create_subnetworks        = false
  routing_mode                   = "GLOBAL"
  delete_default_routes_on_create = false
  mtu                           = 1460

  description = "Multi-region VPC for Nephoran Intent Operator"
}

# Firewall rules
resource "google_compute_firewall" "allow_internal" {
  name    = "${var.project_id}-allow-internal"
  network = google_compute_network.vpc.self_link

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }

  source_ranges = [for subnet in google_compute_subnetwork.subnet : subnet.ip_cidr_range]
  
  priority = 1000
}

resource "google_compute_firewall" "allow_health_checks" {
  name    = "${var.project_id}-allow-health-checks"
  network = google_compute_network.vpc.self_link

  allow {
    protocol = "tcp"
    ports    = ["80", "443", "8080", "8443"]
  }

  source_ranges = [
    "35.191.0.0/16",  # Google Cloud health check ranges
    "130.211.0.0/22"
  ]
  
  priority = 1000
}

# Regional subnets
resource "google_compute_subnetwork" "subnet" {
  for_each = var.regions

  name          = "${var.project_id}-subnet-${each.key}"
  network       = google_compute_network.vpc.self_link
  region        = each.value.name
  ip_cidr_range = cidrsubnet("10.0.0.0/8", 8, index(keys(var.regions), each.key))

  private_ip_google_access = true

  secondary_ip_range {
    range_name    = "${each.key}-pods"
    ip_cidr_range = cidrsubnet("172.16.0.0/12", 4, index(keys(var.regions), each.key))
  }

  secondary_ip_range {
    range_name    = "${each.key}-services"
    ip_cidr_range = cidrsubnet("192.168.0.0/16", 8, index(keys(var.regions), each.key))
  }

  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling       = 0.5
    metadata           = "INCLUDE_ALL_METADATA"
  }
}

# Cloud NAT for each region
resource "google_compute_router" "router" {
  for_each = var.regions

  name    = "${var.project_id}-router-${each.key}"
  network = google_compute_network.vpc.self_link
  region  = each.value.name

  bgp {
    asn = 64514 + index(keys(var.regions), each.key)
  }
}

resource "google_compute_router_nat" "nat" {
  for_each = var.regions

  name                               = "${var.project_id}-nat-${each.key}"
  router                            = google_compute_router.router[each.key].name
  region                           = each.value.name
  nat_ip_allocate_option           = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# Private Service Connection for Google APIs
resource "google_compute_global_address" "private_ip_address" {
  name          = "${var.project_id}-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

# VPC Peering for cross-region connectivity (if needed)
resource "google_compute_network_peering" "peering" {
  for_each = {
    for pair in setproduct(keys(var.regions), keys(var.regions)) : 
    "${pair[0]}-${pair[1]}" => pair
    if pair[0] < pair[1]  # Only create one peering per pair
  }

  name         = "peering-${each.value[0]}-to-${each.value[1]}"
  network      = google_compute_network.vpc.self_link
  peer_network = google_compute_network.vpc.self_link
  
  export_custom_routes = true
  import_custom_routes = true
}

# Private Service Connect endpoints for Google APIs
resource "google_compute_address" "psc_ilb_consumer_address" {
  for_each = var.regions

  name         = "${var.project_id}-psc-${each.key}"
  region       = each.value.name
  address_type = "INTERNAL"
  subnetwork   = google_compute_subnetwork.subnet[each.key].id
  purpose      = "GCE_ENDPOINT"
}

resource "google_compute_forwarding_rule" "psc_ilb_consumer" {
  for_each = var.regions

  name                  = "${var.project_id}-psc-fr-${each.key}"
  region               = each.value.name
  target               = "all-apis"
  load_balancing_scheme = ""
  network              = google_compute_network.vpc.id
  ip_address           = google_compute_address.psc_ilb_consumer_address[each.key].id
}

# DNS configuration for Private Service Connect
resource "google_dns_managed_zone" "private" {
  name        = "${var.project_id}-private-zone"
  dns_name    = "googleapis.com."
  description = "Private DNS zone for Google APIs"

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.vpc.id
    }
  }
}

resource "google_dns_record_set" "private_googleapis" {
  for_each = toset([
    "storage",
    "bigquery",
    "bigtable",
    "dataflow",
    "dataproc",
    "cloudkms",
    "pubsub",
    "spanner",
    "sqladmin",
    "compute",
    "container",
    "containerregistry",
    "containeranalysis",
    "gkeconnect",
    "gkehub",
    "logging",
    "monitoring",
    "servicecontrol",
    "servicemanagement"
  ])

  name         = "*.${each.value}.googleapis.com."
  managed_zone = google_dns_managed_zone.private.name
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["${each.value}.googleapis.com."]
}

# Network Connectivity Center Hub
resource "google_network_connectivity_hub" "hub" {
  name        = "${var.project_id}-hub"
  description = "Network Connectivity Center hub for multi-region connectivity"
  
  labels = var.common_labels
}

# Spokes for each region
resource "google_network_connectivity_spoke" "spoke" {
  for_each = var.regions

  name        = "${var.project_id}-spoke-${each.key}"
  location    = each.value.name
  description = "Spoke for ${each.value.name} region"
  hub         = google_network_connectivity_hub.hub.id
  
  labels = merge(var.common_labels, {
    region = each.value.name
  })

  linked_vpn_tunnels {
    uris = []  # Add VPN tunnel URIs if using VPN
    site_to_site_data_transfer = true
  }
}

# Outputs
output "vpc_id" {
  description = "VPC network ID"
  value       = google_compute_network.vpc.id
}

output "vpc_self_link" {
  description = "VPC network self link"
  value       = google_compute_network.vpc.self_link
}

output "subnet_ids" {
  description = "Subnet IDs by region"
  value       = { for k, v in google_compute_subnetwork.subnet : k => v.id }
}

output "subnet_self_links" {
  description = "Subnet self links by region"
  value       = { for k, v in google_compute_subnetwork.subnet : k => v.self_link }
}

output "pods_range_names" {
  description = "Pod IP range names by region"
  value       = { for k, v in google_compute_subnetwork.subnet : k => "${k}-pods" }
}

output "services_range_names" {
  description = "Service IP range names by region"
  value       = { for k, v in google_compute_subnetwork.subnet : k => "${k}-services" }
}

output "router_ids" {
  description = "Router IDs by region"
  value       = { for k, v in google_compute_router.router : k => v.id }
}

output "nat_ips" {
  description = "NAT IPs by region"
  value       = { for k, v in google_compute_router_nat.nat : k => v.nat_ips }
}

output "private_vpc_connection" {
  description = "Private VPC connection for Google services"
  value       = google_service_networking_connection.private_vpc_connection.id
}

output "sql_private_ip" {
  description = "Reserved IP range for Cloud SQL"
  value       = google_compute_global_address.private_ip_address.address
}

output "hub_id" {
  description = "Network Connectivity Center hub ID"
  value       = google_network_connectivity_hub.hub.id
}