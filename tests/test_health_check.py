# tests/test_health_check.py
"""
This module contains the tests for the health check endpoint.
"""
from aws_cdk import App, Duration
from aws_cdk.assertions import Template
from referee_lights_cdk.referee_lights_cdk_stack import RefereeLightsCdkStack

def test_health_check_configuration():
	app = App()
	stack = RefereeLightsCdkStack(app, "TestStack", env={"region": "ap-southeast-2"})
	template = Template.from_stack(stack)

	# Assert the container health check in the ECS TaskDefinition
	template.has_resource_properties("AWS::ECS::TaskDefinition", {
		"ContainerDefinitions": [{
			"HealthCheck": {
				"Command": ["CMD-SHELL", "curl -f http://127.0.0.1:8080/health || exit 1"],
				"Interval": 10,         # Update these values if you changed them
				"Timeout": 3,
				"Retries": 2,
				"StartPeriod": 10
			}
		}]
	})

	# Assert the target group's health check configuration
	template.has_resource_properties("AWS::ElasticLoadBalancingV2::TargetGroup", {
		"HealthCheckPath": "/health",
		"HealthCheckIntervalSeconds": 10,
		"HealthCheckTimeoutSeconds": 3,
		"HealthyThresholdCount": 2,
		"UnhealthyThresholdCount": 2,
	})
