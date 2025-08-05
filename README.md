# Go Project

## Table of Contents
- [Description](#description)
- [Requirements](#requirements)
- [Setup](#setup)
- [YouTube API Integration](#youtube-api-integration)
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)

## Description
This project is a go starter project with YouTube API integration for managing videos, channels, comments, and more.

## Requirements
- Go 1.21+
- MySQL 5.7
- MongoDB (optional)
- Liquibase 4.3.5
- Docker 20.10.7
- Docker Compose 1.29.2
- Google Cloud Console account (for YouTube API)

## YouTube API Integration

This project includes a complete YouTube API integration that allows you to:
- 🔐 OAuth2 authentication with Google
- 📹 Get your YouTube videos and channel information
- 🔍 Search YouTube videos
- 📤 Upload videos to YouTube
- 💬 Manage comments (get, add, update, delete)
- 👍 Like/dislike videos and comments
- 📊 Access video analytics and statistics

### Quick Setup
1. Copy environment template: `cp .env.example .env`
2. Configure your Google Cloud credentials in `.env`
3. Follow the detailed setup guide: [YOUTUBE_API_SETUP.md](./YOUTUBE_API_SETUP.md)

### API Endpoints
- **Auth**: `/auth/youtube`, `/auth/youtube/callback`
- **Videos**: `/api/youtube/videos/*`
- **Search**: `/api/youtube/search`
- **Channel**: `/api/youtube/channel`
- **Comments**: `/api/youtube/comments/*`

For complete setup instructions, see [YOUTUBE_API_SETUP.md](./YOUTUBE_API_SETUP.md).
- Postman 8.10.0
- Git 2.25.1
- GitHub

## Setup
1. Install Go from https://golang.org/dl/
2. Install MySQL from https://dev.mysql.com/downloads/mysql/
3. Install Liquibase from https://www.liquibase.org/download
4. Install Docker from https://docs.docker.com/get-docker/
5. Install Docker Compose from https://docs.docker.com/compose/install/
6. Install GoLand from https://www.jetbrains.com/go/download/
7. Install Postman from https://www.postman.com/downloads/
8. Install Git from https://git-scm.com/downloads
9. Install GitHub from https://desktop.github.com/

## Installation
```aiignore
liquibase init project --project-dir=ticketing-system --changelog-file=example-changelog --format=[sql|xml|json|yaml|yml] --project-defaults-file=[liquibase.properties] --url=jdbc:mysql://localhost:3306/ticketing_system --username=project --password=[Password]
```

## Usage
1. Using the terminal, navigate to the project directory.
```aiignore
ENV=stage go run .
```
2. If you want to use docker compose, run the following command:
```aiignore
docker-compose up --build
```
3. To reset docker compose, run the following command:
```aiignore
docker-compose down
```
4. Using MySQL Workbench, connect to the MySQL database using the following credentials:
```aiignore
Host: localhost
Port: 3308
Username: root
Password: root123
```
5. Using Postman, send a POST request to /login with the following JSON body:
```json
{
  "user_name": "lamboktulus1379",
  "password": "password123"
}
```
6. Using Postman, send a POST request to /regiter with the following JSON body:
```json
{
  "user_name": "lamboktulus1379",
  "name": "Lambok Tulus Simamora",
  "password": "password123"
}
```

## License
MIT License
```