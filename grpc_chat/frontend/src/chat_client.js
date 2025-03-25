import { ChatServiceClient } from "./proto_gen/chat_grpc_web_pb.js"; // ✅ Sekarang menggunakan `import` ESM
import * as chatMessages from "./proto_gen/chat_pb.js"; // ✅ Masih bisa menggunakan `*` karena ini hasil ESM


// Inisialisasi client gRPC-Web
const client = new ChatServiceClient("http://localhost:8080", null, null);

// Fungsi untuk login user
export function loginUser(username, callback) {
  const request = new chatMessages.LoginRequest(); // ✅ Tidak perlu `proto.chat`
  request.setUsername(username);

  client.login(request, {}, (err, response) => {
    if (err) {
      console.error("Login failed:", err.message);
      return;
    }
    callback(response.getMessage());
  });
}

// Fungsi untuk mengirim pesan chat
export function sendMessage(username, message, callback) {
  const request = new chatMessages.MessageRequest(); // ✅ Tidak perlu `proto.chat`
  request.setUsername(username);
  request.setMessage(message);

  client.sendMessage(request, {}, (err, response) => { // ✅ Pastikan metode sesuai `chat.proto`
    if (err) {
      console.error("Error sending message:", err.message);
      return;
    }
    callback(response.getMessage());
  });
}
