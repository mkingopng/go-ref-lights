import aws_cdk as core
import aws_cdk.assertions as assertions

from go_ref_lights.go_ref_lights_stack import GoRefLightsStack

# example tests. To run these tests, uncomment this file along with the example
# resource in go_ref_lights/go_ref_lights_stack.py
def test_sqs_queue_created():
    app = core.App()
    stack = GoRefLightsStack(app, "go-ref-lights")
    template = assertions.Template.from_stack(stack)

#     template.has_resource_properties("AWS::SQS::Queue", {
#         "VisibilityTimeout": 300
#     })
