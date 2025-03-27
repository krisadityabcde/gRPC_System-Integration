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

function startChat() {
    // Make sure we're setting the global username from cookie if possible
    if (!username) {
        username = getCookie('username') || "Anonymous";
        console.log("Using username from cookie:", username);
    }
    
    // Start the event source for receiving messages
    if (!eventSource) {
        eventSource = new EventSource("/stream");
        eventSource.onmessage = (event) => {
            let msg = event.data;
            if (!isMessageDuplicate(msg)) {
                addMessageToChat(msg);
                
                // Extract username from message format: <username> message
                const match = msg.match(/<([^>]+)>/);
                if (match && match[1]) {
                    addActiveUser(match[1].trim());
                }
            }
        };
        
        // Handle reconnection
        eventSource.onerror = () => {
            console.log("EventSource connection failed, reconnecting...");
            if (eventSource) {
                eventSource.close();
                eventSource = null;
            }
            // Try to reconnect after a short delay
            setTimeout(startChat, 2000);
        };
    }
    
    // Fetch active users
    fetchActiveUsers();
    setInterval(fetchActiveUsers, 10000); // Update every 10 seconds
}

// Function to fetch active users from the server
function fetchActiveUsers() {
    fetch('/active-users')
        .then(response => response.json())
        .then(data => {
            // Clear the current active users set
            activeUsers.clear();
            
            // Add all the users from the server
            if (data.users && Array.isArray(data.users)) {
                data.users.forEach(user => {
                    addActiveUser(user);
                });
            }
            
            // Update the UI
            updateActiveUsersList();
        })
        .catch(error => console.error('Error fetching active users:', error));
}

function sendMessage() {
    let input = document.getElementById("message");
    let messageText = input.value.trim();

    if (messageText !== "") {
        // Get username from cookie if not already set
        if (!username) {
            username = getCookie('username') || "Anonymous";
        }
        
        // Format the message with the username
        let formattedMessage = "<" + username + "> " + messageText;
        
        // Prevent duplication on client side
        if (!isMessageDuplicate(formattedMessage)) {
            addMessageToChat(formattedMessage);
        }
        
        // Send to server
        fetch("/send?message=" + encodeURIComponent(messageText));
        input.value = "";
    }
}

// Helper function to get cookie value
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

function addMessageToChat(message) {
    let chatBox = document.getElementById("chat-box");
    let messageElement = document.createElement("p");
    
    // Apply styling to username in <username> format
    const usernameMatch = message.match(/^<([^>]+)>/);
    if (usernameMatch) {
        // Create a styled message with username in yellow
        const username = usernameMatch[1];
        const messageContent = message.substring(usernameMatch[0].length).trim();
        
        // Create HTML with span for username
        messageElement.innerHTML = `<span class="username-highlight">&lt;${username}&gt;</span> ${messageContent}`;
    } else {
        // Just add the message as plain text if it doesn't match the pattern
        messageElement.textContent = message;
    }
    
    messageElement.style.marginBottom = "5px";
    chatBox.appendChild(messageElement);
    
    // Ensure scroll to bottom happens after DOM update
    requestAnimationFrame(() => {
        chatBox.scrollTop = chatBox.scrollHeight;
    });
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

// Function to log out
function logout() {
    if (confirm("Are you sure you want to log out?")) {
        fetch("/logout").then(response => {
            if (response.ok) {
                if (eventSource) {
                    eventSource.close();
                }
                // Force reload to get index.html with a cache-busting parameter
                window.location.href = '/?t=' + new Date().getTime();
            }
        });
    }
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
    if (!activeUsers.has(username)) {
        activeUsers.add(username);
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
    // Initialize chat immediately
    startChat();
    
    // Add current user to active users list
    if (username) {
        addActiveUser(username);
    }
    
    // Update date and time
    updateDateTime();
    setInterval(updateDateTime, 1000);
    
    // Set up image alternating every 1 second
    setInterval(alternateProfileImage, 1000);
    
    // Setup periodic update of active users list
    fetchActiveUsers();
    setInterval(fetchActiveUsers, 5000); // Check every 5 seconds
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