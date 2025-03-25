import firebase_admin
from firebase_admin import credentials, firestore

# Load Firebase credentials
cred = credentials.Certificate("grpc-system-integration-firebase-adminsdk-fbsvc-44e0212c60.json")  # Unduh dari Firebase Console
firebase_admin.initialize_app(cred)

# Koneksi ke Firestore
db = firestore.client()
messages_ref = db.collection("messages")
users_ref = db.collection("users")
