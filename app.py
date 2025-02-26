"""
boilerplate to make CDK work
"""
import aws_cdk as cdk
from referee_lights_cdk.referee_lights_cdk_stack import RefereeLightsCdkStack

app = cdk.App()
RefereeLightsCdkStack(
	app,
	"RefereeLightsCdkStack",
	env=cdk.Environment(account="001499655372", region="ap-southeast-2")
)

app.synth()
