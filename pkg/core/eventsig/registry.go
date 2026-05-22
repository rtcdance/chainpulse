package eventsig

import (
	"strings"
	"sync"
)

// knownEventSignatureNames maps keccak256 event signature hashes to human-readable names.
// These are the canonical topic0 values for standard ERC events.
var knownEventSignatureNames = map[string]string{
	// ERC-20 / ERC-721 Transfer(address indexed from, address indexed to, uint256/value indexed tokenId)
	"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef": "Transfer",
	// ERC-20 Approval(address indexed owner, address indexed spender, uint256 value)
	"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925": "Approval",
	// ERC-721 ApprovalForAll(address indexed owner, address indexed operator, bool approved)
	"0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31": "ApprovalForAll",
	// ERC-1155 TransferSingle
	"0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62": "TransferSingle",
	// ERC-1155 TransferBatch
	"0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb": "TransferBatch",
	// ERC-1155 URI
	"0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b": "URI",
	// ChainPulse EventEmitter Ping(address,uint256)
	"0xfd8d0c1dc3ab254ec49463a1192bb2423b3b851adedec1aa94dcd362dc063c9d": "Ping",
	// Ping(uint256)
	"0x48257dc961b6f792c2b78a080dacfed693b660960a702de21cee364e20270e2f": "Ping",
	// Uniswap V3 Swap
	"0x08a8a95dd902fa4dc62bec5011d413fe4dd0e10e393f28d895f156bac4f7c4ea": "Swap",
	// Aave V3 Supply
	"0xb50860d64b2bfaf120facad3881ea8fb330317b4a328f7cba5157950aec1d2de": "Supply",
	// Aave V3 Withdraw
	"0x3115d1449a7b732c986cba18244e897a450f61e1bb8d589cd2e69e6c8924f9f7": "Withdraw",
	// Aave V3 Borrow
	"0x8f6e65b68bdd0d3185cbfd43a003120c1ba5e6852290aa02c463c387e1b4ce43": "Borrow",
	// Aave V3 Repay
	"0x248995ad2b9ddda6559fb3b0858726ee6e63b6714191c268b529a6b49f0f2f19": "Repay",
	// Aave V3 LiquidationCall
	"0x3a84f64446e8eada995aa9da2ddbfcd9b5d5d650503b19f024096d04c05ef2a9": "LiquidationCall",
	// Aave V3 ReserveDataUpdated
	"0x804c9b842b2748a22bb64b345453a3de7ca54a6ca45ce00d415894979e22897a": "ReserveDataUpdated",
	// Compound V3 Supply
	"0xd1cf3d156d5f8f0d50f6c122ed609cec09d35c9b9fb3fff6ea0959134dae424e": "CometSupply",
	// Compound V3 Withdraw
	"0x9b1bfa7fa9ee420a16e124f794c35ac9f90472acc99140eb2f6447c714cad8eb": "CometWithdraw",
	// Compound V3 Borrow
	"0xe1979fe4c35e0cef342fef5668e2c8e7a7e9f5d5d1ca8fee0ac6c427fa4153af": "CometBorrow",
	// Compound V3 Repay
	"0x05f2eeda0e08e4b437f487c8d7d29b14537d15e3488170dc3de5dbdf8dac4684": "CometRepay",
	// Compound V3 Liquidate
	"0x18e26ca5d5b0337cefc3ebe9565d62dee00c8817c44218a9ab8ddbaca099e060": "CometLiquidate",
	// Uniswap V2 Swap
	"0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822": "UniswapV2Swap",
	// Uniswap V2 Sync
	"0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1": "Sync",
	// Uniswap V2 PairCreated
	"0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9": "PairCreated",
	// Curve TokenExchange
	"0x064fac08aeecb25ac880c1cccb23c6a87b185421974a4d1e840beb61d7cbe180": "TokenExchange",
	// Curve AddLiquidity
	"0x62564c79f591d504e76d01528198a58d31c18cf591368c1af192abf184781228": "AddLiquidity",
	// Balancer V2 Swap
	"0xfa2dda1cc1b86e41239702756b13effbc1a092b5c57e3ad320fbe4f3b13fe235": "BalancerSwap",
	// OpenZeppelin Governor ProposalCreated
	"0x7d84a6263ae0d98d3329bd7b46bb4e8d6f98cd35a7adb45c274c8b7fd5ebd5e0": "ProposalCreated",
	// OpenZeppelin Governor VoteCast
	"0xb8e138887d0aa13bab447e82de9d5c1777041ecd21ca36ba824ff1e6c07ddda4": "VoteCast",
	// OpenZeppelin Governor ProposalExecuted
	"0x712ae1383f79ac853f8d882153778e0260ef8f03b504e2866e0593e04d2b291f": "ProposalExecuted",
	// OpenZeppelin Governor ProposalCanceled
	"0x789cf55be980739dad1d0699b93b58e806b51c9d96619bfa8fe0a28abaa7b30c": "ProposalCanceled",

	// ERC-4337 Account Abstraction
	"0x6dba8cdfd943e2a28e46977657068af6df2740dbd35e2399f42ff97fc1f98466": "UserOperationEvent",
	"0xa1a13ce46f8df72442ef06cc8a7ace7841a36a8fc672f0c8188415bcca90362c": "AccountDeployed",
	"0xbb47ee3e183a558b1a2ff0874b079f3fc5478b7454eacf2bfc5af2ff5878f972": "BeforeExecution",
	"0x0915899468dc9b80020e1ab1aae70b1564db08ba1d4151ad4116bbfe20712069": "AfterExecution",

	// Cross-Chain Bridge Events
	"0x7904e91da6835a7ca49ba15e27ff92991d80f921780362f400ff6ff616a52":    "PacketSent",
	"0xff3f87a0e0ed53b98bca6f7518f772269fac51bc6a121527a80061a9bea063f9": "PacketDelivered",
	"0xb93c37389233beb85a3a726c3f15c2d15533ee74cb602f20f490dfffef775937": "LogMessagePublished",
	"0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a": "SentMessage",
	"0xc1d1490cf25c3b40d600dfb27c7680340ed1ab901b7e8f3551280968a3b372b0": "TxToL2",
	"0xbf56ee474f26307e809eb65f483345ec94ea2335f33fbb5667cf518080487424": "L2TxCreated",
}

// knownNameToSignatures provides reverse lookup from event name to canonical signature hash.
var knownNameToSignatures map[string]string

var initNameToSigsOnce sync.Once

func ensureNameToSignaturesInitialized() {
	initNameToSigsOnce.Do(func() {
		knownNameToSignatures = make(map[string]string, len(knownEventSignatureNames))
		for sig, name := range knownEventSignatureNames {
			if _, exists := knownNameToSignatures[name]; !exists {
				knownNameToSignatures[name] = sig
			}
		}
	})
}

// ResolveEventNameFromTopic resolves a keccak256 event topic (topic0) to a human-readable
// event name. If the topic is not recognized, it returns the raw hex string unchanged.
func ResolveEventNameFromTopic(topic string) string {
	normalized := strings.ToLower(strings.TrimSpace(topic))
	if normalized == "" {
		return ""
	}
	if name, ok := knownEventSignatureNames[normalized]; ok {
		return name
	}
	return topic
}

// ResolveTopicFromName performs a reverse lookup from an event name to its canonical
// keccak256 signature hash. Returns empty string if the name is not recognized.
func ResolveTopicFromName(name string) string {
	ensureNameToSignaturesInitialized()
	if sig, ok := knownNameToSignatures[name]; ok {
		return sig
	}
	return ""
}

// IsKnownEventSignature checks if a topic0 hash is a recognized event signature.
func IsKnownEventSignature(topic string) bool {
	_, ok := knownEventSignatureNames[strings.ToLower(strings.TrimSpace(topic))]
	return ok
}
