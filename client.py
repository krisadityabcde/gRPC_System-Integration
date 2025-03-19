import grpc
import chat_pb2
import chat_pb2_grpc

def login():
    channel = grpc.insecure_channel("localhost:50051")
    stub = chat_pb2_grpc.ChatServiceStub(channel)

    username = input("Masukkan username: ")
    request = chat_pb2.LoginRequest(username=username)
    response = stub.Login(request)

    if response.success:
        print("Login berhasil!")
        return username
    else:
        print("Login gagal:", response.message)
        return None

def get_recent_messages():
    channel = grpc.insecure_channel("localhost:50051")
    stub = chat_pb2_grpc.ChatServiceStub(channel)

    request = chat_pb2.Empty()
    responses = stub.GetRecentMessages(request)

    print("\n=== Pesan Terbaru ===")
    for msg in responses:
        print(f"{msg.username}: {msg.message}")

def send_multiple_messages(username):
    channel = grpc.insecure_channel("localhost:50051")
    stub = chat_pb2_grpc.ChatServiceStub(channel)

    def message_stream():
        while True:
            message = input("Masukkan pesan (atau ketik 'exit' untuk selesai): ")
            if message.lower() == "exit":
                break
            yield chat_pb2.MessageRequest(username=username, message=message)

    response = stub.SendMultipleMessages(message_stream())
    print(f"Server Response: {response.status}")

def chat_stream(username):
    channel = grpc.insecure_channel("localhost:50051")
    stub = chat_pb2_grpc.ChatServiceStub(channel)

    def generate_messages():
        while True:
            message = input("Masukkan pesan (atau ketik 'exit' untuk keluar): ")
            if message.lower() == "exit":
                break
            yield chat_pb2.MessageRequest(username=username, message=message)

    responses = stub.ChatStream(generate_messages())

    print("\n=== Live Chat ===")
    for response in responses:
        print(f"{response.username}: {response.message}")

if __name__ == "__main__":
    username = login()
    if username:
        while True:
            print("\n1. Lihat Pesan Terbaru (Server Streaming)")
            print("2. Kirim Banyak Pesan Sekaligus (Client Streaming)")
            print("3. Live Chat (Bidirectional Streaming)")
            print("4. Keluar")
            choice = input("Pilih opsi: ")

            if choice == "1":
                get_recent_messages()
            elif choice == "2":
                send_multiple_messages(username)
            elif choice == "3":
                chat_stream(username)
            elif choice == "4":
                print("Keluar dari chat...")
                break
            else:
                print("Pilihan tidak valid, coba lagi.")
