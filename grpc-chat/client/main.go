package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"context"
	"time"

	pb "grpc-chat/proto"
	"google.golang.org/grpc"
)

var tmpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chat Room</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; }
        #chat-box { width: 50%; margin: auto; height: 300px; overflow-y: scroll; border: 1px solid #ccc; padding: 10px; }
        input, button { margin-top: 10px; }
    </style>
</head>
<body>
    <h2>Chat Room</h2>
    <input type="text" id="username" placeholder="Enter your username">
    <button onclick="login()">Login</button>
    <div id="online-status"></div>
    <div id="chat-box"></div>
    <input type="text" id="message" placeholder="Type a message">
    <button onclick="sendMessage()">Send</button>

    <script>
        let username = "";

        function login() {
            username = document.getElementById("username").value;
            fetch("/login?username=" + username)
                .then(res => res.text())
                .then(msg => {
                    alert(msg);
                    startOnlineStatus();
                });
        }

        function startOnlineStatus() {
            const eventSource = new EventSource("/online-status");
            eventSource.onmessage = function(event) {
                document.getElementById("online-status").innerText = event.data;
            };
        }

        function sendMessage() {
            const message = document.getElementById("message").value;
            fetch("/send?username=" + username + "&message=" + message)
                .then(res => res.text())
                .then(msg => {
                    document.getElementById("chat-box").innerHTML += "<p><b>" + username + ":</b> " + message + "</p>";
                    document.getElementById("message").value = "";
                });
        }
    </script>
</body>
</html>
`

type Client struct {
	conn   *grpc.ClientConn
	client pb.ChatServiceClient
	mu     sync.Mutex
	users  []string
}

func NewClient() *Client {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewChatServiceClient(conn),
		users:  []string{},
	}
}

func (c *Client) LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")

	_, err := c.client.Login(context.Background(), &pb.LoginRequest{Username: username})
	if err != nil {
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}

	c.mu.Lock()
	c.users = append(c.users, username)
	c.mu.Unlock()

	fmt.Fprint(w, "Login successful!")
}

func (c *Client) ChatHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")

	// Buat stream dengan server
	stream, err := c.client.ChatStream(context.Background())
	if err != nil {
		http.Error(w, "Failed to start chat stream", http.StatusInternalServerError)
		return
	}

	// Kirim pesan dari client ke server secara asynchronous
	go func() {
		for {
			message := r.URL.Query().Get("message")
			if message == "" {
				continue
			}

			err := stream.Send(&pb.ChatMessage{
				Username:  username,
				Message:   message,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			if err != nil {
				log.Println("Error sending message:", err)
				return
			}
			time.Sleep(500 * time.Millisecond) // Hindari pengiriman terlalu cepat
		}
	}()

	// Terima pesan dari server dan kirim ke client
	for {
		in, err := stream.Recv()
		if err != nil {
			http.Error(w, "Error receiving message", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%s [%s]: %s\n", in.Username, in.Timestamp, in.Message)
	}
}

func (c *Client) OnlineStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	stream, err := c.client.OnlineStatus(context.Background(), &pb.OnlineStatusRequest{})
	if err != nil {
		http.Error(w, "Failed to start online status stream", http.StatusInternalServerError)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Fprintf(w, "data: %s is online: %v\n\n", resp.Username, resp.Online)
		w.(http.Flusher).Flush()
	}
}

func (c *Client) ServeHTML(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("chat").Parse(tmpl))
	t.Execute(w, nil)
}

func main() {
	client := NewClient()
	defer client.conn.Close()

	http.HandleFunc("/", client.ServeHTML)
	http.HandleFunc("/login", client.LoginHandler)
	http.HandleFunc("/send", client.ChatHandler)
	http.HandleFunc("/online-status", client.OnlineStatusHandler)

	fmt.Println("Client running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
