"""
Hash credentials for meet admins (including secondary and superuser)
"""
import bcrypt
import json

def hash_password(password):
	"""
	Hash a plaintext password using bcrypt
	"""
	return bcrypt.hashpw(password.encode(), bcrypt.gensalt()).decode()

# Load JSON file
json_path = "../config/meet_creds_2.json"
output_path = "../config/meet_creds.json"

with open(json_path, "r", encoding="utf-8") as f:
	data = json.load(f)

# Ensure "meets" is present
if "meets" not in data or not isinstance(data["meets"], list):
	raise ValueError("Invalid JSON format: missing or incorrect 'meets' array")

# Hash all admin and secondary admin passwords
for meet in data["meets"]:
	admin = meet.get("admin", {})
	if isinstance(admin, dict) and isinstance(admin.get("password"), str):
		if not admin["password"].startswith("$2b$12$"):
			admin["password"] = hash_password(admin["password"])

	# Handle secondaryAdmins
	for sa in meet.get("secondaryAdmins", []):
		if isinstance(sa, dict) and isinstance(sa.get("password"), str):
			if not sa["password"].startswith("$2b$12$"):
				sa["password"] = hash_password(sa["password"])

# Hash superuser if present
superuser = data.get("superuser")
if isinstance(superuser, dict) and isinstance(superuser.get("password"), str):
	if not superuser["password"].startswith("$2b$12$"):
		superuser["password"] = hash_password(superuser["password"])

# Save updated JSON
with open(output_path, "w", encoding="utf-8") as f:
	json.dump(data, f, indent=4)

print("✅ All passwords hashed and saved to meet_creds.json")
