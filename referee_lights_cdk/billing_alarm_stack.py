# billing_alarm_stack.py
from aws_cdk import (
	Stack,
	aws_cloudwatch as cloudwatch,
	aws_cloudwatch_actions as actions,
	aws_sns as sns,
	aws_sns_subscriptions as sns_subs,
	aws_chatbot as chatbot,
	aws_logs as logs,
	Duration
)
from constructs import Construct

class BillingAlarmStack(Stack):
	def __init__(self, scope: Construct, construct_id: str, **kwargs):
		super().__init__(scope, construct_id, **kwargs)

		# -------------------------------------------------------
		# 1) Create a Billing Alarm that triggers at USD $50
		#    Must be in us-east-1 to see AWS/Billing metrics
		# -------------------------------------------------------
		billing_alarm = cloudwatch.Alarm(
			self,
			"BillingAlarm",
			metric=cloudwatch.Metric(
				namespace="AWS/Billing",
				metric_name="EstimatedCharges",
				dimensions_map={"Currency": "USD"},
				period=Duration.hours(6),
			),
			evaluation_periods=1,
			threshold=50,  # Adjust as you like
			comparison_operator=cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
			alarm_description="Triggers if monthly AWS charges exceed $50",
		)

		# -------------------------------------------------------
		# 2) SNS Topic for Billing Alerts
		# -------------------------------------------------------
		billing_alerts_topic = sns.Topic(
			self,
			"BillingAlertsTopic",
			display_name="RefereeLightsBillingAlerts",
			topic_name="referee-lights-billing-alerts"
		)

		# Add an email subscription
		billing_alerts_topic.add_subscription(
			sns_subs.EmailSubscription("michael.kenneth.kingston@gmail.com")
		)

		# Attach SNS action to the alarm
		billing_alarm.add_alarm_action(actions.SnsAction(billing_alerts_topic))

		# -------------------------------------------------------
		# 3) Slack notification via AWS Chatbot
		# -------------------------------------------------------
		# Make sure you have already authorized your Slack workspace
		# in the AWS Chatbot console
		slack_channel = chatbot.SlackChannelConfiguration(
			self,
			"SlackChannelConfig",
			slack_channel_configuration_name="RefereeLightsSlackChannel",
			slack_workspace_id="T046HLUH064",
			slack_channel_id="C045R1QDBMK",
			notification_topics=[billing_alerts_topic],
			log_retention=logs.RetentionDays.ONE_WEEK
		)
