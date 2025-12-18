from aws_cdk import (
    Stack,
    Duration,
    CfnOutput,
    aws_lambda as lambda_,
    aws_apigatewayv2 as apigw,
    aws_apigatewayv2_integrations as integrations,
    aws_iam as iam,
    aws_logs as logs,
)
from constructs import Construct


class BedrockApiStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, **kwargs):
        super().__init__(scope, construct_id, **kwargs)

        # Get configuration from context or use defaults
        aws_region = self.node.try_get_context("aws_region") or "us-east-1"
        embedding_model = self.node.try_get_context("embedding_model") or "amazon.titan-embed-text-v2"
        knowledge_base_id = self.node.try_get_context("knowledge_base_id") or "LCMAYLRJ7D"
        max_question_length = self.node.try_get_context("max_question_length") or "1000"
        retry_attempts = self.node.try_get_context("retry_attempts") or "3"

        # IAM role for Lambda with Bedrock permissions
        lambda_role = iam.Role(
            self,
            "BedrockApiLambdaRole",
            assumed_by=iam.ServicePrincipal("lambda.amazonaws.com"),
            managed_policies=[
                iam.ManagedPolicy.from_aws_managed_policy_name(
                    "service-role/AWSLambdaBasicExecutionRole"
                )
            ],
        )

        # Add Bedrock permissions
        lambda_role.add_to_policy(
            iam.PolicyStatement(
                effect=iam.Effect.ALLOW,
                actions=[
                    "bedrock:InvokeModel",
                    "bedrock:InvokeModelWithResponseStream",
                ],
                resources=[
                    f"arn:aws:bedrock:{aws_region}::foundation-model/{embedding_model}",
                    f"arn:aws:bedrock:{aws_region}::foundation-model/*",
                ],
            )
        )

        # Add Bedrock Agent Runtime permissions for Knowledge Base
        lambda_role.add_to_policy(
            iam.PolicyStatement(
                effect=iam.Effect.ALLOW,
                actions=[
                    "bedrock:Retrieve",
                    "bedrock:RetrieveAndGenerate",
                ],
                resources=[
                    f"arn:aws:bedrock:{aws_region}:{self.account}:knowledge-base/{knowledge_base_id}",
                ],
            )
        )

        # Lambda function for Go API using custom runtime
        api_lambda = lambda_.Function(
            self,
            "BedrockApiFunction",
            runtime=lambda_.Runtime.PROVIDED_AL2023,
            handler="bootstrap",
            code=lambda_.Code.from_asset("../lambda-build"),
            role=lambda_role,
            timeout=Duration.seconds(30),
            memory_size=512,
            architecture=lambda_.Architecture.X86_64,
            environment={
                "BEDROCK_REGION": aws_region,
                "BEDROCK_EMBEDDING_MODEL": embedding_model,
                "BEDROCK_KB_ID": knowledge_base_id,
                "MAX_QUESTION_LENGTH": max_question_length,
                "RETRY_ATTEMPTS": retry_attempts,
                "AWS_LWA_INVOKE_MODE": "response_stream",
            },
            log_retention=logs.RetentionDays.ONE_WEEK,
            description="Bedrock Question Search API Lambda Function",
        )

        # HTTP API Gateway
        http_api = apigw.HttpApi(
            self,
            "BedrockHttpApi",
            api_name="bedrock-question-search-api",
            description="Bedrock Question Search API",
            cors_preflight=apigw.CorsPreflightOptions(
                allow_origins=["*"],
                allow_methods=[apigw.CorsHttpMethod.GET, apigw.CorsHttpMethod.POST],
                allow_headers=["Content-Type", "Authorization"],
            ),
        )

        # Lambda integration
        lambda_integration = integrations.HttpLambdaIntegration(
            "LambdaIntegration",
            api_lambda,
        )

        # Add catch-all route to forward all requests to Lambda
        http_api.add_routes(
            path="/{proxy+}",
            methods=[apigw.HttpMethod.ANY],
            integration=lambda_integration,
        )

        # Outputs
        CfnOutput(
            self,
            "ApiUrl",
            value=http_api.url or "",
            description="HTTP API Gateway URL",
        )

        CfnOutput(
            self,
            "LambdaFunctionName",
            value=api_lambda.function_name,
            description="Lambda function name",
        )
