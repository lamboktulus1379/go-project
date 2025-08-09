# Go Project - API Documentation

## Table of Contents
- [Description](#description)
- [Requirements](#requirements)
- [API Endpoints](#api-endpoints)
- [Authentication](#authentication)
- [User Management](#user-management)
- [YouTube API Integration](#youtube-api-integration)
- [Setup](#setup)
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)
 - [Migrations](#migrations)

## Description
This project is a Go REST API with comprehensive user management and YouTube API integration. It provides endpoints for user authentication, video management, channel operations, and comment handling.

**Base URL:** `http://localhost:10001`

## Requirements
- Go 1.21+
- PostgreSQL 5.7+ (Primary database)
- MySQL 5.7 (Optional)
- MongoDB (Optional)
- Liquibase 4.3.5
- Docker 20.10.7
- Docker Compose 1.29.2
- Google Cloud Console account (for YouTube API)

## API Endpoints

### 🔑 Authentication

#### Register User
```bash
curl -X POST http://localhost:10001/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "user_name": "johndoe",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "response_code": "200",
  "response_message": "Success",
  "data": {
    "name": "John Doe",
    "user_name": "johndoe"
  }
}
```

#### Login User
```bash
curl -X POST http://localhost:10001/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "johndoe",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "response_code": "200",
  "response_message": "Success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "name": "John Doe",
      "user_name": "johndoe"
    }
  }
}
```

#### Health Check
```bash
curl -X POST http://localhost:10001/healthz
```

### 🎥 YouTube API Integration

> **Note:** All YouTube API endpoints require authentication. Include the JWT token in the Authorization header.

#### YouTube OAuth2 Authentication

##### Get Authorization URL
```bash
curl -X GET http://localhost:10001/auth/youtube
```

**Response:**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?client_id=...",
  "message": "Visit this URL to authorize the application"
}
```

##### OAuth2 Callback
```bash
# This endpoint is called by Google after user authorization
# The frontend should handle the redirect from Google
curl -X GET "http://localhost:10001/auth/youtube/callback?code=AUTH_CODE&state=STATE"
```

#### Video Operations

##### Get My Videos
```bash
curl -X GET "http://localhost:10001/api/youtube/videos?max_results=10&order=date" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Query Parameters:**
- `max_results` (int): Maximum number of results (1-50, default: 25)
- `page_token` (string): Token for pagination
- `order` (string): Sort order (date, rating, relevance, title, viewCount)
- `published_after` (string): RFC3339 timestamp
- `published_before` (string): RFC3339 timestamp

##### Get Video Details
```bash
curl -X GET http://localhost:10001/api/youtube/videos/VIDEO_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Upload Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "title=My Video Title" \
  -F "description=Video description here" \
  -F "privacy=private" \
  -F "tags=golang,tutorial,api" \
  -F "file=@/path/to/video.mp4"
```

##### Search Videos
```bash
curl -X GET "http://localhost:10001/api/youtube/search?q=golang%20tutorial&max_results=5&type=video" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Query Parameters:**
- `q` (string, required): Search query
- `max_results` (int): Maximum results (1-50, default: 25)
- `page_token` (string): Pagination token
- `order` (string): Sort order (date, rating, relevance, title, viewCount)
- `type` (string): Resource type (video, channel, playlist)
- `channel_id` (string): Search within specific channel
- `duration` (string): Video duration (short, medium, long)
- `definition` (string): Video quality (high, standard)

#### Video Rating Operations

##### Like Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/VIDEO_ID/like \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Dislike Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/VIDEO_ID/dislike \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Remove Video Rating
```bash
curl -X DELETE http://localhost:10001/api/youtube/videos/VIDEO_ID/rating \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Comment Operations

##### Get Video Comments
```bash
curl -X GET "http://localhost:10001/api/youtube/videos/VIDEO_ID/comments?max_results=20&order=time" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Add Comment
```bash
curl -X POST http://localhost:10001/api/youtube/comments \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "video_id": "VIDEO_ID_HERE",
    "text": "Great video! Thanks for sharing."
  }'
```

##### Update Comment
```bash
curl -X PUT http://localhost:10001/api/youtube/comments/COMMENT_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Updated comment text here"
  }'
```

##### Delete Comment
```bash
curl -X DELETE http://localhost:10001/api/youtube/comments/COMMENT_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Like Comment
```bash
curl -X POST http://localhost:10001/api/youtube/comments/COMMENT_ID/like \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Channel Operations

##### Get My Channel
```bash
curl -X GET http://localhost:10001/api/youtube/channel \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Get Channel Details
```bash
curl -X GET http://localhost:10001/api/youtube/channels/CHANNEL_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Playlist Operations

##### Get My Playlists
```bash
curl -X GET "http://localhost:10001/api/youtube/playlists?max_results=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Create Playlist
```bash
curl -X POST http://localhost:10001/api/youtube/playlists \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My New Playlist",
    "description": "A collection of my favorite videos",
    "privacy": "public"
  }'
```

### 🧪 Test Endpoints (No Authentication Required)

These endpoints are available for testing without authentication:

```bash
# Test: Get My Videos
curl -X GET "http://localhost:10001/test/youtube/videos?max_results=5"

# Test: Search Videos
curl -X GET "http://localhost:10001/test/youtube/search?q=programming&max_results=3"

# Test: Get My Channel
curl -X GET http://localhost:10001/test/youtube/channel
```

## User Management

### Error Responses

All endpoints return standardized error responses:

```json
{
  "response_code": "400",
  "response_message": "Bad Request",
  "error": "Validation failed",
  "details": "user_name is required"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created (for uploads)
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (invalid or missing auth)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource doesn't exist)
- `500` - Internal Server Error

## YouTube API Integration

### Authentication Flow for Frontend

1. **Get Authorization URL:**
   ```typescript
   // Angular service example
   getAuthUrl(): Observable<any> {
     return this.http.get(`${this.baseUrl}/auth/youtube`);
   }
   ```

2. **Redirect User to Google:**
   ```typescript
   authorizeYouTube() {
     this.youtubeService.getAuthUrl().subscribe(response => {
       window.location.href = response.auth_url;
     });
   }
   ```

3. **Handle Callback:**
   ```typescript
   // The callback URL will be handled by your Go backend
   // User will be redirected back to your Angular app with tokens
   ```

### Making Authenticated Requests

```typescript
// Angular HTTP Interceptor example
@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    const token = localStorage.getItem('jwt_token');
    if (token) {
      const authReq = req.clone({
        headers: req.headers.set('Authorization', `Bearer ${token}`)
      });
      return next.handle(authReq);
    }
    return next.handle(req);
  }
}
```

### YouTube API Rate Limits

- **Default quota:** 10,000 units per day
- **Video uploads:** 1600 units each
- **Search operations:** 100 units each
- **Video details:** 1 unit each
- **Comments:** 1-50 units depending on operation

### CORS Configuration

The API supports CORS for frontend integration. Current configuration allows:
- Origins: `https://tulus.tech` (update for your domain)
- Methods: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`
- Headers: `Origin`, `Content-Type`, `Authorization`
- Credentials: Enabled

Update CORS settings in `/server/router.go` for your frontend domain.

## Setup

### Prerequisites
1. Install Go from https://golang.org/dl/
2. Install PostgreSQL from https://www.postgresql.org/download/
3. Install MySQL from https://dev.mysql.com/downloads/mysql/ (optional)
4. Install MongoDB from https://www.mongodb.com/try/download/community (optional)
5. Install Liquibase from https://www.liquibase.org/download
6. Install Docker from https://docs.docker.com/get-docker/
7. Install Docker Compose from https://docs.docker.com/compose/install/

### Environment Setup
1. Copy environment template:
   ```bash
   cp .env.example .env
   cp config.env.example config.env
   ```

2. Configure your environment variables in `.env` and `config.env`

3. For YouTube API integration, follow the detailed setup guide: [YOUTUBE_API_SETUP.md](./YOUTUBE_API_SETUP.md)

### Database Setup
1. Create PostgreSQL database:
   ```bash
   createdb -h localhost -p 5432 -U postgres my_project
   ```

2. Run Liquibase migrations:
  #### PostgreSQL (current changelog used in this repo)
  The project uses an SQL formatted changelog at `liquibase/my_project_postgres/sql/my-project-changelog.sql`.

  Common commands (run from repo root):
  ```bash
  # Preview SQL without applying
  (cd liquibase/my_project_postgres/sql && liquibase updateSQL)

  # Apply all pending changesets
  (cd liquibase/my_project_postgres/sql && liquibase update)

  # Clear and recalculate checksums (only if a committed changeset was edited)
  (cd liquibase/my_project_postgres/sql && liquibase clearChecksums)

  # Roll back last changeset (example)
  (cd liquibase/my_project_postgres/sql && liquibase rollbackCount 1)
  ```

  Properties file (`liquibase/my_project_postgres/sql/liquibase.properties`) supplies:
  ```properties
  changeLogFile=my-project-changelog.sql
  liquibase.command.url=jdbc:postgresql://localhost:5432/my_project
  liquibase.command.username=project
  liquibase.command.password=MyPassword_123
  ```

  #### MySQL (alternative / legacy)
  A MySQL variant changelog exists at `liquibase/my-project/sql/my-project-changelog.sql` (includes share tables).

  ```bash
  # Preview
  (cd liquibase/my-project/sql && liquibase updateSQL)
  # Apply
  (cd liquibase/my-project/sql && liquibase update)
  ```

  NOTE: The application ShareRepository currently uses the MySQL connection; if you decide to switch sharing to PostgreSQL ensure the repository SQL and DSN align.

## Migrations
For full details (adding changesets, rollback, dual-DB strategy) see [docs/MIGRATIONS.md](./docs/MIGRATIONS.md).

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd go-project
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build -o app main.go
   ```

## Usage

### Development Mode
1. Start the server in development mode:
   ```bash
   go run main.go
   ```

2. Or start with hot reload (Air):
  ```bash
  ./start-air.sh
  ```
  This uses the existing `.air.toml` configuration. If Air isn't installed the script installs it (requires Go) or falls back to `go run`.

2. The API will be available at: `http://localhost:10001`

### Production Mode
1. Build and run:
   ```bash
   go build -o app main.go
   ./app
   ```

### Docker Deployment
1. Build and run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

2. Stop the services:
   ```bash
   docker-compose down
   ```

### Database Connection
- **PostgreSQL** (Primary): `localhost:5432`
- **MySQL** (Optional): `localhost:3306` 
- **MongoDB** (Optional): `localhost:27017`

## Angular Frontend Integration

### Service Example
```typescript
// auth.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private baseUrl = 'http://localhost:10001';

  constructor(private http: HttpClient) {}

  register(userData: any): Observable<any> {
    return this.http.post(`${this.baseUrl}/register`, userData);
  }

  login(credentials: any): Observable<any> {
    return this.http.post(`${this.baseUrl}/login`, credentials);
  }

  getAuthHeaders(): HttpHeaders {
    const token = localStorage.getItem('jwt_token');
    return new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });
  }
}

// youtube.service.ts
@Injectable({
  providedIn: 'root'
})
export class YouTubeService {
  private baseUrl = 'http://localhost:10001/api/youtube';

  constructor(private http: HttpClient, private authService: AuthService) {}

  getMyVideos(params?: any): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.get(`${this.baseUrl}/videos`, { headers, params });
  }

  searchVideos(query: string, maxResults = 25): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const params = { q: query, max_results: maxResults };
    return this.http.get(`${this.baseUrl}/search`, { headers, params });
  }

  uploadVideo(videoData: FormData): Observable<any> {
    const token = localStorage.getItem('jwt_token');
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
      // Don't set Content-Type for FormData, let browser set it
    });
    return this.http.post(`${this.baseUrl}/videos/upload`, videoData, { headers });
  }

  likeVideo(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/videos/${videoId}/like`, {}, { headers });
  }

  addComment(videoId: string, text: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const body = { video_id: videoId, text: text };
    return this.http.post(`${this.baseUrl}/comments`, body, { headers });
  }
}
```

### Component Example
```typescript
// video-list.component.ts
export class VideoListComponent implements OnInit {
  videos: any[] = [];
  loading = false;

  constructor(private youtubeService: YouTubeService) {}

  ngOnInit() {
    this.loadMyVideos();
  }

  loadMyVideos() {
    this.loading = true;
    this.youtubeService.getMyVideos({ max_results: 20, order: 'date' })
      .subscribe({
        next: (response) => {
          this.videos = response.data || [];
          this.loading = false;
        },
        error: (error) => {
          console.error('Error loading videos:', error);
          this.loading = false;
        }
      });
  }

  likeVideo(videoId: string) {
    this.youtubeService.likeVideo(videoId)
      .subscribe({
        next: () => {
          console.log('Video liked successfully');
          // Update UI or reload videos
        },
        error: (error) => {
          console.error('Error liking video:', error);
        }
      });
  }
}
```

## Testing

### API Testing with curl
You can test all endpoints using the curl examples provided above. Make sure to:

1. Start the server: `go run main.go`
2. Register a user first
3. Login to get a JWT token
4. Use the JWT token in Authorization headers for protected endpoints

### Example Test Flow
```bash
# 1. Register
curl -X POST http://localhost:10001/register \
  -H "Content-Type: application/json" \
  -d '{"name": "Test User", "user_name": "testuser", "password": "test123"}'

# 2. Login
curl -X POST http://localhost:10001/login \
  -H "Content-Type: application/json" \
  -d '{"user_name": "testuser", "password": "test123"}'

# 3. Use the token from login response in subsequent requests
export JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 4. Test YouTube endpoints
curl -X GET http://localhost:10001/api/youtube/videos \
  -H "Authorization: Bearer $JWT_TOKEN"
```

## License
MIT License