package store

import (
	"time"

	"gorm.io/gorm"
)

// BaseEvent contains common fields for all event models.
// Embed this in your event structs for consistent indexing.
type BaseEvent struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	BlockNumber uint64    `gorm:"index;not null"`
	TxHash      string    `gorm:"type:varchar(66);index;not null"`
	TxIndex     uint      `gorm:"not null"`
	LogIndex    uint      `gorm:"not null"`
	Timestamp   time.Time `gorm:"index;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// BeforeCreate sets the timestamp if not already set.
func (b *BaseEvent) BeforeCreate(tx *gorm.DB) error {
	if b.Timestamp.IsZero() {
		b.Timestamp = time.Now()
	}
	return nil
}

// SyncStatus represents the synchronization status for the indexer.
type SyncStatus struct {
	Contract        string    `gorm:"primaryKey;type:varchar(100)"`
	LastBlockNumber uint64    `gorm:"not null"`
	LastBlockHash   string    `gorm:"type:varchar(66)"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

// IndexerMeta stores metadata about the indexer instance.
type IndexerMeta struct {
	Key       string    `gorm:"primaryKey;type:varchar(100)"`
	Value     string    `gorm:"type:text"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Transfer represents an ERC20 Transfer event.
// Stores USDC transfers on Linea with indexed fields for efficient queries.
type Transfer struct {
	BaseEvent
	From  string `gorm:"type:varchar(42);index;not null"`
	To    string `gorm:"type:varchar(42);index;not null"`
	Value string `gorm:"type:numeric(78);not null"` // uint256 max is 78 digits
}

// TableName returns the table name for Transfer.
func (Transfer) TableName() string {
	return "transfers"
}
