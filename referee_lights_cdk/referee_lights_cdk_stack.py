# referee_lights_cdk/referee_lights_cdk_stack.py
"""
cdk deployment for go referee lights
"""
from aws_cdk import (
    aws_ec2 as ec2,
    aws_iam as iam,
    aws_logs as logs,
    aws_ecr_assets as ecr_assets,
    aws_ecs as ecs,
    aws_ecs_patterns as ecs_patterns,
    aws_elasticloadbalancingv2 as elbv2,
    aws_certificatemanager as acm,
    CfnOutput,
    Duration,
    Stack,
    RemovalPolicy,
)
from constructs import Construct
import pathlib

project_root = pathlib.Path(__file__).resolve().parent.parent

class RefereeLightsCdkStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # define Variables
        domain_name = "referee-lights.michaelkingston.com.au"

        # create VPC
        vpc = ec2.Vpc(
            self,
            "RefereeLightsVPC",
            vpc_name="referee-lights-vpc",
            max_azs=2,
            nat_gateways=1,
        )

        # create ECS Cluster
        cluster = ecs.Cluster(
            self,
            "RefereeLightsCluster",
            cluster_name="referee-lights-cluster",
            vpc=vpc,
        )

        # define IAM Roles (task_role)
        task_role = iam.Role(
            self, "TaskRole",
            assumed_by=iam.ServicePrincipal("ecs-tasks.amazonaws.com")
        )

        # define IAM roles (execution_role)
        execution_role = iam.Role(
            self, "ExecutionRole",
            assumed_by=iam.ServicePrincipal("ecs-tasks.amazonaws.com")
        )

        execution_role.add_managed_policy(
            iam.ManagedPolicy.from_aws_managed_policy_name(
                "service-role/AmazonECSTaskExecutionRolePolicy"
            )
        )

        # build Docker Image
        docker_image_asset = ecr_assets.DockerImageAsset(
            self,
            "RefereeLightsDockerImage",
            directory=str(project_root),
        )

        # define Fargate Task Definition
        task_definition = ecs.FargateTaskDefinition(
            self, "RefereeLightsTaskDef",
            family="referee-lights-task",
            memory_limit_mib=512,
            cpu=256,
            task_role=task_role,
            execution_role=execution_role
        )

        # add Container to Task Definition
        container = task_definition.add_container(
            "RefereeLightsContainer",
            image=ecs.ContainerImage.from_docker_image_asset(docker_image_asset),
            logging=ecs.LogDrivers.aws_logs(
                stream_prefix="referee-lights",
                log_group=logs.LogGroup(
                    self,
                    "RefereeLightsLogGroup",
                    log_group_name="/ecs/referee-lights-app-container",
                    retention=logs.RetentionDays.ONE_WEEK,
                    removal_policy=RemovalPolicy.DESTROY,
                ),
            ),
            environment={
                "ENV": "production",
                "APPLICATION_URL": f"https://{domain_name}",
                "WEBSOCKET_URL": f"wss://{domain_name}/referee-updates",
            },
            # health_check=ecs.HealthCheck(
            #     command=["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
            #     interval=Duration.seconds(30),
            #     retries=3,
            #     timeout=Duration.seconds(5),
            #     start_period=Duration.seconds(60),
            # ),
        )

        container.add_port_mappings(
            ecs.PortMapping(container_port=8080),
        )

        # import ACM Certificate
        certificate = acm.Certificate.from_certificate_arn(
            self,
            "RefereeLightsCertificate",
            certificate_arn="arn:aws:acm:ap-southeast-2:001499655372:certificate/d644df5b-c471-423c-962c-afcc6d86568c"
        )

        # create Application Load Balanced Fargate Service
        fargate_service = ecs_patterns.ApplicationLoadBalancedFargateService(
            self,
            "RefereeLightsFargateService",
            service_name="referee-lights-service",
            cluster=cluster,
            task_definition=task_definition,
            public_load_balancer=True,
            desired_count=1,
            listener_port=443,
            certificate=certificate,
            protocol=elbv2.ApplicationProtocol.HTTPS,
            redirect_http=True,
        )

        # configure Health Check
        fargate_service.target_group.configure_health_check(
            path="/health",
            protocol=elbv2.Protocol.HTTP,
            port="traffic-port",
            interval=Duration.seconds(30),
            healthy_threshold_count=2,
            unhealthy_threshold_count=5,
        )

        # output ALB DNS Name
        self.output_alb_dns = CfnOutput(
            self,
            "ALBDNS",
            value=fargate_service.load_balancer.load_balancer_dns_name,
            description="The DNS address of the load balancer"
        )
