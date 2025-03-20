compile
```bash
go build ./...
```

```go
go run .
```

run app locally
```bash
ENV=development go run main.go
```

or run from docker:
```bash
docker run -e ENV=development -p 8080:8080 referee-lights
```

run all tests
```bash
go test -v ./...
```

run unit tests
```bash
go test -v -tags=unit ./...
```

run unit tests in a specific directory
```bash
go test -v -tags=unit ./websocket
```

test coverage
```bash
go test -v -tags=unit -coverprofile=cover.out ./controllers
```
or
```bash
go test -v -tags=unit -cover ./controllers
```

run precommit hooks:
```bash
pre-commit run --all-files
```

before committing, run:
```bash
poetry run pre-commit run --all-files
go test -v -tags=unit ./...
```

run integration tests
```bash
go test -v -tags=integration ./...
```

run integration tests in a specific directory
```bash
go test -v -tags=integration ./websocket
```


go's race detector:
```bash
go test -race ./...
```

---
# Best Practices for Maintaining Unit Tests**
Since we’re writing **a large number of tests**, here are **best practices** to
ensure long-term maintainability:

### Follow the Given-When-Then Structure
Write tests in a **clear, structured way**:
- **Keeps tests readable** and **maintains consistency**.

### Use Mocks for External Dependencies
- **Use mock services** instead of real implementations.
- **Minimize reliance on actual databases, API calls, or network connections**.
- Speeds up tests and reduces flakiness

### Separate Unit vs. Integration vs. Smoke Tests
- **Unit Tests** → Test individual functions in isolation.
- **Integration Tests** → Test how multiple components work together.
- **Smoke Tests** → Run a minimal test to check if the system starts up without errors.
- test under load to ensure that the system performs well under high traffic
- **Avoid confusion** between different test types.

### Run Tests in CI/CD Pipelines
- **Ensure all tests run automatically** before merging code.
- **Use GitHub Actions, GitLab CI, or Jenkins** to automate testing.
- **Catches issues before they reach production**.

### Keep Tests Fast
- **Optimize tests** to avoid slow performance.
- **Mock external services** instead of making real calls.
- **Use table-driven tests** to avoid redundant code.
- Encourages running tests frequently

### Write Self-Contained Tests
- **Each test should be independent**.
- **Tests should NOT rely on global state** (e.g. shared session data, database entries).
- **Prevents flaky test failures**.

### Use Meaningful Test Names

### Ensure Clean Test Data
- **Reset mock services after each test**.
- **Use `defer` to clean up test artifacts**.
- **Prevents one test from interfering with another**.

### Prioritize High-Coverage Areas
- **Start with testing critical business logic**.
- **Cover edge cases (invalid input, errors, permissions, etc.).**
- **Focus on high-risk areas first**.

# tasks
1. Integration tests
    - Login + Session Management → Ensure login persists a session.
    - Meet Creation + Role Assignment → Verify a user can create a meet and
      assign roles.
    - Position Claiming + Websocket Broadcast → User claims a position → UI
      updates via broadcast.
    - Referee Actions + State Updates → A referee gives a lift decision →
      The state updates correctly.
    - broadcast_integration_test.go
    - auth_integration_test.go
    - page_integration_test.go
    - position_integration_tst.go
    - api_integration_test.go

#  Test Upgrades
1. Smoke tests
2. Load tests (K6)
3. update precommit hooks
    - golangci-lint (code quality)
    - gofmt (formatting)
    - govet (detect common issues)
    - prettier for frontend parts (if applicable)
4. CI/CD: Automate tests and deploys via GitHub Actions.
    - Run unit tests.
    - Run integration tests.
    - Run smoke tests.
    - Deploy if all pass.
5. improved formatting
6. admin page
7. logout from anywhere
8. reset meet state
9. sudo page

---

# Improvements and bugs
1. When a referee is in position, if they look at another page or app on
   their phone, it can cause the health check to fail. Ideally, the
   healthy connection should be maintained regardless of what the user does
   on their phone, as long as the browser window is open. However, if, for
   whatever reason, the connection becomes unhealthy, the user should be
   able to refresh the page and rejoin the meet without any issues. This is
   not always happening as it should. In many cases, when I hit refresh, I get
   a 404 error. Refer to the screenshots attached
2. There are many mechanisms for the referee to vacate their position,
   or log out however only the admin panel is working correctly. The other
   mechanisms don't work correctly.
   - On the referee screen, there is a button called vacate position. see
     attached image. When this button is pressed it should vacate the position
     and take the user back to /index. However, it does not do this. The
     user gets a 404 error. Refer to the screenshots attached. This is not
     the correct behaviour. We need to correct this. What should happen is a
     redirect to /index.
   - The referee screen has a button called logout. When this button is
     pressed, the user should be logged out and taken back to the login
     page. This doesn't happen. The user gets a 404 error. Refer to the
     screenshot. we need to fix this. What should happen is the user is
     logged out from that when the button is pressed, the referee is taken
     back to /index where they can take on a new position.
   - the referee page has a button called Home. we should remove this button
     as it is not needed.
   - When the referee logs out the meet persists
   - When the admin logs out the meet resets
3. There needs to be a super-user or sudo role who can log into
   any meet and take control as a fall-back position. This is not yet built in
   to the functionality. Not sure how to implement this yet.
4. Need to implement dynamic logo. Most meets currently use the APL logo,
   however more and more meets will use specific logos. In anticipation of
   this i have included logo in the meet.go data structure but it is not
   used anywhere yet. Need to implement this.
5. Review the CDK code and optimise. Consider how to scale to zero

-------

# CDK upgrades
- scheduling
- scale to zero

```python
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

        # add comprehensive tags for better cost tracking
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

        # create billing alarm
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
            threshold=50,  # set threshold in USD
            comparison_operator=cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
        )

        # create an SNS topic for billing alerts
        sns_topic = sns.Topic(
            self,
            "BillingAlertsTopic",
            display_name="Billing Alerts for Referee Lights"
        )

        # add an email subscription
        sns_topic.add_subscription(sns_subs.EmailSubscription("michael.kenneth.kingston@gmail.com"))

        # attach SNS action to CloudWatch billing alarm
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

        # define Fargate task definition
        task_definition = ecs.FargateTaskDefinition(
            self, "RefereeLightsTaskDef",
            family="referee-lights-task",
            memory_limit_mib=512,
            cpu=256,
            task_role=task_role,
            execution_role=execution_role
        )

        # add Container to task definition
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
            health_check=ecs.HealthCheck(
                command=["CMD-SHELL", "curl -f http://0.0.0.0:8080/health || exit 1"],
                interval=Duration.seconds(30),
                timeout=Duration.seconds(5),
                retries=3,
                start_period=Duration.seconds(120)
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

        # add explicit security group rule for health checks
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

        # scale up for weekend (Friday night to Sunday night)
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

        # scale down for weekday development/testing (Monday - Friday)
        scaling.scale_on_schedule(
            "WeekdayDevelopmentScaling",
            schedule=appscaling.Schedule.cron(
                week_day="Mon-FRI",
                hour="24",
                minute="0"
            ),
            min_capacity=0,
            max_capacity=1
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
```
