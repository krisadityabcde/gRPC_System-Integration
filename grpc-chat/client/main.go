package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	pb "grpc-chat/proto"
)

var client pb.ChatServiceClient
var username string
var stream pb.ChatService_ChatStreamClient
var clientPort int
var messageChannels []chan *pb.ChatMessage

// Ambil port unik untuk client baru
func getNextClientPort() int {
	data, err := os.ReadFile("client_count.txt")
	if err != nil {
		os.WriteFile("client_count.txt", []byte("0"), 0644)
		return 8080
	}

	count, _ := strconv.Atoi(string(data))
	nextPort := 8080 + count

	os.WriteFile("client_count.txt", []byte(strconv.Itoa(count+1)), 0644)

	return nextPort
}

// Tampilkan UI
func renderHTML(w http.ResponseWriter, r *http.Request) {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Chat Room</title>
		<style>
			body { font-family: Arial, sans-serif; text-align: center; }
			#chat-box { width: 400px; height: 300px; border: 1px solid #ccc; overflow-y: scroll; padding: 10px; margin: auto; text-align: left; }
			#input-box { margin-top: 10px; }
			#input-box input { width: 300px; padding: 5px; }
			#input-box button { padding: 5px; }
		</style>
	</head>
	<body>
		<h2>Chat Room (Port: {{.Port}})</h2>
		<div id="login-box">
			<input id="username" type="text" placeholder="Enter username" />
			<button onclick="login()">Login</button>
		</div>

		<div id="chat-area" style="display:none;">
			<div id="chat-box"></div>
			<div id="input-box">
				<input id="message" type="text" placeholder="Type a message..." />
				<button onclick="sendMessage()">Send</button>
			</div>
		</div>

		<script>
			let username = "";
			let eventSource;
			let lastMessage = "";

			async function login() {
				let input = document.getElementById("username");
				let response = await fetch("/login?username=" + input.value);
				let result = await response.json();
				if (result.username) {
					username = result.username;
					document.getElementById("login-box").style.display = "none";
					document.getElementById("chat-area").style.display = "block";
					startChat();
				} else {
					alert("Login failed");
				}
			}

			function startChat() {
				eventSource = new EventSource("/stream");
				eventSource.onmessage = (event) => {
					let msg = event.data;
					if (!isMessageDuplicate(msg)) {
						addMessageToChat(msg);
					}
				};
			}

			function sendMessage() {
				let input = document.getElementById("message");
				let messageText = input.value.trim();

				if (messageText !== "") {
					let formattedMessage = "[" + getCurrentTime() + "] " + username + ": " + messageText;
					
					// Cegah duplikasi di sisi client
					if (!isMessageDuplicate(formattedMessage)) {
						addMessageToChat(formattedMessage);
					}
					
					fetch("/send?username=" + username + "&message=" + messageText);
					input.value = "";
				}
			}

			function addMessageToChat(message) {
				let chatBox = document.getElementById("chat-box");
				let messageElement = document.createElement("p");
				messageElement.textContent = message;
				chatBox.appendChild(messageElement);
				chatBox.scrollTop = chatBox.scrollHeight;
			}

			function isMessageDuplicate(message) {
				if (message === lastMessage) {
					return true;
				}
				lastMessage = message;
				return false;
			}

			function getCurrentTime() {
				let now = new Date();
				return now.getHours().toString().padStart(2, '0') + ":" +
					now.getMinutes().toString().padStart(2, '0') + ":" +
					now.getSeconds().toString().padStart(2, '0');
			}
		</script>
	</body>
	</html>`

	tmplObj := template.Must(template.New("chat").Parse(tmpl))
	tmplObj.Execute(w, map[string]int{"Port": clientPort})
}

// Handler login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	username = r.URL.Query().Get("username")

	resp, err := client.Login(context.Background(), &pb.LoginRequest{Username: username})
	if err != nil {
		http.Error(w, "Login gagal", http.StatusUnauthorized)
		return
	}

	// Buat stream untuk chat jika belum ada
	if stream == nil {
		stream, err = client.ChatStream(context.Background())
		if err != nil {
			http.Error(w, "Tidak bisa streaming chat", http.StatusInternalServerError)
			return
		}
	}

	// Kirim pesan bahwa user bergabung
	err = stream.Send(&pb.ChatMessage{Sender: username, Message: "joined the chat", Timestamp: time.Now().Format("15:04:05")})
	if err != nil {
		http.Error(w, "Gagal mengirim pesan ke server", http.StatusInternalServerError)
		return
	}

	// Jalankan goroutine untuk menerima pesan dari server
	go receiveMessages()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"username":"%s","message":"%s"}`, resp.Username, resp.Message)
}

// Handler untuk mengirim pesan
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("message")

	if stream == nil {
		http.Error(w, "Anda belum login", http.StatusUnauthorized)
		return
	}

	chatMessage := &pb.ChatMessage{
		Sender:    username,
		Message:   msg,
		Timestamp: time.Now().Format("15:04:05"),
	}

	// Kirim pesan ke server
	err := stream.Send(chatMessage)
	if err != nil {
		http.Error(w, "Gagal mengirim pesan", http.StatusInternalServerError)
		return
	}

	// Kirim pesan ke semua client
	for _, ch := range messageChannels {
		ch <- chatMessage
	}
}


// Fungsi untuk menerima pesan dari server
func receiveMessages() {
	for {
		if stream == nil {
			log.Println("Stream belum tersedia")
			return
		}

		msg, err := stream.Recv()
		if err != nil {
			log.Println("Gagal menerima pesan:", err)
			return
		}

		// Kirim ke semua client yang mendengarkan /stream
		for _, ch := range messageChannels {
			ch <- msg
		}
	}
}


// Streaming ke UI untuk semua client
func streamMessagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	// Buat channel khusus untuk client ini
	msgChannel := make(chan *pb.ChatMessage, 100)
	messageChannels = append(messageChannels, msgChannel)

	// Hapus channel saat client disconnect
	defer func() {
		for i, ch := range messageChannels {
			if ch == msgChannel {
				messageChannels = append(messageChannels[:i], messageChannels[i+1:]...)
				break
			}
		}
		close(msgChannel)
	}()

	// Kirim pesan yang diterima ke client ini
	for msg := range msgChannel {
		fmt.Fprintf(w, "data: [%s] %s: %s\n\n", msg.Timestamp, msg.Sender, msg.Message)
		flusher.Flush()
	}
}


func main() {
	clientPort = getNextClientPort()
	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
	client = pb.NewChatServiceClient(conn)

	http.HandleFunc("/", renderHTML)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/send", sendMessageHandler)
	http.HandleFunc("/stream", streamMessagesHandler)

	log.Printf("Client berjalan di http://localhost:%d", clientPort)
	http.ListenAndServe(fmt.Sprintf(":%d", clientPort), nil)
}
