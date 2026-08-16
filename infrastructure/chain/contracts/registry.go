// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// CredentialRegistryBatchIssueCredentialsWithSignatureParams is an auto generated low-level Go binding around an user-defined struct.
type CredentialRegistryBatchIssueCredentialsWithSignatureParams struct {
	Issuer      common.Address
	Credentials []CredentialRegistryCredentialIssuance
	Nonce       *big.Int
	Signature   []byte
}

// CredentialRegistryBatchRevokeCredentialsWithSignatureParams is an auto generated low-level Go binding around an user-defined struct.
type CredentialRegistryBatchRevokeCredentialsWithSignatureParams struct {
	Revoker       common.Address
	CredentialIds []*big.Int
	Nonce         *big.Int
	Signature     []byte
}

// CredentialRegistryCredential is an auto generated low-level Go binding around an user-defined struct.
type CredentialRegistryCredential struct {
	Id        *big.Int
	Holder    common.Address
	Hash      string
	Issuer    common.Address
	Revoker   common.Address
	IssuedAt  *big.Int
	RevokedAt *big.Int
	ExpiresAt *big.Int
	Uri       string
}

// CredentialRegistryCredentialHashStatus is an auto generated low-level Go binding around an user-defined struct.
type CredentialRegistryCredentialHashStatus struct {
	Hash   [32]byte
	Status uint8
}

// CredentialRegistryCredentialIssuance is an auto generated low-level Go binding around an user-defined struct.
type CredentialRegistryCredentialIssuance struct {
	Holder    common.Address
	Hash      string
	Uri       string
	IssuedAt  uint64
	ExpiresAt uint64
}

// RegistryMetaData contains all meta data concerning the Registry contract.
var RegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"CredentialNotFoundError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CredentialTransferError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ECDSAInvalidSignature\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"ECDSAInvalidSignatureLength\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"ECDSAInvalidSignatureS\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"ERC721IncorrectOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ERC721InsufficientApproval\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"approver\",\"type\":\"address\"}],\"name\":\"ERC721InvalidApprover\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"ERC721InvalidOperator\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"ERC721InvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"}],\"name\":\"ERC721InvalidReceiver\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"ERC721InvalidSender\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ERC721NonexistentToken\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAddressError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidNonceError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignatureError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IssuedCredentialError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxBatchExceededError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDeployerError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RevokeRevokedCredentialError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleBelowAdminError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleBelowIssuerError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleNotSuperAdminError\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"approved\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"issuedAt\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"expiresAt\",\"type\":\"uint64\"}],\"name\":\"CredentialIssued\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"}],\"name\":\"CredentialRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_BATCH_CREDENTIAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"issuedAt\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"expiresAt\",\"type\":\"uint64\"}],\"internalType\":\"structCredentialRegistry.CredentialIssuance[]\",\"name\":\"credentials\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structCredentialRegistry.BatchIssueCredentialsWithSignatureParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"batchIssueCredentialsWithSignature\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"credentialIds\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structCredentialRegistry.BatchRevokeCredentialsWithSignatureParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"batchRevokeCredentialsWithSignature\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"config\",\"outputs\":[{\"internalType\":\"contractCredentialConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"credentialHashToStatus\",\"outputs\":[{\"internalType\":\"enumCredentialRegistry.CredentialStatus\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"credentialIdToCredential\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"issuedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"revokedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiresAt\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"credentials\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"findCredential\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"issuedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"revokedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiresAt\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"}],\"internalType\":\"structCredentialRegistry.Credential\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"hashes\",\"type\":\"bytes32[]\"}],\"name\":\"getCredentialHashStatuses\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"enumCredentialRegistry.CredentialStatus\",\"name\":\"status\",\"type\":\"uint8\"}],\"internalType\":\"structCredentialRegistry.CredentialHashStatus[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"}],\"name\":\"getCredentialsByIds\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"issuedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"revokedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiresAt\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"}],\"internalType\":\"structCredentialRegistry.Credential[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"holderToCredentialIds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_config\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"credentialIds\",\"type\":\"uint256[]\"}],\"name\":\"isHolderOfCredentialIds\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"limit\",\"type\":\"uint256\"}],\"name\":\"paginateCredentials\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"issuedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"revokedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiresAt\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"}],\"internalType\":\"structCredentialRegistry.Credential[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"limit\",\"type\":\"uint256\"}],\"name\":\"paginateCredentialsByHolder\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"holder\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"hash\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"issuer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"issuedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"revokedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiresAt\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"uri\",\"type\":\"string\"}],\"internalType\":\"structCredentialRegistry.Credential[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"tokenURI\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userToNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// MAXBATCHCREDENTIAL is a free data retrieval call binding the contract method 0x55e469d5.
//
// Solidity: function MAX_BATCH_CREDENTIAL() view returns(uint256)
func (_Registry *RegistryCaller) MAXBATCHCREDENTIAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "MAX_BATCH_CREDENTIAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBATCHCREDENTIAL is a free data retrieval call binding the contract method 0x55e469d5.
//
// Solidity: function MAX_BATCH_CREDENTIAL() view returns(uint256)
func (_Registry *RegistrySession) MAXBATCHCREDENTIAL() (*big.Int, error) {
	return _Registry.Contract.MAXBATCHCREDENTIAL(&_Registry.CallOpts)
}

// MAXBATCHCREDENTIAL is a free data retrieval call binding the contract method 0x55e469d5.
//
// Solidity: function MAX_BATCH_CREDENTIAL() view returns(uint256)
func (_Registry *RegistryCallerSession) MAXBATCHCREDENTIAL() (*big.Int, error) {
	return _Registry.Contract.MAXBATCHCREDENTIAL(&_Registry.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Registry *RegistryCaller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Registry *RegistrySession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _Registry.Contract.BalanceOf(&_Registry.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Registry *RegistryCallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _Registry.Contract.BalanceOf(&_Registry.CallOpts, owner)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Registry *RegistryCaller) Config(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "config")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Registry *RegistrySession) Config() (common.Address, error) {
	return _Registry.Contract.Config(&_Registry.CallOpts)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Registry *RegistryCallerSession) Config() (common.Address, error) {
	return _Registry.Contract.Config(&_Registry.CallOpts)
}

// CredentialHashToStatus is a free data retrieval call binding the contract method 0x98e564c2.
//
// Solidity: function credentialHashToStatus(bytes32 ) view returns(uint8)
func (_Registry *RegistryCaller) CredentialHashToStatus(opts *bind.CallOpts, arg0 [32]byte) (uint8, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "credentialHashToStatus", arg0)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// CredentialHashToStatus is a free data retrieval call binding the contract method 0x98e564c2.
//
// Solidity: function credentialHashToStatus(bytes32 ) view returns(uint8)
func (_Registry *RegistrySession) CredentialHashToStatus(arg0 [32]byte) (uint8, error) {
	return _Registry.Contract.CredentialHashToStatus(&_Registry.CallOpts, arg0)
}

// CredentialHashToStatus is a free data retrieval call binding the contract method 0x98e564c2.
//
// Solidity: function credentialHashToStatus(bytes32 ) view returns(uint8)
func (_Registry *RegistryCallerSession) CredentialHashToStatus(arg0 [32]byte) (uint8, error) {
	return _Registry.Contract.CredentialHashToStatus(&_Registry.CallOpts, arg0)
}

// CredentialIdToCredential is a free data retrieval call binding the contract method 0xe112e1fd.
//
// Solidity: function credentialIdToCredential(uint256 ) view returns(uint256 id, address holder, string hash, address issuer, address revoker, uint256 issuedAt, uint256 revokedAt, uint256 expiresAt, string uri)
func (_Registry *RegistryCaller) CredentialIdToCredential(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Id        *big.Int
	Holder    common.Address
	Hash      string
	Issuer    common.Address
	Revoker   common.Address
	IssuedAt  *big.Int
	RevokedAt *big.Int
	ExpiresAt *big.Int
	Uri       string
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "credentialIdToCredential", arg0)

	outstruct := new(struct {
		Id        *big.Int
		Holder    common.Address
		Hash      string
		Issuer    common.Address
		Revoker   common.Address
		IssuedAt  *big.Int
		RevokedAt *big.Int
		ExpiresAt *big.Int
		Uri       string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Id = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Holder = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Hash = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.Issuer = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Revoker = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.IssuedAt = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.RevokedAt = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.ExpiresAt = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)
	outstruct.Uri = *abi.ConvertType(out[8], new(string)).(*string)

	return *outstruct, err

}

// CredentialIdToCredential is a free data retrieval call binding the contract method 0xe112e1fd.
//
// Solidity: function credentialIdToCredential(uint256 ) view returns(uint256 id, address holder, string hash, address issuer, address revoker, uint256 issuedAt, uint256 revokedAt, uint256 expiresAt, string uri)
func (_Registry *RegistrySession) CredentialIdToCredential(arg0 *big.Int) (struct {
	Id        *big.Int
	Holder    common.Address
	Hash      string
	Issuer    common.Address
	Revoker   common.Address
	IssuedAt  *big.Int
	RevokedAt *big.Int
	ExpiresAt *big.Int
	Uri       string
}, error) {
	return _Registry.Contract.CredentialIdToCredential(&_Registry.CallOpts, arg0)
}

// CredentialIdToCredential is a free data retrieval call binding the contract method 0xe112e1fd.
//
// Solidity: function credentialIdToCredential(uint256 ) view returns(uint256 id, address holder, string hash, address issuer, address revoker, uint256 issuedAt, uint256 revokedAt, uint256 expiresAt, string uri)
func (_Registry *RegistryCallerSession) CredentialIdToCredential(arg0 *big.Int) (struct {
	Id        *big.Int
	Holder    common.Address
	Hash      string
	Issuer    common.Address
	Revoker   common.Address
	IssuedAt  *big.Int
	RevokedAt *big.Int
	ExpiresAt *big.Int
	Uri       string
}, error) {
	return _Registry.Contract.CredentialIdToCredential(&_Registry.CallOpts, arg0)
}

// Credentials is a free data retrieval call binding the contract method 0xe0574e3f.
//
// Solidity: function credentials(uint256 ) view returns(uint256)
func (_Registry *RegistryCaller) Credentials(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "credentials", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Credentials is a free data retrieval call binding the contract method 0xe0574e3f.
//
// Solidity: function credentials(uint256 ) view returns(uint256)
func (_Registry *RegistrySession) Credentials(arg0 *big.Int) (*big.Int, error) {
	return _Registry.Contract.Credentials(&_Registry.CallOpts, arg0)
}

// Credentials is a free data retrieval call binding the contract method 0xe0574e3f.
//
// Solidity: function credentials(uint256 ) view returns(uint256)
func (_Registry *RegistryCallerSession) Credentials(arg0 *big.Int) (*big.Int, error) {
	return _Registry.Contract.Credentials(&_Registry.CallOpts, arg0)
}

// FindCredential is a free data retrieval call binding the contract method 0x7a0876cd.
//
// Solidity: function findCredential(uint256 id) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string))
func (_Registry *RegistryCaller) FindCredential(opts *bind.CallOpts, id *big.Int) (CredentialRegistryCredential, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "findCredential", id)

	if err != nil {
		return *new(CredentialRegistryCredential), err
	}

	out0 := *abi.ConvertType(out[0], new(CredentialRegistryCredential)).(*CredentialRegistryCredential)

	return out0, err

}

// FindCredential is a free data retrieval call binding the contract method 0x7a0876cd.
//
// Solidity: function findCredential(uint256 id) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string))
func (_Registry *RegistrySession) FindCredential(id *big.Int) (CredentialRegistryCredential, error) {
	return _Registry.Contract.FindCredential(&_Registry.CallOpts, id)
}

// FindCredential is a free data retrieval call binding the contract method 0x7a0876cd.
//
// Solidity: function findCredential(uint256 id) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string))
func (_Registry *RegistryCallerSession) FindCredential(id *big.Int) (CredentialRegistryCredential, error) {
	return _Registry.Contract.FindCredential(&_Registry.CallOpts, id)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Registry *RegistryCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Registry *RegistrySession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Registry.Contract.GetApproved(&_Registry.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Registry *RegistryCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Registry.Contract.GetApproved(&_Registry.CallOpts, tokenId)
}

// GetCredentialHashStatuses is a free data retrieval call binding the contract method 0x784fde2b.
//
// Solidity: function getCredentialHashStatuses(bytes32[] hashes) view returns((bytes32,uint8)[])
func (_Registry *RegistryCaller) GetCredentialHashStatuses(opts *bind.CallOpts, hashes [][32]byte) ([]CredentialRegistryCredentialHashStatus, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getCredentialHashStatuses", hashes)

	if err != nil {
		return *new([]CredentialRegistryCredentialHashStatus), err
	}

	out0 := *abi.ConvertType(out[0], new([]CredentialRegistryCredentialHashStatus)).(*[]CredentialRegistryCredentialHashStatus)

	return out0, err

}

// GetCredentialHashStatuses is a free data retrieval call binding the contract method 0x784fde2b.
//
// Solidity: function getCredentialHashStatuses(bytes32[] hashes) view returns((bytes32,uint8)[])
func (_Registry *RegistrySession) GetCredentialHashStatuses(hashes [][32]byte) ([]CredentialRegistryCredentialHashStatus, error) {
	return _Registry.Contract.GetCredentialHashStatuses(&_Registry.CallOpts, hashes)
}

// GetCredentialHashStatuses is a free data retrieval call binding the contract method 0x784fde2b.
//
// Solidity: function getCredentialHashStatuses(bytes32[] hashes) view returns((bytes32,uint8)[])
func (_Registry *RegistryCallerSession) GetCredentialHashStatuses(hashes [][32]byte) ([]CredentialRegistryCredentialHashStatus, error) {
	return _Registry.Contract.GetCredentialHashStatuses(&_Registry.CallOpts, hashes)
}

// GetCredentialsByIds is a free data retrieval call binding the contract method 0xfb87720d.
//
// Solidity: function getCredentialsByIds(uint256[] ids) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCaller) GetCredentialsByIds(opts *bind.CallOpts, ids []*big.Int) ([]CredentialRegistryCredential, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getCredentialsByIds", ids)

	if err != nil {
		return *new([]CredentialRegistryCredential), err
	}

	out0 := *abi.ConvertType(out[0], new([]CredentialRegistryCredential)).(*[]CredentialRegistryCredential)

	return out0, err

}

// GetCredentialsByIds is a free data retrieval call binding the contract method 0xfb87720d.
//
// Solidity: function getCredentialsByIds(uint256[] ids) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistrySession) GetCredentialsByIds(ids []*big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.GetCredentialsByIds(&_Registry.CallOpts, ids)
}

// GetCredentialsByIds is a free data retrieval call binding the contract method 0xfb87720d.
//
// Solidity: function getCredentialsByIds(uint256[] ids) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCallerSession) GetCredentialsByIds(ids []*big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.GetCredentialsByIds(&_Registry.CallOpts, ids)
}

// HolderToCredentialIds is a free data retrieval call binding the contract method 0xf464ff94.
//
// Solidity: function holderToCredentialIds(address , uint256 ) view returns(uint256)
func (_Registry *RegistryCaller) HolderToCredentialIds(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "holderToCredentialIds", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HolderToCredentialIds is a free data retrieval call binding the contract method 0xf464ff94.
//
// Solidity: function holderToCredentialIds(address , uint256 ) view returns(uint256)
func (_Registry *RegistrySession) HolderToCredentialIds(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Registry.Contract.HolderToCredentialIds(&_Registry.CallOpts, arg0, arg1)
}

// HolderToCredentialIds is a free data retrieval call binding the contract method 0xf464ff94.
//
// Solidity: function holderToCredentialIds(address , uint256 ) view returns(uint256)
func (_Registry *RegistryCallerSession) HolderToCredentialIds(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Registry.Contract.HolderToCredentialIds(&_Registry.CallOpts, arg0, arg1)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Registry *RegistryCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isApprovedForAll", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Registry *RegistrySession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _Registry.Contract.IsApprovedForAll(&_Registry.CallOpts, owner, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Registry *RegistryCallerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _Registry.Contract.IsApprovedForAll(&_Registry.CallOpts, owner, operator)
}

// IsHolderOfCredentialIds is a free data retrieval call binding the contract method 0xadc94c56.
//
// Solidity: function isHolderOfCredentialIds(address holder, uint256[] credentialIds) view returns(bool)
func (_Registry *RegistryCaller) IsHolderOfCredentialIds(opts *bind.CallOpts, holder common.Address, credentialIds []*big.Int) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isHolderOfCredentialIds", holder, credentialIds)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsHolderOfCredentialIds is a free data retrieval call binding the contract method 0xadc94c56.
//
// Solidity: function isHolderOfCredentialIds(address holder, uint256[] credentialIds) view returns(bool)
func (_Registry *RegistrySession) IsHolderOfCredentialIds(holder common.Address, credentialIds []*big.Int) (bool, error) {
	return _Registry.Contract.IsHolderOfCredentialIds(&_Registry.CallOpts, holder, credentialIds)
}

// IsHolderOfCredentialIds is a free data retrieval call binding the contract method 0xadc94c56.
//
// Solidity: function isHolderOfCredentialIds(address holder, uint256[] credentialIds) view returns(bool)
func (_Registry *RegistryCallerSession) IsHolderOfCredentialIds(holder common.Address, credentialIds []*big.Int) (bool, error) {
	return _Registry.Contract.IsHolderOfCredentialIds(&_Registry.CallOpts, holder, credentialIds)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Registry *RegistryCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Registry *RegistrySession) Name() (string, error) {
	return _Registry.Contract.Name(&_Registry.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Registry *RegistryCallerSession) Name() (string, error) {
	return _Registry.Contract.Name(&_Registry.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Registry *RegistryCaller) OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "ownerOf", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Registry *RegistrySession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _Registry.Contract.OwnerOf(&_Registry.CallOpts, tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Registry *RegistryCallerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _Registry.Contract.OwnerOf(&_Registry.CallOpts, tokenId)
}

// PaginateCredentials is a free data retrieval call binding the contract method 0x120c1e93.
//
// Solidity: function paginateCredentials(uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCaller) PaginateCredentials(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "paginateCredentials", offset, limit)

	if err != nil {
		return *new([]CredentialRegistryCredential), err
	}

	out0 := *abi.ConvertType(out[0], new([]CredentialRegistryCredential)).(*[]CredentialRegistryCredential)

	return out0, err

}

// PaginateCredentials is a free data retrieval call binding the contract method 0x120c1e93.
//
// Solidity: function paginateCredentials(uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistrySession) PaginateCredentials(offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.PaginateCredentials(&_Registry.CallOpts, offset, limit)
}

// PaginateCredentials is a free data retrieval call binding the contract method 0x120c1e93.
//
// Solidity: function paginateCredentials(uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCallerSession) PaginateCredentials(offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.PaginateCredentials(&_Registry.CallOpts, offset, limit)
}

// PaginateCredentialsByHolder is a free data retrieval call binding the contract method 0x7026b14d.
//
// Solidity: function paginateCredentialsByHolder(address holder, uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCaller) PaginateCredentialsByHolder(opts *bind.CallOpts, holder common.Address, offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "paginateCredentialsByHolder", holder, offset, limit)

	if err != nil {
		return *new([]CredentialRegistryCredential), err
	}

	out0 := *abi.ConvertType(out[0], new([]CredentialRegistryCredential)).(*[]CredentialRegistryCredential)

	return out0, err

}

// PaginateCredentialsByHolder is a free data retrieval call binding the contract method 0x7026b14d.
//
// Solidity: function paginateCredentialsByHolder(address holder, uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistrySession) PaginateCredentialsByHolder(holder common.Address, offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.PaginateCredentialsByHolder(&_Registry.CallOpts, holder, offset, limit)
}

// PaginateCredentialsByHolder is a free data retrieval call binding the contract method 0x7026b14d.
//
// Solidity: function paginateCredentialsByHolder(address holder, uint256 offset, uint256 limit) view returns((uint256,address,string,address,address,uint256,uint256,uint256,string)[])
func (_Registry *RegistryCallerSession) PaginateCredentialsByHolder(holder common.Address, offset *big.Int, limit *big.Int) ([]CredentialRegistryCredential, error) {
	return _Registry.Contract.PaginateCredentialsByHolder(&_Registry.CallOpts, holder, offset, limit)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Registry *RegistryCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Registry *RegistrySession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Registry.Contract.SupportsInterface(&_Registry.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Registry *RegistryCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Registry.Contract.SupportsInterface(&_Registry.CallOpts, interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Registry *RegistryCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Registry *RegistrySession) Symbol() (string, error) {
	return _Registry.Contract.Symbol(&_Registry.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Registry *RegistryCallerSession) Symbol() (string, error) {
	return _Registry.Contract.Symbol(&_Registry.CallOpts)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Registry *RegistryCaller) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "tokenURI", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Registry *RegistrySession) TokenURI(tokenId *big.Int) (string, error) {
	return _Registry.Contract.TokenURI(&_Registry.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Registry *RegistryCallerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _Registry.Contract.TokenURI(&_Registry.CallOpts, tokenId)
}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Registry *RegistryCaller) UserToNonce(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "userToNonce", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Registry *RegistrySession) UserToNonce(arg0 common.Address) (*big.Int, error) {
	return _Registry.Contract.UserToNonce(&_Registry.CallOpts, arg0)
}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Registry *RegistryCallerSession) UserToNonce(arg0 common.Address) (*big.Int, error) {
	return _Registry.Contract.UserToNonce(&_Registry.CallOpts, arg0)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactor) Approve(opts *bind.TransactOpts, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "approve", to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Registry *RegistrySession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.Approve(&_Registry.TransactOpts, to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactorSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.Approve(&_Registry.TransactOpts, to, tokenId)
}

// BatchIssueCredentialsWithSignature is a paid mutator transaction binding the contract method 0x50e0c6b5.
//
// Solidity: function batchIssueCredentialsWithSignature((address,(address,string,string,uint64,uint64)[],uint256,bytes) params) returns()
func (_Registry *RegistryTransactor) BatchIssueCredentialsWithSignature(opts *bind.TransactOpts, params CredentialRegistryBatchIssueCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "batchIssueCredentialsWithSignature", params)
}

// BatchIssueCredentialsWithSignature is a paid mutator transaction binding the contract method 0x50e0c6b5.
//
// Solidity: function batchIssueCredentialsWithSignature((address,(address,string,string,uint64,uint64)[],uint256,bytes) params) returns()
func (_Registry *RegistrySession) BatchIssueCredentialsWithSignature(params CredentialRegistryBatchIssueCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.Contract.BatchIssueCredentialsWithSignature(&_Registry.TransactOpts, params)
}

// BatchIssueCredentialsWithSignature is a paid mutator transaction binding the contract method 0x50e0c6b5.
//
// Solidity: function batchIssueCredentialsWithSignature((address,(address,string,string,uint64,uint64)[],uint256,bytes) params) returns()
func (_Registry *RegistryTransactorSession) BatchIssueCredentialsWithSignature(params CredentialRegistryBatchIssueCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.Contract.BatchIssueCredentialsWithSignature(&_Registry.TransactOpts, params)
}

// BatchRevokeCredentialsWithSignature is a paid mutator transaction binding the contract method 0x23b1812d.
//
// Solidity: function batchRevokeCredentialsWithSignature((address,uint256[],uint256,bytes) params) returns()
func (_Registry *RegistryTransactor) BatchRevokeCredentialsWithSignature(opts *bind.TransactOpts, params CredentialRegistryBatchRevokeCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "batchRevokeCredentialsWithSignature", params)
}

// BatchRevokeCredentialsWithSignature is a paid mutator transaction binding the contract method 0x23b1812d.
//
// Solidity: function batchRevokeCredentialsWithSignature((address,uint256[],uint256,bytes) params) returns()
func (_Registry *RegistrySession) BatchRevokeCredentialsWithSignature(params CredentialRegistryBatchRevokeCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.Contract.BatchRevokeCredentialsWithSignature(&_Registry.TransactOpts, params)
}

// BatchRevokeCredentialsWithSignature is a paid mutator transaction binding the contract method 0x23b1812d.
//
// Solidity: function batchRevokeCredentialsWithSignature((address,uint256[],uint256,bytes) params) returns()
func (_Registry *RegistryTransactorSession) BatchRevokeCredentialsWithSignature(params CredentialRegistryBatchRevokeCredentialsWithSignatureParams) (*types.Transaction, error) {
	return _Registry.Contract.BatchRevokeCredentialsWithSignature(&_Registry.TransactOpts, params)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _config) returns()
func (_Registry *RegistryTransactor) Initialize(opts *bind.TransactOpts, _config common.Address) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "initialize", _config)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _config) returns()
func (_Registry *RegistrySession) Initialize(_config common.Address) (*types.Transaction, error) {
	return _Registry.Contract.Initialize(&_Registry.TransactOpts, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _config) returns()
func (_Registry *RegistryTransactorSession) Initialize(_config common.Address) (*types.Transaction, error) {
	return _Registry.Contract.Initialize(&_Registry.TransactOpts, _config)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "safeTransferFrom", from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistrySession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SafeTransferFrom(&_Registry.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactorSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SafeTransferFrom(&_Registry.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Registry *RegistryTransactor) SafeTransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "safeTransferFrom0", from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Registry *RegistrySession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Registry.Contract.SafeTransferFrom0(&_Registry.TransactOpts, from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Registry *RegistryTransactorSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Registry.Contract.SafeTransferFrom0(&_Registry.TransactOpts, from, to, tokenId, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Registry *RegistryTransactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Registry *RegistrySession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Registry.Contract.SetApprovalForAll(&_Registry.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Registry *RegistryTransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Registry.Contract.SetApprovalForAll(&_Registry.TransactOpts, operator, approved)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "transferFrom", from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistrySession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.TransferFrom(&_Registry.TransactOpts, from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Registry *RegistryTransactorSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.TransferFrom(&_Registry.TransactOpts, from, to, tokenId)
}

// RegistryApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Registry contract.
type RegistryApprovalIterator struct {
	Event *RegistryApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryApproval represents a Approval event raised by the Registry contract.
type RegistryApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, approved []common.Address, tokenId []*big.Int) (*RegistryApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &RegistryApprovalIterator{contract: _Registry.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *RegistryApproval, owner []common.Address, approved []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryApproval)
				if err := _Registry.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) ParseApproval(log types.Log) (*RegistryApproval, error) {
	event := new(RegistryApproval)
	if err := _Registry.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the Registry contract.
type RegistryApprovalForAllIterator struct {
	Event *RegistryApprovalForAll // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryApprovalForAll)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryApprovalForAll)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryApprovalForAll represents a ApprovalForAll event raised by the Registry contract.
type RegistryApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Registry *RegistryFilterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*RegistryApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &RegistryApprovalForAllIterator{contract: _Registry.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Registry *RegistryFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *RegistryApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryApprovalForAll)
				if err := _Registry.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Registry *RegistryFilterer) ParseApprovalForAll(log types.Log) (*RegistryApprovalForAll, error) {
	event := new(RegistryApprovalForAll)
	if err := _Registry.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryCredentialIssuedIterator is returned from FilterCredentialIssued and is used to iterate over the raw logs and unpacked data for CredentialIssued events raised by the Registry contract.
type RegistryCredentialIssuedIterator struct {
	Event *RegistryCredentialIssued // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryCredentialIssuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryCredentialIssued)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryCredentialIssued)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryCredentialIssuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryCredentialIssuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryCredentialIssued represents a CredentialIssued event raised by the Registry contract.
type RegistryCredentialIssued struct {
	Id        *big.Int
	Holder    common.Address
	Issuer    common.Address
	IssuedAt  uint64
	ExpiresAt uint64
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCredentialIssued is a free log retrieval operation binding the contract event 0xc9489fecb5499f17e8ff515fcfbe1dbb22eb52abb09422be0b461b5d0d951b10.
//
// Solidity: event CredentialIssued(uint256 indexed id, address indexed holder, address indexed issuer, uint64 issuedAt, uint64 expiresAt)
func (_Registry *RegistryFilterer) FilterCredentialIssued(opts *bind.FilterOpts, id []*big.Int, holder []common.Address, issuer []common.Address) (*RegistryCredentialIssuedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var holderRule []interface{}
	for _, holderItem := range holder {
		holderRule = append(holderRule, holderItem)
	}
	var issuerRule []interface{}
	for _, issuerItem := range issuer {
		issuerRule = append(issuerRule, issuerItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "CredentialIssued", idRule, holderRule, issuerRule)
	if err != nil {
		return nil, err
	}
	return &RegistryCredentialIssuedIterator{contract: _Registry.contract, event: "CredentialIssued", logs: logs, sub: sub}, nil
}

// WatchCredentialIssued is a free log subscription operation binding the contract event 0xc9489fecb5499f17e8ff515fcfbe1dbb22eb52abb09422be0b461b5d0d951b10.
//
// Solidity: event CredentialIssued(uint256 indexed id, address indexed holder, address indexed issuer, uint64 issuedAt, uint64 expiresAt)
func (_Registry *RegistryFilterer) WatchCredentialIssued(opts *bind.WatchOpts, sink chan<- *RegistryCredentialIssued, id []*big.Int, holder []common.Address, issuer []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var holderRule []interface{}
	for _, holderItem := range holder {
		holderRule = append(holderRule, holderItem)
	}
	var issuerRule []interface{}
	for _, issuerItem := range issuer {
		issuerRule = append(issuerRule, issuerItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "CredentialIssued", idRule, holderRule, issuerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryCredentialIssued)
				if err := _Registry.contract.UnpackLog(event, "CredentialIssued", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCredentialIssued is a log parse operation binding the contract event 0xc9489fecb5499f17e8ff515fcfbe1dbb22eb52abb09422be0b461b5d0d951b10.
//
// Solidity: event CredentialIssued(uint256 indexed id, address indexed holder, address indexed issuer, uint64 issuedAt, uint64 expiresAt)
func (_Registry *RegistryFilterer) ParseCredentialIssued(log types.Log) (*RegistryCredentialIssued, error) {
	event := new(RegistryCredentialIssued)
	if err := _Registry.contract.UnpackLog(event, "CredentialIssued", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryCredentialRevokedIterator is returned from FilterCredentialRevoked and is used to iterate over the raw logs and unpacked data for CredentialRevoked events raised by the Registry contract.
type RegistryCredentialRevokedIterator struct {
	Event *RegistryCredentialRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryCredentialRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryCredentialRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryCredentialRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryCredentialRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryCredentialRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryCredentialRevoked represents a CredentialRevoked event raised by the Registry contract.
type RegistryCredentialRevoked struct {
	Id      *big.Int
	Revoker common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterCredentialRevoked is a free log retrieval operation binding the contract event 0x936526f4a3155895d9f01699716f2e0c548d726a24a1437e5ce92b3dacaad35b.
//
// Solidity: event CredentialRevoked(uint256 indexed id, address indexed revoker)
func (_Registry *RegistryFilterer) FilterCredentialRevoked(opts *bind.FilterOpts, id []*big.Int, revoker []common.Address) (*RegistryCredentialRevokedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var revokerRule []interface{}
	for _, revokerItem := range revoker {
		revokerRule = append(revokerRule, revokerItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "CredentialRevoked", idRule, revokerRule)
	if err != nil {
		return nil, err
	}
	return &RegistryCredentialRevokedIterator{contract: _Registry.contract, event: "CredentialRevoked", logs: logs, sub: sub}, nil
}

// WatchCredentialRevoked is a free log subscription operation binding the contract event 0x936526f4a3155895d9f01699716f2e0c548d726a24a1437e5ce92b3dacaad35b.
//
// Solidity: event CredentialRevoked(uint256 indexed id, address indexed revoker)
func (_Registry *RegistryFilterer) WatchCredentialRevoked(opts *bind.WatchOpts, sink chan<- *RegistryCredentialRevoked, id []*big.Int, revoker []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var revokerRule []interface{}
	for _, revokerItem := range revoker {
		revokerRule = append(revokerRule, revokerItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "CredentialRevoked", idRule, revokerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryCredentialRevoked)
				if err := _Registry.contract.UnpackLog(event, "CredentialRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCredentialRevoked is a log parse operation binding the contract event 0x936526f4a3155895d9f01699716f2e0c548d726a24a1437e5ce92b3dacaad35b.
//
// Solidity: event CredentialRevoked(uint256 indexed id, address indexed revoker)
func (_Registry *RegistryFilterer) ParseCredentialRevoked(log types.Log) (*RegistryCredentialRevoked, error) {
	event := new(RegistryCredentialRevoked)
	if err := _Registry.contract.UnpackLog(event, "CredentialRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Registry contract.
type RegistryInitializedIterator struct {
	Event *RegistryInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryInitialized represents a Initialized event raised by the Registry contract.
type RegistryInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Registry *RegistryFilterer) FilterInitialized(opts *bind.FilterOpts) (*RegistryInitializedIterator, error) {

	logs, sub, err := _Registry.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &RegistryInitializedIterator{contract: _Registry.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Registry *RegistryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *RegistryInitialized) (event.Subscription, error) {

	logs, sub, err := _Registry.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryInitialized)
				if err := _Registry.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Registry *RegistryFilterer) ParseInitialized(log types.Log) (*RegistryInitialized, error) {
	event := new(RegistryInitialized)
	if err := _Registry.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Registry contract.
type RegistryTransferIterator struct {
	Event *RegistryTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryTransfer represents a Transfer event raised by the Registry contract.
type RegistryTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, tokenId []*big.Int) (*RegistryTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &RegistryTransferIterator{contract: _Registry.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *RegistryTransfer, from []common.Address, to []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryTransfer)
				if err := _Registry.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Registry *RegistryFilterer) ParseTransfer(log types.Log) (*RegistryTransfer, error) {
	event := new(RegistryTransfer)
	if err := _Registry.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
