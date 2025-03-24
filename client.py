import grpc
import chat_pb2
import chat_pb2_grpc
import time

def login(stub, username):
    response = stub.Login(chat_pb2.LoginRequest(username=username))
    print(response.message)

def get_recent_messages(stub):
    print("\nRecent Messages:")
    for message in stub.GetRecentMessages(chat_pb2.Empty()):
        print(f"{message.username}: {message.message}")

def broadcast_chat(stub, username):
    print("\nKetik pesan untuk broadcast (ketik 'exit' untuk berhenti):")

    def message_generator():
        while True:
            msg = input("> ")
            if msg.lower() == "exit":
                break
            yield chat_pb2.MessageRequest(username=username, message=msg)

    response = stub.BroadcastChat(message_generator())
    print(f"Total {response.count} pesan dikirim.")

def chat_stream(stub, username):
    print("\nMulai chat real-time (ketik 'exit' untuk keluar)")

    def generate_messages():
        while True:
            msg = input("> ")
            if msg.lower() == "exit":
                yield chat_pb2.MessageRequest(username=username, message=msg)
                break
            yield chat_pb2.MessageRequest(username=username, message=msg)

    responses = stub.ChatStream(generate_messages())

    try:
        for response in responses:
            print(f"{response.username}: {response.message}")  # Terima pesan dari client lain
    except grpc._channel._Rendezvous:
        print("Chat session ended.")



if __name__ == "__main__":
    channel = grpc.insecure_channel("localhost:50051")
    stub = chat_pb2_grpc.ChatServiceStub(channel)

    username = input("Masukkan username: ")
    login(stub, username)

    while True:
        print("\n1. Get Recent Messages\n2. Broadcast Chat\n3. Real-time Chat\n4. Keluar")
        choice = input("Pilih opsi: ")

        if choice == "1":
            get_recent_messages(stub)
        elif choice == "2":
            broadcast_chat(stub, username)
        elif choice == "3":
            chat_stream(stub, username)
        elif choice == "4":
            break
