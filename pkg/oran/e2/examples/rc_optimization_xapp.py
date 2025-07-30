#!/usr/bin/env python3
"""
RC Optimization xApp - Python Implementation
Demonstrates RAN Control optimization using the Nephoran xApp Python SDK
"""

import asyncio
import json
import logging
import os
import signal
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
import numpy as np

# Import the xApp SDK
import sys
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from python_sdk.xapp_sdk import (
    XAppSDK, XAppConfig, XAppResourceLimits, XAppHealthConfig,
    XAppAction, XAppState, RICIndication, RICControlRequest, RICControlAcknowledge,
    create_rc_control_message, parse_kmp_indication
)


@dataclass
class OptimizationTarget:
    """Optimization target definition"""
    target_id: str
    metric_name: str
    target_value: float
    weight: float = 1.0
    tolerance: float = 0.1


@dataclass
class ControlAction:
    """RAN control action definition"""
    action_id: str
    action_type: str  # "qos_flow_mapping", "traffic_steering", etc.
    parameters: Dict[str, Any]
    priority: int = 1
    execution_time: Optional[datetime] = None


@dataclass
class OptimizationResult:
    """Optimization algorithm result"""
    result_id: str
    objective_value: float
    actions: List[ControlAction]
    confidence: float
    timestamp: datetime = field(default_factory=datetime.now)


class MLOptimizer:
    """Machine Learning based optimization engine"""
    
    def __init__(self):
        self.model_weights = {}
        self.learning_rate = 0.01
        self.historical_data = []
        self.feature_scaler = {"mean": {}, "std": {}}
    
    def train(self, features: Dict[str, float], target: float):
        """Train the optimization model with new data"""
        # Simplified ML training - in practice would use proper ML libraries
        self.historical_data.append({"features": features, "target": target})
        
        # Keep only last 1000 samples for training
        if len(self.historical_data) > 1000:
            self.historical_data = self.historical_data[-1000:]
        
        # Update feature scaling
        self._update_feature_scaling()
        
        # Simple gradient descent update
        self._update_weights(features, target)
    
    def predict(self, features: Dict[str, float]) -> float:
        """Predict optimization score for given features"""
        if not self.model_weights:
            return 0.5  # Default prediction
        
        # Normalize features
        normalized_features = self._normalize_features(features)
        
        # Simple linear model prediction
        prediction = 0.0
        for feature_name, value in normalized_features.items():
            weight = self.model_weights.get(feature_name, 0.0)
            prediction += weight * value
        
        # Apply sigmoid activation
        return 1.0 / (1.0 + np.exp(-prediction))
    
    def optimize(self, current_state: Dict[str, float], 
                targets: List[OptimizationTarget]) -> List[ControlAction]:
        """Generate optimal control actions"""
        actions = []
        
        for target in targets:
            current_value = current_state.get(target.metric_name, 0.0)
            error = target.target_value - current_value
            
            if abs(error) > target.tolerance:
                # Generate control action based on error
                action = self._generate_control_action(target, error)
                if action:
                    actions.append(action)
        
        return actions
    
    def _update_feature_scaling(self):
        """Update feature mean and standard deviation for normalization"""
        if not self.historical_data:
            return
        
        all_features = {}
        for sample in self.historical_data:
            for feature_name, value in sample["features"].items():
                if feature_name not in all_features:
                    all_features[feature_name] = []
                all_features[feature_name].append(value)
        
        for feature_name, values in all_features.items():
            self.feature_scaler["mean"][feature_name] = np.mean(values)
            self.feature_scaler["std"][feature_name] = np.std(values) or 1.0
    
    def _normalize_features(self, features: Dict[str, float]) -> Dict[str, float]:
        """Normalize features using computed mean and std"""
        normalized = {}
        for feature_name, value in features.items():
            mean = self.feature_scaler["mean"].get(feature_name, 0.0)
            std = self.feature_scaler["std"].get(feature_name, 1.0)
            normalized[feature_name] = (value - mean) / std
        return normalized
    
    def _update_weights(self, features: Dict[str, float], target: float):
        """Update model weights using gradient descent"""
        normalized_features = self._normalize_features(features)
        prediction = self.predict(features)
        error = target - prediction
        
        # Update weights
        for feature_name, value in normalized_features.items():
            if feature_name not in self.model_weights:
                self.model_weights[feature_name] = 0.0
            
            gradient = error * value
            self.model_weights[feature_name] += self.learning_rate * gradient
    
    def _generate_control_action(self, target: OptimizationTarget, 
                                error: float) -> Optional[ControlAction]:
        """Generate control action to address optimization target"""
        action_type = self._select_action_type(target.metric_name)
        if not action_type:
            return None
        
        # Generate parameters based on error magnitude and direction
        parameters = self._calculate_action_parameters(action_type, error, target)
        
        return ControlAction(
            action_id=f"opt_{target.target_id}_{int(time.time())}",
            action_type=action_type,
            parameters=parameters,
            priority=1 if abs(error) > target.tolerance * 2 else 2
        )
    
    def _select_action_type(self, metric_name: str) -> Optional[str]:
        """Select appropriate control action type for metric"""
        action_mapping = {
            "DRB.UEThpDl": "qos_flow_mapping",
            "DRB.UEThpUl": "qos_flow_mapping",
            "RRU.PrbUsedDl": "traffic_steering",
            "RRU.PrbUsedUl": "traffic_steering",
            "DRB.RlcSduDelayDl": "dual_connectivity",
            "DRB.RlcSduDelayUl": "dual_connectivity",
        }
        return action_mapping.get(metric_name)
    
    def _calculate_action_parameters(self, action_type: str, error: float, 
                                   target: OptimizationTarget) -> Dict[str, Any]:
        """Calculate action parameters based on optimization needs"""
        base_parameters = {
            "target_metric": target.metric_name,
            "target_value": target.target_value,
            "current_error": error,
            "urgency": min(abs(error) / target.tolerance, 10.0)
        }
        
        if action_type == "qos_flow_mapping":
            return {
                **base_parameters,
                "qos_class": "guaranteed_bitrate" if error < 0 else "non_guaranteed_bitrate",
                "bitrate_adjustment": min(abs(error) * 0.1, 1.0),
                "scheduling_priority": 1 if abs(error) > target.tolerance * 2 else 3
            }
        elif action_type == "traffic_steering":
            return {
                **base_parameters,
                "steering_ratio": min(abs(error) * 0.05, 0.3),
                "target_cell": "neighbor_cell" if error > 0 else "serving_cell",
                "load_balancing": True
            }
        elif action_type == "dual_connectivity":
            return {
                **base_parameters,
                "secondary_cell_group": True if error > 0 else False,
                "bearer_split_ratio": min(abs(error) * 0.1, 0.5),
                "latency_optimization": True
            }
        else:
            return base_parameters


class RCOptimizationXApp:
    """RAN Control Optimization xApp"""
    
    def __init__(self):
        self.sdk: Optional[XAppSDK] = None
        self.optimizer = MLOptimizer()
        self.current_measurements: Dict[str, float] = {}
        self.optimization_targets: List[OptimizationTarget] = []
        self.active_controls: Dict[str, ControlAction] = {}
        self.optimization_history: List[OptimizationResult] = []
        self.logger = logging.getLogger(__name__)
        
        # Initialize optimization targets
        self._initialize_targets()
    
    def _initialize_targets(self):
        """Initialize optimization targets"""
        self.optimization_targets = [
            OptimizationTarget(
                target_id="throughput_dl",
                metric_name="DRB.UEThpDl",
                target_value=150.0,  # Target 150 Mbps
                weight=1.0,
                tolerance=10.0
            ),
            OptimizationTarget(
                target_id="throughput_ul",
                metric_name="DRB.UEThpUl",
                target_value=75.0,   # Target 75 Mbps
                weight=1.0,
                tolerance=5.0
            ),
            OptimizationTarget(
                target_id="prb_utilization_dl",
                metric_name="RRU.PrbUsedDl",
                target_value=0.7,    # Target 70% utilization
                weight=0.8,
                tolerance=0.1
            ),
            OptimizationTarget(
                target_id="prb_utilization_ul",
                metric_name="RRU.PrbUsedUl",
                target_value=0.6,    # Target 60% utilization
                weight=0.8,
                tolerance=0.1
            ),
            OptimizationTarget(
                target_id="delay_dl",
                metric_name="DRB.RlcSduDelayDl",
                target_value=10.0,   # Target 10ms delay
                weight=1.2,
                tolerance=2.0
            ),
        ]
    
    async def initialize(self):
        """Initialize the xApp"""
        config = XAppConfig(
            xapp_name="rc-optimization-xapp",
            xapp_version="1.0.0",
            xapp_description="RAN Control Optimization xApp with ML",
            e2_node_id=os.getenv("E2_NODE_ID", "gnb-001"),
            near_rt_ric_url=os.getenv("NEAR_RT_RIC_URL", "http://near-rt-ric:8080"),
            service_models=["KPM", "RC"],
            environment={
                "LOG_LEVEL": "INFO",
                "OPTIMIZATION_MODE": "ml_based",
                "LEARNING_RATE": "0.01"
            },
            resource_limits=XAppResourceLimits(
                max_memory_mb=1024,
                max_cpu_cores=2.0,
                max_subscriptions=5,
                request_timeout=30.0
            ),
            health_check=XAppHealthConfig(
                enabled=True,
                check_interval=30.0,
                failure_threshold=3,
                health_endpoint="/health"
            )
        )
        
        self.sdk = XAppSDK(config)
        
        # Register handlers
        self.sdk.register_indication_handler("default", self._handle_indication)
        
        self.logger.info("RC Optimization xApp initialized")
    
    async def start(self):
        """Start the xApp"""
        if not self.sdk:
            await self.initialize()
        
        await self.sdk.start()
        
        # Create subscriptions for KMP measurements
        await self._create_kmp_subscription()
        
        # Start optimization loop
        asyncio.create_task(self._optimization_loop())
        
        self.logger.info("RC Optimization xApp started")
    
    async def stop(self):
        """Stop the xApp"""
        if self.sdk:
            await self.sdk.stop()
        self.logger.info("RC Optimization xApp stopped")
    
    async def _create_kmp_subscription(self):
        """Create KMP subscription for performance measurements"""
        # Create event trigger for performance metrics
        event_trigger = {
            "measurement_types": [
                "DRB.UEThpDl",
                "DRB.UEThpUl", 
                "RRU.PrbUsedDl",
                "RRU.PrbUsedUl",
                "DRB.RlcSduDelayDl",
                "DRB.RlcSduDelayUl",
                "TB.ErrTotNbrDl",
                "TB.ErrTotNbrUl"
            ],
            "granularity_period": "1000ms",
            "collection_start_time": datetime.now().isoformat(),
            "collection_duration": "continuous",
            "reporting_format": "CHOICE"
        }
        
        actions = [XAppAction(
            action_id=1,
            action_type="report",
            definition={"format": "json", "compression": False}
        )]
        
        subscription = await self.sdk.subscribe(
            node_id=self.sdk.config.e2_node_id,
            ran_function_id=1,  # KMP service model
            event_trigger=event_trigger,
            actions=actions
        )
        
        if subscription:
            self.logger.info(f"Created KMP subscription: {subscription.subscription_id}")
        else:
            self.logger.error("Failed to create KMP subscription")
    
    def _handle_indication(self, indication: RICIndication):
        """Handle incoming RIC indications"""
        try:
            # Parse KMP indication
            kmp_data = parse_kmp_indication(indication)
            if not kmp_data:
                return
            
            measurements = kmp_data.get("measurements", {})
            
            # Update current measurements
            for metric_name, value in measurements.items():
                if isinstance(value, (int, float)):
                    self.current_measurements[metric_name] = float(value)
            
            self.logger.info(f"Updated measurements: {len(measurements)} metrics")
            
            # Train optimizer with new data
            self._train_optimizer(measurements)
            
        except Exception as e:
            self.logger.error(f"Error handling indication: {e}")
    
    def _train_optimizer(self, measurements: Dict[str, Any]):
        """Train the ML optimizer with new measurements"""
        try:
            # Calculate overall performance score
            performance_score = self._calculate_performance_score(measurements)
            
            # Train the optimizer
            features = {k: float(v) for k, v in measurements.items() 
                       if isinstance(v, (int, float))}
            self.optimizer.train(features, performance_score)
            
            self.logger.debug(f"Trained optimizer with performance score: {performance_score}")
            
        except Exception as e:
            self.logger.error(f"Error training optimizer: {e}")
    
    def _calculate_performance_score(self, measurements: Dict[str, Any]) -> float:
        """Calculate overall network performance score"""
        score = 0.0
        total_weight = 0.0
        
        for target in self.optimization_targets:
            value = measurements.get(target.metric_name)
            if value is not None and isinstance(value, (int, float)):
                # Calculate normalized score for this target
                error = abs(float(value) - target.target_value)
                normalized_error = min(error / target.target_value, 1.0)
                target_score = 1.0 - normalized_error
                
                score += target_score * target.weight
                total_weight += target.weight
        
        return score / total_weight if total_weight > 0 else 0.5
    
    async def _optimization_loop(self):
        """Main optimization loop"""
        while self.sdk and self.sdk.get_state() == XAppState.RUNNING:
            try:
                await asyncio.sleep(10)  # Run optimization every 10 seconds
                
                if not self.current_measurements:
                    continue
                
                # Generate optimization actions
                actions = self.optimizer.optimize(
                    self.current_measurements, 
                    self.optimization_targets
                )
                
                if actions:
                    await self._execute_control_actions(actions)
                
                # Clean up old control actions
                self._cleanup_expired_actions()
                
            except Exception as e:
                self.logger.error(f"Error in optimization loop: {e}")
    
    async def _execute_control_actions(self, actions: List[ControlAction]):
        """Execute control actions"""
        for action in actions:
            try:
                # Create RIC control request
                control_header = json.dumps({
                    "action_id": action.action_id,
                    "action_type": action.action_type,
                    "priority": action.priority,
                    "timestamp": datetime.now().isoformat()
                }).encode('utf-8')
                
                control_message = create_rc_control_message(
                    action.action_type, 
                    action.parameters
                )
                
                control_request = RICControlRequest(
                    ran_function_id=2,  # RC service model
                    ric_request_id=action.action_id,
                    control_header=control_header,
                    control_message=control_message
                )
                
                # Send control message
                response = await self.sdk.send_control_message(control_request)
                
                if response:
                    action.execution_time = datetime.now()
                    self.active_controls[action.action_id] = action
                    self.logger.info(f"Executed control action: {action.action_type}")
                else:
                    self.logger.error(f"Failed to execute control action: {action.action_type}")
                    
            except Exception as e:
                self.logger.error(f"Error executing control action {action.action_id}: {e}")
    
    def _cleanup_expired_actions(self):
        """Clean up expired control actions"""
        now = datetime.now()
        expired_actions = []
        
        for action_id, action in self.active_controls.items():
            if action.execution_time:
                age = now - action.execution_time
                if age > timedelta(minutes=5):  # Actions expire after 5 minutes
                    expired_actions.append(action_id)
        
        for action_id in expired_actions:
            del self.active_controls[action_id]
            self.logger.debug(f"Cleaned up expired action: {action_id}")
    
    def get_status(self) -> Dict[str, Any]:
        """Get current xApp status"""
        return {
            "state": self.sdk.get_state().value if self.sdk else "STOPPED",
            "subscriptions": len(self.sdk.get_subscriptions()) if self.sdk else 0,
            "current_measurements": len(self.current_measurements),
            "optimization_targets": len(self.optimization_targets),
            "active_controls": len(self.active_controls),
            "optimizer_weights": len(self.optimizer.model_weights),
            "metrics": self.sdk.get_metrics().__dict__ if self.sdk else {}
        }


async def main():
    """Main function"""
    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Create xApp
    xapp = RCOptimizationXApp()
    
    # Setup signal handlers for graceful shutdown
    shutdown_event = asyncio.Event()
    
    def signal_handler():
        logging.info("Received shutdown signal")
        shutdown_event.set()
    
    # Register signal handlers
    for sig in [signal.SIGINT, signal.SIGTERM]:
        asyncio.get_event_loop().add_signal_handler(sig, signal_handler)
    
    try:
        # Start xApp
        await xapp.start()
        
        # Wait for shutdown signal
        await shutdown_event.wait()
        
    except Exception as e:
        logging.error(f"Error running xApp: {e}")
    finally:
        # Stop xApp
        await xapp.stop()
        logging.info("RC Optimization xApp shutdown complete")


if __name__ == "__main__":
    asyncio.run(main())