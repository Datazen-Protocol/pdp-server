package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/glebarez/sqlite"
	"github.com/storacha/piri/pkg/store/blobstore"
	"github.com/your-org/pdp-server/pkg/api"
	myBlobstore "github.com/your-org/pdp-server/pkg/blobstore"
	"github.com/your-org/pdp-server/pkg/config"
	"github.com/your-org/pdp-server/pkg/piece"
	"github.com/your-org/pdp-server/pkg/piri"
	"github.com/your-org/pdp-server/pkg/proofset"
	"github.com/your-org/pdp-server/pkg/service"
	"github.com/your-org/pdp-server/pkg/wallet"
	"github.com/your-org/pdp-server/pkg/watcher"
	"gorm.io/gorm"

	"github.com/your-org/pdp-server/pkg/models"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize PDP server using Piri components
	pdpServer, err := initializePDPServer(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to initialize PDP server: %v", err)
	}

	// Start the PDP server
	if err := pdpServer.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start PDP server: %v", err)
	}

	// Register routes
	api.RegisterRoutes(pdpServer.Echo, pdpServer)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting PDP server on %s", addr)

	if err := pdpServer.Echo.Start(addr); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func initializePDPServer(ctx context.Context, config *config.Config) (*api.PDPServer, error) {
	// Initialize our own database for production deployment
	dataDir := config.PDP.DataDir
	if dataDir == "" {
		dataDir = "./data" // Default to local data directory
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Initialize our own SQLite database
	dbPath := filepath.Join(dataDir, "pdp_server.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Auto-migrate our database schema
	if err := db.AutoMigrate(&models.ParkedPiece{}, &models.ParkedPieceRef{}, &models.PDPPieceRef{}, &models.MessageWaitsEth{}, &models.PDPProofSet{}, &models.PDPProofSetCreate{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Printf("Using isolated database at: %s", dbPath)

	// Initialize blob store
	blobStore, err := myBlobstore.NewFileBlobstore(filepath.Join(dataDir, "blobs"))
	if err != nil {
		return nil, fmt.Errorf("failed to create blob store: %v", err)
	}

	// Initialize Piri FsBlobstore for uploads (used by upload service)
	blobRoot := filepath.Join(dataDir, "blobs")
	blobTmp := filepath.Join(dataDir, "tmp")
	piriBlobStore, err := blobstore.NewFsBlobstore(blobRoot, blobTmp)
	if err != nil {
		return nil, fmt.Errorf("failed to create fs blob store: %v", err)
	}

	// Wallet setup
	wm, err := wallet.NewWalletManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("wallet init: %w", err)
	}
	addr, err := wm.ImportKey(ctx, config.PDP.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("wallet import: %w", err)
	}
	if has, err := wm.HasAddress(ctx, addr); err != nil || !has {
		return nil, fmt.Errorf("wallet missing address: %v", err)
	}

	// Piri server (provides PDP service and state DB)
	piriCfg := piri.Config{
		DataDir:    dataDir,
		LotusURL:   config.PDP.LotusURL,
		EthAddress: common.HexToAddress(addr.Hex()),
		Wallet:     wm.GetWallet(),
	}
	piriServer, err := piri.NewServer(piriCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init piri: %v", err)
	}

	// Wire services with our isolated database but Piri's PDP service
	piriService := piriServer.GetPDPService()
	proofSetSvc := proofset.NewProofSetService(piriService, db, common.HexToAddress(addr.Hex())) // Use our isolated DB
	simpleProofSvc := proofset.NewSimpleProofSetService(db)                                      // Simple proof set service for our isolated DB
	adapter := service.NewPiriServiceAdapter(piriService)
	pieceSvc := piece.NewPieceService(adapter, blobStore, db) // Use our isolated DB

	// Initialize transaction watcher
	piriDB := piriServer.GetDB() // Get Piri's database for checking transaction status
	txWatcher := watcher.NewTransactionWatcher(db, piriDB)

	log.Printf("Initialized services with isolated database and transaction watcher")

	// Create PDP server
	pdpServer := api.NewPDPServer(piriServer, piriBlobStore, dataDir, proofSetSvc, simpleProofSvc, pieceSvc, txWatcher)

	return pdpServer, nil
}
