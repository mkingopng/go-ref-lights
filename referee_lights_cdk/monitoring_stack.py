from aws_cdk import (
	Stack,
	Duration,
	aws_cloudwatch as cw,
	aws_sns as sns,
	aws_sns_subscriptions as subs,
	aws_cloudwatch_actions as cw_actions,
)
from constructs import Construct

class MonitoringStack(Stack):

	def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
		super().__init__(scope, construct_id, **kwargs)

		# Dashboard
		dashboard = cw.Dashboard(self, "RefVisionMonitoringDashboard")

		# Metrics
		namespace = "RefVision"

		metrics = {
			"RefereeConnections": cw.Metric(namespace=namespace, metric_name="RefereeConnections", period=Duration.minutes(1)),
			"DecisionLatencyMs": cw.Metric(namespace=namespace, metric_name="DecisionLatencyMs", statistic="Average", period=Duration.minutes(1)),
			"BroadcastQueueDepth": cw.Metric(namespace=namespace, metric_name="BroadcastQueueDepth", period=Duration.minutes(1)),
		}

		# Widgets
		dashboard.add_widgets(
			cw.GraphWidget(
				title="Referee Connections",
				left=[metrics["RefereeConnections"]]
			),
			cw.GraphWidget(
				title="Decision Latency (ms)",
				left=[metrics["DecisionLatencyMs"]]
			),
			cw.GraphWidget(
				title="Broadcast Queue Depth",
				left=[metrics["BroadcastQueueDepth"]]
			),
		)

		# Alarms
		conn_alarm = metrics["RefereeConnections"].create_alarm(
			self, "LowRefConnections",
			threshold=3,
			evaluation_periods=1,
			comparison_operator=cw.ComparisonOperator.LESS_THAN_THRESHOLD
		)

		latency_alarm = metrics["DecisionLatencyMs"].create_alarm(
			self, "HighLatency",
			threshold=1500,
			evaluation_periods=1,
			comparison_operator=cw.ComparisonOperator.GREATER_THAN_THRESHOLD
		)

		backlog_alarm = metrics["BroadcastQueueDepth"].create_alarm(
			self, "HighBacklog",
			threshold=10,
			evaluation_periods=1,
			comparison_operator=cw.ComparisonOperator.GREATER_THAN_THRESHOLD
		)

		# Notification system
		topic = sns.Topic(self, "RefVisionAlertTopic")

		# Example email subscription
		topic.add_subscription(subs.EmailSubscription("your@email.com"))

		# To send to Slack:
		# 1. Create an HTTPS endpoint (API Gateway + Lambda or Slack Incoming Webhook)
		# 2. Add it using: topic.add_subscription(subs.UrlSubscription("https://hooks.slack.com/services/..."))

		# Bind alarms to notifications
		for alarm in [conn_alarm, latency_alarm, backlog_alarm]:
			alarm.add_alarm_action(cw_actions.SnsAction(topic))
