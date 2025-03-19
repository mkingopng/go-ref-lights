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
    aws_cloudwatch as cloudwatch,
    aws_sns as sns,
    aws_sns_subscriptions as sns_subs,
    aws_cloudwatch_actions as actions,
    aws_applicationautoscaling as appscaling,
    CfnOutput,
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

        # add tags to the stack
        # Add more comprehensive tags for better cost tracking
        Tags.of(self).add("Project", "RefereeLightsApp")
        Tags.of(self).add("Environment", "Production")
        Tags.of(self).add("Owner", "Michael_Kingston")
        Tags.of(self).add("CostCenter", "RefereeLights")
        Tags.of(self).add("Application", "referee-lights")
        Tags.of(self).add("AutoScale", "enabled")

        # define domain name variable
        domain_name = "referee-lights.michaelkingston.com.au"

        # create VPC
        vpc = ec2.Vpc(
            self,
            "RefereeLightsVPC",
            vpc_name="referee-lights-vpc",
            max_azs=2,
            nat_gateways=1,
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    name="Public",
                    subnet_type=ec2.SubnetType.PUBLIC,
                    cidr_mask=24
                ),
                ec2.SubnetConfiguration(
                    name="Private",
                    subnet_type=ec2.SubnetType.PRIVATE_WITH_EGRESS,
                    cidr_mask=24
                )
            ]
        )

        vpc.add_flow_log("FlowLogs", destination=ec2.FlowLogDestination.to_cloud_watch_logs())

        # Create billing alarm
        billing_alarm = cloudwatch.Alarm(
            self,
            "BillingAlarm",
            metric=cloudwatch.Metric(
                namespace="AWS/Billing",
                metric_name="EstimatedCharges",
                dimensions_map={"Currency": "USD"},
                period=Duration.hours(6)
            ),
            evaluation_periods=1,
            threshold=50,  # Set your threshold in USD
            comparison_operator=cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
        )

        # Create an SNS topic for billing alerts
        sns_topic = sns.Topic(
            self,
            "BillingAlertsTopic",
            display_name="Billing Alerts for Referee Lights"
        )

        # Add an email subscription (Replace with your email)
        sns_topic.add_subscription(sns_subs.EmailSubscription("michael.kenneth.kingston@gmail.com"))

        # Attach SNS action to CloudWatch billing alarm
        billing_alarm.add_alarm_action(actions.SnsAction(sns_topic))

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
            self, "ExecutionRole",
            assumed_by=iam.ServicePrincipal("ecs-tasks.amazonaws.com")
        )

        execution_role.add_managed_policy(
            iam.ManagedPolicy.from_aws_managed_policy_name(
                "service-role/AmazonECSTaskExecutionRolePolicy"
            )
        )

        # build Docker image
        docker_image_asset = ecr_assets.DockerImageAsset(
            self,
            "RefereeLightsDockerImage",
            directory=str(project_root),
            exclude=["cdk.out", "cdk.context.json", "cdk*.json", "cdk.staging", "**/cdk.out/**"]
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
                "LOG_LEVEL": "DEBUG",
            },
            health_check=ecs.HealthCheck(
                command=["CMD-SHELL", "curl -f http://127.0.0.1:8080/health || exit 1"],
                interval=Duration.seconds(10),
                timeout=Duration.seconds(3),
                retries=2,
                start_period=Duration.seconds(10)
            )
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
            capacity_provider_strategies=[
                ecs.CapacityProviderStrategy(
                    capacity_provider="FARGATE_SPOT",
                    weight=1
                )
            ]
        )

        # Add explicit security group rule for health checks
        fargate_service.service.connections.allow_from(
            fargate_service.load_balancer,
            ec2.Port.tcp(8080),
            "Allow health check from ALB"
        )

        scaling = fargate_service.service.auto_scale_task_count(
            max_capacity=2,
            min_capacity=0
        )

        scaling.scale_on_cpu_utilization(
            "CpuScaling",
            target_utilization_percent=50,
            scale_in_cooldown=Duration.seconds(180),
            scale_out_cooldown=Duration.seconds(30),
        )

        scaling.scale_on_request_count(
            "RequestCountScaling",
            requests_per_target=100,
            target_group=fargate_service.target_group,
            scale_in_cooldown=Duration.seconds(300),
            scale_out_cooldown=Duration.seconds(60)
        )

        # Scale up for **weekend production** (Friday night to Sunday night)
        scaling.scale_on_schedule(
            "WeekendProductionScaling",
            schedule=appscaling.Schedule.cron(
                week_day="FRI-SUN",
                hour="18",
                minute="0"
            ),
            min_capacity=1,
            max_capacity=2
        )

        # Scale down for **weekday development/testing** (Monday morning)
        scaling.scale_on_schedule(
            "WeekdayDevelopmentScaling",
            schedule=appscaling.Schedule.cron(
                week_day="Mon-FRI",
                hour="18",
                minute="0"
            ),
            min_capacity=1
        )

        # configure Health Check
        fargate_service.target_group.configure_health_check(
            path="/health",
            protocol=elbv2.Protocol.HTTP,
            port="8080",
            interval=Duration.seconds(10),
            timeout=Duration.seconds(3),
            healthy_threshold_count=2,
            unhealthy_threshold_count=2,
            healthy_http_codes="200-299",
        )

        # set idle timeout
        fargate_service.load_balancer.set_attribute(
            "idle_timeout.timeout_seconds",
            "3600" # 60 minutes
            # "1800" # 30 minutes
            # "300"
        )

        # output ALB DNS Name
        self.output_alb_dns = CfnOutput(
            self,
            "ALBDNS",
            value=fargate_service.load_balancer.load_balancer_dns_name,
            description="The DNS address of the load balancer"
        )
