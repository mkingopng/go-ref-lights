# referee_lights_cdk/referee_lights_cdk_stack.py
"""
cdk deployment
"""
from aws_cdk import (
    aws_ec2 as ec2,
    aws_ecs as ecs,
    aws_ecs_patterns as ecs_patterns,
    aws_elasticloadbalancingv2 as elbv2,
    aws_certificatemanager as acm,
    aws_route53 as route53,
    aws_route53_targets as targets,
    RemovalPolicy,
    CfnOutput,
    Duration,
    Stack,
)
from constructs import Construct

class RefereeLightsCdkStack(Stack):

    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # ----------------------------
        # 1. Define Variables
        # ----------------------------

        # Replace with your actual values
        domain_name = ""
        hosted_zone_name = ""
        ecr_repository_name = "referee-lights"
        ecr_image_tag = "latest"
        aws_account_id = "123456789012"  # Replace with your AWS Account ID
        aws_region = "ap-southeast-2"

        # ----------------------------
        # 2. Lookup Hosted Zone
        # ----------------------------

        hosted_zone = route53.HostedZone.from_lookup(
            self, "HostedZone",
            domain_name=hosted_zone_name
        )

        # ----------------------------
        # 3. Create VPC
        # ----------------------------

        vpc = ec2.Vpc(
            self, "RefereeLightsVPC",
            max_azs=3,
            nat_gateways=1,
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    name="public",
                    subnet_type=ec2.SubnetType.PUBLIC,
                    cidr_mask=24
                ),
                ec2.SubnetConfiguration(
                    name="private",
                    subnet_type=ec2.SubnetType.PRIVATE_WITH_EGRESS,
                    cidr_mask=24
                )
            ]
        )

        # ----------------------------
        # 4. Create ECS Cluster
        # ----------------------------

        cluster = ecs.Cluster(
            self, "RefereeLightsCluster",
            vpc=vpc,
            cluster_name="referee-lights-cluster"
        )

        # ----------------------------
        # 5. Define Fargate Task Definition
        # ----------------------------

        task_definition = ecs.FargateTaskDefinition(
            self, "RefereeLightsTaskDef",
            memory_limit_mib=512,
            cpu=256,
        )

        # Add Container to Task Definition
        container = task_definition.add_container(
            "RefereeLightsContainer",
            image=ecs.ContainerImage.from_registry(
                f"{aws_account_id}.dkr.ecr.{aws_region}.amazonaws.com/{ecr_repository_name}:{ecr_image_tag}"
            ),
            logging=ecs.LogDrivers.aws_logs(stream_prefix="referee-lights"),
            environment={
                # Add environment variables if needed
                "ENV": "production"
            }
        )

        container.add_port_mappings(
            ecs.PortMapping(
                container_port=8080
            )
        )

        # ----------------------------
        # 6. Create ECR Repository (Optional)
        # ----------------------------

        # If you haven't created the ECR repository manually, uncomment the following:
        """
        ecr_repository = ecs.Repository.from_repository_name(
            self, "RefereeLightsECR",
            repository_name=ecr_repository_name
        )
        """

        # ----------------------------
        # 7. Create ACM Certificate
        # ----------------------------

        certificate = acm.Certificate(
            self, "RefereeLightsCertificate",
            domain_name=domain_name,
            validation=acm.CertificateValidation.from_dns(hosted_zone),
            removal_policy=RemovalPolicy.DESTROY  # Change to RETAIN for production
        )

        # ----------------------------
        # 8. Create Application Load Balancer (ALB) with HTTPS
        # ----------------------------

        # Create an Application Load Balancer
        alb = elbv2.ApplicationLoadBalancer(
            self, "RefereeLightsALB",
            vpc=vpc,
            internet_facing=True,
            load_balancer_name="referee-lights-alb"
        )

        # Add HTTPS Listener
        https_listener = alb.add_listener(
            "HTTPSListener",
            port=443,
            certificates=[certificate],
            default_action=elbv2.ListenerAction.fixed_response(
                status_code=404,
                message_body="Not Found",
                content_type="text/plain"
            )
        )

        # Add HTTP Listener for Redirection
        http_listener = alb.add_listener(
            "HTTPListener",
            port=80,
            open=True
        )

        # Redirect HTTP to HTTPS
        http_listener.add_action(
            "RedirectToHTTPS",
            action=elbv2.ListenerAction.redirect(
                protocol="HTTPS",
                port="443",
                permanent=True
            )
        )

        # ----------------------------
        # 9. Create Fargate Service with ALB Integration
        # ----------------------------

        fargate_service = ecs_patterns.ApplicationLoadBalancedFargateService(
            self, "RefereeLightsFargateService",
            cluster=cluster,
            task_definition=task_definition,
            desired_count=2,
            public_load_balancer=False,  # ALB is already internet-facing
            listener_port=8080,  # The container port
            load_balancer=alb,
            listener=https_listener,
            protocol=elbv2.ApplicationProtocol.HTTP,  # Internal communication
            domain_name=domain_name,
            domain_zone=hosted_zone,
        )

        # Optional: Configure Auto Scaling
        scaling = fargate_service.service.auto_scale_task_count(
            max_capacity=5,
            min_capacity=2
        )

        scaling.scale_on_cpu_utilization(
            "CpuScaling",
            target_utilization_percent=70,
            scale_in_cooldown=Duration.seconds(60),
            scale_out_cooldown=Duration.seconds(60)
        )

        # ----------------------------
        # 10. Add Route53 DNS Record (Already handled by CDK Patterns)
        # ----------------------------

        # The `ecs_patterns.ApplicationLoadBalancedFargateService` automatically creates the necessary DNS records
        # since we provided `domain_name` and `domain_zone`

        # ----------------------------
        # 11. Output ALB DNS Name
        # ----------------------------

        self.output_alb_dns = CfnOutput(
            self, "ALBDNS",
            value=alb.load_balancer_dns_name,
            description="The DNS address of the load balancer"
        )
