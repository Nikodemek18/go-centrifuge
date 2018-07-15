package coredocument

import (
	"context"
	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/coredocument"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/identity"
	logging "github.com/ipfs/go-log"
	"fmt"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/p2p"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/anchor"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/tools"
	"github.com/CentrifugeInc/centrifuge-protobufs/gen/go/p2p"
	"github.com/CentrifugeInc/go-centrifuge/centrifuge/errors"
	goerrors "github.com/go-errors/errors"
)

var log = logging.Logger("coredocument")

// ----- ERROR -----
type ErrInconsistentState struct {
	message string
}

func NewErrInconsistentState(message string) *ErrInconsistentState {
	padded := ""
	if len(message) > 0 {
		padded = fmt.Sprintf(": %s", message)
	}
	return &ErrInconsistentState{
		message: fmt.Sprintf("Inconsistent CoreDocument state%s", padded),
	}
}
func (e *ErrInconsistentState) Error() string {
	return e.message
}

// ----- END ERROR -----

// CoreDocumentProcessor is the processor that can deal with CoreDocuments and performs actions on them such as
// anchoring, sending on the p2p level, or signing.
type CoreDocumentProcessor struct {
}

// CoreDocumentProcessorer identifies an implementation, which can do a bunch of things with a CoreDocument.
// E.g. send, anchor, etc.
type CoreDocumentProcessorer interface {
	Send(coreDocument *coredocumentpb.CoreDocument, ctx context.Context, recipient string) (err error)
	Anchor(document *coredocumentpb.CoreDocument) (err error)
}

func GetDefaultCoreDocumentProcessor() (CoreDocumentProcessorer) {
	return &CoreDocumentProcessor{}
}

// Send sends the given CoreDocumentProcessor to the given recipient on the P2P layer
func (cdp *CoreDocumentProcessor) Send(coreDocument *coredocumentpb.CoreDocument, ctx context.Context, recipient string) (err error) {
	if coreDocument == nil {
		return errors.GenerateNilParameterError(coreDocument)
	}

	peerId, err := identity.ResolveP2PEthereumIdentityForId(recipient)
	if err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}


	if len(peerId.Keys[1]) == 0 {
		return goerrors.Wrap("Identity doesn't have p2p key", 1)
	}

	// Default to last key of that type
	lastb58Key, err := peerId.GetLastB58KeyForType(1)
	if err != nil {
		return err
	}
	log.Infof("Sending Document to CentID [%v] with Key [%v]\n", recipient, lastb58Key)
	clientWithProtocol := fmt.Sprintf("/ipfs/%s", lastb58Key)
	client := p2p.OpenClient(clientWithProtocol)
	log.Infof("Done opening connection against [%s]\n", lastb58Key)

	hostInstance := p2p.GetHost()
	bSenderId, err := hostInstance.ID().ExtractPublicKey().Bytes()
	if err != nil {
		return err
	}
	_, err = client.Post(context.Background(), &p2ppb.P2PMessage{Document: coreDocument, SenderCentrifugeId: bSenderId})
	if err != nil {
		return err
	}
	return
}

// Anchor anchors the given CoreDocument
func (cd *CoreDocumentProcessor) Anchor(document *coredocumentpb.CoreDocument) (error) {
	if document == nil {
		return errors.GenerateNilParameterError(document)
	}
	log.Infof("Anchoring document %v", document)

	id, err := tools.ByteArrayToByte32(document.CurrentIdentifier)
	if err != nil {
		log.Error(err)
		return err
	}
	rootHash, err := tools.ByteArrayToByte32(document.DocumentRoot)
	if err != nil {
		log.Error(err)
		return err
	}

	confirmations := make(chan *anchor.WatchAnchor, 1)
	err = anchor.RegisterAsAnchor(id, rootHash, confirmations)
	if err != nil {
		log.Error(err)
		return err
	}
	anchorWatch := <-confirmations
	err = anchorWatch.Error
	return err
}

func (cd *CoreDocumentProcessor) Sign() {
	//signingService := cc.Node.GetSigningService()
	//signingService.Sign(cd.Document)
	return
}