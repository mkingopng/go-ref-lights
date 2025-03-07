# controllers/hash_creds.py
"""
hash credentials
"""
import bcrypt
import json

def hash_password(password):
	"""
	hash credentials
	:param password:
	:return: encrypted password
	"""
	return bcrypt.hashpw(password.encode(), bcrypt.gensalt()).decode()

# Load existing JSON
with open("./../config/meet_creds2.json") as f:
	data = json.load(f)

# Hash passwords
for meet in data["meets"]:
	for user in meet["users"]:
		if not user["password"].startswith("$2b$12$"):
			user["password"] = hash_password(user["password"])

# Save back
with open("./../config/meet_creds.json", "w") as f:
	json.dump(data, f, indent=4)

print("✅ Passwords successfully hashed!")
