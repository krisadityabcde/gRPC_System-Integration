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

// Add a global variable for the active users stream
var activeUsersStream pb.ChatService_ActiveUsersStreamClient

// Add map to track the client session IDs for more reliable identification
var clientSessions = make(map[string]string) // Maps cookie ID to clientIP
var sessionsMutex sync.RWMutex               // Mutex for thread-safe access to sessions

// Add a mutex for the processedMessages map to prevent concurrent access
var (
	processedMessages = make(map[string]bool) // Track message IDs to avoid duplicates
	processMutex      sync.Mutex              // Mutex to protect processedMessages map
)

// Generate a unique ID for messages
func generateMessageID(sender, message, timestamp string) string {
	return fmt.Sprintf("%s_%s_%s", sender, message, timestamp)
}

// Extract client identifier from request using cookies first, then IP as fallback
func getClientIdentifier(r *http.Request) string {
	// First try to get the client ID from port-specific cookie
	cookie, err := r.Cookie(fmt.Sprintf("client_id_port%d", clientPort))
	if err == nil && cookie != nil && cookie.Value != "" {
		sessionsMutex.RLock()
		clientIP, exists := clientSessions[cookie.Value]
		sessionsMutex.RUnlock()

		if exists {
			return clientIP
		}
	}

	// Fallback to IP address
	return r.RemoteAddr
}

// Create a new session ID for a client
func createClientSession(w http.ResponseWriter, clientIP string) string {
	sessionID := fmt.Sprintf("session_%d_%s", time.Now().UnixNano(), clientIP)

	// Store the mapping between session ID and client IP
	sessionsMutex.Lock()
	clientSessions[sessionID] = clientIP
	sessionsMutex.Unlock()

	// Set a cookie with the session ID - make it port-specific
	cookie := &http.Cookie{
		Name:     fmt.Sprintf("client_id_port%d", clientPort), // Include port in cookie name
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600 * 24, // 1 day
	}
	http.SetCookie(w, cookie)

	log.Printf("Created new session ID %s for client %s on port %d", sessionID, clientIP, clientPort)
	return sessionID
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
	// Get client identifier
	clientIP := getClientIdentifier(r)

	// Create a session if one doesn't exist
	_, err := r.Cookie(fmt.Sprintf("client_id_port%d", clientPort))
	if err != nil {
		createClientSession(w, clientIP)
	}

	// Check for cookie-based authentication instead of IP - use port-specific cookie
	cookie, err := r.Cookie(fmt.Sprintf("username_port%d", clientPort))
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
	clientIP := getClientIdentifier(r)

	// Create a client session if one doesn't exist
	sessionCookie, err := r.Cookie(fmt.Sprintf("client_id_port%d", clientPort))
	if err != nil || sessionCookie == nil || sessionCookie.Value == "" {
		createClientSession(w, clientIP)
	}

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

	// Set a cookie to track login state with the username - make it port-specific
	cookie := &http.Cookie{
		Name:     fmt.Sprintf("username_port%d", clientPort), // Include port in cookie name
		Value:    loginUsername,
		Path:     "/",
		HttpOnly: false,
		MaxAge:   3600 * 24, // 1 day
	}
	http.SetCookie(w, cookie)

	log.Printf("User %s logged in successfully from client %s", loginUsername, clientIP)
	log.Printf("Current logged in users: %v", usernames)

	// Start streaming active users for this client - do this first to ensure immediate display
	go startActiveUsersStream(loginUsername)

	// Brief delay to allow the active users list to be processed
	time.Sleep(100 * time.Millisecond)

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

// Handler untuk mengirim pesan
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("message")
	clientIP := getClientIdentifier(r)

	// Log that we're handling a message
	log.Printf("Handling message request from client %s", clientIP)

	// Get correct username for this specific client
	sender, ok := usernames[clientIP]
	if !ok {
		// Get from cookie as fallback - use port-specific cookie
		cookie, err := r.Cookie(fmt.Sprintf("username_port%d", clientPort))
		if err == nil && cookie != nil {
			sender = cookie.Value
			// Update the mapping for next time
			usernames[clientIP] = sender
			log.Printf("Retrieved username %s from port-specific cookie for client %s", sender, clientIP)
		} else {
			// Return an error if we can't identify the user
			log.Printf("Could not identify user for client %s", clientIP)
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}
	}

	log.Printf("Message from %s (Client: %s): %s", sender, clientIP, msg)

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

	log.Printf("Message sent successfully from %s with timestamp %s", sender, timestamp)

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

	// Get client identifier from cookie or IP
	clientIP := getClientIdentifier(r)

	// Get username for this client to verify message ownership
	clientUsername := usernames[clientIP]

	log.Printf("New SSE connection from client %s (username: %s)", clientIP, clientUsername)

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
	log.Printf("Test message sent directly to client %s", clientIP)

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
				log.Printf("Message channel closed for client %s", clientIP)
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
				log.Printf("Error sending message to client %s: %v", clientIP, err)
				return
			}
			flusher.Flush()
			log.Printf("Message flushed to client %s", clientIP)
		}
	}
}

// Add a logout handler to properly handle user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIdentifier(r)
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

		// Make sure we process the leave message before closing connection
		time.Sleep(100 * time.Millisecond)
	}

	wasLoggedIn := loggedInUsers[clientIP]

	// Remove user from all maps
	delete(loggedInUsers, clientIP)
	delete(usernames, clientIP)
	delete(userStreams, clientIP)

	// Clear the cookie - make it port-specific
	cookie := &http.Cookie{
		Name:   fmt.Sprintf("username_port%d", clientPort),
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	// Also clear the client_id cookie
	clientIdCookie := &http.Cookie{
		Name:   fmt.Sprintf("client_id_port%d", clientPort),
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, clientIdCookie)

	log.Printf("Logout for client %s (%s), was logged in: %v", loggedInUsername, clientIP, wasLoggedIn)

	// Return success response with cache control to prevent browser caching
	setStandardHeaders(w)
	fmt.Fprintf(w, `{"success":true}`)
}

// Add a check-session handler
func checkSessionHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIdentifier(r)
	isLoggedIn := loggedInUsers[clientIP]

	log.Printf("Session check for %s: logged in = %v", clientIP, isLoggedIn)

	setStandardHeaders(w)
	fmt.Fprintf(w, `{"loggedIn":%t}`, isLoggedIn)
}

// Add a periodic task to update all users seen in messages
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

// Handle for ping requests - streamline duplicate error checking pattern
func pingHandler(w http.ResponseWriter, r *http.Request) {
	// Record start time for server processing
	startTime := time.Now()

	// Get client identifier for debugging
	clientIP := getClientIdentifier(r)
	log.Printf("Ping request from client %s", clientIP)

	// Calculate processing time
	serverTime := time.Since(startTime).Milliseconds()

	// Return pong response with timing info - use a common header setting pattern
	setStandardHeaders(w)
	fmt.Fprintf(w, `{"message":"pong","serverTime":%d}`, serverTime)
}

// Extract common header setting into a function to reduce duplication
func setStandardHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// Add a function to start streaming active users
func startActiveUsersStream(username string) {
	// Close any existing stream first
	if activeUsersStream != nil {
		activeUsersStream.CloseSend()
	}

	var err error
	log.Printf("Starting active users stream for %s", username)

	// Create a stream for active users
	activeUsersStream, err = client.ActiveUsersStream(context.Background(), &pb.ActiveUsersRequest{
		Username: username,
	})

	if err != nil {
		log.Printf("Failed to create active users stream: %v", err)
		return
	}

	// Start a goroutine to receive active user updates
	go receiveActiveUserUpdates()
}

// Add a function to receive active user updates - remove duplicate stream existence check
func receiveActiveUserUpdates() {
	for {
		// Check if stream exists
		if activeUsersStream == nil {
			log.Printf("Active users stream is nil, exiting receiver")
			return
		}

		// Receive updates
		update, err := activeUsersStream.Recv()
		if err != nil {
			log.Printf("Error receiving active users update: %v", err)
			if err == io.EOF {
				// Stream closed normally
				return
			}
			// Try to reconnect after a delay
			time.Sleep(5 * time.Second)
			return
		}

		// Share code for broadcasting messages to all clients
		broadcastSystemMessage := func(msg string) {
			for _, ch := range messageChannels {
				systemMsg := &pb.ChatMessage{
					Sender:    "System",
					Message:   msg,
					Timestamp: time.Now().Format("15:04:05"),
				}

				select {
				case ch <- systemMsg:
					// Message sent successfully
				default:
					log.Printf("Channel buffer full, skipping update")
				}
			}
		}

		// Process the update based on the type
		switch update.UpdateType {
		case pb.ActiveUsersUpdate_FULL_LIST:
			// Full list of active users
			log.Printf("Received full active users list: %v", update.Users)

			// Create a copy of update.Users to avoid any potential race conditions
			usersCopy := make([]string, len(update.Users))
			copy(usersCopy, update.Users)

			// Use the shared function
			broadcastSystemMessage(fmt.Sprintf("ActiveUsersList: %s", strings.Join(usersCopy, ", ")))

		case pb.ActiveUsersUpdate_JOIN:
			// User joined
			log.Printf("User joined: %s", update.Username)
			broadcastSystemMessage(fmt.Sprintf("UserJoined: %s", update.Username))

		case pb.ActiveUsersUpdate_LEAVE:
			// User left
			log.Printf("User left: %s", update.Username)
			broadcastSystemMessage(fmt.Sprintf("UserLeft: %s", update.Username))
		}
	}
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
	http.HandleFunc("/cleanup", cleanupHandler) // Add cleanup handler
	http.HandleFunc("/ping", pingHandler)       // Add the ping handler

	// Make sure there's no active-users HTTP endpoint here

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
