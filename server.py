import grpc
import time
import chat_pb2
import chat_pb2_grpc
from concurrent import futures

# Menyimpan daftar client yang sedang terhubung
active_clients = []
recent_messages = []

class ChatService(chat_pb2_grpc.ChatServiceServicer):

    # Unary RPC: Login user
    def Login(self, request, context):
        print(f"User {request.username} logged in.")
        return chat_pb2.LoginResponse(success=True, message=f"Welcome {request.username}!")

    # Server Streaming: Mengirim recent messages
    def GetRecentMessages(self, request, context):
        for msg in recent_messages:
            yield chat_pb2.MessageResponse(username=msg["username"], message=msg["message"])
            time.sleep(1)

    # Client Streaming: Menerima broadcast chat messages
    def BroadcastChat(self, request_iterator, context):
        count = 0
        for msg in request_iterator:
            print(f"Broadcast dari {msg.username}: {msg.message}")
            count += 1
        return chat_pb2.BroadcastResponse(count=count)

    # Bidirectional Streaming: Chat real-time
    def ChatStream(self, request_iterator, context):
        global active_clients

        # Stream untuk client ini
        response_queue = []

        # Tambahkan client ke daftar aktif
        active_clients.append(response_queue)

        try:
            for msg in request_iterator:
                if msg.message.lower() == "exit":
                    print(f"{msg.username} keluar dari chat.")
                    break

                print(f"[{msg.username}] {msg.message}")

                # Kirim pesan ke semua client lain
                for queue in active_clients:
                    if queue is not response_queue:  # Jangan kirim ke pengirim sendiri
                        queue.append(chat_pb2.MessageResponse(username=msg.username, message=msg.message))

        except Exception as e:
            print(f"Error di ChatStream: {e}")

        # Hapus client dari daftar aktif setelah keluar
        active_clients.remove(response_queue)

        return chat_pb2.MessageResponse(username="Server", message="Chat session ended.")


# Menjalankan server
def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    chat_pb2_grpc.add_ChatServiceServicer_to_server(ChatService(), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    print("Server berjalan di port 50051...")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
