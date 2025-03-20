"""
Hash credentials for meet admins
"""
import bcrypt
import json

def hash_password(password):
	"""
	Hash a plaintext password using bcrypt
	:param password:
	:return:
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

# Process each meet and hash its admin password if needed
for meet in data["meets"]:
	admin = meet.get("admin", {})

	if not isinstance(admin, dict):
		raise ValueError(f"Meet '{meet.get('name', 'UNKNOWN')}' has an invalid 'admin' object")

	if "password" not in admin or not isinstance(admin["password"], str):
		raise ValueError(f"Meet '{meet['name']}' admin has an invalid password format")

	# Hash only if the password is not already hashed
	if not admin["password"].startswith("$2b$12$"):
		admin["password"] = hash_password(admin["password"])

# Save updated JSON
with open(output_path, "w", encoding="utf-8") as f:
	json.dump(data, f, indent=4)

print("✅ Passwords successfully hashed and saved in meet_creds.json!")
