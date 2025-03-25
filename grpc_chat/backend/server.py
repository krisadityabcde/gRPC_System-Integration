import os
import sys

# Tambahkan proto_gen ke path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "proto_gen/")))

import grpc
import time
from concurrent import futures
from firebase_config import db, messages_ref, users_ref
import chat_pb2 # type: ignore
import chat_pb2_grpc # type: ignore

active_users = set()  # Menyimpan daftar user online

class ChatService(chat_pb2_grpc.ChatServiceServicer):

    def Login(self, request, context):
        active_users.add(request.username)
        users_ref.document(request.username).set({"status": "online"})
        print(f"User {request.username} logged in.")
        return chat_pb2.LoginResponse(success=True, message=f"Welcome {request.username}!")

    def GetOnlineUsers(self, request, context):
        while True:
            yield chat_pb2.OnlineUsersResponse(users=list(active_users))
            time.sleep(5)

    def UserTyping(self, request_iterator, context):
        for msg in request_iterator:
            print(f"{msg.username} sedang mengetik...")
            time.sleep(1)
        return chat_pb2.TypingResponse(message=f"{msg.username} berhenti mengetik.")

    def ChatStream(self, request_iterator, context):
        for msg in request_iterator:
            print(f"[{msg.username}] {msg.message}")

            # Simpan pesan ke Firebase Firestore
            messages_ref.add({"username": msg.username, "message": msg.message, "timestamp": time.time()})

            yield chat_pb2.MessageResponse(username=msg.username, message=msg.message)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    chat_pb2_grpc.add_ChatServiceServicer_to_server(ChatService(), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    print("Server berjalan di port 50051...")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
