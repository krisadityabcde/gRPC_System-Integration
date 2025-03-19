import grpc
import chat_pb2
import chat_pb2_grpc
from concurrent import futures
import time

# Simpan data user yang login dan pesan dalam list sederhana
logged_users = set()
chat_messages = []

# Implementasi layanan ChatService
class ChatService(chat_pb2_grpc.ChatServiceServicer):
    # Unary RPC - User mendaftar/melakukan login
    def Login(self, request, context):
        if request.username in logged_users:
            return chat_pb2.LoginResponse(success=False, message="Username sudah digunakan.")
        
        logged_users.add(request.username)
        print(f"[Login] {request.username} telah login.")
        return chat_pb2.LoginResponse(success=True, message="Login berhasil!")

    # Server Streaming RPC - Saat user masuk chat room, server mengirim pesan terbaru
    def GetRecentMessages(self, request, context):
        print("[Server Streaming] Mengirim pesan terbaru...")
        for user, msg in chat_messages[-10:]:  # Kirim 10 pesan terbaru
            yield chat_pb2.ChatMessage(username=user, message=msg)
            time.sleep(0.5)  # Simulasi delay agar terasa streaming

    # Client Streaming RPC - User bisa mengirim beberapa pesan sebelum mendapat respons
    def SendMultipleMessages(self, request_iterator, context):
        for request in request_iterator:
            chat_messages.append((request.username, request.message))
            print(f"[Client Streaming] {request.username}: {request.message}")

        return chat_pb2.MessageResponse(status="Semua pesan berhasil dikirim!")

    # Bidirectional Streaming RPC - Chat real-time antara banyak user
    def ChatStream(self, request_iterator, context):
        print("[Bidirectional Streaming] Live chat dimulai...")
        for request in request_iterator:
            chat_messages.append((request.username, request.message))
            print(f"[Bidirectional] {request.username}: {request.message}")
            yield chat_pb2.ChatMessage(username=request.username, message=request.message)

# Menjalankan server gRPC
def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    chat_pb2_grpc.add_ChatServiceServicer_to_server(ChatService(), server)
    server.add_insecure_port("[::]:50051")
    print("Server berjalan di port 50051...")
    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
