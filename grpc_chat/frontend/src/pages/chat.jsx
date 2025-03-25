import React, { useState } from "react";
import { loginUser, sendMessage } from "../chat_client";
import "bootstrap/dist/css/bootstrap.min.css";

const Chat = () => {
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");
  const [chatHistory, setChatHistory] = useState([]);

  const handleLogin = () => {
    loginUser(username, (response) => {
      console.log("Login sukses:", response);
    });
  };

  const handleSendMessage = () => {
    if (message.trim() === "") return;

    sendMessage(username, message, (response) => {
      setChatHistory([...chatHistory, { username, message }]);
      setMessage("");
    });
  };

  return (
    <div className="container mt-4">
      <h2>gRPC Chat</h2>
      <div className="mb-3">
        <input
          type="text"
          className="form-control"
          placeholder="Enter username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
        <button className="btn btn-primary mt-2" onClick={handleLogin}>
          Login
        </button>
      </div>
      <div className="mb-3">
        <input
          type="text"
          className="form-control"
          placeholder="Type a message"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
        />
        <button className="btn btn-success mt-2" onClick={handleSendMessage}>
          Send
        </button>
      </div>
      <div className="chat-box border p-3">
        {chatHistory.map((chat, index) => (
          <div key={index}>
            <strong>{chat.username}:</strong> {chat.message}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Chat;
