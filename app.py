# app.py
"""
boilerplate to make CDK work
"""
import aws_cdk as cdk
from referee_lights_cdk.referee_lights_cdk_stack import RefereeLightsCdkStack
from billing_alarm_stack import BillingAlarmStack
# from monitoring_stack import MonitoringStack

app = cdk.App()

# Deploy the billing alarm stack to us-east-1
# BillingAlarmStack(
# 	app,
# 	"BillingAlarmStack",
# 	env=cdk.Environment(region="us-east-1"),
# )

# Deploy your main application to ap-southeast-2
RefereeLightsCdkStack(
	app,
	"RefereeLightsCdkStack",
	env=cdk.Environment(account="001499655372", region="ap-southeast-2")
)

app.synth()
