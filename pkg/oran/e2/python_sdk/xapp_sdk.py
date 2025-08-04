#!/usr/bin/env python3
"""
Nephoran xApp SDK for Python
Provides Python SDK for developing E2 xApps with lifecycle management
"""

import asyncio
import json
import logging
import time
import threading
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Dict, List, Optional, Callable, Any
import aiohttp
import uuid


class XAppState(Enum):
    """xApp lifecycle states"""
    INITIALIZING = "INITIALIZING"
    RUNNING = "RUNNING"
    STOPPING = "STOPPING"
    STOPPED = "STOPPED"
    ERROR = "ERROR"


class XAppLifecycleEvent(Enum):
    """xApp lifecycle events"""
    STARTUP = "STARTUP"
    SHUTDOWN = "SHUTDOWN"
    ERROR = "ERROR"
    SUBSCRIBED = "SUBSCRIBED"
    INDICATION = "INDICATION"


class XAppSubscriptionStatus(Enum):
    """Subscription status enumeration"""
    PENDING = "PENDING"
    ACTIVE = "ACTIVE"
    FAILED = "FAILED"
    CANCELLING = "CANCELLING"
    CANCELLED = "CANCELLED"


@dataclass
class XAppConfig:
    """Configuration for xApp"""
    xapp_name: str
    xapp_version: str
    xapp_description: str
    e2_node_id: str
    near_rt_ric_url: str
    service_models: List[str] = field(default_factory=list)
    environment: Dict[str, str] = field(default_factory=dict)
    resource_limits: Optional['XAppResourceLimits'] = None
    health_check: Optional['XAppHealthConfig'] = None


@dataclass
class XAppResourceLimits:
    """Resource constraints for xApp"""
    max_memory_mb: int = 512
    max_cpu_cores: float = 1.0
    max_subscriptions: int = 10
    request_timeout: float = 30.0


@dataclass
class XAppHealthConfig:
    """Health check configuration"""
    enabled: bool = True
    check_interval: float = 30.0
    failure_threshold: int = 3
    health_endpoint: str = "/health"


@dataclass
class XAppAction:
    """Subscription action definition"""
    action_id: int
    action_type: str  # "report", "insert", "policy"
    definition: Dict[str, Any]
    handler: str = "default"


@dataclass
class XAppSubscription:
    """Active E2 subscription"""
    subscription_id: str
    node_id: str
    ran_function_id: int
    event_trigger: Dict[str, Any]
    actions: List[XAppAction]
    status: XAppSubscriptionStatus = XAppSubscriptionStatus.PENDING
    created_at: datetime = field(default_factory=datetime.now)
    last_indication: Optional[datetime] = None
    indication_count: int = 0


@dataclass
class XAppMetrics:
    """xApp performance metrics"""
    subscriptions_active: int = 0
    indications_received: int = 0
    control_requests_sent: int = 0
    error_count: int = 0
    average_processing_time: float = 0.0
    throughput_per_second: float = 0.0
    last_metrics_update: datetime = field(default_factory=datetime.now)
    custom_metrics: Dict[str, float] = field(default_factory=dict)


class E2Message:
    """Base class for E2AP messages"""
    
    def __init__(self, message_type: str, data: Dict[str, Any]):
        self.message_type = message_type
        self.data = data
        self.timestamp = datetime.now()
    
    def to_json(self) -> str:
        """Convert message to JSON"""
        return json.dumps({
            "message_type": self.message_type,
            "data": self.data,
            "timestamp": self.timestamp.isoformat()
        })
    
    @classmethod
    def from_json(cls, json_str: str) -> 'E2Message':
        """Create message from JSON"""
        data = json.loads(json_str)
        return cls(data["message_type"], data["data"])


class RICIndication(E2Message):
    """RIC Indication message"""
    
    def __init__(self, ran_function_id: int, ric_action_id: int, 
                 indication_header: bytes, indication_message: bytes):
        super().__init__("RIC_INDICATION", {
            "ran_function_id": ran_function_id,
            "ric_action_id": ric_action_id,
            "indication_header": indication_header.hex(),
            "indication_message": indication_message.hex()
        })
        self.ran_function_id = ran_function_id
        self.ric_action_id = ric_action_id
        self.indication_header = indication_header
        self.indication_message = indication_message


class RICControlRequest(E2Message):
    """RIC Control Request message"""
    
    def __init__(self, ran_function_id: int, ric_request_id: str,
                 control_header: bytes, control_message: bytes):
        super().__init__("RIC_CONTROL_REQUEST", {
            "ran_function_id": ran_function_id,
            "ric_request_id": ric_request_id,
            "control_header": control_header.hex(),
            "control_message": control_message.hex()
        })
        self.ran_function_id = ran_function_id
        self.ric_request_id = ric_request_id
        self.control_header = control_header
        self.control_message = control_message


class RICControlAcknowledge(E2Message):
    """RIC Control Acknowledge message"""
    
    def __init__(self, ran_function_id: int, ric_request_id: str,
                 control_outcome: Optional[bytes] = None):
        super().__init__("RIC_CONTROL_ACKNOWLEDGE", {
            "ran_function_id": ran_function_id,
            "ric_request_id": ric_request_id,
            "control_outcome": control_outcome.hex() if control_outcome else None
        })
        self.ran_function_id = ran_function_id
        self.ric_request_id = ric_request_id
        self.control_outcome = control_outcome


# Handler type definitions
IndicationHandler = Callable[[RICIndication], None]
ControlHandler = Callable[[RICControlRequest], RICControlAcknowledge]
LifecycleHandler = Callable[[XAppLifecycleEvent, Any], None]


class XAppLifecycle:
    """Manages xApp lifecycle events and state"""
    
    def __init__(self):
        self._state = XAppState.INITIALIZING
        self._start_time = datetime.now()
        self._last_activity = datetime.now()
        self._handlers: Dict[XAppLifecycleEvent, List[LifecycleHandler]] = {}
        self._lock = threading.RLock()
    
    def register_handler(self, event: XAppLifecycleEvent, handler: LifecycleHandler):
        """Register a lifecycle event handler"""
        with self._lock:
            if event not in self._handlers:
                self._handlers[event] = []
            self._handlers[event].append(handler)
    
    def trigger_event(self, event: XAppLifecycleEvent, data: Any = None):
        """Trigger a lifecycle event"""
        with self._lock:
            self._last_activity = datetime.now()
            handlers = self._handlers.get(event, [])
        
        for handler in handlers:
            try:
                handler(event, data)
            except Exception as e:
                logging.error(f"Lifecycle handler error for {event}: {e}")
    
    def set_state(self, state: XAppState):
        """Set the current xApp state"""
        with self._lock:
            self._state = state
            self._last_activity = datetime.now()
    
    def get_state(self) -> XAppState:
        """Get the current xApp state"""
        with self._lock:
            return self._state
    
    @property
    def uptime(self) -> float:
        """Get xApp uptime in seconds"""
        return (datetime.now() - self._start_time).total_seconds()


class E2Client:
    """E2 interface client for communication with Near-RT RIC"""
    
    def __init__(self, near_rt_ric_url: str, timeout: float = 30.0):
        self.near_rt_ric_url = near_rt_ric_url
        self.timeout = timeout
        self.session: Optional[aiohttp.ClientSession] = None
        
    async def __aenter__(self):
        self.session = aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=self.timeout))
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def setup_connection(self, node_id: str) -> bool:
        """Setup E2 connection with Near-RT RIC"""
        try:
            setup_request = {
                "message_type": "E2_SETUP_REQUEST",
                "global_e2_node_id": node_id,
                "ran_functions_list": []
            }
            
            async with self.session.post(
                f"{self.near_rt_ric_url}/e2/setup",
                json=setup_request
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    return result.get("success", False)
                return False
        except Exception as e:
            logging.error(f"E2 setup failed: {e}")
            return False
    
    async def subscribe(self, subscription_request: Dict[str, Any]) -> bool:
        """Create an E2 subscription"""
        try:
            async with self.session.post(
                f"{self.near_rt_ric_url}/e2/subscription",
                json=subscription_request
            ) as response:
                return response.status == 200
        except Exception as e:
            logging.error(f"E2 subscription failed: {e}")
            return False
    
    async def unsubscribe(self, subscription_id: str) -> bool:
        """Cancel an E2 subscription"""
        try:
            async with self.session.delete(
                f"{self.near_rt_ric_url}/e2/subscription/{subscription_id}"
            ) as response:
                return response.status == 200
        except Exception as e:
            logging.error(f"E2 unsubscription failed: {e}")
            return False
    
    async def send_control(self, control_request: RICControlRequest) -> Optional[RICControlAcknowledge]:
        """Send a control message"""
        try:
            async with self.session.post(
                f"{self.near_rt_ric_url}/e2/control",
                json=control_request.data
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    return RICControlAcknowledge(
                        ran_function_id=result["ran_function_id"],
                        ric_request_id=result["ric_request_id"],
                        control_outcome=bytes.fromhex(result["control_outcome"]) if result.get("control_outcome") else None
                    )
                return None
        except Exception as e:
            logging.error(f"Control message failed: {e}")
            return None


class XAppSDK:
    """Main xApp SDK class for Python"""
    
    def __init__(self, config: XAppConfig):
        self.config = config
        self.lifecycle = XAppLifecycle()
        self.metrics = XAppMetrics()
        self.subscriptions: Dict[str, XAppSubscription] = {}
        self.indication_handlers: Dict[str, IndicationHandler] = {}
        self.control_handlers: Dict[str, ControlHandler] = {}
        
        # Initialize logging
        self.logger = logging.getLogger(f"xapp.{config.xapp_name}")
        self.logger.setLevel(logging.INFO)
        
        # Initialize E2 client
        self.e2_client: Optional[E2Client] = None
        
        # Background tasks
        self._running = False
        self._metrics_task: Optional[asyncio.Task] = None
        self._health_task: Optional[asyncio.Task] = None
        
        # Setup lifecycle handlers
        self.lifecycle.register_handler(XAppLifecycleEvent.STARTUP, self._handle_startup)
        self.lifecycle.register_handler(XAppLifecycleEvent.SHUTDOWN, self._handle_shutdown)
    
    async def start(self):
        """Start the xApp"""
        self.logger.info(f"Starting xApp: {self.config.xapp_name} v{self.config.xapp_version}")
        
        # Initialize E2 client
        self.e2_client = E2Client(self.config.near_rt_ric_url)
        
        # Trigger startup event
        self.lifecycle.trigger_event(XAppLifecycleEvent.STARTUP, self.config)
        
        # Setup E2 connection
        async with self.e2_client as client:
            if await client.setup_connection(self.config.e2_node_id):
                self.logger.info("E2 connection established")
            else:
                self.logger.warning("Failed to establish E2 connection")
        
        # Start background tasks
        self._running = True
        self._metrics_task = asyncio.create_task(self._metrics_loop())
        
        if self.config.health_check and self.config.health_check.enabled:
            self._health_task = asyncio.create_task(self._health_loop())
        
        self.lifecycle.set_state(XAppState.RUNNING)
        self.logger.info(f"xApp {self.config.xapp_name} started successfully")
    
    async def stop(self):
        """Stop the xApp"""
        self.logger.info(f"Stopping xApp: {self.config.xapp_name}")
        self.lifecycle.set_state(XAppState.STOPPING)
        
        self._running = False
        
        # Cancel all subscriptions
        for subscription_id in list(self.subscriptions.keys()):
            await self.unsubscribe(subscription_id)
        
        # Cancel background tasks
        if self._metrics_task:
            self._metrics_task.cancel()
        if self._health_task:
            self._health_task.cancel()
        
        # Trigger shutdown event
        self.lifecycle.trigger_event(XAppLifecycleEvent.SHUTDOWN, None)
        
        self.lifecycle.set_state(XAppState.STOPPED)
        self.logger.info(f"xApp {self.config.xapp_name} stopped")
    
    async def subscribe(self, node_id: str, ran_function_id: int, 
                       event_trigger: Dict[str, Any], actions: List[XAppAction]) -> Optional[XAppSubscription]:
        """Create a new E2 subscription"""
        
        # Check resource limits
        if len(self.subscriptions) >= self.config.resource_limits.max_subscriptions:
            self.logger.error(f"Maximum subscriptions limit reached: {self.config.resource_limits.max_subscriptions}")
            return None
        
        subscription_id = str(uuid.uuid4())
        subscription = XAppSubscription(
            subscription_id=subscription_id,
            node_id=node_id,
            ran_function_id=ran_function_id,
            event_trigger=event_trigger,
            actions=actions,
            status=XAppSubscriptionStatus.PENDING
        )
        
        # Create subscription request
        subscription_request = {
            "subscription_id": subscription_id,
            "node_id": node_id,
            "ran_function_id": ran_function_id,
            "event_trigger": event_trigger,
            "actions": [
                {
                    "action_id": action.action_id,
                    "action_type": action.action_type,
                    "definition": action.definition
                }
                for action in actions
            ]
        }
        
        # Send subscription via E2 client
        if self.e2_client:
            async with self.e2_client as client:
                success = await client.subscribe(subscription_request)
                if success:
                    subscription.status = XAppSubscriptionStatus.ACTIVE
                    self.subscriptions[subscription_id] = subscription
                    self.metrics.subscriptions_active += 1
                    
                    self.lifecycle.trigger_event(XAppLifecycleEvent.SUBSCRIBED, subscription)
                    self.logger.info(f"Created subscription: {subscription_id} for node: {node_id}")
                    return subscription
                else:
                    subscription.status = XAppSubscriptionStatus.FAILED
                    return subscription
        
        return None
    
    async def unsubscribe(self, subscription_id: str) -> bool:
        """Cancel an E2 subscription"""
        subscription = self.subscriptions.get(subscription_id)
        if not subscription:
            self.logger.error(f"Subscription not found: {subscription_id}")
            return False
        
        subscription.status = XAppSubscriptionStatus.CANCELLING
        
        # Cancel via E2 client
        if self.e2_client:
            async with self.e2_client as client:
                success = await client.unsubscribe(subscription_id)
                if success:
                    del self.subscriptions[subscription_id]
                    subscription.status = XAppSubscriptionStatus.CANCELLED
                    self.metrics.subscriptions_active -= 1
                    self.logger.info(f"Cancelled subscription: {subscription_id}")
                    return True
        
        return False
    
    async def send_control_message(self, control_request: RICControlRequest) -> Optional[RICControlAcknowledge]:
        """Send a control message to an E2 node"""
        self.metrics.control_requests_sent += 1
        
        if self.e2_client:
            async with self.e2_client as client:
                response = await client.send_control(control_request)
                if response:
                    self.logger.info(f"Sent control message, function: {control_request.ran_function_id}")
                    return response
                else:
                    self.metrics.error_count += 1
        
        return None
    
    def register_indication_handler(self, action_type: str, handler: IndicationHandler):
        """Register a handler for RIC indications"""
        self.indication_handlers[action_type] = handler
    
    def register_control_handler(self, control_type: str, handler: ControlHandler):
        """Register a handler for RIC control requests"""
        self.control_handlers[control_type] = handler
    
    async def handle_indication(self, indication: RICIndication):
        """Process incoming RIC indications"""
        self.metrics.indications_received += 1
        
        # Update subscription metrics
        for subscription in self.subscriptions.values():
            if subscription.ran_function_id == indication.ran_function_id:
                subscription.last_indication = datetime.now()
                subscription.indication_count += 1
                break
        
        # Find and call appropriate handler
        handler = self.indication_handlers.get("default")
        if handler:
            try:
                handler(indication)
                self.lifecycle.trigger_event(XAppLifecycleEvent.INDICATION, indication)
            except Exception as e:
                self.logger.error(f"Indication handler error: {e}")
                self.metrics.error_count += 1
        else:
            self.logger.warning(f"No indication handler registered for function: {indication.ran_function_id}")
    
    def get_subscriptions(self) -> Dict[str, XAppSubscription]:
        """Get all active subscriptions"""
        return self.subscriptions.copy()
    
    def get_metrics(self) -> XAppMetrics:
        """Get current metrics"""
        return self.metrics
    
    def get_state(self) -> XAppState:
        """Get current xApp state"""
        return self.lifecycle.get_state()
    
    def get_config(self) -> XAppConfig:
        """Get xApp configuration"""
        return self.config
    
    # Private methods
    
    def _handle_startup(self, event: XAppLifecycleEvent, data: Any):
        """Handle startup event"""
        config = data
        self.logger.info(f"xApp startup: {config.xapp_name} v{config.xapp_version}")
    
    def _handle_shutdown(self, event: XAppLifecycleEvent, data: Any):
        """Handle shutdown event"""
        self.logger.info("xApp shutdown initiated")
    
    async def _metrics_loop(self):
        """Background metrics collection loop"""
        while self._running:
            try:
                await asyncio.sleep(30)  # Update every 30 seconds
                self._update_metrics()
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Metrics loop error: {e}")
    
    async def _health_loop(self):
        """Background health check loop"""
        if not self.config.health_check:
            return
        
        failure_count = 0
        
        while self._running:
            try:
                await asyncio.sleep(self.config.health_check.check_interval)
                
                # Perform health check
                if self._check_health():
                    failure_count = 0
                else:
                    failure_count += 1
                    if failure_count >= self.config.health_check.failure_threshold:
                        self.logger.error("Health check failed - triggering error state")
                        self.lifecycle.set_state(XAppState.ERROR)
                        self.lifecycle.trigger_event(XAppLifecycleEvent.ERROR, "Health check failed")
                        break
                        
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Health check error: {e}")
    
    def _update_metrics(self):
        """Update performance metrics"""
        now = datetime.now()
        duration = (now - self.metrics.last_metrics_update).total_seconds()
        
        if duration > 0:
            # Calculate throughput (indications per second)
            self.metrics.throughput_per_second = self.metrics.indications_received / duration
        
        self.metrics.last_metrics_update = now
    
    def _check_health(self) -> bool:
        """Perform health check"""
        try:
            # Basic health checks
            if self.lifecycle.get_state() not in [XAppState.RUNNING, XAppState.INITIALIZING]:
                return False
            
            # Check if we have active subscriptions (if expected)
            if len(self.subscriptions) == 0 and self.config.service_models:
                return False
            
            # Check error rate
            if self.metrics.error_count > 100:  # Arbitrary threshold
                return False
            
            return True
        except Exception:
            return False


# Utility functions for xApp development

def create_kmp_subscription(measurement_types: List[str], 
                           granularity_period: str = "1000ms") -> Dict[str, Any]:
    """Create a KMP (Key Performance Measurement) subscription event trigger"""
    return {
        "measurement_types": measurement_types,
        "granularity_period": granularity_period,
        "collection_start_time": datetime.now().isoformat(),
        "collection_duration": "continuous",
        "reporting_format": "CHOICE"
    }


def create_rc_control_message(control_action: str, 
                             parameters: Dict[str, Any]) -> bytes:
    """Create an RC (RAN Control) control message"""
    control_data = {
        "action": control_action,
        "parameters": parameters,
        "timestamp": datetime.now().isoformat()
    }
    return json.dumps(control_data).encode('utf-8')


def parse_kmp_indication(indication: RICIndication) -> Dict[str, Any]:
    """Parse a KMP indication message"""
    try:
        message_data = json.loads(indication.indication_message.decode('utf-8'))
        return {
            "measurements": message_data.get("measurements", {}),
            "timestamp": message_data.get("timestamp"),
            "cell_id": message_data.get("cell_id"),
            "measurement_period": message_data.get("measurement_period")
        }
    except Exception as e:
        logging.error(f"Failed to parse KMP indication: {e}")
        return {}


# Example usage
if __name__ == "__main__":
    # This would be used in an actual xApp implementation
    async def example_xapp():
        config = XAppConfig(
            xapp_name="example-kmp-xapp",
            xapp_version="1.0.0",
            xapp_description="Example KMP analytics xApp",
            e2_node_id="gnb-001",
            near_rt_ric_url="http://near-rt-ric:8080",
            service_models=["KPM"],
            resource_limits=XAppResourceLimits(max_subscriptions=5)
        )
        
        sdk = XAppSDK(config)
        
        # Register indication handler
        def kmp_handler(indication: RICIndication):
            data = parse_kmp_indication(indication)
            print(f"Received KMP data: {data}")
        
        sdk.register_indication_handler("default", kmp_handler)
        
        try:
            await sdk.start()
            
            # Create KMP subscription
            event_trigger = create_kmp_subscription([
                "DRB.UEThpDl",
                "DRB.UEThpUl",
                "RRU.PrbUsedDl",
                "RRU.PrbUsedUl"
            ])
            
            actions = [XAppAction(
                action_id=1,
                action_type="report",
                definition={"format": "json"}
            )]
            
            subscription = await sdk.subscribe("gnb-001", 1, event_trigger, actions)
            if subscription:
                print(f"Created subscription: {subscription.subscription_id}")
            
            # Keep running
            while sdk.get_state() == XAppState.RUNNING:
                await asyncio.sleep(1)
                
        finally:
            await sdk.stop()
    
    # Run example
    # asyncio.run(example_xapp())