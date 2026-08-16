package chain

import (
	"context"
	"math/big"
	"testing"

	"CredChain_Golang/infrastructure/chain/contracts"
	"CredChain_Golang/tests/fixtures"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRegistryService_IssueCredentials_PassesBusinessDatesToContract asserts
// issuedAt and expiresAt actually reach the contract call arguments instead
// of silently defaulting to zero values.
func TestRegistryService_IssueCredentials_PassesBusinessDatesToContract(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}

	regMock.On("UserToNonce", mock.Anything, mock.Anything).Return(big.NewInt(0), nil)
	var got contracts.CredentialRegistryBatchIssueCredentialsWithSignatureParams
	regMock.On("BatchIssueCredentialsWithSignature", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			got = args.Get(1).(contracts.CredentialRegistryBatchIssueCredentialsWithSignatureParams)
		}).
		Return(&types.Transaction{}, nil)

	svc := NewRegistryService(mkClient(authMock, regMock), mkConfig())
	rs := svc.(*registryService)
	rs.waitMined = func(ctx context.Context, b bind.DeployBackend, tx *types.Transaction) (*types.Receipt, error) {
		return &types.Receipt{Status: 1}, nil
	}

	wallet := fixtures.NewWallet(t)
	issuedAt := uint64(1750000000)
	expiry := uint64(1735689600)

	ids, err := svc.IssueCredentials(context.Background(), wallet,
		CredentialIssuance{HolderAddress: wallet.Address, Hash: "0xhash1", URI: "u1", IssuedAt: issuedAt, ExpiresAt: expiry},
		CredentialIssuance{HolderAddress: wallet.Address, Hash: "0xhash2", URI: "u2", IssuedAt: 0, ExpiresAt: 0},
	)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	require.Len(t, got.Credentials, 2)
	assert.Equal(t, issuedAt, got.Credentials[0].IssuedAt, "issuedAt must reach the contract call args")
	assert.Equal(t, expiry, got.Credentials[0].ExpiresAt, "expiresAt must reach the contract call args, not default to 0")
	assert.Equal(t, uint64(0), got.Credentials[1].IssuedAt)
	assert.Equal(t, uint64(0), got.Credentials[1].ExpiresAt, "no-expiry issuance writes 0 on chain")
}
