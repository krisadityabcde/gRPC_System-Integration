package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

var client pb.ChatServiceClient
var clientPort int
var messageChannels []chan *pb.ChatMessage
var baseDir string
var loggedInUsers = make(map[string]bool)                          // Track logged in users by their IP
var usernames = make(map[string]string)                            // Maps IP to username
var userStreams = make(map[string]pb.ChatService_ChatStreamClient) // Each user gets their own stream

// Global variables - add a map for tracking active users across clients
var activeUserTimestamps = make(map[string]time.Time) // Track last seen times for users
var activeUsersMutex sync.RWMutex                     // Mutex for thread-safe access

// Global variables for tracking connections
// Removed redundant declaration of clientMessageChannels

// Add a mutex for the processedMessages map to prevent concurrent access
var (
	processedMessages = make(map[string]bool) // Track message IDs to avoid duplicates
	processMutex      sync.Mutex              // Mutex to protect processedMessages map
)

// Generate a unique ID for messages
func generateMessageID(sender, message, timestamp string) string {
	return fmt.Sprintf("%s_%s_%s", sender, message, timestamp)
}

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
	// Don't store in global variable, just get from request
	loginUsername := r.URL.Query().Get("username")
	clientIP := r.RemoteAddr

	log.Printf("Login attempt from %s with username: %s", clientIP, loginUsername)

	resp, err := client.Login(context.Background(), &pb.LoginRequest{Username: loginUsername})
	if err != nil {
		log.Printf("Login failed for %s: %v", loginUsername, err)
		http.Error(w, "Login gagal", http.StatusUnauthorized)
		return
	}

	// Create a dedicated stream for this client
	newStream, err := client.ChatStream(context.Background())
	if err != nil {
		log.Printf("Failed to create chat stream: %v", err)
		http.Error(w, "Tidak bisa streaming chat", http.StatusInternalServerError)
		return
	}

	// Store the stream for this specific client
	userStreams[clientIP] = newStream

	// Store username with IP for this client
	usernames[clientIP] = loginUsername

	// Kirim pesan bahwa user bergabung
	err = newStream.Send(&pb.ChatMessage{Sender: loginUsername, Message: "joined the chat", Timestamp: time.Now().Format("15:04:05")})
	if err != nil {
		log.Printf("Failed to send join message: %v", err)
		http.Error(w, "Gagal mengirim pesan ke server", http.StatusInternalServerError)
		return
	}

	// Mark user as logged in
	loggedInUsers[clientIP] = true

	// Set a cookie to track login state
	cookie := &http.Cookie{
		Name:     "username",
		Value:    loginUsername,
		Path:     "/",
		HttpOnly: false,
		MaxAge:   3600 * 24, // 1 day
	}
	http.SetCookie(w, cookie)

	log.Printf("User %s logged in successfully from %s", loginUsername, clientIP)
	log.Printf("Current logged in users: %v", usernames)

	// Update active users when someone logs in
	activeUsersMutex.Lock()
	activeUserTimestamps[loginUsername] = time.Now()
	activeUsersMutex.Unlock()

	// Jalankan goroutine untuk menerima pesan dari this user's stream
	go receiveMessagesForUser(clientIP, newStream)

	// Return with redirect flag
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"username":"%s","message":"%s","redirect":true}`, resp.Username, resp.Message)
}

// New function for message deduplication with thread safety
func isMessageDuplicate(msg *pb.ChatMessage) bool {
	// Generate a unique message ID
	messageID := generateMessageID(msg.Sender, msg.Message, msg.Timestamp)

	// Use mutex to protect map access
	processMutex.Lock()
	defer processMutex.Unlock()

	// Check if we've seen this message before
	if _, exists := processedMessages[messageID]; exists {
		log.Printf("Duplicate message detected: %s", messageID)
		return true
	}

	// Mark as processed
	processedMessages[messageID] = true

	// Clean up old entries periodically
	if len(processedMessages) > 5000 {
		// Create a fresh map with only the last 1000 messages
		newProcessed := make(map[string]bool)
		count := 0

		// Copy the newest entries (assuming the map is accessed in insertion order)
		for id, val := range processedMessages {
			if count >= 4000 { // Skip the oldest 4000
				newProcessed[id] = val
			}
			count++
		}

		processedMessages = newProcessed
	}

	return false
}

// New function to handle receiving messages for a specific user
func receiveMessagesForUser(clientIP string, stream pb.ChatService_ChatStreamClient) {
	log.Printf("Starting message receiver for client %s", clientIP)

	for {
		// Check if client is still logged in
		if !loggedInUsers[clientIP] {
			log.Printf("Client %s no longer logged in, stopping message receiver", clientIP)
			return
		}

		// Receive message from server
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving message for client %s: %v", clientIP, err)
			return
		}

		// Skip duplicate messages
		if isMessageDuplicate(msg) {
			continue
		}

		log.Printf("RECEIVED FROM SERVER: [%s] %s: %s", msg.Timestamp, msg.Sender, msg.Message)

		// Special handling for "left the chat" messages - remove from active users
		if msg.Message == "left the chat" {
			activeUsersMutex.Lock()
			delete(activeUserTimestamps, msg.Sender)
			log.Printf("Removed user %s from active users due to 'left the chat' message", msg.Sender)
			activeUsersMutex.Unlock()
		} else if msg.Message == "left the chat (client shutdown)" {
			activeUsersMutex.Lock()
			delete(activeUserTimestamps, msg.Sender)
			log.Printf("Removed user %s from active users due to 'left the chat' message", msg.Sender)
			activeUsersMutex.Unlock()
		} else {
			// Regular message, update active users
			updateActiveUsersFromMessages(msg)
		}

		// Broadcast to active message channels
		broadcastCount := 0
		for _, ch := range messageChannels {
			select {
			case ch <- msg:
				broadcastCount++
			default:
				log.Printf("Channel buffer full, skipping")
			}
		}
		log.Printf("Message broadcast to %d channels", broadcastCount)
	}
}

// Replace existing receiveMessages function
func receiveMessages() {
	// This function is deprecated, log a warning if it's called
	log.Println("WARNING: Legacy receiveMessages() function called, should use receiveMessagesForUser instead")
}

// Handler untuk mengirim pesan
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("message")
	clientIP := r.RemoteAddr

	// Log that we're handling a message
	log.Printf("Handling message request from %s", clientIP)

	// Get correct username for this specific client
	sender, ok := usernames[clientIP]
	if !ok {
		// Get from cookie as fallback
		cookie, err := r.Cookie("username")
		if err == nil && cookie != nil {
			sender = cookie.Value
			// Update the mapping for next time
			usernames[clientIP] = sender
		} else {
			// Return an error if we can't identify the user
			log.Printf("Could not identify user for %s", clientIP)
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}
	}

	log.Printf("Message from %s (IP: %s): %s", sender, clientIP, msg)

	// Get the stream for this client
	stream, ok := userStreams[clientIP]
	if !ok || stream == nil {
		// Create a new stream if needed
		var err error
		stream, err = client.ChatStream(context.Background())
		if err != nil {
			log.Printf("Failed to create new stream for %s: %v", clientIP, err)
			http.Error(w, "Chat service unavailable", http.StatusServiceUnavailable)
			return
		}
		userStreams[clientIP] = stream
		// Start a receiver for this new stream
		go receiveMessagesForUser(clientIP, stream)
	}

	timestamp := time.Now().Format("15:04:05")
	chatMessage := &pb.ChatMessage{
		Sender:    sender,
		Message:   msg,
		Timestamp: timestamp,
	}

	// Generate message ID for deduplication with proper locking
	messageID := generateMessageID(sender, msg, timestamp)

	// Use mutex to protect map access
	processMutex.Lock()
	processedMessages[messageID] = true
	processMutex.Unlock()

	// Send the message to the server
	err := stream.Send(chatMessage)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// IMPORTANT: Also deliver the message locally to ensure the sender sees it
	// But don't mark it as a duplicate since we want it to be received from the server too
	go func() {
		// Add slight delay to let the local UI update first
		time.Sleep(50 * time.Millisecond)

		// Broadcast directly to the sender's channel
		if senderChannel, exists := clientMessageChannels[clientIP]; exists && senderChannel != nil {
			select {
			case senderChannel <- chatMessage:
				log.Printf("Local message echo sent to sender %s", sender)
			default:
				log.Printf("Failed to send local echo to sender %s", sender)
			}
		}
	}()

	log.Printf("Message sent successfully from %s with timestamp %s", sender, timestamp)

	// Mark this user as active when they send a message
	activeUsersMutex.Lock()
	activeUserTimestamps[sender] = time.Now()
	activeUsersMutex.Unlock()

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprintf(w, `{"success":true,"sender":"%s","timestamp":"%s"}`, sender, timestamp)
}

// Ensure each client gets a unique message channel to prevent duplicates
var clientMessageChannels = make(map[string]chan *pb.ChatMessage)

// Streaming ke UI untuk semua client
func streamMessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Set necessary headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering if using nginx

	// Force immediate flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	// Log client connection
	clientIP := r.RemoteAddr
	log.Printf("New SSE connection from %s", clientIP)

	// Check if this client already has a channel and close the old one
	if existingChannel, found := clientMessageChannels[clientIP]; found {
		close(existingChannel)
		// Remove from global channels array if it exists
		for i, ch := range messageChannels {
			if ch == existingChannel {
				messageChannels = append(messageChannels[:i], messageChannels[i+1:]...)
				break
			}
		}
	}

	// Create new message channel for this client
	msgChannel := make(chan *pb.ChatMessage, 100)
	messageChannels = append(messageChannels, msgChannel)
	clientMessageChannels[clientIP] = msgChannel

	// Test message directly to browser
	fmt.Fprintf(w, "data: <System> Connection established\n\n")
	flusher.Flush()
	log.Printf("Test message sent directly to %s", clientIP)

	// Immediately send another test message after a small delay
	time.Sleep(200 * time.Millisecond)
	fmt.Fprintf(w, "data: <System> Chat session ready\n\n")
	flusher.Flush()

	// Only send a single welcome message
	fmt.Fprintf(w, "data: <System> Welcome to chat\n\n")
	flusher.Flush()

	// Create a cleanup handler that properly handles errors
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in streamMessagesHandler: %v", r)
		}

		log.Printf("Client %s disconnected from SSE", clientIP)

		// Safe cleanup with mutex protection
		delete(clientMessageChannels, clientIP)

		// Find and remove the message channel safely
		for i, ch := range messageChannels {
			if ch == msgChannel {
				// Use safe slice operation
				if i < len(messageChannels)-1 {
					messageChannels = append(messageChannels[:i], messageChannels[i+1:]...)
				} else {
					messageChannels = messageChannels[:i]
				}
				break
			}
		}

		close(msgChannel)
	}()

	// Keep-alive ticker with longer interval (reduce frequency of keep-alive messages)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Process messages as they arrive
	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client %s connection closed by browser", clientIP)
			return
		case <-ticker.C:
			// Send an empty comment as keep-alive
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-msgChannel:
			if !ok {
				log.Printf("Message channel closed for %s", clientIP)
				return
			}

			// Only display non-system messages or important system messages
			if msg.Sender == "System" && strings.Contains(msg.Message, "Ping") {
				continue // Skip system ping messages
			}

			// Format as SSE message and send
			chatMsg := fmt.Sprintf("data: <%s> %s\n\n", msg.Sender, msg.Message)
			log.Printf("Sending to client %s: %s", clientIP, chatMsg)

			_, err := fmt.Fprint(w, chatMsg)
			if err != nil {
				log.Printf("Error sending message to %s: %v", clientIP, err)
				return
			}
			flusher.Flush()
			log.Printf("Message flushed to %s", clientIP)
		}
	}
}

// Add a logout handler to properly handle user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	loggedInUsername := usernames[clientIP]

	// Close the gRPC stream for this client if it exists
	if stream, ok := userStreams[clientIP]; ok && loggedInUsername != "" {
		// Try to send a leave message - this will trigger removal from active users
		log.Printf("Sending leave message for user %s", loggedInUsername)
		leaveMsg := &pb.ChatMessage{
			Sender:    loggedInUsername,
			Message:   "left the chat",
			Timestamp: time.Now().Format("15:04:05"),
		}

		err := stream.Send(leaveMsg)
		if err != nil {
			log.Printf("Error sending leave message: %v", err)
		}

		// Remove from active users tracking
		activeUsersMutex.Lock()
		delete(activeUserTimestamps, loggedInUsername)
		activeUsersMutex.Unlock()

		// Make sure we process the leave message before closing connection
		time.Sleep(100 * time.Millisecond)
	}

	wasLoggedIn := loggedInUsers[clientIP]

	// Remove user from all maps
	delete(loggedInUsers, clientIP)
	delete(usernames, clientIP)
	delete(userStreams, clientIP)

	// Clear the cookie
	cookie := &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	log.Printf("Logout for client %s (%s), was logged in: %v", loggedInUsername, clientIP, wasLoggedIn)

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

// Add an endpoint to get active users - completely revise this function
func activeUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Use mutex for thread safety
	activeUsersMutex.RLock()
	defer activeUsersMutex.RUnlock()

	// Create an array to hold active users
	var activeUsers []string

	// Get current time for timestamp comparison
	now := time.Now()

	// Add all users that were active in the last 2 minutes
	// This is more reliable than using the loggedInUsers map
	for username, lastSeen := range activeUserTimestamps {
		if now.Sub(lastSeen) < 2*time.Minute {
			activeUsers = append(activeUsers, username)
		}
	}

	// Generate JSON response with all active users
	w.Header().Set("Content-Type", "application/json")
	if len(activeUsers) > 0 {
		fmt.Fprintf(w, `{"users":["%s"]}`, strings.Join(activeUsers, `","`))
	} else {
		fmt.Fprintf(w, `{"users":[]}`)
	}
}

// Add a periodic task to update all users seen in messages
func updateActiveUsersFromMessages(msg *pb.ChatMessage) {
	if msg != nil && msg.Sender != "" && msg.Sender != "System" {
		activeUsersMutex.Lock()
		activeUserTimestamps[msg.Sender] = time.Now()
		activeUsersMutex.Unlock()
	}
}

// Decrement the client counter when this client exits
func decrementClientCount() {
	counterFile := filepath.Join(baseDir, "client_count.txt")

	// Use file lock to ensure atomic operation
	file, err := os.OpenFile(counterFile, os.O_RDWR, 0644)
	if err != nil {
		log.Printf("Error opening client count file: %v", err)
		return
	}
	defer file.Close()

	// Read current count
	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading client count: %v", err)
		return
	}

	// Parse count
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		log.Printf("Error parsing client count: %v", err)
		return
	}

	// Decrement count, ensuring it doesn't go below 0
	if count > 0 {
		count--
	}

	// Write back to file
	if _, err := file.Seek(0, 0); err != nil {
		log.Printf("Error seeking file: %v", err)
		return
	}

	if err := file.Truncate(0); err != nil {
		log.Printf("Error truncating file: %v", err)
		return
	}

	if _, err := file.WriteString(strconv.Itoa(count)); err != nil {
		log.Printf("Error writing decremented count: %v", err)
		return
	}

	log.Printf("Client count decremented to %d", count)
}

func main() {
	// Set the base directory for all file operations
	baseDir = getClientDir()
	log.Printf("Using base directory: %s", baseDir)

	// Setup signal handling before doing anything else
	setupSignalHandling()

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
	http.HandleFunc("/cleanup", cleanupHandler)          // Add cleanup handler

	// Start the active users cleaner goroutine
	startActiveUsersCleaner()

	log.Printf("Client berjalan di http://localhost:%d", clientPort)
	http.ListenAndServe(fmt.Sprintf(":%d", clientPort), nil)
}

// Setup signal handling for graceful shutdown
func setupSignalHandling() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		log.Printf("Received signal %v, performing cleanup...", sig)

		// Decrement client count
		decrementClientCount()

		// Perform other cleanup like sending "left the chat" messages
		for clientIP, username := range usernames {
			if stream, ok := userStreams[clientIP]; ok && username != "" {
				log.Printf("Sending leave message for user %s", username)
				stream.Send(&pb.ChatMessage{
					Sender:    username,
					Message:   "left the chat (client shutdown)",
					Timestamp: time.Now().Format("15:04:05"),
				})
			}
		}

		// Allow some time for the leave messages to be sent
		time.Sleep(100 * time.Millisecond)

		log.Println("Cleanup complete, exiting...")
		os.Exit(0)
	}()
}

// Add a cleanup endpoint handler
func cleanupHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Cleanup handler called - decrementing client count")
	decrementClientCount()
	w.WriteHeader(http.StatusOK)
}

// Add this function at the end of main()
func startActiveUsersCleaner() {
	// Every 30 seconds, clean up old users
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			now := time.Now()
			activeUsersMutex.Lock()

			// Remove users who haven't been seen in 2 minutes
			for username, lastSeen := range activeUserTimestamps {
				if now.Sub(lastSeen) > 2*time.Minute {
					delete(activeUserTimestamps, username)
					log.Printf("Removed inactive user: %s", username)
				}
			}
			activeUsersMutex.Unlock()
		}
	}()
}
