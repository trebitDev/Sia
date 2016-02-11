package modules

import (
	"github.com/NebulousLabs/Sia/types"
)

const (
	// AcceptResponse defines the response that is sent to a successful RPC.
	AcceptResponse = "accept"

	// HostDir names the directory that contains the host persistence.
	HostDir = "host"

	// MaxFileContractSetLen determines the maximum allowed size of a
	// transaction set that can be sent when trying to negotiate a file
	// contract. The transaction set will contain all of the unconfirmed
	// dependencies of the file contract, meaning that it can be quite large.
	// The transaction pool's size limit for transaction sets has been chosen
	// as a reasonable guideline for determining what is too large.
	MaxFileContractSetLen = TransactionSetSizeLimit - 1e3
)

var (
	// RPCSettings is the specifier for requesting settings from the host.
	RPCSettings = types.Specifier{'S', 'e', 't', 't', 'i', 'n', 'g', 's'}

	// RPCUpload is the specifier for initiating an upload with the host.
	RPCUpload = types.Specifier{'U', 'p', 'l', 'o', 'a', 'd'}

	// RPCRenew is the specifier to renewing an existing contract.
	RPCRenew = types.Specifier{'R', 'e', 'n', 'e', 'w'}

	// RPCRevise is the specifier for revising an existing file contract.
	RPCRevise = types.Specifier{'R', 'e', 'v', 'i', 's', 'e'}

	// RPCDownload is the specifier for downloading a file from a host.
	RPCDownload = types.Specifier{'D', 'o', 'w', 'n', 'l', 'o', 'a', 'd'}

	// PrefixHostAnnouncement is used to indicate that a transaction's
	// Arbitrary Data field contains a host announcement. The encoded
	// announcement will follow this prefix.
	PrefixHostAnnouncement = types.Specifier{'H', 'o', 's', 't', 'A', 'n', 'n', 'o', 'u', 'n', 'c', 'e', 'm', 'e', 'n', 't'}
)

type (
	// A DownloadRequest is used to retrieve a particular segment of a file from a
	// host.
	DownloadRequest struct {
		Offset uint64
		Length uint64
	}

	// HostAnnouncement declares a nodes intent to be a host, providing a net
	// address that can be used to contact the host.
	HostAnnouncement struct {
		IPAddress NetAddress
	}

	// HostSettings are the parameters advertised by the host. These are the
	// values that the renter will request from the host in order to build its
	// database.
	HostSettings struct {
		AcceptingContracts bool              `json:"acceptingcontracts"`
		MaxDuration        types.BlockHeight `json:"maxduration"`
		NetAddress         NetAddress        `json:"netaddress"`
		RemainingStorage   uint64            `json:"remainingstorage"` // Cannot be directly changed
		TotalStorage       uint64            `json:"totalstorage"`     // Cannot be directly changed
		UnlockHash         types.UnlockHash  `json:"unlockhash"`       // Cannot be directly changed
		WindowSize         types.BlockHeight `json:"windowsize"`

		Collateral             types.Currency `json:"collateral"`
		ContractPrice          types.Currency `json:"contractprice"`
		DownloadBandwidthPrice types.Currency `json:"downloadbandwidthprice"` // The cost for a renter to download something (meaning the host is uploading).
		StoragePrice           types.Currency `json:"storageprice"`
		UploadBandwidthPrice   types.Currency `json:"uploadbandwidthprice"` // The cost for a renter to upload something (meaning the host is downloading).
	}

	// HostRPCMetrics reports the quantity of each type of RPC call that has
	// been made to the host.
	HostRPCMetrics struct {
		ErrorCalls        uint64 `json:"errorcalls"` // Calls that resulted in an error.
		UnrecognizedCalls uint64 `json:"unrecognizedcalls"`
		DownloadCalls     uint64 `json:"downloadcalls"`
		RenewCalls        uint64 `json:"renewcalls"`
		ReviseCalls       uint64 `json:"revisecalls"`
		SettingsCalls     uint64 `json:"settingscalls"`
		UploadCalls       uint64 `json:"uploadcalls"`
	}

	// Host can take storage from disk and offer it to the network, managing things
	// such as announcements, settings, and implementing all of the RPCs of the
	// host protocol.
	Host interface {
		// Announce submits a host announcement to the blockchain, returning an
		// error if its external IP address is unknown. After announcing, the
		// host will begin accepting contracts.
		Announce() error

		// AnnounceAddress behaves like Announce, but allows the caller to
		// specify the address announced. Like Announce, this will cause the
		// host to start accepting contracts.
		AnnounceAddress(NetAddress) error

		// Capacity returns the amount of storage still available on the
		// machine. The amount can be negative if the total capacity was
		// reduced to below the active capacity.
		Capacity() uint64

		// Contracts returns the number of unresolved file contracts that the
		// host is responsible for.
		Contracts() uint64

		// DeleteContract deletes a file contract. The revenue and collateral
		// on the file contract will be lost, and the data will be removed.
		DeleteContract(types.FileContractID) error

		// NetAddress returns the host's network address
		NetAddress() NetAddress

		// Revenue returns the amount of revenue that the host has lined up,
		// the amount of revenue the host has successfully captured, and the
		// amount of revenue the host has lost.
		//
		// TODO: This function will eventually include two more numbers, one
		// representing current collateral at risk, and one representing total
		// collateral lost.
		Revenue() (unresolved, resolved, lost types.Currency)

		// RPCMetrics returns information on the types of RPC calls that have
		// been made to the host.
		RPCMetrics() HostRPCMetrics

		// SetConfig sets the hosting parameters of the host.
		SetSettings(HostSettings) error

		// Settings returns the host's settings.
		Settings() HostSettings

		// Close saves the state of the host and stops its listener process.
		Close() error
	}
)

// BandwidthPriceToConsensus converts a human bandwidth price, having the unit
// 'Siacoins per Terabyte', to a consensus storage price, having the unit
// 'Hastings per Byte'.
func BandwidthPriceToConsensus(siacoinsTB uint64) (hastingsByte types.Currency) {
	hastingsTB := types.NewCurrency64(siacoinsTB).Mul(types.SiacoinPrecision)
	return hastingsTB.Div(types.NewCurrency64(1e12))
}

// BandwidthPriceToHuman converts a consensus bandwidth price, having the unit
// 'Hastings per Byte' to a human bandwidth price, having the unit 'Siacoins
// per Terabyte'.
func BandwidthPriceToHuman(hastingsByte types.Currency) (siacoinsTB uint64, err error) {
	hastingsTB := hastingsByte.Mul(types.NewCurrency64(1e12))
	if hastingsTB.Cmp(types.SiacoinPrecision.Div(types.NewCurrency64(2))) < 0 {
		// The result of the final division is going to be less than 0.5,
		// therefore 0 should be returned.
		return 0, nil
	}
	if hastingsTB.Cmp(types.SiacoinPrecision) < 0 {
		// The result of the final division is going to be greater than or
		// equal to 0.5, but less than 1, therefore 1 should be returned.
		return 1, nil
	}
	return hastingsTB.Div(types.SiacoinPrecision).Uint64()
}

// StoragePriceToConsensus converts a human storage price, having the unit
// 'Siacoins per Month per Terabyte', to a consensus storage price, having the
// unit 'Hastings per Block per Byte'.
func StoragePriceToConsensus(siacoinsMonthTB uint64) (hastingsBlockByte types.Currency) {
	// Perform multiplication first to preserve precision.
	hastingsMonthTB := types.NewCurrency64(siacoinsMonthTB).Mul(types.SiacoinPrecision)
	hastingsBlockTB := hastingsMonthTB.Div(types.NewCurrency64(4320))
	return hastingsBlockTB.Div(types.NewCurrency64(1e12))
}

// StoragePriceToHuman converts a consensus storage price, having the unit
// 'Hastings per Block per Byte', to a human storage price, having the unit
// 'Siacoins per Month per Terabyte'. An error is returned if the result would
// overflow a uint64. If the result is between 0 and 1, the value is rounded to
// the nearest value.
func StoragePriceToHuman(hastingsBlockByte types.Currency) (siacoinsMonthTB uint64, err error) {
	// Perform multiplication first to preserve precision.
	hastingsMonthByte := hastingsBlockByte.Mul(types.NewCurrency64(4320))
	hastingsMonthTB := hastingsMonthByte.Mul(types.NewCurrency64(1e12))
	if hastingsMonthTB.Cmp(types.SiacoinPrecision.Div(types.NewCurrency64(2))) < 0 {
		// The result of the final division is going to be less than 0.5,
		// therefore 0 should be returned.
		return 0, nil
	}
	if hastingsMonthTB.Cmp(types.SiacoinPrecision) < 0 {
		// The result of the final division is going to be greater than or
		// equal to 0.5, but less than 1, therefore 1 should be returned.
		return 1, nil
	}
	return hastingsMonthTB.Div(types.SiacoinPrecision).Uint64()
}
