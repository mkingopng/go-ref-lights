# analyze_logs.py
"""
Reads all logs/*.json files and summarizes log message types,
error patterns, volume per log type, and timeline breakdowns.
"""
import os
import json
import pandas as pd
from glob import glob
from datetime import datetime

LOG_DIR = "logs"
all_files = sorted(glob(os.path.join(LOG_DIR, "*.json")))

print(f"🔍 Loading {len(all_files)} log files...")

records = []

for file in all_files:
	try:
		with open(file) as f:
			data = json.load(f)
			for entry in data:
				if "timestamp" in entry and "message" in entry:
					ts = datetime.utcfromtimestamp(entry["timestamp"] / 1000.0)
					msg = entry["message"]
					prefix = msg.split("]")[0][1:] if msg.startswith("[") else "UNKNOWN"
					records.append({
						"timestamp": ts,
						"type": prefix,
						"message": msg
					})
	except Exception as e:
		print(f"⚠️ Error reading {file}: {e}")

print(f"✅ Loaded {len(records)} log messages.")

df = pd.DataFrame(records)

# Count log types
type_counts = df["type"].value_counts().reset_index()
type_counts.columns = ["log_type", "count"]
type_counts.to_csv("summary_log_types.csv", index=False)
print("📊 Top log types saved to summary_log_types.csv")

# Extract and save all errors/warnings
df_error = df[df["message"].str.contains("error|fail|drop|unexpected", case=False, na=False)]
df_error.to_csv("log_issues_only.csv", index=False)
print(f"❗ Found {len(df_error)} potential issues — saved to log_issues_only.csv")

# Optional: Save all logs as CSV for timeline review
df.to_csv("all_logs_flat.csv", index=False)
print("🕒 Full log timeline saved to all_logs_flat.csv")

print("\n✅ Done. You can now open the CSVs in Excel, Pycharm, or pandas.")
