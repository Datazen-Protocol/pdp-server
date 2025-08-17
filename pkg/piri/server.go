package piri

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"gorm.io/gorm"

	"github.com/storacha/piri/pkg/database"
	"github.com/storacha/piri/pkg/database/gormdb"
	"github.com/storacha/piri/pkg/pdp"
	"github.com/storacha/piri/pkg/pdp/service"
	"github.com/storacha/piri/pkg/pdp/service/contract"
	"github.com/storacha/piri/pkg/pdp/store"
	"github.com/storacha/piri/pkg/store/blobstore"
	"github.com/storacha/piri/pkg/wallet"
)

// Server wraps Piri's PDP server functionality
type Server struct {
	piriServer *pdp.Server
	wallet     *wallet.LocalWallet
	pdpService *service.PDPService
	db         *gorm.DB
}

// Config holds Piri server configuration
type Config struct {
	DataDir    string
	LotusURL   string
	EthAddress common.Address
	Wallet     *wallet.LocalWallet
}

// NewServer creates a new Piri PDP server
func NewServer(cfg Config) (*Server, error) {
	// Parse endpoint URL (we'll use a local endpoint for now)
	_, err := url.Parse("http://localhost:3001")
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	// Create the components manually to capture service and database
	datastorePath := filepath.Join(cfg.DataDir, "datastore")
	log.Printf("Creating datastore at: %s", datastorePath)
	ds, err := leveldb.NewDatastore(datastorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore at %s: %w", datastorePath, err)
	}

	blobStore := blobstore.NewTODO_DsBlobstore(namespace.Wrap(ds, datastore.NewKey("blobs")))
	stashStore, err := store.NewStashStore(filepath.Join(cfg.DataDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create stash store: %w", err)
	}

	// Check wallet
	if has, err := cfg.Wallet.Has(context.Background(), cfg.EthAddress); err != nil {
		return nil, fmt.Errorf("failed to read wallet for address %s: %w", cfg.EthAddress, err)
	} else if !has {
		return nil, fmt.Errorf("wallet for address %s not found", cfg.EthAddress)
	}

	// Create Lotus client
	lotusURL, err := url.Parse(cfg.LotusURL)
	if err != nil {
		return nil, fmt.Errorf("parsing lotus client address: %w", err)
	}
	if lotusURL.Scheme != "ws" && lotusURL.Scheme != "wss" {
		return nil, fmt.Errorf("lotus client address must be 'ws' or 'wss', got %s", lotusURL.Scheme)
	}

	chainClient, _, err := client.NewFullNodeRPCV1(context.Background(), lotusURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain client: %w", err)
	}

	ethClient, err := ethclient.Dial(cfg.LotusURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to eth client: %w", err)
	}

	// Create state database
	stateDir := filepath.Join(cfg.DataDir, "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	stateDB, err := gormdb.New(filepath.Join(stateDir, "state.db"),
		database.WithJournalMode(database.JournalModeWAL),
		database.WithForeignKeyConstraintsEnable(true),
		database.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to create state database: %w", err)
	}

	// Create PDP service
	pdpService, err := service.NewPDPService(stateDB, cfg.EthAddress, cfg.Wallet, blobStore, stashStore, chainClient, ethClient, &contract.PDPContract{})
	if err != nil {
		return nil, fmt.Errorf("failed to create PDP service: %w", err)
	}

	// Create a minimal Piri server wrapper (we don't need the full server since we're using our own API)
	// We'll create an empty server since we're using our own API layer
	piriServer := &pdp.Server{}

	return &Server{
		piriServer: piriServer,
		wallet:     cfg.Wallet,
		pdpService: pdpService,
		db:         stateDB,
	}, nil
}

// Start starts the Piri PDP server
func (s *Server) Start(ctx context.Context) error {
	// Start the PDP service which includes task engine and watchers
	if s.pdpService != nil {
		if err := s.pdpService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start PDP service: %w", err)
		}
	}
	return nil
}

// Stop stops the Piri PDP server
func (s *Server) Stop(ctx context.Context) error {
	// Stop the PDP service which includes task engine and watchers
	if s.pdpService != nil {
		if err := s.pdpService.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop PDP service: %w", err)
		}
	}
	return nil
}

// GetPiriServer returns the underlying Piri server
func (s *Server) GetPiriServer() *pdp.Server {
	return s.piriServer
}

// GetPDPService returns the PDP service
func (s *Server) GetPDPService() *service.PDPService {
	return s.pdpService
}

// GetDB returns the database
func (s *Server) GetDB() *gorm.DB {
	return s.db
}
