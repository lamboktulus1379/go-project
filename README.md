# Go Project

## Table of Contents
- [Description](#description)
- [Requirements](#requirements)
- [Setup](#setup)
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)

## Description
This project is a go starter project.

## Requirements
- Go 1.16
- MySQL 5.7
- Liquibase 4.3.5
- Docker 20.10.7
- Docker Compose 1.29.2
- GoLand 2021.1.3
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