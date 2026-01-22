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

### 1. Authentication

#### Register
- **Endpoint**: `POST /api/auth/register`
- **Body**:
  ```json
  {
    "username": "user3",
    "email": "user3@example.com",
    "password": "password123",
    "full_name": "User Three"
  }
  ```

#### Login
- **Endpoint**: `POST /api/auth/login`
- **Body**:
  ```json
  {
    "email": "user1@example.com",
    "password": "password123"
  }
  ```
- **Response**: Returns a `token` (JWT).

### 2. Chat

#### Initialize Private Chat
Creates or retrieves a chat room with another user.
- **Endpoint**: `POST /api/chat/private`
- **Header**: `Authorization: Bearer <TOKEN>`
- **Body**:
  ```json
  {
    "target_user_id": 2
  }
  ```
- **Response**: `{ "room_id": 1, "created": false }`

### 3. WebSocket

- **URL**: `ws://localhost:8000/ws?token=<JWT_TOKEN>`
- **Events (JSON)**:

  **Send Message (Client -> Server):**
  ```json
  {
    "type": "chat",
    "chat_room_id": 1,
    "content": "Hello World"
  }
  ```

  **Receive Message (Server -> Client):**
  ```json
  {
    "type": "chat",
    "message": {
        "id": 12,
        "content": "Hello World",
        "sender_id": 1
    }
  }
  ```

  **Send Read Receipt (Client -> Server):**
  *Automatically sent by client when displaying a message.*
  ```json
  {
    "type": "read",
    "message_id": 12
  }
  ```
  *Effect: The message is permanently deleted from the database.*

  **Receive Read Receipt (Server -> Client):**
  *Sent to the Sender when their message is read/deleted.*
  ```json
  {
    "type": "read_receipt",
    "message_id": 12,
    "chat_room_id": 1,
    "read_by": 2
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
6.  **Window B**: Enter User ID `1` and click **Start Chat**.
7.  Send messages!

### What you will see:
- **Grey Ticks**: Message sent.
- **Blue Ticks**: Message read by recipient (and deleted from DB).
- **Offline Messages**: If you send a message to an offline user, they will receive it immediately upon connecting.

