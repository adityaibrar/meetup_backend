# Meetup Backend & Chat System

This repository contains a backend built with **Go (Fiber)** and a **MySQL** database, featuring a real-time **WebSocket Chat System** with ephemeral messaging (messages are deleted after being read) and Read Receipts.

A simple **HTML/JS Client Demo** is also included.

## 🚀 Features

- **Authentication**: JWT-based Register & Login.
- **Real-time Chat**: WebSocket communication.
- **Ephemeral Messaging**: Messages are stored in DB and **hard deleted** immediately after the recipient reads them.
- **Offline Retrieval**: Unread messages are delivered immediately when a user connects.
- **Read Receipts**: Blue ticks (Real-time notification) when a message is read/deleted.
- **Private Rooms**: 1-on-1 chat rooms.
- **Client Demo**: Simple UI to test the chat flow.

## 🛠 Prerequisites

- Go 1.25+
- MySQL
- Air (Optional, for live reload)

## ⚙️ Setup & Installation

1.  **Clone the repository**
2.  **Configure Environment**:
    Make sure you have a `.env` file (or set variables):
    ```env
    DB_HOST=127.0.0.1
    DB_PORT=3306
    DB_USER=root
    DB_PASSWORD=
    DB_NAME=meetup_database
    JWT_SECRET=secret_key
    PORT=8000
    ```

3.  **Run with Seeding (First Time / Reset)**:
    This command wipes the database, migrates tables, and seeds 2 default users.
    ```bash
    go run main.go -reset
    ```
    *Default Users:*
    *   **User 1**: `user1@example.com` / `password123`
    *   **User 2**: `user2@example.com` / `password123`

4.  **Run Normally**:
    ```bash
    go run main.go
    # OR using Air
    air
    ```

## 📖 API Usage

**Full API Documentation**: See [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for detailed endpoint usage.

**Base URL**: `http://localhost:8000`

### Quick Start

#### Register
Create a new user account.
- **URL**: `/api/auth/register`
- **Method**: `POST`
- **Headers**: `Content-Type: application/json`
- **Body**:
  ```json
  {
    "username": "user3",
    "email": "user3@example.com",
    "password": "password123",
    "full_name": "User Three"
  }
  ```
- **Response (201 Created)**:
  ```json
  {
    "message": "User registered successfully",
    "user": { ... }
  }
  ```

#### Login
Authenticate and receive a JWT token.
- **URL**: `/api/auth/login`
- **Method**: `POST`
- **Headers**: `Content-Type: application/json`
- **Body**:
  ```json
  {
    "email": "user1@example.com",
    "password": "password123"
  }
  ```
- **Response (200 OK)**:
  ```json
  {
    "status": "success",
    "token": "eyJhbGciOiJIUzI1...",
    "user": {
      "id": 1,
      "email": "user1@example.com",
      "username": "user1"
    }
  }
  ```

### 2. Chat

#### Initialize Private Chat
Check if a private chat room exists with a user, or create one if not.
- **URL**: `/api/chat/private`
- **Method**: `POST`
- **Headers**: 
  - `Content-Type: application/json`
  - `Authorization: Bearer <YOUR_JWT_TOKEN>`
- **Body**:
  ```json
  {
    "target_user_id": 2
  }
  ```
- **Response (200 OK / 201 Created)**:
  ```json
  {
    "room_id": 1,
    "created": false
  }
  ```

### 3. WebSocket

Connect to the real-time chat server.

- **URL**: `ws://localhost:8000/ws?token=<YOUR_JWT_TOKEN>`
- **Method**: `GET` (WebSocket Upgrade)

#### Events (Received from Server)

**1. Incoming Message**
Received when someone sends you a message.
```json
{
  "type": "chat",
  "message": {
      "id": 12,
      "chat_room_id": 1,
      "sender_id": 1,
      "content": "Hello World",
      "created_at": "2026-01-22T10:00:00Z"
  }
}
```

**2. Read Receipt Notification**
Received when your message has been read by the recipient (and thus deleted).
```json
{
  "type": "read_receipt",
  "message_id": 12,
  "chat_room_id": 1,
  "read_by": 2
}
```

**3. User Status Update**
Received when a user comes online or goes offline.
```json
{
  "type": "user_status",
  "user_id": 2,
  "is_online": true
}
```

#### Events (Sent by Client)

**1. Send Message**
```json
{
  "type": "chat",
  "chat_room_id": 1,
  "content": "Hello User 2"
}
```

**2. Send Read Receipt**
Send this when the user views the message. This triggers the ephemeral "Delete on Read" logic.
```json
{
  "type": "read",
  "message_id": 12
}
```

## 🖥 Client Demo

A standalone HTML client is located in `client_demo/`.

1.  Ensure backend is running.
2.  Open `client_demo/index.html` in your browser.
    *   *Tip: Open two private/incognito windows to simulate two users.*
3.  **Login** as `user1@example.com` in Window A.
4.  **Login** as `user2@example.com` in Window B.
5.  **Window A**: Enter User ID `2` and click **Start Chat**.
    *   *Note: User 2's status (Online/Offline) will appear in the header.*
6.  **Window B**: Enter User ID `1` and click **Start Chat**.
7.  Send messages!

### What you will see:
- **Grey Ticks (✓✓)**: Message sent.
- **Blue Ticks (✓✓)**: Message read by recipient.
- **Offline Messages**: If recipient is offline, they receive messages upon connecting.
- **Presence**: Green "Online" indicator updates in real-time.

