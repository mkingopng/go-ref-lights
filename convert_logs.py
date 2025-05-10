#!/usr/bin/env python3
import re
import json
import sys

# Pattern for application logs
log_pattern = re.compile(
	r'(?P<level>INFO|WARN|ERROR|DEBUG):\s*'
	r'(?P<timestamp>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) '
	r'(?P<source>[^:]+):(?P<line>\d+): (?P<message>.*)'
)

# Pattern for Gin logs
gin_pattern = re.compile(
	r'\[GIN\]\s*'
	r'(?P<date>\d{4}/\d{2}/\d{2})\s*-\s*'
	r'(?P<time>\d{2}:\d{2}:\d{2})\s*\|\s*'
	r'(?P<status>\d{3})\s*\|\s*'
	r'(?P<duration>[\d\.µm]+s)\s*\|\s*'
	r'(?P<ip>[\d\.]+)\s*\|\s*'
	r'(?P<method>GET|POST|HEAD)\s*"(?P<path>[^"]+)"'
)

def parse_line(line):
	# Try application log pattern
	m = log_pattern.match(line)
	if m:
		gd = m.groupdict()
		return {
			"type": "app_log",
			"level": gd["level"],
			"timestamp": gd["timestamp"],
			"source": gd["source"],
			"line": int(gd["line"]),
			"message": gd["message"],
		}

	# Try GIN access log pattern
	m2 = gin_pattern.search(line)
	if m2:
		gd = m2.groupdict()
		return {
			"type": "gin_log",
			"timestamp": f"{gd['date']} {gd['time']}",
			"status": int(gd["status"]),
			"duration": gd["duration"],
			"ip": gd["ip"],
			"method": gd["method"],
			"path": gd["path"],
		}

	# Skip unrecognized lines
	return None

def main():
	if len(sys.argv) != 3:
		print("Usage: python convert_logs.py input.txt output.jsonl")
		sys.exit(1)

	infile, outfile = sys.argv[1], sys.argv[2]
	with open(infile, 'r') as f_in, open(outfile, 'w') as f_out:
		for line in f_in:
			parsed = parse_line(line)
			if parsed:
				f_out.write(json.dumps(parsed) + "\n")

if __name__ == "__main__":
	main()
