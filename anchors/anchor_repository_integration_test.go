// +build integration

package anchors_test

import (
	"os"
	"testing"
	"time"

	"github.com/centrifuge/go-centrifuge/anchors"
	"github.com/centrifuge/go-centrifuge/bootstrap"
	cc "github.com/centrifuge/go-centrifuge/bootstrap/bootstrappers/testingbootstrap"
	"github.com/centrifuge/go-centrifuge/config"
	"github.com/centrifuge/go-centrifuge/testingutils/config"
	"github.com/centrifuge/go-centrifuge/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"

	"golang.org/x/crypto/blake2b"
)

var (
	anchorRepo anchors.AnchorRepository
	cfg        config.Configuration
)

func TestMain(m *testing.M) {
	ctx := cc.TestFunctionalEthereumBootstrap()
	anchorRepo = ctx[anchors.BootstrappedAnchorRepo].(anchors.AnchorRepository)
	cfg = ctx[bootstrap.BootstrappedConfig].(config.Configuration)
	result := m.Run()
	cc.TestFunctionalEthereumTearDown()
	os.Exit(result)
}

func TestPreCommitAnchor_Integration(t *testing.T) {
	t.Parallel()
	anchorID := utils.RandomSlice(32)
	signingRoot := utils.RandomSlice(32)
	preCommitAnchor(t, anchorID, signingRoot)
}

func TestPreCommit_CommitAnchor_Integration(t *testing.T) {
	t.Parallel()
	anchorIDPreImage := utils.RandomSlice(32)
	h, err := blake2b.New256(nil)
	assert.NoError(t, err)
	_, err = h.Write(anchorIDPreImage)
	assert.NoError(t, err)
	var anchorID []byte
	anchorID = h.Sum(anchorID)
	proofStr := []string{"0xc0c38dd1635b279af306bc04900559fc346970ad8f654106bfced202b067a10e"}
	signingRootStr := "0x3f274cf97a0c166e6e3fa1c10a3353e260b3cb162aff873fa01a49deafc65ec8"
	documentRootStr := "0xeefa76542337d4c1456819f4f01f362455ab0c47f7514a0a7f7fb99efd64ce82"

	signingRoot, err := hexutil.Decode(signingRootStr)
	assert.NoError(t, err)

	documentRoot, err := hexutil.Decode(documentRootStr)
	assert.NoError(t, err)

	proof1, err := hexutil.Decode(proofStr[0])
	assert.NoError(t, err)

	var proofB1 [32]byte
	copy(proofB1[:], proof1)

	anchorIDTyped, err := anchors.ToAnchorID(anchorID)
	assert.NoError(t, err)
	preCommitAnchor(t, anchorID, signingRoot)

	docRootTyped, _ := anchors.ToDocumentRoot(documentRoot)
	commitAnchor(t, anchorIDPreImage, documentRoot, proofB1)
	gotDocRoot, _, err := anchorRepo.GetAnchorData(anchorIDTyped)
	assert.Nil(t, err)
	assert.Equal(t, docRootTyped, gotDocRoot)
}

func TestCommitAnchor_Integration(t *testing.T) {
	t.Parallel()
	anchorIDPreImage := utils.RandomSlice(32)
	h, err := blake2b.New256(nil)
	assert.NoError(t, err)
	_, err = h.Write(anchorIDPreImage)
	assert.NoError(t, err)
	var anchorID []byte
	anchorID = h.Sum(nil)
	documentRoot := utils.RandomSlice(32)

	anchorIDTyped, err := anchors.ToAnchorID(anchorID)
	assert.NoError(t, err)
	docRootTyped, _ := anchors.ToDocumentRoot(documentRoot)
	commitAnchor(t, anchorIDPreImage, documentRoot, utils.RandomByte32())
	gotDocRoot, hval, err := anchorRepo.GetAnchorData(anchorIDTyped)
	assert.Nil(t, err)
	assert.Equal(t, docRootTyped, gotDocRoot)
	assert.True(t, time.Now().After(hval))
}

func commitAnchor(t *testing.T, anchorID, documentRoot []byte, documentProof [32]byte) {
	anchorIDTyped, err := anchors.ToAnchorID(anchorID)
	assert.NoError(t, err)
	docRootTyped, _ := anchors.ToDocumentRoot(documentRoot)

	ctx := testingconfig.CreateAccountContext(t, cfg)
	done, err := anchorRepo.CommitAnchor(ctx, anchorIDTyped, docRootTyped, documentProof)
	assert.Nil(t, err)
	doneErr := <-done
	assert.NoError(t, doneErr, "no error")
}

func preCommitAnchor(t *testing.T, anchorID, documentRoot []byte) {
	anchorIDTyped, err := anchors.ToAnchorID(anchorID)
	assert.NoError(t, err)
	docRootTyped, _ := anchors.ToDocumentRoot(documentRoot)

	ctx := testingconfig.CreateAccountContext(t, cfg)
	done, err := anchorRepo.PreCommitAnchor(ctx, anchorIDTyped, docRootTyped)
	assert.Nil(t, err)
	doneErr := <-done
	assert.NoError(t, doneErr, "no error")
}

func TestPreCommitAnchor_Integration_Concurrent(t *testing.T) {
	t.Parallel()

	var doneList [5]chan error

	ctx := testingconfig.CreateAccountContext(t, cfg)

	for ix := 0; ix < 5; ix++ {
		anchorID := utils.RandomSlice(32)
		signingRoot := utils.RandomSlice(32)
		anchorIDTyped, err := anchors.ToAnchorID(anchorID)
		assert.NoError(t, err)
		docRootTyped, _ := anchors.ToDocumentRoot(signingRoot)
		doneList[ix], err = anchorRepo.PreCommitAnchor(ctx, anchorIDTyped, docRootTyped)
		if err != nil {
			t.Fatalf("Error precommit anchor %v", err)
		}
	}

	for ix := 0; ix < 5; ix++ {
		doneErr := <-doneList[ix]
		assert.NoError(t, doneErr)
	}

}

func TestCommitAnchor_Integration_Concurrent(t *testing.T) {
	t.Parallel()
	var commitDataList [5]*anchors.CommitData
	var doneList [5]chan error

	for ix := 0; ix < 5; ix++ {
		anchorIDPreImage := utils.RandomSlice(32)
		anchorIDPreImageID, err := anchors.ToAnchorID(anchorIDPreImage)
		assert.NoError(t, err)
		h, err := blake2b.New256(nil)
		assert.NoError(t, err)
		_, err = h.Write(anchorIDPreImage)
		assert.NoError(t, err)
		var cAnchorId []byte
		cAnchorId = h.Sum(cAnchorId)
		currentAnchorId, err := anchors.ToAnchorID(cAnchorId)
		assert.NoError(t, err)
		currentDocumentRoot := utils.RandomByte32()
		documentProof := utils.RandomByte32()
		commitDataList[ix] = anchors.NewCommitData(currentAnchorId, currentDocumentRoot, documentProof)
		ctx := testingconfig.CreateAccountContext(t, cfg)
		doneList[ix], err = anchorRepo.CommitAnchor(ctx, anchorIDPreImageID, currentDocumentRoot, documentProof)
		if err != nil {
			t.Fatalf("Error commit Anchor %v", err)
		}
		assert.Nil(t, err, "should not error out upon anchor registration")
	}

	for ix := 0; ix < 5; ix++ {
		doneErr := <-doneList[ix]
		assert.NoError(t, doneErr)
		anchorID := commitDataList[ix].AnchorID
		docRoot := commitDataList[ix].DocumentRoot
		gotDocRoot, _, err := anchorRepo.GetAnchorData(anchorID)
		assert.Nil(t, err)
		assert.Equal(t, docRoot, gotDocRoot)
	}
}
