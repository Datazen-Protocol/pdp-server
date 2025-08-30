package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Datazen-Protocol/pdp-server/pkg/piece"
	"github.com/Datazen-Protocol/pdp-server/pkg/piri"
	"github.com/Datazen-Protocol/pdp-server/pkg/proofset"
	"github.com/Datazen-Protocol/pdp-server/pkg/upload"
	"github.com/Datazen-Protocol/pdp-server/pkg/watcher"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/storacha/piri/pkg/store/blobstore"
)

// PDPServer wraps Piri's PDP server functionality
type PDPServer struct {
	piriServer     *piri.Server
	Echo           *echo.Echo // Exported field
	uploadSvc      *upload.UploadService
	proofSetSvc    *proofset.ProofSetService
	simpleProofSvc *proofset.SimpleProofSetService
	pieceSvc       *piece.PieceService
	txWatcher      *watcher.TransactionWatcher
}

// NewPDPServer creates a new PDP server instance
func NewPDPServer(piriServer *piri.Server, blobStore blobstore.Blobstore, dataDir string, proofSetSvc *proofset.ProofSetService, simpleProofSvc *proofset.SimpleProofSetService, pieceSvc *piece.PieceService, txWatcher *watcher.TransactionWatcher) *PDPServer {
	return &PDPServer{
		piriServer:     piriServer,
		Echo:           echo.New(),
		uploadSvc:      upload.NewUploadService(blobStore, dataDir),
		proofSetSvc:    proofSetSvc,
		simpleProofSvc: simpleProofSvc,
		pieceSvc:       pieceSvc,
		txWatcher:      txWatcher,
	}
}

// Start starts the PDP server
func (s *PDPServer) Start(ctx context.Context) error {
	// Start the Piri PDP server if it exists
	if s.piriServer != nil {
		if err := s.piriServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start Piri PDP server: %w", err)
		}
	}

	// Start the transaction watcher
	if s.txWatcher != nil {
		if err := s.txWatcher.Start(ctx); err != nil {
			return fmt.Errorf("failed to start transaction watcher: %w", err)
		}
	}

	return nil
}

// Stop stops the PDP server
func (s *PDPServer) Stop(ctx context.Context) error {
	// Stop the Piri PDP server if it exists
	if s.piriServer != nil {
		if err := s.piriServer.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop Piri PDP server: %w", err)
		}
	}
	return nil
}

// RegisterRoutes registers the API routes
func RegisterRoutes(e *echo.Echo, pdpServer *PDPServer) {
	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

	// Upload endpoint for direct client uploads
	e.POST("/upload", pdpServer.handleUpload)

	// File listing endpoint
	e.GET("/files", pdpServer.handleListFiles)

	// Status endpoint
	e.GET("/status", pdpServer.handleStatus)

	// Proof set management endpoints
	e.POST("/proofsets", pdpServer.handleCreateProofSet)
	e.GET("/proofsets", pdpServer.handleListProofSets)
	e.GET("/proofsets/:id", pdpServer.handleGetProofSet)
	e.POST("/proofsets/:id/roots", pdpServer.handleAddRootsToProofSet)
	e.GET("/proofsets/:id/roots", pdpServer.handleGetProofSetRoots)
	e.GET("/proofsets/:id/status", pdpServer.handleGetProofSetStatus)

	// Piece management endpoints
	e.POST("/pieces", pdpServer.handlePreparePiece)
	e.PUT("/pieces/:pieceID", pdpServer.handleUploadPiece)
	e.GET("/pieces/:pieceID", pdpServer.handleGetPiece)
	e.POST("/pieces/:pieceID/proofset/:proofSetID", pdpServer.handleAddPieceToProofSet)

	// Transaction monitoring endpoints
	e.GET("/pieces/:pieceID/transaction/status", pdpServer.handleGetTransactionStatus)
	e.POST("/pieces/:pieceID/transaction/monitor", pdpServer.handleMonitorTransaction)

	// Piri's piece upload endpoint (for internal use)
	e.PUT("/pdp/piece/upload/:uploadUUID", pdpServer.handlePiriPieceUpload)

	// Proving endpoints
	e.POST("/proofsets/:id/prove", pdpServer.handleProveProofSet)
	e.GET("/proofsets/:id/prove/status", pdpServer.handleGetProveStatus)
}

// handleUpload handles direct file uploads from clients
func (s *PDPServer) handleUpload(c echo.Context) error {
	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No file uploaded",
		})
	}

	// Upload file using upload service
	result, err := s.uploadSvc.UploadFile(c.Request().Context(), file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Upload failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// handleListFiles returns a list of uploaded files
func (s *PDPServer) handleListFiles(c echo.Context) error {
	files, err := s.uploadSvc.ListFiles(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to list files: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"files": files,
	})
}

// handleStatus returns the current status of the PDP server
func (s *PDPServer) handleStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "running",
		"service": "pdp-server",
	})
}

// handleCreateProofSet creates a new proof set
func (s *PDPServer) handleCreateProofSet(c echo.Context) error {
	if s.simpleProofSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	var req proofset.CreateProofSetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	proofSet, err := s.simpleProofSvc.CreateProofSet(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create proof set: %v", err),
		})
	}

	return c.JSON(http.StatusCreated, proofSet)
}

// handleListProofSets lists all proof sets
func (s *PDPServer) handleListProofSets(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	proofSets, err := s.proofSetSvc.ListProofSets(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to list proof sets: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"proof_sets": proofSets,
	})
}

// handleGetProofSet gets a specific proof set by ID
func (s *PDPServer) handleGetProofSet(c echo.Context) error {
	if s.simpleProofSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	proofSet, err := s.simpleProofSvc.GetProofSet(c.Request().Context(), uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Proof set not found: %v", err),
		})
	}

	return c.JSON(http.StatusOK, proofSet)
}

// handleAddRootsToProofSet adds roots to a proof set
func (s *PDPServer) handleAddRootsToProofSet(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	var requests []proofset.AddRootRequest
	if err := c.Bind(&requests); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := s.proofSetSvc.AddRootsToProofSet(c.Request().Context(), id, requests); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to add roots: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Roots added successfully",
	})
}

// handleGetProofSetRoots gets the roots for a proof set
func (s *PDPServer) handleGetProofSetRoots(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	roots, err := s.proofSetSvc.GetProofSetRoots(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get roots: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"roots": roots,
	})
}

// handleGetProofSetStatus gets detailed status of a proof set
func (s *PDPServer) handleGetProofSetStatus(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	status, err := s.proofSetSvc.GetProofSetStatus(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get status: %v", err),
		})
	}

	return c.JSON(http.StatusOK, status)
}

// handlePreparePiece prepares a piece for upload
func (s *PDPServer) handlePreparePiece(c echo.Context) error {
	if s.pieceSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piece service not available",
		})
	}

	var req struct {
		FilePath string `json:"file_path"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.FilePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "file_path is required",
		})
	}

	// Prepare the piece
	pieceInfo, err := s.pieceSvc.PreparePiece(c.Request().Context(), req.FilePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to prepare piece: %v", err),
		})
	}

	return c.JSON(http.StatusCreated, pieceInfo)
}

// handleUploadPiece uploads piece data
func (s *PDPServer) handleUploadPiece(c echo.Context) error {
	if s.pieceSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piece service not available",
		})
	}

	pieceID := c.Param("pieceID")
	if pieceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Piece ID is required",
		})
	}

	// Read file content from multipart form or request body
	var fileContent []byte
	var err error

	// Try multipart form first
	file, err := c.FormFile("file")
	if err == nil {
		// Handle multipart file upload
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to open uploaded file",
			})
		}
		defer src.Close()

		fileContent, err = io.ReadAll(src)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to read uploaded file",
			})
		}
	} else {
		// Try reading from request body
		fileContent, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to read request body",
			})
		}
	}

	if len(fileContent) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No file content provided",
		})
	}

	// Upload the piece
	pieceInfo, err := s.pieceSvc.UploadPiece(c.Request().Context(), pieceID, fileContent)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to upload piece: %v", err),
		})
	}

	return c.JSON(http.StatusOK, pieceInfo)
}

// handleGetPiece retrieves piece data
func (s *PDPServer) handleGetPiece(c echo.Context) error {
	if s.pieceSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piece service not available",
		})
	}

	pieceID := c.Param("pieceID")
	if pieceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Piece ID is required",
		})
	}

	// Get piece information
	pieceInfo, err := s.pieceSvc.GetPiece(c.Request().Context(), pieceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Piece not found: %v", err),
		})
	}

	return c.JSON(http.StatusOK, pieceInfo)
}

// handleProveProofSet triggers proving for a proof set
func (s *PDPServer) handleProveProofSet(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	// For now, return a placeholder response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"proofSetID": id,
		"status":     "proving_initiated",
		"message":    "Proving will be handled by Piri's task engine",
	})
}

// handleGetProveStatus gets the proving status for a proof set
func (s *PDPServer) handleGetProveStatus(c echo.Context) error {
	if s.proofSetSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Proof set service not available",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	// For now, return a placeholder response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"proofSetID": id,
		"status":     "pending",
		"lastProof":  "not_available",
		"nextProof":  "scheduled",
	})
}

// handleAddPieceToProofSet adds a piece to a proof set
func (s *PDPServer) handleAddPieceToProofSet(c echo.Context) error {
	if s.pieceSvc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piece service not available",
		})
	}

	pieceID := c.Param("pieceID")
	if pieceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Piece ID is required",
		})
	}

	proofSetIDStr := c.Param("proofSetID")
	proofSetID, err := strconv.ParseInt(proofSetIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proof set ID",
		})
	}

	if err := s.pieceSvc.AddPieceToProofSet(c.Request().Context(), pieceID, proofSetID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to add piece to proof set: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Piece added to proof set successfully",
	})
}

// handlePiriPieceUpload handles Piri's piece upload endpoint
func (s *PDPServer) handlePiriPieceUpload(c echo.Context) error {
	if s.piriServer == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piri server not available",
		})
	}

	uploadUUID := c.Param("uploadUUID")
	if uploadUUID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Upload UUID is required",
		})
	}

	// Parse UUID
	uuid, err := uuid.Parse(uploadUUID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid upload UUID",
		})
	}

	// Get Piri's PDPService
	pdpService := s.piriServer.GetPDPService()
	if pdpService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Piri PDP service not available",
		})
	}

	// Upload piece using Piri's service
	if _, err := pdpService.UploadPiece(c.Request().Context(), uuid, c.Request().Body); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to upload piece: %v", err),
		})
	}

	// Return 204 No Content as per Piri's API
	return c.NoContent(http.StatusNoContent)
}

// handleGetTransactionStatus returns the transaction status for a piece
func (s *PDPServer) handleGetTransactionStatus(c echo.Context) error {
	pieceID := c.Param("pieceID")
	if pieceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "piece ID is required")
	}

	piece, err := s.pieceSvc.GetPiece(c.Request().Context(), pieceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("piece not found: %v", err))
	}

	response := map[string]interface{}{
		"piece_id":              piece.ID,
		"status":                piece.Status,
		"transaction_hash":      piece.TransactionHash,
		"transaction_timestamp": piece.TransactionTimestamp,
		"error_message":         piece.ErrorMessage,
	}

	return c.JSON(http.StatusOK, response)
}

// handleMonitorTransaction manually triggers transaction monitoring for a piece
func (s *PDPServer) handleMonitorTransaction(c echo.Context) error {
	pieceID := c.Param("pieceID")
	if pieceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "piece ID is required")
	}

	err := s.pieceSvc.MonitorTransactionStatus(c.Request().Context(), pieceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to monitor transaction: %v", err))
	}

	piece, err := s.pieceSvc.GetPiece(c.Request().Context(), pieceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("piece not found: %v", err))
	}

	response := map[string]interface{}{
		"piece_id":              piece.ID,
		"status":                piece.Status,
		"transaction_hash":      piece.TransactionHash,
		"transaction_timestamp": piece.TransactionTimestamp,
		"error_message":         piece.ErrorMessage,
	}

	return c.JSON(http.StatusOK, response)
}
