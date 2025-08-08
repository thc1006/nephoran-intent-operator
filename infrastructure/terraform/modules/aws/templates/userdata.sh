#!/bin/bash

# User data script for EKS worker nodes
# Nephoran Intent Operator - Production-grade node configuration

set -o xtrace

# Update system packages
yum update -y

# Install additional packages for Nephoran workloads
yum install -y \
    wget \
    curl \
    jq \
    htop \
    iotop \
    sysstat \
    tcpdump \
    nmap-ncat \
    bind-utils \
    git \
    unzip \
    tree \
    amazon-cloudwatch-agent \
    collectd

# Configure CloudWatch agent for enhanced monitoring
cat > /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json << 'EOF'
{
    "agent": {
        "metrics_collection_interval": 60,
        "run_as_user": "cwagent"
    },
    "metrics": {
        "namespace": "CWAgent",
        "metrics_collected": {
            "cpu": {
                "measurement": [
                    "cpu_usage_idle",
                    "cpu_usage_iowait",
                    "cpu_usage_user",
                    "cpu_usage_system"
                ],
                "metrics_collection_interval": 60,
                "totalcpu": false
            },
            "disk": {
                "measurement": [
                    "used_percent"
                ],
                "metrics_collection_interval": 60,
                "resources": [
                    "*"
                ]
            },
            "diskio": {
                "measurement": [
                    "io_time"
                ],
                "metrics_collection_interval": 60,
                "resources": [
                    "*"
                ]
            },
            "mem": {
                "measurement": [
                    "mem_used_percent"
                ],
                "metrics_collection_interval": 60
            },
            "netstat": {
                "measurement": [
                    "tcp_established",
                    "tcp_time_wait"
                ],
                "metrics_collection_interval": 60
            },
            "swap": {
                "measurement": [
                    "swap_used_percent"
                ],
                "metrics_collection_interval": 60
            }
        }
    },
    "logs": {
        "logs_collected": {
            "files": {
                "collect_list": [
                    {
                        "file_path": "/var/log/messages",
                        "log_group_name": "/aws/ec2/nephoran-nodes",
                        "log_stream_name": "{instance_id}/messages"
                    },
                    {
                        "file_path": "/var/log/secure",
                        "log_group_name": "/aws/ec2/nephoran-nodes",
                        "log_stream_name": "{instance_id}/secure"
                    },
                    {
                        "file_path": "/var/log/cloud-init.log",
                        "log_group_name": "/aws/ec2/nephoran-nodes",
                        "log_stream_name": "{instance_id}/cloud-init"
                    }
                ]
            }
        }
    }
}
EOF

# Start CloudWatch agent
/opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl \
    -a fetch-config \
    -m ec2 \
    -c file:/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json \
    -s

# Configure kernel parameters for high-performance networking
cat >> /etc/sysctl.conf << 'EOF'
# Nephoran Intent Operator - Network optimizations
net.core.rmem_default = 262144
net.core.rmem_max = 16777216
net.core.wmem_default = 262144
net.core.wmem_max = 16777216
net.core.netdev_max_backlog = 30000
net.core.netdev_budget = 600
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_mtu_probing = 1
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_max_tw_buckets = 2000000
net.ipv4.tcp_fin_timeout = 10
net.ipv4.tcp_keepalive_time = 600
net.ipv4.tcp_keepalive_intvl = 60
net.ipv4.tcp_keepalive_probes = 3

# Memory management optimizations
vm.swappiness = 1
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
vm.vfs_cache_pressure = 50

# File system optimizations
fs.file-max = 2097152
fs.nr_open = 1048576
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 512

# Security enhancements
kernel.dmesg_restrict = 1
kernel.kptr_restrict = 2
kernel.yama.ptrace_scope = 1
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0
net.ipv4.icmp_ignore_bogus_error_responses = 1
net.ipv4.icmp_echo_ignore_broadcasts = 1
EOF

# Apply sysctl settings
sysctl -p

# Configure limits for high-performance applications
cat >> /etc/security/limits.conf << 'EOF'
# Nephoran Intent Operator - Resource limits
* soft nofile 1048576
* hard nofile 1048576
* soft nproc 1048576
* hard nproc 1048576
* soft memlock unlimited
* hard memlock unlimited
root soft nofile 1048576
root hard nofile 1048576
root soft nproc 1048576
root hard nproc 1048576
root soft memlock unlimited
root hard memlock unlimited
EOF

# Configure systemd limits
mkdir -p /etc/systemd/system.conf.d
cat > /etc/systemd/system.conf.d/limits.conf << 'EOF'
[Manager]
DefaultLimitNOFILE=1048576
DefaultLimitNPROC=1048576
DefaultLimitMEMLOCK=infinity
EOF

# Install AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
./aws/install
rm -rf aws awscliv2.zip

# Install kubectl
curl -o kubectl "https://s3.us-west-2.amazonaws.com/amazon-eks/1.28.3/2023-11-14/bin/linux/amd64/kubectl"
chmod +x ./kubectl
mkdir -p /usr/local/bin
mv ./kubectl /usr/local/bin/

# Install Helm
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
rm get_helm.sh

# Configure Docker daemon for optimal performance
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << 'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ],
  "exec-opts": ["native.cgroupdriver=systemd"],
  "live-restore": true,
  "userland-proxy": false,
  "experimental": false,
  "debug": false,
  "registry-mirrors": [],
  "insecure-registries": [],
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 1048576,
      "Soft": 1048576
    },
    "nproc": {
      "Name": "nproc",
      "Hard": 1048576,
      "Soft": 1048576
    }
  }
}
EOF

# Create a script for node health monitoring
cat > /usr/local/bin/nephoran-node-health.sh << 'EOF'
#!/bin/bash

# Nephoran Node Health Check Script
# This script monitors node health and reports to CloudWatch

INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
REGION=$(curl -s http://169.254.169.254/latest/meta-data/placement/region)

# Check disk usage
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Nodes" \
    --metric-data MetricName=DiskUsagePercent,Value="$DISK_USAGE",Unit=Percent,Dimensions=InstanceId="$INSTANCE_ID"

# Check memory usage
MEMORY_USAGE=$(free | awk 'NR==2{printf "%.2f", $3*100/$2}')
aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Nodes" \
    --metric-data MetricName=MemoryUsagePercent,Value="$MEMORY_USAGE",Unit=Percent,Dimensions=InstanceId="$INSTANCE_ID"

# Check load average
LOAD_AVERAGE=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Nodes" \
    --metric-data MetricName=LoadAverage1Min,Value="$LOAD_AVERAGE",Unit=Count,Dimensions=InstanceId="$INSTANCE_ID"

# Check if kubelet is running
if systemctl is-active --quiet kubelet; then
    KUBELET_STATUS=1
else
    KUBELET_STATUS=0
fi

aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Nodes" \
    --metric-data MetricName=KubeletStatus,Value="$KUBELET_STATUS",Unit=Count,Dimensions=InstanceId="$INSTANCE_ID"

# Check container runtime status
if systemctl is-active --quiet containerd; then
    CONTAINERD_STATUS=1
else
    CONTAINERD_STATUS=0
fi

aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Nodes" \
    --metric-data MetricName=ContainerdStatus,Value="$CONTAINERD_STATUS",Unit=Count,Dimensions=InstanceId="$INSTANCE_ID"
EOF

chmod +x /usr/local/bin/nephoran-node-health.sh

# Create systemd service for health monitoring
cat > /etc/systemd/system/nephoran-node-health.service << 'EOF'
[Unit]
Description=Nephoran Node Health Monitor
Wants=nephoran-node-health.timer

[Service]
Type=oneshot
ExecStart=/usr/local/bin/nephoran-node-health.sh
User=root

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/nephoran-node-health.timer << 'EOF'
[Unit]
Description=Run Nephoran Node Health Monitor every 5 minutes
Requires=nephoran-node-health.service

[Timer]
OnCalendar=*:0/5
Persistent=true

[Install]
WantedBy=timers.target
EOF

# Enable and start the health monitoring service
systemctl daemon-reload
systemctl enable nephoran-node-health.timer
systemctl start nephoran-node-health.timer

# Install node exporter for Prometheus monitoring
NODE_EXPORTER_VERSION="1.7.0"
wget "https://github.com/prometheus/node_exporter/releases/download/v$${NODE_EXPORTER_VERSION}/node_exporter-$${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
tar xvfz "node_exporter-$${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
cp "node_exporter-$${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter" /usr/local/bin/
rm -rf "node_exporter-$${NODE_EXPORTER_VERSION}.linux-amd64"*

# Create node_exporter user
useradd --no-create-home --shell /bin/false node_exporter

# Create systemd service for node_exporter
cat > /etc/systemd/system/node_exporter.service << 'EOF'
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter \
    --collector.systemd \
    --collector.processes \
    --collector.interrupts \
    --collector.ksmd \
    --collector.logind \
    --collector.meminfo_numa \
    --collector.mountstats \
    --collector.ntp \
    --collector.perf \
    --collector.systemd \
    --collector.tcpstat \
    --collector.wifi \
    --web.listen-address=:9100

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable node_exporter
systemctl start node_exporter

# Configure log rotation for container logs
cat > /etc/logrotate.d/docker-containers << 'EOF'
/var/lib/docker/containers/*/*.log {
    rotate 7
    daily
    compress
    size=50M
    missingok
    delaycompress
    copytruncate
}
EOF

# Set up custom EKS bootstrap with additional configurations
B64_CLUSTER_CA=${cluster_ca}
API_SERVER_URL=${cluster_endpoint}
K8S_CLUSTER_DNS_IP=10.100.0.10

# Configure kubelet with additional arguments for Nephoran workloads
/etc/eks/bootstrap.sh ${cluster_name} \
    --b64-cluster-ca $B64_CLUSTER_CA \
    --apiserver-endpoint $API_SERVER_URL \
    --dns-cluster-ip $K8S_CLUSTER_DNS_IP \
    --kubelet-extra-args "--node-labels=nephoran.com/node-ready=true,nephoran.com/bootstrap-time=$(date +%s) --max-pods=110 --kube-reserved=cpu=250m,memory=1Gi,ephemeral-storage=1Gi --system-reserved=cpu=250m,memory=0.5Gi,ephemeral-storage=1Gi --eviction-hard=memory.available<500Mi,nodefs.available<10%,imagefs.available<10% --feature-gates=RotateKubeletServerCertificate=true --protect-kernel-defaults=true --read-only-port=0 --anonymous-auth=false --authorization-mode=Webhook --authentication-token-webhook=true --client-ca-file=/etc/kubernetes/pki/ca.crt" \
    ${bootstrap_arguments}

# Install container security tools
# Install Falco for runtime security (if enabled)
if [[ "${enable_falco:-false}" == "true" ]]; then
    curl -s https://falco.org/repo/falcosecurity-packages.asc | apt-key add -
    echo "deb https://download.falco.org/packages/deb stable main" | tee -a /etc/apt/sources.list.d/falcosecurity.list
    apt-get update -y
    apt-get install -y falco
fi

# Performance tuning for AI/ML workloads
if [[ "$(grep -i 'ai-processing\|ml' /proc/cmdline)" ]]; then
    # Enable performance governor
    echo 'performance' | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
    
    # Disable CPU idle states for consistent performance
    for i in /sys/devices/system/cpu/cpu*/cpuidle/state*/disable; do
        echo 1 > $i 2>/dev/null || true
    done
    
    # Configure transparent huge pages
    echo never > /sys/kernel/mm/transparent_hugepage/enabled
    echo never > /sys/kernel/mm/transparent_hugepage/defrag
fi

# Configure for database workloads
if [[ "$(grep -i 'database' /proc/cmdline)" ]]; then
    # Optimize for database I/O
    echo deadline > /sys/block/nvme*/queue/scheduler 2>/dev/null || true
    echo 1 > /sys/block/nvme*/queue/iosched/fifo_batch 2>/dev/null || true
    
    # Configure transparent huge pages for databases
    echo madvise > /sys/kernel/mm/transparent_hugepage/enabled
fi

# Install AWS EBS CSI driver dependencies
yum install -y nvme-cli

# Create directories for local storage
mkdir -p /mnt/nephoran-local-storage
chmod 755 /mnt/nephoran-local-storage

# Configure audit logging for security compliance
mkdir -p /var/log/nephoran-audit
cat > /etc/audit/rules.d/nephoran.rules << 'EOF'
# Nephoran Intent Operator - Audit Rules

# Monitor file system changes in sensitive directories
-w /etc/kubernetes/ -p wa -k nephoran-k8s-config
-w /var/lib/kubelet/ -p wa -k nephoran-kubelet
-w /etc/docker/ -p wa -k nephoran-docker-config
-w /etc/containerd/ -p wa -k nephoran-containerd-config

# Monitor privilege escalation
-a always,exit -F arch=b64 -S setuid -S setgid -S setreuid -S setregid -k nephoran-privilege-escalation
-a always,exit -F arch=b32 -S setuid -S setgid -S setreuid -S setregid -k nephoran-privilege-escalation

# Monitor network configuration changes
-a always,exit -F arch=b64 -S sethostname -S setdomainname -k nephoran-network-config
-a always,exit -F arch=b32 -S sethostname -S setdomainname -k nephoran-network-config

# Monitor process execution
-a always,exit -F arch=b64 -S execve -k nephoran-process-execution
-a always,exit -F arch=b32 -S execve -k nephoran-process-execution
EOF

service auditd restart

# Configure rsyslog for centralized logging
cat >> /etc/rsyslog.conf << 'EOF'

# Nephoran Intent Operator - Custom log forwarding
*.* @@localhost:514
EOF

systemctl restart rsyslog

# Set up custom motd
cat > /etc/motd << 'EOF'
******************************************************************************
*                                                                            *
*  Nephoran Intent Operator - AWS EKS Worker Node                          *
*                                                                            *
*  This system is for authorized use only. All activity is monitored        *
*  and logged for security and compliance purposes.                         *
*                                                                            *
*  Node Type: $(grep -o 'nephoran.com/node-type=[^,]*' /proc/cmdline | cut -d= -f2)
*  Cluster: ${cluster_name}
*  Region: $(curl -s http://169.254.169.254/latest/meta-data/placement/region)
*  Instance ID: $(curl -s http://169.254.169.254/latest/meta-data/instance-id)
*                                                                            *
******************************************************************************
EOF

# Final system update and cleanup
yum update -y
yum clean all

# Log successful completion
echo "$(date): Nephoran EKS node initialization completed successfully" >> /var/log/nephoran-bootstrap.log

# Signal successful completion to CloudFormation (if applicable)
INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
REGION=$(curl -s http://169.254.169.254/latest/meta-data/placement/region)

aws cloudwatch put-metric-data \
    --region "$REGION" \
    --namespace "Nephoran/EKS/Bootstrap" \
    --metric-data MetricName=NodeBootstrapSuccess,Value=1,Unit=Count,Dimensions=InstanceId="$INSTANCE_ID" \
    2>/dev/null || true

exit 0