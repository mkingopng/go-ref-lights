#!/usr/bin/env python3
"""
analyze_logs.py

Reads a single JSON or JSON Lines (JSONL) log file and summarizes:
- Log message types and their counts
- Potential error/warning patterns
- Full log timeline export

Usage:
    python analyze_logs.py --input path/to/logs.jsonl
"""
import argparse
import json
import os
import pandas as pd
from datetime import datetime

def load_logs(input_path):
	records = []
	basename = os.path.basename(input_path)

	# Determine if JSONL (one JSON object per line) or a JSON array
	with open(input_path, 'r') as f:
		first_char = f.read(1)
		f.seek(0)
		if first_char == '[':
			# JSON array
			data = json.load(f)
		else:
			# JSON Lines
			data = [json.loads(line) for line in f if line.strip()]

	for entry in data:
		# Normalize timestamp
		ts = None
		if isinstance(entry.get("timestamp"), (int, float)):
			# assume epoch ms
			ts = datetime.utcfromtimestamp(entry["timestamp"] / 1000.0)
		elif isinstance(entry.get("timestamp"), str):
			for fmt in ("%Y/%m/%d %H:%M:%S", "%Y-%m-%d %H:%M:%S"):
				try:
					ts = datetime.strptime(entry["timestamp"], fmt)
					break
				except:
					continue

		# Determine log type
		log_type = entry.get("type") or entry.get("level") or "UNKNOWN"
		message = entry.get("message", "")
		records.append({
			"timestamp": ts,
			"type": log_type,
			"message": message,
			**{k: v for k, v in entry.items() if k not in ("timestamp", "type", "message")}
		})
	return pd.DataFrame(records)

def main():
	parser = argparse.ArgumentParser(description="Analyze a JSON or JSONL log file.")
	parser.add_argument("--input", "-i", required=True, help="Path to the input JSON/JSONL log file.")
	parser.add_argument("--output-dir", "-o", default=".", help="Directory to save summary CSVs.")
	args = parser.parse_args()

	os.makedirs(args.output_dir, exist_ok=True)

	print(f"🔍 Loading logs from {args.input}...")
	df = load_logs(args.input)
	print(f"✅ Loaded {len(df)} log entries.")

	# Summary of log types
	type_counts = df["type"].fillna("UNKNOWN").value_counts().reset_index()
	type_counts.columns = ["log_type", "count"]
	types_csv = os.path.join(args.output_dir, "mtn_top_rumble_summary_log_types.csv")
	type_counts.to_csv(types_csv, index=False)
	print(f"📊 Log type counts saved to {types_csv}")

	# Extract issues
	issue_mask = df["message"].str.contains(r"error|fail|drop|unexpected", case=False, na=False)
	df_issues = df[issue_mask]
	issues_csv = os.path.join(args.output_dir, "mtn_top_rumble_log_issues_only.csv")
	df_issues.to_csv(issues_csv, index=False)
	print(f"❗ Potential issues ({len(df_issues)}) saved to {issues_csv}")

	# Full log timeline
	timeline_csv = os.path.join(args.output_dir, "mtn_top_rumble_all_logs_flat.csv")
	df.to_csv(timeline_csv, index=False)
	print(f"🕒 Full log timeline saved to {timeline_csv}")

	print("\n✅ Analysis complete.")

if __name__ == "__main__":
	main()
