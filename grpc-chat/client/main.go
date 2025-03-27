package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

var client pb.ChatServiceClient
var username string
var stream pb.ChatService_ChatStreamClient
var clientPort int
var messageChannels []chan *pb.ChatMessage
var baseDir string
var loggedInUsers = make(map[string]bool) // Track logged in users by their IP
var usernames = make(map[string]string)   // Maps IP to username

// Ambil port unik untuk client baru
func getNextClientPort() int {
	counterFile := filepath.Join(baseDir, "client_count.txt")
	data, err := os.ReadFile(counterFile)
	if err != nil {
		os.WriteFile(counterFile, []byte("0"), 0644)
		return 8080
	}

	count, _ := strconv.Atoi(string(data))
	nextPort := 8080 + count

	os.WriteFile(counterFile, []byte(strconv.Itoa(count+1)), 0644)

	return nextPort
}

// Get the directory of the client code
func getClientDir() string {
	// Get the directory of the current file (main.go)
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		return filepath.Dir(filename)
	}

	// Fallback: Try to detect from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Check if we're in the client directory
	if filepath.Base(cwd) == "client" {
		return cwd
	}

	// Check if we're in the parent directory
	possibleClientDir := filepath.Join(cwd, "client")
	if _, err := os.Stat(possibleClientDir); !os.IsNotExist(err) {
		return possibleClientDir
	}

	return cwd
}

// Tampilkan UI
func renderHTML(w http.ResponseWriter, r *http.Request) {
	// Get client IP or other unique identifier
	clientIP := r.RemoteAddr

	// Check for cookie-based authentication instead of IP
	cookie, err := r.Cookie("username")
	isLoggedIn := loggedInUsers[clientIP] && err == nil && cookie != nil

	log.Printf("Checking login status for client %s: %v (Cookie: %v)", clientIP, isLoggedIn, cookie != nil)

	// Determine which template to use
	var tmplFile string
	if isLoggedIn {
		// User is logged in, show chat interface
		tmplFile = filepath.Join(baseDir, "static", "chat.html")
		// Verify chat.html exists
		if _, err := os.Stat(tmplFile); os.IsNotExist(err) {
			log.Printf("ERROR: chat.html does not exist at %s", tmplFile)
			tmplFile = filepath.Join(baseDir, "static", "index.html")
		} else {
			log.Printf("Using chat.html for logged in user at %s", clientIP)
		}
	} else {
		// User is not logged in, show login interface
		tmplFile = filepath.Join(baseDir, "static", "index.html")
		log.Printf("Using index.html for non-logged in user at %s", clientIP)
	}

	// Force browser to not cache the page to ensure template changes are applied
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	log.Printf("Final template decision for %s: %s", clientIP, tmplFile)

	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		log.Printf("Error parsing template %s: %v", tmplFile, err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, map[string]interface{}{
		"Port":         clientPort,
		"TemplatePath": tmplFile,
		"IsLoggedIn":   isLoggedIn,
	})
}

// Handler login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	username = r.URL.Query().Get("username")
	clientIP := r.RemoteAddr

	log.Printf("Login attempt from %s with username: %s", clientIP, username)

	resp, err := client.Login(context.Background(), &pb.LoginRequest{Username: username})
	if err != nil {
		log.Printf("Login failed for %s: %v", username, err)
		http.Error(w, "Login gagal", http.StatusUnauthorized)
		return
	}

	// Buat stream untuk chat jika belum ada
	if stream == nil {
		stream, err = client.ChatStream(context.Background())
		if err != nil {
			log.Printf("Failed to create chat stream: %v", err)
			http.Error(w, "Tidak bisa streaming chat", http.StatusInternalServerError)
			return
		}
	}

	// Kirim pesan bahwa user bergabung
	err = stream.Send(&pb.ChatMessage{Sender: username, Message: "joined the chat", Timestamp: time.Now().Format("15:04:05")})
	if err != nil {
		log.Printf("Failed to send join message: %v", err)
		http.Error(w, "Gagal mengirim pesan ke server", http.StatusInternalServerError)
		return
	}

	// Mark user as logged in and store username
	loggedInUsers[clientIP] = true
	usernames[clientIP] = username

	// Set a cookie to track login state
	cookie := &http.Cookie{
		Name:     "username",
		Value:    username,
		Path:     "/",
		HttpOnly: false,
		MaxAge:   3600 * 24, // 1 day
	}
	http.SetCookie(w, cookie)

	log.Printf("User %s logged in successfully from %s", username, clientIP)
	log.Printf("Current logged in users: %v", loggedInUsers)

	// Jalankan goroutine untuk menerima pesan dari server
	go receiveMessages()

	// Return with redirect flag
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"username":"%s","message":"%s","redirect":true}`, resp.Username, resp.Message)
}

// Handler untuk mengirim pesan
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("message")
	clientIP := r.RemoteAddr

	// Get username from cookie
	var sender string
	cookie, err := r.Cookie("username")
	if err == nil && cookie != nil {
		sender = cookie.Value
	} else {
		// Fallback to global username if cookie not found
		sender = username
	}

	log.Printf("Message from %s (IP: %s): %s", sender, clientIP, msg)

	if stream == nil {
		http.Error(w, "Anda belum login", http.StatusUnauthorized)
		return
	}

	chatMessage := &pb.ChatMessage{
		Sender:    sender,
		Message:   msg,
		Timestamp: time.Now().Format("15:04:05"),
	}

	// Kirim pesan ke server
	err = stream.Send(chatMessage)
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

	// Format has been changed to use <username> message format
	// Kirim pesan yang diterima ke client ini
	for msg := range msgChannel {
		fmt.Fprintf(w, "data: <%s> %s\n\n", msg.Sender, msg.Message)
		flusher.Flush()
	}
}

// Add a logout handler to properly handle user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr

	wasLoggedIn := loggedInUsers[clientIP]

	// Remove user from logged in users and usernames maps
	delete(loggedInUsers, clientIP)
	delete(usernames, clientIP)

	// Clear the cookie
	cookie := &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	log.Printf("Logout for client %s, was logged in: %v", clientIP, wasLoggedIn)
	log.Printf("Current logged in users after logout: %v", loggedInUsers)

	// Return success response with cache control to prevent browser caching
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	fmt.Fprintf(w, `{"success":true}`)
}

// Add a check-session handler
func checkSessionHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	isLoggedIn := loggedInUsers[clientIP]

	log.Printf("Session check for %s: logged in = %v", clientIP, isLoggedIn)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprintf(w, `{"loggedIn":%t}`, isLoggedIn)
}

// Add an endpoint to get active users
func activeUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Create a list of usernames from logged in users
	var users []string
	for ip, isLoggedIn := range loggedInUsers {
		if isLoggedIn {
			// Use the actual username instead of IP-based identifier
			if username, ok := usernames[ip]; ok && username != "" {
				users = append(users, username)
			} else {
				// Fallback in case username isn't stored
				users = append(users, "Anonymous")
			}
		}
	}

	// Remove duplicates from users list
	uniqueUsers := make([]string, 0, len(users))
	seen := make(map[string]bool)

	for _, user := range users {
		if !seen[user] {
			seen[user] = true
			uniqueUsers = append(uniqueUsers, user)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(uniqueUsers) > 0 {
		fmt.Fprintf(w, `{"users":["%s"]}`, strings.Join(uniqueUsers, `","`))
	} else {
		fmt.Fprintf(w, `{"users":[]}`)
	}
}

func main() {
	// Set the base directory for all file operations
	baseDir = getClientDir()
	log.Printf("Using base directory: %s", baseDir)

	clientPort = getNextClientPort()
	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
	client = pb.NewChatServiceClient(conn)

	// Serve static files with absolute path
	staticDir := filepath.Join(baseDir, "static")
	log.Printf("Serving static files from: %s", staticDir)

	// Create a custom file server that logs requests
	fs := http.FileServer(http.Dir(staticDir))
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Log each static file request
		log.Printf("Static file requested: %s", r.URL.Path)

		// For CSS files, explicitly set the content type
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		}

		// Remove /static/ prefix and serve the file
		http.StripPrefix("/static/", fs).ServeHTTP(w, r)
	})

	http.HandleFunc("/", renderHTML)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler) // Add logout handler
	http.HandleFunc("/send", sendMessageHandler)
	http.HandleFunc("/stream", streamMessagesHandler)
	http.HandleFunc("/check-session", checkSessionHandler)
	http.HandleFunc("/active-users", activeUsersHandler) // Add endpoint for active users

	log.Printf("Client berjalan di http://localhost:%d", clientPort)
	http.ListenAndServe(fmt.Sprintf(":%d", clientPort), nil)
}
