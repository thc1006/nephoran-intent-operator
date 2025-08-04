#!/usr/bin/env python3
"""
Cost Optimization Report Generator for Multi-Region Nephoran Deployment
Analyzes GCP costs and provides optimization recommendations
"""

import json
import datetime
import argparse
from typing import Dict, List, Tuple
from google.cloud import bigquery
from google.cloud import monitoring_v3
from google.cloud import compute_v1
from google.cloud import container_v1
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from tabulate import tabulate

class CostOptimizationAnalyzer:
    def __init__(self, project_id: str):
        self.project_id = project_id
        self.bq_client = bigquery.Client(project=project_id)
        self.monitoring_client = monitoring_v3.MetricServiceClient()
        self.compute_client = compute_v1.InstancesClient()
        self.container_client = container_v1.ClusterManagerClient()
        
        self.regions = ["us-central1", "europe-west1", "asia-southeast1"]
        self.services = ["llm-processor", "rag-api", "intent-controller", "edge-controller", "weaviate"]
        
    def analyze_costs(self, days: int = 30) -> Dict:
        """Analyze costs for the specified period"""
        print(f"Analyzing costs for the last {days} days...")
        
        # Query BigQuery billing data
        query = f"""
        SELECT
            service.description as service,
            location.location as region,
            sku.description as sku,
            SUM(cost) as total_cost,
            SUM(usage.amount) as usage_amount,
            usage.unit as usage_unit,
            COUNT(*) as line_items
        FROM `{self.project_id}.billing_export.gcp_billing_export_v1`
        WHERE DATE(_PARTITIONTIME) >= DATE_SUB(CURRENT_DATE(), INTERVAL {days} DAY)
            AND project.id = '{self.project_id}'
        GROUP BY service, region, sku, usage_unit
        ORDER BY total_cost DESC
        """
        
        df = self.bq_client.query(query).to_dataframe()
        
        return {
            'total_cost': df['total_cost'].sum(),
            'by_service': self._aggregate_by_service(df),
            'by_region': self._aggregate_by_region(df),
            'by_resource': self._aggregate_by_resource(df),
            'top_costs': df.head(20).to_dict('records')
        }
    
    def _aggregate_by_service(self, df: pd.DataFrame) -> Dict:
        """Aggregate costs by service"""
        service_costs = df.groupby('service')['total_cost'].sum().sort_values(ascending=False)
        return service_costs.to_dict()
    
    def _aggregate_by_region(self, df: pd.DataFrame) -> Dict:
        """Aggregate costs by region"""
        region_costs = df.groupby('region')['total_cost'].sum().sort_values(ascending=False)
        return region_costs.to_dict()
    
    def _aggregate_by_resource(self, df: pd.DataFrame) -> Dict:
        """Aggregate costs by resource type"""
        resource_mapping = {
            'Compute Engine': ['VM', 'Instance', 'Disk', 'Snapshot'],
            'Kubernetes Engine': ['GKE', 'Cluster', 'Node'],
            'Networking': ['Load Balancing', 'VPN', 'NAT', 'Bandwidth'],
            'Storage': ['Storage', 'Bucket', 'Object'],
            'Database': ['SQL', 'Spanner', 'Firestore', 'Bigtable'],
            'AI/ML': ['Vertex', 'ML', 'AI Platform']
        }
        
        df['resource_type'] = 'Other'
        for resource_type, keywords in resource_mapping.items():
            mask = df['sku'].str.contains('|'.join(keywords), case=False, na=False)
            df.loc[mask, 'resource_type'] = resource_type
        
        return df.groupby('resource_type')['total_cost'].sum().sort_values(ascending=False).to_dict()
    
    def analyze_utilization(self) -> Dict:
        """Analyze resource utilization across regions"""
        print("Analyzing resource utilization...")
        
        utilization_data = {}
        
        for region in self.regions:
            utilization_data[region] = {
                'cpu': self._get_cpu_utilization(region),
                'memory': self._get_memory_utilization(region),
                'storage': self._get_storage_utilization(region),
                'network': self._get_network_utilization(region)
            }
        
        return utilization_data
    
    def _get_cpu_utilization(self, region: str) -> Dict:
        """Get CPU utilization metrics"""
        project_name = f"projects/{self.project_id}"
        interval = monitoring_v3.TimeInterval({
            "end_time": datetime.datetime.now(),
            "start_time": datetime.datetime.now() - datetime.timedelta(days=7)
        })
        
        results = self.monitoring_client.list_time_series(
            request={
                "name": project_name,
                "filter": f'resource.type="k8s_container" AND metric.type="kubernetes.io/container/cpu/core_usage_time" AND resource.label.location="{region}"',
                "interval": interval,
                "view": monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL
            }
        )
        
        cpu_data = []
        for result in results:
            for point in result.points:
                cpu_data.append(point.value.double_value)
        
        if cpu_data:
            return {
                'average': sum(cpu_data) / len(cpu_data),
                'max': max(cpu_data),
                'min': min(cpu_data)
            }
        return {'average': 0, 'max': 0, 'min': 0}
    
    def _get_memory_utilization(self, region: str) -> Dict:
        """Get memory utilization metrics"""
        # Similar implementation to CPU utilization
        return {'average': 0.65, 'max': 0.85, 'min': 0.45}  # Placeholder
    
    def _get_storage_utilization(self, region: str) -> Dict:
        """Get storage utilization metrics"""
        # Implementation for storage metrics
        return {'used_gb': 500, 'total_gb': 1000, 'percentage': 50}  # Placeholder
    
    def _get_network_utilization(self, region: str) -> Dict:
        """Get network utilization metrics"""
        # Implementation for network metrics
        return {'ingress_gb': 100, 'egress_gb': 150, 'total_gb': 250}  # Placeholder
    
    def identify_optimization_opportunities(self, cost_data: Dict, utilization_data: Dict) -> List[Dict]:
        """Identify cost optimization opportunities"""
        print("Identifying optimization opportunities...")
        
        opportunities = []
        
        # 1. Underutilized resources
        for region, metrics in utilization_data.items():
            if metrics['cpu']['average'] < 0.3:
                opportunities.append({
                    'type': 'Underutilized CPU',
                    'region': region,
                    'recommendation': f'Consider reducing CPU allocation in {region}',
                    'potential_savings': self._estimate_cpu_savings(region, metrics['cpu']),
                    'priority': 'High'
                })
            
            if metrics['memory']['average'] < 0.4:
                opportunities.append({
                    'type': 'Underutilized Memory',
                    'region': region,
                    'recommendation': f'Consider reducing memory allocation in {region}',
                    'potential_savings': self._estimate_memory_savings(region, metrics['memory']),
                    'priority': 'Medium'
                })
        
        # 2. Expensive regions
        region_costs = cost_data['by_region']
        avg_cost = sum(region_costs.values()) / len(region_costs)
        for region, cost in region_costs.items():
            if cost > avg_cost * 1.5:
                opportunities.append({
                    'type': 'High-cost Region',
                    'region': region,
                    'recommendation': f'Consider redistributing workloads from {region}',
                    'potential_savings': (cost - avg_cost) * 0.2,
                    'priority': 'Medium'
                })
        
        # 3. Storage optimization
        for region in self.regions:
            opportunities.extend(self._identify_storage_optimizations(region))
        
        # 4. Network optimization
        opportunities.extend(self._identify_network_optimizations(cost_data))
        
        # 5. Spot/Preemptible instances
        opportunities.extend(self._identify_spot_opportunities())
        
        # 6. Reserved instances/committed use
        opportunities.extend(self._identify_commitment_opportunities(cost_data))
        
        return sorted(opportunities, key=lambda x: x['potential_savings'], reverse=True)
    
    def _estimate_cpu_savings(self, region: str, cpu_metrics: Dict) -> float:
        """Estimate potential CPU cost savings"""
        # Simplified calculation - in reality would be more complex
        underutilization = 1 - cpu_metrics['average']
        estimated_monthly_cpu_cost = 500  # Placeholder
        return underutilization * estimated_monthly_cpu_cost * 0.7
    
    def _estimate_memory_savings(self, region: str, memory_metrics: Dict) -> float:
        """Estimate potential memory cost savings"""
        underutilization = 1 - memory_metrics['average']
        estimated_monthly_memory_cost = 300  # Placeholder
        return underutilization * estimated_monthly_memory_cost * 0.7
    
    def _identify_storage_optimizations(self, region: str) -> List[Dict]:
        """Identify storage optimization opportunities"""
        opportunities = []
        
        # Check for old snapshots
        opportunities.append({
            'type': 'Old Snapshots',
            'region': region,
            'recommendation': f'Delete snapshots older than 30 days in {region}',
            'potential_savings': 50,  # Placeholder
            'priority': 'Low'
        })
        
        # Check for unattached disks
        opportunities.append({
            'type': 'Unattached Disks',
            'region': region,
            'recommendation': f'Remove unattached persistent disks in {region}',
            'potential_savings': 100,  # Placeholder
            'priority': 'Medium'
        })
        
        return opportunities
    
    def _identify_network_optimizations(self, cost_data: Dict) -> List[Dict]:
        """Identify network optimization opportunities"""
        opportunities = []
        
        # Cross-region traffic optimization
        opportunities.append({
            'type': 'Cross-region Traffic',
            'region': 'Global',
            'recommendation': 'Implement regional caching to reduce cross-region data transfer',
            'potential_savings': 200,  # Placeholder
            'priority': 'High'
        })
        
        return opportunities
    
    def _identify_spot_opportunities(self) -> List[Dict]:
        """Identify opportunities to use spot/preemptible instances"""
        opportunities = []
        
        for service in ['edge-controller', 'batch-processing']:
            opportunities.append({
                'type': 'Spot Instance Opportunity',
                'region': 'All',
                'recommendation': f'Use spot instances for {service}',
                'potential_savings': 150,  # Placeholder
                'priority': 'Medium'
            })
        
        return opportunities
    
    def _identify_commitment_opportunities(self, cost_data: Dict) -> List[Dict]:
        """Identify committed use discount opportunities"""
        opportunities = []
        
        total_compute_cost = cost_data.get('by_resource', {}).get('Compute Engine', 0)
        if total_compute_cost > 5000:  # Monthly threshold
            opportunities.append({
                'type': 'Committed Use Discount',
                'region': 'All',
                'recommendation': 'Purchase 1-year committed use contracts for stable workloads',
                'potential_savings': total_compute_cost * 0.37,  # 37% discount
                'priority': 'High'
            })
        
        return opportunities
    
    def generate_report(self, output_file: str = 'cost_optimization_report.html'):
        """Generate comprehensive cost optimization report"""
        print("Generating cost optimization report...")
        
        # Gather data
        cost_data = self.analyze_costs()
        utilization_data = self.analyze_utilization()
        opportunities = self.identify_optimization_opportunities(cost_data, utilization_data)
        
        # Calculate summary metrics
        total_cost = cost_data['total_cost']
        total_potential_savings = sum(opp['potential_savings'] for opp in opportunities)
        savings_percentage = (total_potential_savings / total_cost) * 100 if total_cost > 0 else 0
        
        # Generate visualizations
        self._create_visualizations(cost_data, utilization_data, opportunities)
        
        # Generate HTML report
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>Nephoran Multi-Region Cost Optimization Report</title>
            <style>
                body {{ font-family: Arial, sans-serif; margin: 40px; }}
                h1, h2, h3 {{ color: #333; }}
                .summary {{ background-color: #f0f0f0; padding: 20px; border-radius: 5px; }}
                .metric {{ display: inline-block; margin: 10px 20px; }}
                .metric-value {{ font-size: 24px; font-weight: bold; color: #0066cc; }}
                table {{ border-collapse: collapse; width: 100%; margin: 20px 0; }}
                th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
                th {{ background-color: #4CAF50; color: white; }}
                tr:nth-child(even) {{ background-color: #f2f2f2; }}
                .high-priority {{ color: #d32f2f; font-weight: bold; }}
                .medium-priority {{ color: #f57c00; }}
                .low-priority {{ color: #388e3c; }}
                .chart {{ margin: 20px 0; text-align: center; }}
                .recommendation {{ background-color: #e3f2fd; padding: 15px; margin: 10px 0; border-radius: 5px; }}
            </style>
        </head>
        <body>
            <h1>Nephoran Multi-Region Cost Optimization Report</h1>
            <p>Generated on: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
            
            <div class="summary">
                <h2>Executive Summary</h2>
                <div class="metric">
                    <div>Total Monthly Cost</div>
                    <div class="metric-value">${total_cost:,.2f}</div>
                </div>
                <div class="metric">
                    <div>Potential Monthly Savings</div>
                    <div class="metric-value">${total_potential_savings:,.2f}</div>
                </div>
                <div class="metric">
                    <div>Savings Percentage</div>
                    <div class="metric-value">{savings_percentage:.1f}%</div>
                </div>
            </div>
            
            <h2>Cost Breakdown</h2>
            <div class="chart">
                <img src="cost_by_service.png" alt="Cost by Service">
                <img src="cost_by_region.png" alt="Cost by Region">
            </div>
            
            <h2>Top Cost Items</h2>
            <table>
                <tr>
                    <th>Service</th>
                    <th>Region</th>
                    <th>SKU</th>
                    <th>Cost</th>
                    <th>Usage</th>
                </tr>
                {''.join(f'''
                <tr>
                    <td>{item['service']}</td>
                    <td>{item['region']}</td>
                    <td>{item['sku']}</td>
                    <td>${item['total_cost']:.2f}</td>
                    <td>{item['usage_amount']:.2f} {item['usage_unit']}</td>
                </tr>
                ''' for item in cost_data['top_costs'][:10])}
            </table>
            
            <h2>Resource Utilization</h2>
            <div class="chart">
                <img src="utilization_by_region.png" alt="Utilization by Region">
            </div>
            
            <h2>Optimization Opportunities</h2>
            <p>Total identified opportunities: {len(opportunities)}</p>
            
            {''.join(f'''
            <div class="recommendation">
                <h3>{opp['type']} - <span class="{opp['priority'].lower()}-priority">{opp['priority']} Priority</span></h3>
                <p><strong>Region:</strong> {opp['region']}</p>
                <p><strong>Recommendation:</strong> {opp['recommendation']}</p>
                <p><strong>Potential Savings:</strong> ${opp['potential_savings']:.2f}/month</p>
            </div>
            ''' for opp in opportunities[:10])}
            
            <h2>Implementation Roadmap</h2>
            <ol>
                <li><strong>Immediate Actions (Week 1)</strong>
                    <ul>
                        <li>Remove unattached disks and old snapshots</li>
                        <li>Right-size underutilized instances</li>
                        <li>Enable autoscaling for variable workloads</li>
                    </ul>
                </li>
                <li><strong>Short-term Actions (Month 1)</strong>
                    <ul>
                        <li>Implement spot instances for fault-tolerant workloads</li>
                        <li>Optimize cross-region data transfer</li>
                        <li>Review and adjust resource allocations</li>
                    </ul>
                </li>
                <li><strong>Long-term Actions (Quarter 1)</strong>
                    <ul>
                        <li>Purchase committed use contracts</li>
                        <li>Implement automated cost optimization</li>
                        <li>Redesign architecture for cost efficiency</li>
                    </ul>
                </li>
            </ol>
            
            <h2>Monitoring and Alerts</h2>
            <p>Recommended alerts to implement:</p>
            <ul>
                <li>Daily cost exceeds $500</li>
                <li>Resource utilization below 30% for 24 hours</li>
                <li>Unattached resources detected</li>
                <li>Cross-region traffic exceeds 1TB/day</li>
            </ul>
        </body>
        </html>
        """
        
        with open(output_file, 'w') as f:
            f.write(html_content)
        
        print(f"Report generated: {output_file}")
        print(f"Total potential savings identified: ${total_potential_savings:,.2f}/month ({savings_percentage:.1f}%)")
    
    def _create_visualizations(self, cost_data: Dict, utilization_data: Dict, opportunities: List[Dict]):
        """Create visualization charts"""
        # Cost by service pie chart
        plt.figure(figsize=(10, 6))
        services = list(cost_data['by_service'].keys())[:10]
        costs = [cost_data['by_service'][s] for s in services]
        plt.pie(costs, labels=services, autopct='%1.1f%%')
        plt.title('Cost Distribution by Service')
        plt.savefig('cost_by_service.png')
        plt.close()
        
        # Cost by region bar chart
        plt.figure(figsize=(10, 6))
        regions = list(cost_data['by_region'].keys())
        costs = list(cost_data['by_region'].values())
        plt.bar(regions, costs)
        plt.xlabel('Region')
        plt.ylabel('Cost ($)')
        plt.title('Cost by Region')
        plt.savefig('cost_by_region.png')
        plt.close()
        
        # Utilization heatmap
        plt.figure(figsize=(12, 8))
        util_matrix = []
        metrics = ['cpu', 'memory']
        for region in self.regions:
            row = []
            for metric in metrics:
                row.append(utilization_data[region][metric]['average'])
            util_matrix.append(row)
        
        sns.heatmap(util_matrix, 
                   xticklabels=metrics, 
                   yticklabels=self.regions,
                   annot=True, 
                   fmt='.2f',
                   cmap='RdYlGn_r')
        plt.title('Resource Utilization by Region')
        plt.savefig('utilization_by_region.png')
        plt.close()

def main():
    parser = argparse.ArgumentParser(description='Generate cost optimization report for Nephoran deployment')
    parser.add_argument('--project-id', required=True, help='GCP Project ID')
    parser.add_argument('--days', type=int, default=30, help='Number of days to analyze')
    parser.add_argument('--output', default='cost_optimization_report.html', help='Output file path')
    
    args = parser.parse_args()
    
    analyzer = CostOptimizationAnalyzer(args.project_id)
    analyzer.generate_report(args.output)

if __name__ == '__main__':
    main()