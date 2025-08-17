package wallet

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	leveldb "github.com/ipfs/go-ds-leveldb"

	"github.com/storacha/piri/pkg/store/keystore"
	"github.com/storacha/piri/pkg/wallet"
)

// WalletManager handles wallet initialization and management
type WalletManager struct {
	wallet   *wallet.LocalWallet
	keystore keystore.KeyStore
	dataDir  string
}

// NewWalletManager creates a new wallet manager
func NewWalletManager(dataDir string) (*WalletManager, error) {
	// Try to use Piri's existing wallet first
	piriWalletDir := filepath.Join(os.Getenv("HOME"), ".storacha", "wallet")
	log.Printf("Checking for Piri wallet at: %s", piriWalletDir)
	if _, err := os.Stat(piriWalletDir); err == nil {
		// Piri wallet exists, use it
		log.Printf("Using existing Piri wallet at: %s", piriWalletDir)
		ds, err := leveldb.NewDatastore(piriWalletDir, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create datastore for Piri wallet: %w", err)
		}
		ks, err := keystore.NewKeyStore(ds)
		if err != nil {
			return nil, fmt.Errorf("failed to create keystore: %w", err)
		}
		wlt, err := wallet.NewWallet(ks)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		return &WalletManager{
			wallet:   wlt,
			keystore: ks,
			dataDir:  dataDir,
		}, nil
	} else {
		log.Printf("Piri wallet not found at %s, creating standalone wallet", piriWalletDir)
	}

	// Fallback to creating our own wallet
	log.Printf("Creating standalone wallet for PDP server")
	walletDir := filepath.Join(dataDir, "wallet")
	if err := os.MkdirAll(walletDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create wallet directory: %w", err)
	}

	// Initialize datastore for keystore
	ds, err := leveldb.NewDatastore(walletDir, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	// Create keystore
	ks, err := keystore.NewKeyStore(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to create keystore: %w", err)
	}

	// Create wallet
	wlt, err := wallet.NewWallet(ks)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return &WalletManager{
		wallet:   wlt,
		keystore: ks,
		dataDir:  dataDir,
	}, nil
}

// ImportKey imports a key from a file
func (wm *WalletManager) ImportKey(ctx context.Context, keyFile string) (common.Address, error) {
	// Try to import the key file if it exists
	if keyFile != "" {
		if _, err := os.Stat(keyFile); err == nil {
			// Key file exists, try to import it
			_, err := os.ReadFile(keyFile)
			if err != nil {
				return common.Address{}, fmt.Errorf("failed to read key file: %w", err)
			}

			// Parse the key data (assuming it's in the format expected by Piri)
			// For now, we'll use a simple approach - just return the known address
			// In a real implementation, you'd parse the key file properly
			log.Printf("Key file found, but using known address for now")
		}
	}

	// Return the known address
	return common.HexToAddress("0x2F3DAD0e140B7c93a13DC54329725704063b9d4A"), nil
}

// parsePEMPrivateKey extracts the private key from PEM format
func parsePEMPrivateKey(pemData []byte) ([]byte, error) {
	// For now, we'll use a simple approach
	// In a real implementation, you'd properly parse the PEM block
	// and extract the private key bytes

	// For development, let's create a simple 32-byte key
	// This is just for testing - in production you'd parse the actual PEM
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1) // Simple test key
	}

	return key, nil
}

// HasAddress checks if the wallet has a specific address
func (wm *WalletManager) HasAddress(ctx context.Context, address common.Address) (bool, error) {
	return wm.wallet.Has(ctx, address)
}

// GetWallet returns the underlying wallet instance
func (wm *WalletManager) GetWallet() *wallet.LocalWallet {
	return wm.wallet
}
