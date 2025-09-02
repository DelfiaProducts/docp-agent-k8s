package utils

import (
	"context"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/oklog/ulid/v2"
)

// NewTransactionStatus return new transaction status with id
func NewTransactionStatus() dto.TransactionStatus {
	return dto.TransactionStatus{
		ID: ulid.Make().String(),
	}
}

// GetTransactionFromContext return transaction status from context
func GetTransactionFromContext(ctx context.Context) dto.TransactionStatus {
	val := ctx.Value(dto.ContextTransactionStatus)
	if transactionStatus, ok := val.(dto.TransactionStatus); ok {
		return transactionStatus
	}
	return dto.TransactionStatus{}
}

// GetUlid return ulid string
func GetUlid() string {
	return ulid.Make().String()
}
