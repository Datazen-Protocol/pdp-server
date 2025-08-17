# PDP Server API Documentation

## Base URL
```
http://localhost:8081
```

## Authentication
Currently no authentication required. Future versions will include JWT-based authentication.

## Content Types
- Request: `application/json`, `multipart/form-data` (for file uploads)
- Response: `application/json`

---

## Health & Status Endpoints

### GET /health
Health check endpoint for monitoring systems.

**Response:**
```json
{
  "status": "healthy"
}
```

**Status Codes:**
- `200 OK`: Service is healthy
- `500 Internal Server Error`: Service is unhealthy

---

### GET /status
Detailed server status information.

**Response:**
```json
{
  "server": {
    "status": "running",
    "uptime": "2h30m",
    "version": "1.0.0"
  },
  "database": {
    "status": "connected",
    "type": "sqlite"
  },
  "piri": {
    "status": "connected",
    "blockchain": "filecoin-calibnet"
  }
}
```

---

## File Management Endpoints

### POST /upload
Upload a file directly to the server.

**Request:**
```bash
curl -X POST \
  -F "file=@example.txt" \
  http://localhost:8081/upload
```

**Response:**
```json
{
  "success": true,
  "file": {
    "name": "example.txt",
    "size": 1024,
    "cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
    "upload_id": "uuid-string"
  }
}
```

---

### GET /files
List all uploaded files.

**Response:**
```json
{
  "files": [
    {
      "name": "example.txt",
      "size": 1024,
      "cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
      "upload_date": "2024-08-17T01:30:00Z"
    }
  ]
}
```

---

## Piece Management Endpoints

### POST /pieces
Prepare a piece for upload.

**Request:**
```json
{
  "check": {
    "name": "sha2-256",
    "hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "size": 1024
  }
}
```

**Response:**
```json
{
  "piece_id": "piece-uuid",
  "upload_url": "/pieces/piece-uuid",
  "status": "prepared"
}
```

---

### PUT /pieces/:pieceID
Upload piece data.

**Request:**
```bash
curl -X PUT \
  -H "Content-Type: application/octet-stream" \
  --data-binary @piece-data.bin \
  http://localhost:8081/pieces/piece-uuid
```

**Response:**
```json
{
  "success": true,
  "piece": {
    "id": "piece-uuid",
    "cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
    "size": 1024,
    "commp": "baga6ea4seaqhxjzqhcnmjqb4jz7q6z7q6z7q6z7q6z7q6z7q6z7q6z7q6z7q",
    "status": "uploaded"
  }
}
```

---

### GET /pieces/:pieceID
Get piece information.

**Response:**
```json
{
  "piece": {
    "id": "piece-uuid",
    "cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
    "size": 1024,
    "commp": "baga6ea4seaqhxjzqhcnmjqb4jz7q6z7q6z7q6z7q6z7q6z7q6z7q6z7q6z7q",
    "status": "uploaded",
    "created_at": "2024-08-17T01:30:00Z"
  }
}
```

---

### POST /pieces/:pieceID/proofset/:proofSetID
Add a piece to a proof set.

**Response:**
```json
{
  "success": true,
  "transaction_id": "tx-hash",
  "status": "pending"
}
```

---

## Proof Set Management Endpoints

### POST /proofsets
Create a new proof set on the blockchain.

**Response:**
```json
{
  "success": true,
  "proofset": {
    "id": 1,
    "proofset_id": "0x49e212e1ab77b7260eb91e19be1b1e4b8e7ed6ce77685e28295974c2d9560ba3",
    "status": "created",
    "transaction_id": "tx-hash",
    "created_at": "2024-08-17T01:30:00Z"
  }
}
```

---

### GET /proofsets
List all proof sets.

**Response:**
```json
{
  "proofsets": [
    {
      "id": 1,
      "proofset_id": "0x49e212e1ab77b7260eb91e19be1b1e4b8e7ed6ce77685e28295974c2d9560ba3",
      "status": "active",
      "created_at": "2024-08-17T01:30:00Z"
    }
  ]
}
```

---

### GET /proofsets/:id
Get specific proof set details.

**Response:**
```json
{
  "proofset": {
    "id": 1,
    "proofset_id": "0x49e212e1ab77b7260eb91e19be1b1e4b8e7ed6ce77685e28295974c2d9560ba3",
    "status": "active",
    "transaction_id": "tx-hash",
    "roots_count": 5,
    "created_at": "2024-08-17T01:30:00Z"
  }
}
```

---

### POST /proofsets/:id/roots
Add roots to a proof set.

**Request:**
```json
{
  "roots": [
    {
      "root_cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
      "subroot_cids": [
        "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
      ]
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "transaction_id": "tx-hash",
  "status": "pending"
}
```

---

### GET /proofsets/:id/roots
Get all roots for a proof set.

**Response:**
```json
{
  "roots": [
    {
      "root_cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
      "subroot_cids": [
        "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
      ],
      "added_at": "2024-08-17T01:30:00Z"
    }
  ]
}
```

---

### GET /proofsets/:id/status
Get detailed proof set status.

**Response:**
```json
{
  "proofset": {
    "id": 1,
    "proofset_id": "0x49e212e1ab77b7260eb91e19be1b1e4b8e7ed6ce77685e28295974c2d9560ba3",
    "status": "active",
    "blockchain_status": "confirmed",
    "transaction_id": "tx-hash",
    "block_number": 12345,
    "confirmation_count": 10,
    "roots_count": 5,
    "last_updated": "2024-08-17T01:30:00Z"
  }
}
```

---

## Proving Endpoints

### POST /proofsets/:id/prove
Trigger proving for a proof set.

**Response:**
```json
{
  "success": true,
  "proving_job": {
    "id": "proving-job-uuid",
    "status": "started",
    "started_at": "2024-08-17T01:30:00Z"
  }
}
```

---

### GET /proofsets/:id/prove/status
Get proving status for a proof set.

**Response:**
```json
{
  "proving_job": {
    "id": "proving-job-uuid",
    "status": "completed",
    "progress": 100,
    "started_at": "2024-08-17T01:30:00Z",
    "completed_at": "2024-08-17T01:45:00Z",
    "proof_cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
  }
}
```

---

## Transaction Monitoring Endpoints

### GET /pieces/:pieceID/transaction/status
Get transaction status for a piece operation.

**Response:**
```json
{
  "transaction": {
    "id": "tx-hash",
    "status": "confirmed",
    "block_number": 12345,
    "confirmation_count": 10,
    "gas_used": 21000,
    "created_at": "2024-08-17T01:30:00Z",
    "confirmed_at": "2024-08-17T01:32:00Z"
  }
}
```

---

### POST /pieces/:pieceID/transaction/monitor
Start monitoring a transaction.

**Request:**
```json
{
  "transaction_id": "tx-hash"
}
```

**Response:**
```json
{
  "success": true,
  "monitoring": {
    "transaction_id": "tx-hash",
    "status": "monitoring",
    "started_at": "2024-08-17T01:30:00Z"
  }
}
```

---

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "The request is invalid",
    "details": "Specific error details here"
  }
}
```

### Common Error Codes

- `INVALID_REQUEST`: Request format or parameters are invalid
- `NOT_FOUND`: Requested resource not found
- `INTERNAL_ERROR`: Server internal error
- `BLOCKCHAIN_ERROR`: Blockchain operation failed
- `STORAGE_ERROR`: File storage operation failed
- `VALIDATION_ERROR`: Data validation failed

### HTTP Status Codes

- `200 OK`: Successful operation
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `502 Bad Gateway`: External service error
- `503 Service Unavailable`: Service temporarily unavailable

---

## Rate Limiting

Current implementation has no rate limiting. Future versions will include:
- Per-IP rate limiting
- Per-endpoint rate limiting
- Configurable rate limits

## Pagination

For list endpoints that may return large datasets, pagination will be implemented:

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)

**Response Format:**
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

## WebSocket Support (Future)

Future versions will include WebSocket endpoints for real-time updates:
- Transaction status updates
- Proving progress updates
- System status notifications
