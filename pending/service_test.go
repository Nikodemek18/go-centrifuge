// +build unit

package pending

import (
	"context"
	"testing"

	"github.com/centrifuge/go-centrifuge/contextutil"
	"github.com/centrifuge/go-centrifuge/documents"
	"github.com/centrifuge/go-centrifuge/errors"
	"github.com/centrifuge/go-centrifuge/identity"
	"github.com/centrifuge/go-centrifuge/jobs"
	testingconfig "github.com/centrifuge/go-centrifuge/testingutils/config"
	testingdocuments "github.com/centrifuge/go-centrifuge/testingutils/documents"
	testingidentity "github.com/centrifuge/go-centrifuge/testingutils/identity"
	"github.com/centrifuge/go-centrifuge/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
	Repository
}

func (m *mockRepo) Get(accID, id []byte) (documents.Model, error) {
	args := m.Called(accID, id)
	doc, _ := args.Get(0).(documents.Model)
	return doc, args.Error(1)
}

func (m *mockRepo) Delete(accID, id []byte) error {
	args := m.Called(accID, id)
	return args.Error(0)
}

func (m *mockRepo) Create(accID, id []byte, doc documents.Model) error {
	args := m.Called(accID, id, doc)
	return args.Error(0)
}

func (m *mockRepo) Update(accID, id []byte, doc documents.Model) error {
	args := m.Called(accID, id, doc)
	return args.Error(0)
}

func TestService_Commit(t *testing.T) {
	s := service{}

	// missing did
	ctx := context.Background()
	docID := utils.RandomSlice(32)
	_, _, err := s.Commit(ctx, docID)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// missing model
	ctx = testingconfig.CreateAccountContext(t, cfg)
	repo := new(mockRepo)
	repo.On("Get", did[:], docID).Return(nil, errors.New("not found")).Once()
	s.pendingRepo = repo
	_, _, err = s.Commit(ctx, docID)
	assert.Error(t, err)

	// failed commit
	doc := new(documents.MockModel)
	repo.On("Get", did[:], docID).Return(doc, nil)
	docSrv := new(testingdocuments.MockService)
	docSrv.On("Commit", ctx, doc).Return(nil, errors.New("failed to commit")).Once()
	s.docSrv = docSrv
	_, _, err = s.Commit(ctx, docID)
	assert.Error(t, err)

	// success
	jobID := jobs.NewJobID()
	repo.On("Delete", did[:], docID).Return(nil)
	docSrv.On("Commit", ctx, doc).Return(jobID, nil)
	m, jid, err := s.Commit(ctx, docID)
	assert.NoError(t, err)
	assert.Equal(t, jobID, jid)
	assert.NotNil(t, m)
	docSrv.AssertExpectations(t)
	doc.AssertExpectations(t)
}

func TestService_Create(t *testing.T) {
	s := service{}

	// missing did
	ctx := context.Background()
	payload := documents.UpdatePayload{}
	_, err := s.Create(ctx, payload)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// derive failed
	ctx = testingconfig.CreateAccountContext(t, cfg)
	docSrv := new(testingdocuments.MockService)
	docSrv.On("Derive", ctx, payload).Return(nil, errors.New("failed to derive")).Once()
	s.docSrv = docSrv
	_, err = s.Create(ctx, payload)
	assert.Error(t, err)

	// already existing document
	payload.DocumentID = utils.RandomSlice(32)
	repo := new(mockRepo)
	repo.On("Get", did[:], payload.DocumentID).Return(new(documents.MockModel), nil).Once()
	s.pendingRepo = repo
	_, err = s.Create(ctx, payload)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(ErrPendingDocumentExists, err))

	// success
	repo.On("Get", did[:], payload.DocumentID).Return(nil, errors.New("missing")).Once()
	doc := new(documents.MockModel)
	doc.On("ID").Return(payload.DocumentID).Once()
	repo.On("Create", did[:], payload.DocumentID, doc).Return(nil).Once()
	docSrv.On("Derive", ctx, payload).Return(doc, nil).Once()
	gdoc, err := s.Create(ctx, payload)
	assert.NoError(t, err)
	assert.Equal(t, doc, gdoc)
	doc.AssertExpectations(t)
	docSrv.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestService_Get(t *testing.T) {
	// not pending document
	st := documents.Committed
	s := service{}
	ctx := context.Background()
	docID := utils.RandomSlice(32)
	docSrv := new(testingdocuments.MockService)
	docSrv.On("GetCurrentVersion", docID).Return(new(documents.MockModel), nil).Once()
	s.docSrv = docSrv
	doc, err := s.Get(ctx, docID, st)
	assert.NoError(t, err)
	assert.NotNil(t, doc)

	// pending doc
	// missing did from context
	st = documents.Pending
	_, err = s.Get(ctx, docID, st)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// success
	repo := new(mockRepo)
	repo.On("Get", did[:], docID).Return(doc, nil).Once()
	s.pendingRepo = repo
	ctx = testingconfig.CreateAccountContext(t, cfg)
	gdoc, err := s.Get(ctx, docID, st)
	assert.NoError(t, err)
	assert.Equal(t, doc, gdoc)
	docSrv.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestService_GetVersion(t *testing.T) {
	// found in docService
	doc := new(documents.MockModel)
	docID, versionID := utils.RandomSlice(32), utils.RandomSlice(32)
	s := service{}
	ctx := context.Background()
	docSrv := new(testingdocuments.MockService)
	docSrv.On("GetVersion", docID, versionID).Return(doc, nil).Once()
	s.docSrv = docSrv
	gdoc, err := s.GetVersion(ctx, docID, versionID)
	assert.NoError(t, err)
	assert.Equal(t, doc, gdoc)

	// not found in docService
	// ctx with no did
	docSrv.On("GetVersion", docID, versionID).Return(nil, documents.ErrDocumentNotFound)
	_, err = s.GetVersion(ctx, docID, versionID)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// different current version
	ctx = testingconfig.CreateAccountContext(t, cfg)
	repo := new(mockRepo)
	repo.On("Get", did[:], docID).Return(doc, nil)
	doc.On("CurrentVersion").Return(utils.RandomSlice(32)).Once()
	s.pendingRepo = repo
	_, err = s.GetVersion(ctx, docID, versionID)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(documents.ErrDocumentNotFound, err))

	// successful retrieval
	doc.On("CurrentVersion").Return(versionID).Once()
	gdoc, err = s.GetVersion(ctx, docID, versionID)
	assert.NoError(t, err)
	assert.Equal(t, doc, gdoc)
	docSrv.AssertExpectations(t)
	repo.AssertExpectations(t)
	doc.AssertExpectations(t)
}

func TestService_Update(t *testing.T) {
	s := service{}

	// missing did
	ctx := context.Background()
	payload := documents.UpdatePayload{}
	_, err := s.Update(ctx, payload)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// document doesnt exist yet
	repo := new(mockRepo)
	repo.On("Get", did[:], payload.DocumentID).Return(nil, errors.New("not found")).Once()
	s.pendingRepo = repo
	ctx = testingconfig.CreateAccountContext(t, cfg)
	_, err = s.Update(ctx, payload)
	assert.Error(t, err)

	// Patch error
	oldModel := new(documents.MockModel)
	oldModel.On("Patch", payload).Return(errors.New("error patching")).Once()
	repo.On("Get", did[:], payload.DocumentID).Return(oldModel, nil)
	_, err = s.Update(ctx, payload)
	assert.Error(t, err)

	// Success
	oldModel.On("ID").Return(payload.DocumentID).Once()
	oldModel.On("Patch", payload).Return(nil).Once()
	repo.On("Update", did[:], payload.DocumentID, oldModel).Return(nil).Once()
	_, err = s.Update(ctx, payload)
	assert.NoError(t, err)
}

func TestService_AddSignedAttribute(t *testing.T) {
	s := service{}
	label := "signed_attribute"
	value := utils.RandomSlice(32)
	docID := utils.RandomSlice(32)
	versionID := utils.RandomSlice(32)

	// missing did
	ctx := context.Background()
	_, err := s.AddSignedAttribute(ctx, docID, label, value)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// missing document
	ctx = testingconfig.CreateAccountContext(t, cfg)
	prepo := new(mockRepo)
	prepo.On("Get", did[:], docID).Return(nil, errors.New("Missing")).Once()
	s.pendingRepo = prepo
	_, err = s.AddSignedAttribute(ctx, docID, label, value)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(documents.ErrDocumentNotFound, err))

	// failed to get new attribute
	doc := new(documents.MockModel)
	doc.On("ID").Return(docID)
	doc.On("CurrentVersion").Return(versionID)
	prepo.On("Get", did[:], docID).Return(doc, nil)
	_, err = s.AddSignedAttribute(ctx, docID, "", value)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(documents.ErrEmptyAttrLabel, err))

	// failed to add attribute to document
	doc.On("AddAttributes", mock.Anything, false, mock.Anything).Return(errors.New("failed to add")).Once()
	_, err = s.AddSignedAttribute(ctx, docID, label, value)
	assert.Error(t, err)

	// success
	doc.On("AddAttributes", mock.Anything, false, mock.Anything).Return(nil).Once()
	prepo.On("Update", did[:], docID, doc).Return(nil).Once()
	doc1, err := s.AddSignedAttribute(ctx, docID, label, value)
	assert.NoError(t, err)
	assert.Equal(t, doc, doc1)
	prepo.AssertExpectations(t)
	doc.AssertExpectations(t)
}

func TestService_RemoveCollaborators(t *testing.T) {
	s := service{}

	// missing did from context
	ctx := context.Background()
	docID := utils.RandomSlice(32)
	_, err := s.RemoveCollaborators(ctx, docID, nil)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(contextutil.ErrDIDMissingFromContext, err))

	// missing doc
	ctx = testingconfig.CreateAccountContext(t, cfg)
	repo := new(mockRepo)
	repo.On("Get", did[:], docID).Return(nil, errors.New("failed")).Once()
	s.pendingRepo = repo
	_, err = s.RemoveCollaborators(ctx, docID, nil)
	assert.Error(t, err)
	assert.True(t, errors.IsOfType(documents.ErrDocumentNotFound, err))

	// failed to remove collaborators
	d := new(documents.MockModel)
	d.On("RemoveCollaborators", mock.Anything).Return(errors.New("failed")).Once()
	repo.On("Get", did[:], docID).Return(d, nil)
	_, err = s.RemoveCollaborators(ctx, docID, []identity.DID{testingidentity.GenerateRandomDID()})
	assert.Error(t, err)

	// success
	d.On("RemoveCollaborators", mock.Anything).Return(nil)
	repo.On("Update", did[:], docID, d).Return(nil).Once()
	d1, err := s.RemoveCollaborators(ctx, docID, []identity.DID{testingidentity.GenerateRandomDID()})
	assert.NoError(t, err)
	assert.Equal(t, d, d1)
	repo.AssertExpectations(t)
	d.AssertExpectations(t)
}
