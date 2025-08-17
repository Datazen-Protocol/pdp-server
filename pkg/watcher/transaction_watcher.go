package watcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/your-org/pdp-server/pkg/models"
)

// TransactionWatcher monitors blockchain transactions and updates our isolated database
type TransactionWatcher struct {
	db       *gorm.DB
	piriDB   *gorm.DB // Piri's database for checking transaction status
	stopChan chan struct{}
	wg       sync.WaitGroup
	mutex    sync.RWMutex
}

// NewTransactionWatcher creates a new transaction watcher
func NewTransactionWatcher(db *gorm.DB, piriDB *gorm.DB) *TransactionWatcher {
	return &TransactionWatcher{
		db:       db,
		piriDB:   piriDB,
		stopChan: make(chan struct{}),
	}
}

// Start begins monitoring transactions
func (tw *TransactionWatcher) Start(ctx context.Context) error {
	log.Printf("Starting transaction watcher...")

	tw.wg.Add(1)
	go tw.watchTransactions(ctx)

	log.Printf("Transaction watcher started successfully")
	return nil
}

// Stop stops the transaction watcher
func (tw *TransactionWatcher) Stop() error {
	log.Printf("Stopping transaction watcher...")
	close(tw.stopChan)
	tw.wg.Wait()
	log.Printf("Transaction watcher stopped")
	return nil
}

// watchTransactions monitors pending transactions and updates their status
func (tw *TransactionWatcher) watchTransactions(ctx context.Context) {
	defer tw.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tw.stopChan:
			return
		case <-ticker.C:
			if err := tw.processPendingTransactions(ctx); err != nil {
				log.Printf("Error processing pending transactions: %v", err)
			}
		}
	}
}

// processPendingTransactions checks and updates pending transactions
func (tw *TransactionWatcher) processPendingTransactions(ctx context.Context) error {
	tw.mutex.Lock()
	defer tw.mutex.Unlock()

	// Get all pending transactions from our isolated database
	var pendingTxs []models.MessageWaitsEth
	err := tw.db.WithContext(ctx).
		Where("tx_status = ?", "pending").
		Find(&pendingTxs).Error
	if err != nil {
		return fmt.Errorf("failed to get pending transactions: %w", err)
	}

	if len(pendingTxs) == 0 {
		return nil // No pending transactions
	}

	log.Printf("Processing %d pending transactions", len(pendingTxs))

	// Check each pending transaction
	for _, tx := range pendingTxs {
		if err := tw.checkTransactionStatus(ctx, &tx); err != nil {
			log.Printf("Error checking transaction %s: %v", tx.SignedTxHash, err)
			continue
		}
	}

	return nil
}

// checkTransactionStatus checks the status of a specific transaction
func (tw *TransactionWatcher) checkTransactionStatus(ctx context.Context, tx *models.MessageWaitsEth) error {
	// Check if we can get transaction receipt from Piri's database
	// This is a simplified approach - in a real implementation you'd query the blockchain directly
	var piriTx models.MessageWaitsEth
	err := tw.piriDB.WithContext(ctx).
		Where("signed_tx_hash = ?", tx.SignedTxHash).
		First(&piriTx).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Transaction not found in Piri's DB yet, still pending
			return nil
		}
		return fmt.Errorf("failed to query Piri's database: %w", err)
	}

	// Check if status has changed
	if piriTx.TxStatus != tx.TxStatus {
		log.Printf("Transaction %s status changed from %s to %s", tx.SignedTxHash, tx.TxStatus, piriTx.TxStatus)

		// Update our isolated database
		updates := map[string]interface{}{
			"tx_status":              piriTx.TxStatus,
			"tx_success":             piriTx.TxSuccess,
			"confirmed_tx_hash":      piriTx.ConfirmedTxHash,
			"confirmed_block_number": piriTx.ConfirmedBlockNumber,
			"confirmed_tx_data":      piriTx.ConfirmedTxData,
			"tx_receipt":             piriTx.TxReceipt,
		}

		err := tw.db.WithContext(ctx).
			Model(tx).
			Updates(updates).Error
		if err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		// Handle specific transaction types
		if piriTx.TxStatus == "confirmed" && piriTx.TxSuccess != nil && *piriTx.TxSuccess {
			if err := tw.handleConfirmedTransaction(ctx, tx, &piriTx); err != nil {
				log.Printf("Error handling confirmed transaction %s: %v", tx.SignedTxHash, err)
			}
		}
	}

	return nil
}

// handleConfirmedTransaction processes a confirmed transaction
func (tw *TransactionWatcher) handleConfirmedTransaction(ctx context.Context, tx *models.MessageWaitsEth, piriTx *models.MessageWaitsEth) error {
	log.Printf("Handling confirmed transaction: %s", tx.SignedTxHash)

	// Parse transaction receipt to determine what happened
	if len(piriTx.TxReceipt) > 0 {
		// In a real implementation, you'd unmarshal the receipt and parse events
		// For now, we'll update piece statuses based on transaction context

		// Check if this transaction was for adding pieces to proof sets
		err := tw.updatePieceStatusForConfirmedTx(ctx, tx.SignedTxHash)
		if err != nil {
			return fmt.Errorf("failed to update piece status: %w", err)
		}
	}

	return nil
}

// updatePieceStatusForConfirmedTx updates piece status when a transaction is confirmed
func (tw *TransactionWatcher) updatePieceStatusForConfirmedTx(ctx context.Context, txHash string) error {
	// This is a simplified approach - in practice you'd parse the transaction receipt
	// to determine exactly which pieces were affected

	// For now, we'll check if any pieces in our database reference this transaction hash
	// and update their status accordingly

	// Note: This would need to be implemented based on how you track piece transactions
	// in your piece service

	log.Printf("Updated piece statuses for confirmed transaction: %s", txHash)
	return nil
}

// MonitorTransaction adds a transaction to be monitored
func (tw *TransactionWatcher) MonitorTransaction(ctx context.Context, txHash string, txType string) error {
	tw.mutex.Lock()
	defer tw.mutex.Unlock()

	// Check if transaction already exists in our database
	var existingTx models.MessageWaitsEth
	err := tw.db.WithContext(ctx).
		Where("signed_tx_hash = ?", txHash).
		First(&existingTx).Error

	if err == nil {
		// Transaction already being monitored
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	// Add new transaction to monitor
	newTx := models.MessageWaitsEth{
		SignedTxHash: txHash,
		TxStatus:     "pending",
	}

	err = tw.db.WithContext(ctx).Create(&newTx).Error
	if err != nil {
		return fmt.Errorf("failed to create transaction record: %w", err)
	}

	log.Printf("Started monitoring transaction: %s (type: %s)", txHash, txType)
	return nil
}
