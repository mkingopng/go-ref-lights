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
    Duration,
    Stack,
    RemovalPolicy,
    Tags,
    IAspect,
)
from constructs import Construct
import pathlib

project_root = pathlib.Path(__file__).resolve().parent.parent

class TaggingAspect(IAspect):
    def __init__(self, key: str, value: str):
        self.key = key
        self.value = value

class RefereeLightsCdkStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # add comprehensive tags for better cost tracking
        Tags.of(self).add("Project", "RefereeLightsApp")
        Tags.of(self).add("Environment", "Production")
        Tags.of(self).add("Owner", "Michael_Kingston")
        Tags.of(self).add("CostCenter", "RefereeLights")
        Tags.of(self).add("Application", "referee-lights")
        Tags.of(self).add("AutoScale", "enabled")

        # define domain name variable
        domain_name = "referee-lights.michaelkingston.com.au"

        vpc = ec2.Vpc(
            self,
            "RefereeLightsVPC",
            ip_addresses=ec2.IpAddresses.cidr("10.0.0.0/16"),
            max_azs=2,  # Ensure two different AZs
            nat_gateways=0,
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    name="PublicSubnet",
                    subnet_type=ec2.SubnetType.PUBLIC,
                    cidr_mask=24
                )
            ]
        )

        # Ensure an Internet Gateway exists for Public Subnets
        vpc.add_gateway_endpoint(
            "S3Endpoint",
            service=ec2.GatewayVpcEndpointAwsService.S3
        )

        # create Security Group for ECS Tasks
        security_group = ec2.SecurityGroup(
            self,
            "RefereeLightsSecurityGroup",
            vpc=vpc,
            description="Security group for Referee Lights ECS tasks",
            allow_all_outbound=True
        )

        # add inbound rules to security group as needed
        security_group.add_ingress_rule(
            peer=ec2.Peer.any_ipv4(),
            connection=ec2.Port.tcp(80),
            description="Allow HTTP inbound"
        )

        security_group.add_ingress_rule(
            peer=ec2.Peer.any_ipv4(),
            connection=ec2.Port.tcp(443),
            description="Allow HTTPS inbound"
        )

        # create ECS Cluster
        cluster = ecs.Cluster(
            self,
            "RefereeLightsCluster",
            cluster_name="referee-lights-cluster",
            vpc=vpc,
        )

        # define IAM task role
        task_role = iam.Role(
            self, "TaskRole",
            assumed_by=iam.ServicePrincipal("ecs-tasks.amazonaws.com")
        )

        # define IAM execution role
        execution_role = iam.Role(
            self,
            "ExecutionRole",
            assumed_by=iam.ServicePrincipal("ecs-tasks.amazonaws.com")
        )

        # add necessary policy attachments to roles here
        execution_role.add_managed_policy(
            iam.ManagedPolicy.from_aws_managed_policy_name(
                "service-role/AmazonECSTaskExecutionRolePolicy"
            )
        )

        # build docker image
        docker_image_asset = ecr_assets.DockerImageAsset(
            self,
            "RefereeLightsDockerImage",
            directory=str(project_root),
            exclude=["cdk.out", "cdk.context.json", "cdk*.json", "cdk.staging", "**/cdk.out/**"]
        )

        # define fargate task definition
        task_definition = ecs.FargateTaskDefinition(
            self,
            "RefereeLightsTaskDef",
            family="referee-lights-task",
            memory_limit_mib=1024,
            cpu=512,
            task_role=task_role,
            execution_role=execution_role
        )

        # add container to task definition
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
                "LOG_LEVEL": "DEBUG",
                "HOST": "0.0.0.0",
                "PORT": "8080"
            },
            health_check=None,
            # ecs.HealthCheck(
            #     command=["CMD-SHELL", "curl -f http://0.0.0.0:8080/health || exit 1"],
            #     interval=Duration.seconds(30),
            #     timeout=Duration.seconds(10),
            #     retries=3,
            #     start_period=Duration.seconds(120)
            # )
        )

        container.add_port_mappings(
            ecs.PortMapping(container_port=8080),
        )

        # import ACM certificate
        certificate = acm.Certificate.from_certificate_arn(
            self,
            "RefereeLightsCertificate",
            certificate_arn="arn:aws:acm:ap-southeast-2:001499655372:certificate/d644df5b-c471-423c-962c-afcc6d86568c"
        )

        # create application load balanced fargate service
        fargate_service = ecs_patterns.ApplicationLoadBalancedFargateService(
            self,
            "RefereeLightsFargateService",
            service_name="referee-lights-service",
            cluster=cluster,
            task_definition=task_definition,
            public_load_balancer=True,
            assign_public_ip=True,  # Ensure ECS tasks get an internet IP
            desired_count=1,
            listener_port=443,
            certificate=certificate,
            protocol=elbv2.ApplicationProtocol.HTTPS,
            redirect_http=True,
            capacity_provider_strategies=[
                ecs.CapacityProviderStrategy(
                    capacity_provider="FARGATE",
                    weight=1
                )
            ],
            security_groups=[security_group]
        )

        fargate_service.load_balancer.connections.security_groups[0].add_ingress_rule(
            peer=ec2.Peer.any_ipv4(),
            connection=ec2.Port.tcp(443),
            description="Allow HTTPS inbound"
        )

        # ALB must allow communication with ECS tasks
        fargate_service.service.connections.allow_from(
            fargate_service.load_balancer,
            ec2.Port.tcp(8080),
            "Allow ALB to communicate with ECS service"
        )

        # configure target group health check
        fargate_service.target_group.configure_health_check(
            path="/health",
            protocol=elbv2.Protocol.HTTP,
            port="8080",
            interval=Duration.seconds(120),
            timeout=Duration.seconds(10),
            healthy_threshold_count=2,
            unhealthy_threshold_count=5,
            healthy_http_codes="200-299",
        )

        fargate_service.service.health_check_grace_period = Duration.seconds(300)

        # configure load balancer attributes
        fargate_service.load_balancer.set_attribute(
            "idle_timeout.timeout_seconds",
            "3600"  # 60 minutes
        )

        # add additional load balancer attributes for better performance
        fargate_service.load_balancer.set_attribute(
            "routing.http2.enabled",
            "true"
        )

        fargate_service.load_balancer.set_attribute("idle_timeout.timeout_seconds", "3600")
        fargate_service.load_balancer.set_attribute("routing.http2.enabled", "true")
        fargate_service.load_balancer.apply_removal_policy(RemovalPolicy.DESTROY)
