# 📖 API Documentation

**Base URL**: `http://localhost:8000`

This document provides detailed usage instructions for the Meetup Backend API, including authentication, user management, products, categories, uploads, and chat functionality.

---

## 1. Health Check (`/health`)

### Check API Status
Simple endpoint to verify if the server is running.

- **URL**: `/health`
- **Method**: `GET`
- **Response (200 OK)**:
  ```json
  {
    "status": "success",
    "message": "API is healthy"
  }
  ```

---

## 2. Authentication (`/api/auth`)

### Register
Create a new user account.

- **URL**: `/api/auth/register`
- **Method**: `POST`
- **Headers**: `Content-Type: application/json`
- **Body**:
  ```json
  {
    "username": "johndoe",
    "email": "johndoe@example.com",
    "password": "securepassword",
    "full_name": "John Doe"
  }
  ```
- **Response (201 Created)**:
  ```json
  {
    "message": "User registered successfully"
  }
  ```
- **Response (Err)**:
  ```json
  {
    "error": "User already exists"
  }
  ```

### Login
Authenticate and receive a JWT token.

- **URL**: `/api/auth/login`
- **Method**: `POST`
- **Headers**: `Content-Type: application/json`
- **Body**:
  ```json
  {
    "email": "johndoe@example.com",
    "password": "securepassword"
  }
  ```
- **Response (200 OK)**:
  ```json
  {
    "token": "eyJhbGciOiJIUzI1Ni...",
    "user": {
      "id": 1,
      "username": "johndoe",
      "email": "johndoe@example.com",
      "role": "user",
      "image_url": ""
    }
  }
  ```

---

## 3. Users (`/api/users`)
*Requires Authentication (`Authorization: Bearer <token>`).*

### Search Users
Search for users by username or email (excluding self).

- **URL**: `/api/users/search`
- **Method**: `GET`
- **Query Params**:
  - `q`: Search keyword (required)
- **Example**: `/api/users/search?q=jane`
- **Response (200 OK)**:
  ```json
  {
    "data": [
      {
        "id": 2,
        "username": "janedoe",
        "email": "jane@example.com",
        "full_name": "Jane Doe",
        "image_url": ""
      }
    ]
  }
  ```

---

## 4. Categories (`/api/categories`)

### Get All Categories
List all available product categories.

- **URL**: `/api/categories`
- **Method**: `GET`
- **Response (200 OK)**:
  ```json
  {
    "data": [
      { "id": 1, "name": "Electronics", "slug": "electronics" },
      { "id": 2, "name": "Clothing", "slug": "clothing" }
    ]
  }
  ```

---

## 5. Products (`/api/products`)

### Get All Products (Public)
List all available products with optional filtering.

- **URL**: `/api/products`
- **Method**: `GET`
- **Query Params**:
  - `category`: Filter by category slug (e.g., `electronics`)
  - `q`: Search by title
- **Response (200 OK)**:
  ```json
  {
    "data": [
      {
        "id": 1,
        "title": "iPhone 15",
        "price": 999,
        "image_url": "/uploads/products/image.jpg",
        "seller": { "username": "seller1", ... }
      }
    ]
  }
  ```

### Get Product Detail (Public)
Get detailed information about a specific product.

- **URL**: `/api/products/:id`
- **Method**: `GET`
- **Response (200 OK)**:
  ```json
  {
    "data": {
      "id": 1,
      "title": "iPhone 15",
      "description": "Brand new...",
      "price": 999,
      "images": ["url1", "url2"],
      "seller": { "email": "seller@example.com", ... }
    }
  }
  ```

### Create Product (Protected)
- **URL**: `/api/products`
- **Method**: `POST`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: application/json`
- **Body**:
  ```json
  {
    "title": "MacBook Pro",
    "description": "M3 Chip, 16GB RAM",
    "price": 2000,
    "category": "electronics",
    "condition": "new",
    "image_url": "/uploads/products/image.jpg",
    "images": ["/uploads/products/image.jpg"]
  }
  ```
- **Response (201 Created)**:
  ```json
  {
    "message": "Product created",
    "data": { ... }
  }
  ```

### Get My Products (Protected)
List products listed by the logged-in user.

- **URL**: `/api/my-products`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <token>`
- **Response (200 OK)**:
  ```json
  {
    "data": [
      {
        "id": 1,
        "title": "My Item",
        "price": 100,
        ...
      }
    ]
  }
  ```

### Update Product (Protected)
Only the seller can update their product.

- **URL**: `/api/products/:id`
- **Method**: `PUT`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: application/json`
- **Body**: Same structure as Create Product.
- **Response (200 OK)**:
  ```json
  { "message": "Product updated", "data": { ... } }
  ```

### Delete Product (Protected)
Only the seller can delete their product.

- **URL**: `/api/products/:id`
- **Method**: `DELETE`
- **Headers**: `Authorization: Bearer <token>`
- **Response (200 OK)**:
  ```json
  { "message": "Product deleted" }
  ```

---

## 6. Uploads (`/api/upload`)
*Requires Authentication.*

### Upload Single Image
- **URL**: `/api/upload`
- **Method**: `POST`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: multipart/form-data`
- **Body**: Form data with key `image` (file).
- **Response (200 OK)**:
  ```json
  { "url": "/uploads/products/170000000.jpg" }
  ```

### Upload Multiple Images
- **URL**: `/api/upload/multiple`
- **Method**: `POST`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: multipart/form-data`
- **Body**: Form data with key `images` (multiple files).
- **Response (200 OK)**:
  ```json
  { "urls": ["/uploads/products/1.jpg", "/uploads/products/2.jpg"] }
  ```

---

## 7. Chat (`/api/chat`)
*Requires Authentication.*

### Init/Get Private Chat
Start a chat or get existing room ID with another user.

- **URL**: `/api/chat/private`
- **Method**: `POST`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: application/json`
- **Body**:
  ```json
  { "target_user_id": 2 }
  ```
- **Response (200 OK / 201 Created)**:
  ```json
  {
    "room_id": 1,
    "created": true // true if new, false if existed
  }
  ```

### Get My Chats
List all chat rooms the user is participating in.

- **URL**: `/api/chat/rooms`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <token>`
- **Response**:
  ```json
  {
    "data": [
      {
        "id": 1,
        "type": "private",
        "last_message": "Hello!",
        "unread_count": 2,
        "other_user_id": 2,
        "other_username": "janedoe",
        "other_image_url": "..."
      }
    ]
  }
  ```

### Get Messages
Get messages for a specific room.

- **URL**: `/api/chat/room/:roomID/messages`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <token>`
- **Query Params**:
  - `limit`: (default 50)
  - `offset`: (default 0)
- **Response**:
  ```json
  {
    "messages": [
      {
        "id": 10,
        "sender_id": 2,
        "content": "Hi there",
        "created_at": "..."
      }
    ]
  }
  ```

### Get Room Status
Check availability of users in a room.

- **URL**: `/api/chat/room/:roomID/status`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <token>`
- **Response**:
  ```json
  {
    "room_id": 1,
    "statuses": [
      { "user_id": 1, "in_room": true, "is_online": true },
      { "user_id": 2, "in_room": false, "is_online": false }
    ]
  }
  ```

### Delete Chat
Leave/Delete a chat room from your list.

- **URL**: `/api/chat/room/:roomID`
- **Method**: `DELETE`
- **Headers**: `Authorization: Bearer <token>`
- **Response**:
  ```json
  { "message": "Chat deleted successfully" }
  ```

---

## 8. WebSocket (`/ws`)
Real-time messaging connection.

- **URL**: `ws://localhost:8000/ws?token=<JWT_TOKEN>`
- **Method**: `GET` (WebSocket Upgrade)

### Client -> Server Events

**1. Send Message**
```json
{
  "type": "chat",
  "chat_room_id": 1,
  "content": "Hello"
}
```

**2. Send Read Receipt**
```json
{
  "type": "read",
  "message_id": 10
}
```

**3. Join Room**
Notify server that user is active in a specific room (active window).
```json
{
  "type": "join_room",
  "chat_room_id": 1
}
```

**4. Leave Room**
Notify server that user left the room (closed window/navigate back).
```json
{
  "type": "leave_room"
}
```

### Server -> Client Events

**1. Incoming Message**
```json
{
  "type": "chat",
  "message": {
      "id": 12,
      "chat_room_id": 1,
      "sender_id": 1,
      "content": "Hello World",
      "created_at": "..."
  }
}
```

**2. Read Receipt Notification**
```json
{
  "type": "read_receipt",
  "message_id": 12,
  "chat_room_id": 1,
  "read_by": 2
}
```

**3. User Status Update**
Notifies when a user comes online/offline.
```json
{
  "type": "user_status",
  "user_id": 2,
  "is_online": true
}
```

**4. Room Status**
Notifies when a user joins or leaves a chat room (real-time active status in room).
```json
{
  "type": "room_status",
  "user_id": 1,
  "chat_room_id": 1,
  "in_room": true
}
```

**5. Online Users List**
Sent immediately after connection established.
```json
{
  "type": "online_users_list",
  "user_ids": [1, 2, 5]
}
```
