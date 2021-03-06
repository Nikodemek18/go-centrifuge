package v2

import (
	"github.com/centrifuge/go-centrifuge/bootstrap"
	"github.com/centrifuge/go-centrifuge/documents"
	"github.com/centrifuge/go-centrifuge/errors"
	"github.com/centrifuge/go-centrifuge/pending"
)

// BootstrappedService key maps to the Service implementation in Bootstrap context.
const BootstrappedService = "V2 Service"

// Bootstrapper implements bootstrap.Bootstrapper.
type Bootstrapper struct{}

// Bootstrap adds transaction.Repository into context.
func (b Bootstrapper) Bootstrap(ctx map[string]interface{}) error {
	pendingDocSrv, ok := ctx[pending.BootstrappedPendingDocumentService].(pending.Service)
	if !ok {
		return errors.New("failed to get %s", pending.BootstrappedPendingDocumentService)
	}

	nftSrv, ok := ctx[bootstrap.BootstrappedInvoiceUnpaid].(documents.TokenRegistry)
	if !ok {
		return errors.New("failed to get %s", bootstrap.BootstrappedInvoiceUnpaid)
	}

	ctx[BootstrappedService] = Service{
		pendingDocSrv: pendingDocSrv,
		tokenRegistry: nftSrv,
	}
	return nil
}
