let username = "";
let eventSource;
let lastMessage = "";

// Boot sequence text
const bootText = `
Initializing system...
Loading kernel modules... OK
Checking file system integrity... OK
Starting Chat Client... OK

System ready.
`;

// Function to simulate typing effect
function typeWriter(text, element, speed, callback) {
    let i = 0;
    function type() {
        if (i < text.length) {
            element.innerHTML += text.charAt(i);
            i++;
            setTimeout(type, speed);
        } else if (callback) {
            callback();
        }
    }
    type();
}

// Boot sequence animation
window.onload = function() {
    // Check if we're already showing the chat UI (logged in)
    const chatArea = document.getElementById('chat-area');
    if (chatArea && chatArea.style.display === 'block') {
        // Skip the boot sequence when already in chat view
        document.getElementById('bootSequence').style.display = 'none';
        return;
    }
    
    const bootElement = document.getElementById('bootSequence');
    if (!bootElement) return; // Safety check
    
    // Start typing boot sequence
    typeWriter(bootText, bootElement, 1, function() {
        // After boot sequence completes, wait a moment and fade out
        setTimeout(function() {
            bootElement.style.transition = 'opacity 1s';
            bootElement.style.opacity = '0';
            
            // Remove boot sequence element after fade
            setTimeout(function() {
                bootElement.style.display = 'none';
            }, 1000);
        }, 500);
    });

    // Allow Enter key to login
    const usernameInput = document.getElementById('username');
    if (usernameInput) {
        usernameInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                login();
            }
        });
    }
    
    // Allow Enter key to send message
    const messageInput = document.getElementById('message');
    if (messageInput) {
        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    }
};

async function login() {
    let input = document.getElementById("username");
    if (!input || !input.value.trim()) {
        alert("Please enter a username");
        return;
    }
    
    let response = await fetch("/login?username=" + encodeURIComponent(input.value.trim()));
    if (!response.ok) {
        alert("Login failed. Please try again.");
        return;
    }
    
    let result = await response.json();
    if (result.username) {
        username = result.username;
        
        // Force a full page refresh to load the chat interface
        window.location.href = '/?t=' + new Date().getTime();
    } else {
        alert("Login failed");
    }
}

// Improved EventSource management
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 3;

function startChat() {
    // Make sure we're setting the global username from cookie if possible
    if (!username) {
        username = getCookie('username') || "Anonymous";
        console.log("Using username from cookie:", username);
    }
    
    // First make sure any existing connection is closed
    if (eventSource) {
        console.log("Closing existing EventSource connection");
        eventSource.close();
        eventSource = null;
    }
    
    // Reset reconnection counter
    reconnectAttempts = 0;
    
    // Start a new EventSource connection
    console.log("Creating new EventSource connection...");
    
    // Add a timestamp to prevent caching
    const timestamp = new Date().getTime();
    const clientId = getCookie('client_id') || generateClientId(); // Get client ID from cookie or generate
    
    // If we generated a new client ID, store it for future use
    if (!getCookie('client_id')) {
        document.cookie = `client_id=${clientId}; path=/; max-age=${60*60*24}`;
    }
    
    // Use a session-unique ID to track messages from this client
    myMessagesPrefix = `client_${Math.random().toString(36).substring(2, 10)}`;
    console.log("Generated unique message tracking prefix:", myMessagesPrefix);
    
    eventSource = new EventSource(`/stream?t=${timestamp}&clientId=${clientId}`);
    
    // Debug any messages coming through
    eventSource.addEventListener('message', function(event) {
        // Process the message data
        const msg = event.data;
        
        // Skip empty messages or ping messages
        if (!msg || msg === "" || (msg.includes("<System>") && msg.includes("Ping"))) {
            return;
        }
        
        console.log("Message received:", msg);
        
        // Extract username from message format: <username> message
        const messageUsername = extractUsernameFromMessage(msg);
        
        // Get our username directly from cookie every time to ensure consistency
        const currentUsername = getCookie('username');
        
        // A message is our own if it matches one of our locally echoed messages
        // Or if the sender username matches our username from cookie
        let isOwnMessage = localEchoMessages.has(msg);
        
        // Add another check comparing usernames
        if (!isOwnMessage && messageUsername === currentUsername) {
            isOwnMessage = true;
            console.log("Message identified as own based on username match");
        }
        
        console.log(`Message from ${messageUsername}, currentUsername: ${currentUsername}, isOwn: ${isOwnMessage}`);
        
        // Add to chat display with proper ownership flag
        addMessageToChat(msg, false, isOwnMessage);
        
        // Add the sender to active users - explicitly exclude System
        if (messageUsername && messageUsername !== "System") {
            addActiveUser(messageUsername);
        }
    });
    
    // Use explicit open handler for debugging
    eventSource.addEventListener('open', function() {
        console.log("SSE connection opened successfully");
        // Reset reconnection counter on successful connection
        reconnectAttempts = 0;
    });
    
    // Improved error handling
    eventSource.addEventListener('error', function(event) {
        console.error("SSE connection error:", event);
        
        if (eventSource.readyState === EventSource.CLOSED) {
            console.log("SSE connection closed");
            
            // Only try reconnecting if we haven't exceeded the maximum attempts
            if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
                reconnectAttempts++;
                console.log(`Reconnection attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}`);
                
                // Close the old connection properly
                if (eventSource) {
                    eventSource.close();
                    eventSource = null;
                }
                
                // Attempt to reconnect with exponential backoff
                const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000);
                setTimeout(startChat, delay);
            } else {
                console.error("Maximum reconnection attempts reached, reloading page");
                // Force a page reload as last resort
                window.location.reload();
            }
        }
    });
    
    // Fetch active users
    fetchActiveUsers();
    setInterval(fetchActiveUsers, 3000); // Check every 3 seconds
}

// Improved function to extract username from message
function extractUsernameFromMessage(message) {
    const match = message.match(/<([^>]+)>/);
    if (match && match[1]) {
        return match[1].trim();
    }
    return null;
}

// Improved function to add messages to chat
function addMessageToChat(message, isLocalEcho = false, isOwnMessage = false) {
    console.log(`Adding message to chat box: ${message} (local echo: ${isLocalEcho}, own message: ${isOwnMessage})`);
    
    // Get the chat box element
    let chatBox = document.getElementById("chat-box");
    if (!chatBox) {
        console.error("CRITICAL ERROR: Chat box element not found!");
        alert("Error: Chat box not found. Please refresh the page.");
        return;
    }
    
    // If this is a non-local echo that matches a local echo, skip it
    if (!isLocalEcho && localEchoMessages.has(message)) {
        console.log(`Skipping duplicate message that was already echoed locally: ${message}`);
        return;
    }
    
    // Add message timestamp for debugging
    const timestamp = new Date().toLocaleTimeString();
    console.log(`[${timestamp}] Processing message: ${message}`);
    
    // Create a new message element
    let messageElement = document.createElement("p");
    messageElement.style.marginBottom = "5px";
    
    // If it's a local echo, mark it as such
    if (isLocalEcho) {
        messageElement.setAttribute('data-local-echo', 'true');
        localEchoMessages.add(message);
    }
    
    try {
        // Try to parse the username format <username> message
        const usernameMatch = message.match(/^<([^>]+)>/);
        if (usernameMatch) {
            const extractedUsername = usernameMatch[1];
            const messageContent = message.substring(usernameMatch[0].length).trim();
            
            console.log(`Formatted message - Username: ${extractedUsername}, Content: ${messageContent}`);
            
            // Check if this is a "left the chat" message
            if (messageContent === "left the chat") {
                console.log(`User ${extractedUsername} has left the chat`);
                removeActiveUser(extractedUsername);
                
                // Add with special style
                messageElement.innerHTML = `<span class="username-highlight" style="color: #ff0; font-weight: bold;">&lt;${extractedUsername}&gt;</span> <span style="color: #f55;">${messageContent}</span>`;
            } else if (messageContent === "left the chat (client shutdown)") {
                console.log(`User ${extractedUsername} has left the chat`);
                removeActiveUser(extractedUsername);
                
                // Add with special style
                messageElement.innerHTML = `<span class="username-highlight" style="color: #ff0; font-weight: bold;">&lt;${extractedUsername}&gt;</span> <span style="color: #f55;">${messageContent}</span>`; 
            }
            else {
                // Regular message - IMPORTANT: Always use the username from the message
                // Don't replace it with the local username even for local echoes
                messageElement.innerHTML = `<span class="username-highlight" style="color: #ff0; font-weight: bold;">&lt;${extractedUsername}&gt;</span> ${messageContent}`;
                
                // Any user who sends a message is active - add them to active users
                if (extractedUsername && extractedUsername !== "System") {
                    addActiveUser(extractedUsername);
                }
            }
        } else {
            // Just add as plain text
            console.log("No username pattern found, using plain text");
            messageElement.textContent = message;
        }
        
        // Add to chat box
        chatBox.appendChild(messageElement);
        
        // Force scroll to bottom
        chatBox.scrollTop = chatBox.scrollHeight;
        
        console.log("Message added successfully!");
    } catch (error) {
        console.error("Error processing message:", error);
        // Fallback: add as plain text
        messageElement.textContent = message;
        chatBox.appendChild(messageElement);
    }
}

// Generate a unique client ID
function generateClientId() {
    const existingId = localStorage.getItem('chat_client_id');
    if (existingId) {
        return existingId;
    }
    
    const newId = 'client_' + Math.random().toString(36).substring(2, 12) + '_' + Date.now();
    localStorage.setItem('chat_client_id', newId);
    return newId;
}

// Add window event handlers to clean up EventSource on page unload
window.addEventListener('beforeunload', function() {
    if (eventSource) {
        console.log("Closing EventSource connection before page unload");
        eventSource.close();
    }
});

// Enhanced beforeunload handler to clean up when the page is closed
window.addEventListener('beforeunload', function() {
    // Call the cleanup endpoint to decrement the client count
    if (navigator.sendBeacon) {
        // Use sendBeacon API for more reliable delivery during page unload
        navigator.sendBeacon('/cleanup');
        console.log("Sent cleanup request via sendBeacon");
    } else {
        // Fallback to synchronous XHR which is less reliable but better than nothing
        try {
            const xhr = new XMLHttpRequest();
            xhr.open('GET', '/cleanup', false); // false makes it synchronous
            xhr.send();
            console.log("Sent cleanup request via synchronous XHR");
        } catch (e) {
            console.error("Failed to send cleanup request:", e);
        }
    }
    
    // Close EventSource connection
    if (eventSource) {
        console.log("Closing EventSource connection before page unload");
        eventSource.close();
        eventSource = null;
    }
});

// Enhanced logout function that properly closes the EventSource
function logout() {
    if (confirm("Are you sure you want to log out?")) {
        // First close the EventSource
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        
        // Then send the logout request
        fetch("/logout", {
            method: "GET", // Changed from POST to GET to match server handler
            credentials: "same-origin",
            headers: {
                "Cache-Control": "no-cache"
            }
        }).then(response => {
            if (response.ok) {
                // Force reload to get index.html with a cache-busting parameter
                window.location.href = '/?t=' + new Date().getTime();
            }
        }).catch(err => {
            console.error("Error during logout:", err);
            // Force reload anyway in case of error
            window.location.href = '/?t=' + new Date().getTime();
        });
    }
}

// Function to fetch active users from the server - improve this function
function fetchActiveUsers() {
    fetch('/active-users')
        .then(response => response.json())
        .then(data => {
            console.log("Active users from server:", data.users);
            
            // Update the active users set
            activeUsers.clear();
            
            // Add all users from the server's response, excluding System
            if (data.users && Array.isArray(data.users)) {
                data.users.forEach(user => {
                    if (user !== "System") {
                        activeUsers.add(user);
                    }
                });
            }
            
            // Always make sure current user is in the list, unless it's System
            if (username && username !== "System") {
                activeUsers.add(username);
            }
            
            // Update the UI with the new list
            updateActiveUsersList();
        })
        .catch(error => console.error('Error fetching active users:', error));
}

function sendMessage() {
    let input = document.getElementById("message");
    let messageText = input.value.trim();

    if (messageText !== "") {
        console.log("Processing message:", messageText);
        
        // Clear input immediately to prevent double-sending
        input.value = "";
        
        // Check if this is a multiple message command
        if (messageText.startsWith("/multiple")) {
            handleMultipleMessages(messageText);
            return;
        }
        
        // Always get username directly from cookie
        const currentUsername = getCookie('username');
        console.log(`Sending message as ${currentUsername}:`, messageText);
        
        // Store the message text for local echo
        const messageToSend = messageText;
        
        // Add timestamp to prevent caching
        const timestamp = new Date().getTime();
        
        // Add a local echo of the message to show it immediately
        // Make sure to use the current username for the local echo
        const localMessage = `<${currentUsername}> ${messageToSend}`;
        addMessageToChat(localMessage, true, true); // true indicates this is a local echo and own message
        
        // Send to server with client id for additional attribution
        fetch(`/send?message=${encodeURIComponent(messageToSend)}&t=${timestamp}`, {
            method: 'GET',
            credentials: 'same-origin',
            headers: {
                'Cache-Control': 'no-cache'
            }
        })
        .then(response => {
            if (response.ok) {
                console.log("Message sent successfully to server");
                return response.json();
            } else {
                // Remove local echo if sending failed
                removeLocalEcho(localMessage);
                console.error("Failed to send message:", response.status);
                alert("Failed to send message. Please try again.");
                throw new Error("Send failed");
            }
        })
        .catch(error => {
            console.error("Error sending message:", error);
        });
    }
}

// Function to handle multiple messages command
function handleMultipleMessages(messageText) {
    // Extract all messages enclosed in parentheses
    const messageRegex = /\(([^)]+)\)/g;
    const matches = [...messageText.matchAll(messageRegex)];
    
    if (!matches || matches.length === 0) {
        addMessageToChat("<System> Invalid multiple message format. Use: /multiple (message 1), (message 2), ...", true, false);
        return;
    }
    
    // Extract the messages from the regex matches
    const messages = matches.map(match => match[1].trim());
    console.log("Multiple messages to send:", messages);
    
    // Add a status message
    addMessageToChat(`<System> Sending ${messages.length} messages...`, true, false);
    
    // Always get username directly from cookie
    const currentUsername = getCookie('username');
    
    // Send messages with a small delay between them to simulate typing
    let delay = 0;
    messages.forEach((msg, index) => {
        // Calculate delay based on message length (simulating typing speed)
        const typingDelay = 500 + (msg.length * 30); // Base delay + ~30ms per character
        
        setTimeout(() => {
            // Add timestamp to prevent caching
            const timestamp = new Date().getTime();
            
            // Add a local echo of the message using current username
            const localMessage = `<${currentUsername}> ${msg}`;
            addMessageToChat(localMessage, true, true);
            
            // Send to server
            fetch("/send?message=" + encodeURIComponent(msg) + "&t=" + timestamp, {
                method: 'GET',
                credentials: 'same-origin',
                headers: {
                    'Cache-Control': 'no-cache'
                }
            })
            .then(response => {
                if (response.ok) {
                    console.log(`Message ${index+1}/${messages.length} sent successfully`);
                    return response.json();
                } else {
                    removeLocalEcho(localMessage);
                    console.error(`Failed to send message ${index+1}/${messages.length}:`, response.status);
                    throw new Error("Send failed");
                }
            })
            .catch(error => {
                console.error(`Error sending message ${index+1}/${messages.length}:`, error);
            });
            
            // If this is the last message, add completion message
            if (index === messages.length - 1) {
                setTimeout(() => {
                    addMessageToChat("<System> All messages sent successfully.", true, false);
                }, 500);
            }
        }, delay);
        
        // Increase delay for next message
        delay += typingDelay;
    });
}

// A set to keep track of local echo messages
const localEchoMessages = new Set();

// Function to remove local echo if server sending fails
function removeLocalEcho(message) {
    const chatBox = document.getElementById("chat-box");
    if (!chatBox) return;
    
    // Find and remove the local echo message
    const children = chatBox.children;
    for (let i = children.length - 1; i >= 0; i--) {
        const child = children[i];
        if (child.getAttribute('data-local-echo') === 'true' && 
            child.textContent.includes(message)) {
            chatBox.removeChild(child);
            break;
        }
    }
    
    localEchoMessages.delete(message);
}

function getCurrentTime() {
    let now = new Date();
    return now.getHours().toString().padStart(2, '0') + ":" +
        now.getMinutes().toString().padStart(2, '0') + ":" +
        now.getSeconds().toString().padStart(2, '0');
}

let activeUsers = new Set();
let isTalking = false; // Track which image is currently shown

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

// Function to alternate between normal and talk images
function alternateProfileImage() {
    const profileImg = document.getElementById('profile-image');
    if (!profileImg) return;
    
    isTalking = !isTalking;
    profileImg.src = isTalking ? '/static/img/talk.png' : '/static/img/normal.png';
}

username = getCookie('username');

// Function to add username to active users
function addActiveUser(username) {
    // Never add System to the active users list
    if (username === "System") {
        return;
    }
    
    if (!activeUsers.has(username)) {
        activeUsers.add(username);
        updateActiveUsersList();
    }
}

// Function to remove a user from the active users list
function removeActiveUser(username) {
    if (activeUsers.has(username)) {
        console.log(`Removing user ${username} from active users list`);
        activeUsers.delete(username);
        updateActiveUsersList();
    }
}

// Function to update active users list in the UI
function updateActiveUsersList() {
    const activeUsersElement = document.getElementById('active-users');
    // Keep the heading
    activeUsersElement.innerHTML = '<h3>Active Users</h3>';
    
    // Add each user as a list item
    activeUsers.forEach(user => {
        const userElement = document.createElement('div');
        userElement.className = 'user-item';
        userElement.textContent = user;
        activeUsersElement.appendChild(userElement);
    });
}

// Event listener for message input
document.getElementById('message').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        sendMessage();
    }
});

document.addEventListener('DOMContentLoaded', function() {
    // Initialize username from cookie right away
    username = getCookie('username') || "";
    console.log("DOMContentLoaded: Using username from cookie:", username);
    
    // Initialize chat immediately
    startChat();
    
    // Add current user to active users list
    if (username && username !== "System") {
        addActiveUser(username);
    }
    
    // Update date and time
    updateDateTime();
    setInterval(updateDateTime, 1000);
    
    // Set up image alternating every 1 second
    setInterval(alternateProfileImage, 1000);
    
    // Setup periodic update of active users list - check more frequently
    fetchActiveUsers();
    setInterval(fetchActiveUsers, 3000); // Check every 3 seconds instead of 5
});

// Additional script for date/time updates
function updateDateTime() {
    const now = new Date();
    
    // Format date: Thursday, 2023-03-21
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    const day = days[now.getDay()];
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const date = String(now.getDate()).padStart(2, '0');
    
    // Format time: 04:24:32 PM (with seconds)
    let hours = now.getHours();
    const ampm = hours >= 12 ? 'PM' : 'AM';
    hours = hours % 12;
    hours = hours ? hours : 12; // the hour '0' should be '12'
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0'); // Add seconds
    
    document.getElementById('current-date').textContent = `${day}, ${year}-${month}-${date}`;
    document.getElementById('current-time').textContent = `${String(hours).padStart(2, '0')}:${minutes}:${seconds} ${ampm}`;
}

// Extract usernames from messages
function extractUsernameFromMessage(message) {
    const match = message.match(/<([^>]+)>/);
    if (match && match[1]) {
        return match[1].trim();
    }
    return null;
}

// Function to test server latency
function pingServer() {
    // Record start time
    const startTime = Date.now();
    
    // Display sending message
    addMessageToChat("<System> Sending ping to server...", true, false);
    
    // Send ping request to server
    fetch(`/ping?t=${startTime}`, {
        method: 'GET',
        credentials: 'same-origin',
        headers: {
            'Cache-Control': 'no-cache'
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error("Server ping failed");
        }
        return response.json();
    })
    .then(data => {
        // Calculate round-trip time
        const endTime = Date.now();
        const latency = endTime - startTime;
        
        // Display result in chat
        addMessageToChat(`<System> Pong! Server latency: ${latency}ms`, true, false);
        
        // Log additional server info if available
        if (data.serverTime) {
            console.log(`Server processing time: ${data.serverTime}ms`);
        }
    })
    .catch(error => {
        console.error("Ping error:", error);
        addMessageToChat("<System> Failed to ping server. Check console for details.", true, false);
    });
}