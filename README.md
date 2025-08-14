# Storage API

A robust file storage service built with Go and Fiber, featuring configurable file validation, flexible storage organization, and comprehensive file management capabilities for local storage.

## Features

-   **File Validation**: Configurable file type restrictions, size limits, and MIME type validation
-   **Flexible Storage Organization**: Date-based, type-based, or custom file organization patterns
-   **Local Storage**: Optimized for local file system storage with automatic directory creation
-   **File Management**: Upload, download, update, delete, and search files
-   **Access Control**: File sharing with tokens, passwords, and download limits
-   **Analytics**: Track file access patterns and usage statistics
-   **Security**: Secure file URLs, referrer validation, and comprehensive access logging

## Configuration

The API is configured via `config/storage.yaml` with the following main sections:

### File Validation

-   Maximum file size limits
-   Allowed/blocked file extensions
-   MIME type validation

### Storage Organization

-   Organization patterns (date/type/user/custom)
-   File naming strategies (UUID, timestamp, original)
-   Date format customization
-   Time-based folder creation

### Local Storage

-   Configurable upload directory
-   Automatic directory creation
-   File and directory permissions
-   Optimized for local file systems

### Security

-   Secure URL generation
-   Authentication requirements
-   Referrer validation
-   File access logging

## API Endpoints

### Files

-   `POST /api/v1/files/upload` - Upload a new file
-   `GET /api/v1/files` - Search and list files
-   `GET /api/v1/files/:id` - Get file information
-   `GET /api/v1/files/:id/download` - Download a file
-   `PUT /api/v1/files/:id` - Update file metadata
-   `DELETE /api/v1/files/:id` - Delete a file

### Health & Monitoring

-   `GET /health` - Service health check
-   `GET /metrics` - Application metrics

## Environment Variables

```bash
# Server
PORT=3003
GO_ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=password
DB_NAME=storage
```

## File Organization

Files are organized based on the configuration pattern:

### Default Pattern: `date/type`

```
uploads/
├── 2024-01-15/
│   ├── images/
│   │   ├── uuid1.jpg
│   │   └── uuid2.png
│   └── documents/
│       ├── uuid3.pdf
│       └── uuid4.docx
└── 2024-01-16/
    └── videos/
        └── uuid5.mp4
```

### Custom Patterns

-   `date` - Organize by date only
-   `type` - Organize by file type only
-   `user` - Organize by user ID
-   `custom` - Use custom organization function

## File Naming Strategies

-   **UUID**: Generate unique identifiers for files
-   **Timestamp**: Use current timestamp as filename
-   **Original**: Preserve original filename
-   **Custom**: Implement custom naming logic

## Security Features

-   File type validation and blocking
-   MIME type verification
-   Access logging and analytics
-   Secure share tokens
-   Password-protected file sharing
-   Download limits and expiration

## Getting Started

1. **Clone the repository**

    ```bash
    git clone <repository-url>
    cd storage-api
    ```

2. **Install dependencies**

    ```bash
    go mod download
    ```

3. **Configure the service**

    - Copy and modify `config/storage.yaml`
    - Set environment variables

4. **Run the service**

    ```bash
    go run main.go
    ```

5. **Using Docker**

    ```bash
    docker-compose up
    ```

## Development

### Prerequisites

-   Go 1.24.5+
-   PostgreSQL
-   Docker (optional)

### Testing

```bash
go test ./...
```

### Building

```bash
go build -o storage-api main.go
```

## Architecture

The service follows a clean architecture pattern:

-   **Handlers**: HTTP request/response handling
-   **Services**: Business logic and file operations
-   **Models**: Data structures and database models
-   **Config**: Configuration management
-   **Database**: Data persistence layer

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.
