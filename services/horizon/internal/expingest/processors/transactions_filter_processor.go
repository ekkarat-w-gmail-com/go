package processors

import (
	"context"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/xdr"
)

// TransactionFilterProcessor is a processor which can be configured to filter away failed transactions
type TransactionFilterProcessor struct {
	IngestFailedTransactions bool
}

func (p *TransactionFilterProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()
	r.IgnoreUpgradeChanges()

	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		txSucceeded := transaction.Result.Result.Result.Code == xdr.TransactionResultCodeTxSuccess
		if p.IngestFailedTransactions || txSucceeded {
			err = w.Write(transaction)
			if err != nil {
				if err == stdio.ErrClosedPipe {
					// Reader does not need more data
					return nil
				}
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *TransactionFilterProcessor) Name() string {
	return "TransactionFilterProcessor"
}

func (p *TransactionFilterProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &TransactionFilterProcessor{}