#!/usr/bin/env python3
import aws_cdk as cdk
from stacks.api_stack import BedrockApiStack

app = cdk.App()

# Deploy to your AWS account/region from environment or defaults
env = cdk.Environment(
    account=app.node.try_get_context("account"),
    region=app.node.try_get_context("region") or "us-east-1"
)

BedrockApiStack(
    app,
    "BedrockApiStack",
    env=env,
    description="Bedrock Question Search API with Lambda and API Gateway"
)

app.synth()
