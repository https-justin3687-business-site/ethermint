// This is a test utility for Ethermint's Web3 JSON-RPC services.
//
// To run these tests please first ensure you have the emintd running
// and have started the RPC service with `emintcli rest-server`.
//
// You can configure the desired ETHERMINT_NODE_HOST and ETHERMINT_INTEGRATION_TEST_MODE
//
// to have it running

package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cosmos/ethermint/version"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const (
	addrA         = "0xc94770007dda54cF92009BFF0dE90c06F603a09f"
	addrAStoreKey = 0
)

var (
	ETHERMINT_INTEGRATION_TEST_MODE = os.Getenv("ETHERMINT_INTEGRATION_TEST_MODE")
	ETHERMINT_NODE_HOST             = os.Getenv("ETHERMINT_NODE_HOST")

	zeroString = "0x0"
)

type Request struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Response struct {
	Error  *RPCError       `json:"error"`
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
}

func TestMain(m *testing.M) {
	if ETHERMINT_INTEGRATION_TEST_MODE != "stable" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip stable test")
		return
	}

	if ETHERMINT_NODE_HOST == "" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip stable test, ETHERMINT_NODE_HOST is not defined")
		return
	}

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func createRequest(method string, params interface{}) Request {
	return Request{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
}

func call(t *testing.T, method string, params interface{}) (*Response, error) {
	req, err := json.Marshal(createRequest(method, params))
	if err != nil {
		return nil, err
	}

	var rpcRes *Response
	time.Sleep(1 * time.Second)
	/* #nosec */
	res, err := http.Post(ETHERMINT_NODE_HOST, "application/json", bytes.NewBuffer(req))
	if err != nil {
		t.Log("could not http.Post, ", "err", err)
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	rpcRes = new(Response)
	err = decoder.Decode(&rpcRes)
	if err != nil {
		t.Log("could not decoder.Decode, ", "err", err)
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		t.Log("could not Body.Close, ", "err", err)
		return nil, err
	}

	if rpcRes.Error != nil {
		t.Log("could not rpcRes.Error, ", "err", err)
		return nil, errors.New(rpcRes.Error.Message)
	}

	return rpcRes, nil

}

func TestEth_protocolVersion(t *testing.T) {
	expectedRes := hexutil.Uint(version.ProtocolVersion)

	rpcRes, err := call(t, "eth_protocolVersion", []string{})
	require.NoError(t, err)

	var res hexutil.Uint
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	t.Logf("Got protocol version: %s\n", res.String())
	require.Equal(t, expectedRes, res, "expected: %s got: %s\n", expectedRes.String(), rpcRes.Result)
}

func TestEth_blockNumber(t *testing.T) {
	rpcRes, err := call(t, "eth_blockNumber", []string{})
	require.NoError(t, err)

	var res hexutil.Uint64
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	t.Logf("Got block number: %s\n", res.String())
}

func TestEth_GetBalance(t *testing.T) {
	rpcRes, err := call(t, "eth_getBalance", []string{addrA, zeroString})
	require.NoError(t, err)

	var res hexutil.Big
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	t.Logf("Got balance %s for %s\n", res.String(), addrA)

	// 0 if x == y; where x is res, y is 0
	if res.ToInt().Cmp(big.NewInt(0)) != 0 {
		t.Errorf("expected balance: %d, got: %s", 0, res.String())
	}
}

func TestEth_GetStorageAt(t *testing.T) {
	expectedRes := hexutil.Bytes{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	rpcRes, err := call(t, "eth_getStorageAt", []string{addrA, string(addrAStoreKey), zeroString})
	require.NoError(t, err)

	var storage hexutil.Bytes
	err = storage.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	t.Logf("Got value [%X] for %s with key %X\n", storage, addrA, addrAStoreKey)

	require.True(t, bytes.Equal(storage, expectedRes), "expected: %d (%d bytes) got: %d (%d bytes)", expectedRes, len(expectedRes), storage, len(storage))
}

func TestEth_GetCode(t *testing.T) {
	expectedRes := hexutil.Bytes{}
	rpcRes, err := call(t, "eth_getCode", []string{addrA, zeroString})
	require.NoError(t, err)

	var code hexutil.Bytes
	err = code.UnmarshalJSON(rpcRes.Result)

	require.NoError(t, err)

	t.Logf("Got code [%X] for %s\n", code, addrA)
	require.True(t, bytes.Equal(expectedRes, code), "expected: %X got: %X", expectedRes, code)
}

func getAddress(t *testing.T) []byte {
	rpcRes, err := call(t, "eth_accounts", []string{})
	require.NoError(t, err)

	var res []hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &res)
	require.NoError(t, err)

	return res[0]
}

func TestEth_SendTransaction(t *testing.T) {
	from := getAddress(t)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["data"] = "0x6080604052348015600f57600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a2603580604b6000396000f3fe6080604052600080fdfea165627a7a723058206cab665f0f557620554bb45adf266708d2bd349b8a4314bdff205ee8440e3c240029"

	rpcRes, err := call(t, "eth_sendTransaction", param)
	require.NoError(t, err)

	var hash hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &hash)
	require.NoError(t, err)
}

func TestEth_NewFilter(t *testing.T) {
	param := make([]map[string][]string, 1)
	param[0] = make(map[string][]string)
	param[0]["topics"] = []string{"0x0000000000000000000000000000000000000000000000000000000012341234"}
	rpcRes, err := call(t, "eth_newFilter", param)
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)
}

func TestEth_NewBlockFilter(t *testing.T) {
	rpcRes, err := call(t, "eth_newBlockFilter", []string{})
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)
}

func TestEth_GetFilterChanges_NoLogs(t *testing.T) {
	param := make([]map[string][]string, 1)
	param[0] = make(map[string][]string)
	param[0]["topics"] = []string{}
	rpcRes, err := call(t, "eth_newFilter", param)
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)

	changesRes, err := call(t, "eth_getFilterChanges", []string{ID.String()})
	require.NoError(t, err)

	var logs []*ethtypes.Log
	err = json.Unmarshal(changesRes.Result, &logs)
	require.NoError(t, err)
}

func TestEth_GetFilterChanges_WrongID(t *testing.T) {
	_, err := call(t, "eth_getFilterChanges", []string{"0x1122334400000077"})
	require.NotNil(t, err)
}

// sendTestTransaction sends a dummy transaction
func sendTestTransaction(t *testing.T) hexutil.Bytes {
	from := getAddress(t)
	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = "0x1122334455667788990011223344556677889900"
	rpcRes, err := call(t, "eth_sendTransaction", param)
	require.NoError(t, err)
	var hash hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &hash)
	require.NoError(t, err)
	return hash
}

func TestEth_GetTransactionReceipt(t *testing.T) {
	hash := sendTestTransaction(t)

	time.Sleep(time.Second * 5)

	param := []string{hash.String()}
	rpcRes, err := call(t, "eth_getTransactionReceipt", param)
	require.NoError(t, err)

	receipt := make(map[string]interface{})
	err = json.Unmarshal(rpcRes.Result, &receipt)
	require.NoError(t, err)
	require.Equal(t, "0x1", receipt["status"].(string))
}

// deployTestContract deploys a contract that emits an event in the constructor
func deployTestContract(t *testing.T) hexutil.Bytes {
	from := getAddress(t)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["data"] = "0x60806040526000805534801561001457600080fd5b5060d2806100236000396000f3fe6080604052348015600f57600080fd5b5060043610603c5760003560e01c80634f2be91f1460415780636deebae31460495780638ada066e146051575b600080fd5b6047606d565b005b604f6080565b005b60576094565b6040518082815260200191505060405180910390f35b6000808154809291906001019190505550565b600080815480929190600190039190505550565b6000805490509056fea265627a7a723158207b1aaa18c3100d8aa67f26a53f3cb83d2c69342d17327bd11e1b17c248957bfa64736f6c634300050c0032"
	param[0]["gas"] = "0x200000"

	rpcRes, err := call(t, "eth_sendTransaction", param)
	require.NoError(t, err)

	var hash hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &hash)
	require.NoError(t, err)

	return hash
}

func TestEth_GetTransactionReceipt_ContractDeployment(t *testing.T) {
	hash := deployTestContract(t)

	time.Sleep(time.Second * 5)

	param := []string{hash.String()}
	rpcRes, err := call(t, "eth_getTransactionReceipt", param)
	require.NoError(t, err)

	receipt := make(map[string]interface{})
	err = json.Unmarshal(rpcRes.Result, &receipt)
	require.NoError(t, err)
	require.Equal(t, "0x1", receipt["status"].(string))

	require.NotEqual(t, ethcmn.Address{}.String(), receipt["contractAddress"].(string))
	require.NotNil(t, receipt["logs"])
}

var ensFactory = "0x608060405234801561001057600080fd5b50612033806100206000396000f3006080604052600436106100405763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663e9358b018114610045575b600080fd5b34801561005157600080fd5b5061007373ffffffffffffffffffffffffffffffffffffffff6004351661009c565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b60008060006100a961057f565b604051809103906000f0801580156100c5573d6000803e3d6000fd5b50604080517f06ab59230000000000000000000000000000000000000000000000000000000081526000600482018190527f4f5b812789fc606be1b3b16908db13fc7a9adf7ca72641f84d75b47069d3d7f06024830152306044830152915192945073ffffffffffffffffffffffffffffffffffffffff8516926306ab59239260648084019391929182900301818387803b15801561016357600080fd5b505af1158015610177573d6000803e3d6000fd5b505050508161018461058f565b73ffffffffffffffffffffffffffffffffffffffff909116815260405190819003602001906000f0801580156101be573d6000803e3d6000fd5b50604080517f06ab59230000000000000000000000000000000000000000000000000000000081527f93cdeb708b7545dc668eb9280176169d1c33cfd8ed6f04690a0bcc88a93fc4ae60048201527f329539a1d23af1810c48a07fe7fc66a3b34fbc8b37e9b3cdb97bb88ceab7e4bf6024820152306044820152905191925073ffffffffffffffffffffffffffffffffffffffff8416916306ab59239160648082019260009290919082900301818387803b15801561027c57600080fd5b505af1158015610290573d6000803e3d6000fd5b5050604080517f1896f70a0000000000000000000000000000000000000000000000000000000081527ffdd5d5de6dd63db72bbc2d487944ba13bf775b50a80805fe6fcaba9b0fba88f5600482015273ffffffffffffffffffffffffffffffffffffffff858116602483015291519186169350631896f70a925060448082019260009290919082900301818387803b15801561032b57600080fd5b505af115801561033f573d6000803e3d6000fd5b5050604080517fd5fa2b000000000000000000000000000000000000000000000000000000000081527ffdd5d5de6dd63db72bbc2d487944ba13bf775b50a80805fe6fcaba9b0fba88f5600482015273ffffffffffffffffffffffffffffffffffffffff851660248201819052915191935063d5fa2b00925060448082019260009290919082900301818387803b1580156103d957600080fd5b505af11580156103ed573d6000803e3d6000fd5b5050604080517f5b0fc9c30000000000000000000000000000000000000000000000000000000081527f93cdeb708b7545dc668eb9280176169d1c33cfd8ed6f04690a0bcc88a93fc4ae600482015273ffffffffffffffffffffffffffffffffffffffff888116602483015291519186169350635b0fc9c3925060448082019260009290919082900301818387803b15801561048857600080fd5b505af115801561049c573d6000803e3d6000fd5b5050604080517f5b0fc9c300000000000000000000000000000000000000000000000000000000815260006004820181905273ffffffffffffffffffffffffffffffffffffffff898116602484015292519287169450635b0fc9c39350604480830193919282900301818387803b15801561051657600080fd5b505af115801561052a573d6000803e3d6000fd5b50506040805173ffffffffffffffffffffffffffffffffffffffff8616815290517fdbfb5ababf63f86424e8df6053dfb90f8b63ea26d7e1e8f68407af4fb2d2c4f29350908190036020019150a15092915050565b60405161064a806105a083390190565b60405161141e80610bea833901905600608060405234801561001057600080fd5b5060008080526020527fad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb58054600160a060020a031916331790556105f1806100596000396000f3006080604052600436106100825763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630178b8bf811461008757806302571be3146100c857806306ab5923146100e057806314ab90381461011657806316a25cbd1461013b5780631896f70a146101705780635b0fc9c3146101a1575b600080fd5b34801561009357600080fd5b5061009f6004356101d2565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b3480156100d457600080fd5b5061009f6004356101fd565b3480156100ec57600080fd5b5061011460043560243573ffffffffffffffffffffffffffffffffffffffff60443516610225565b005b34801561012257600080fd5b5061011460043567ffffffffffffffff60243516610311565b34801561014757600080fd5b506101536004356103e7565b6040805167ffffffffffffffff9092168252519081900360200190f35b34801561017c57600080fd5b5061011460043573ffffffffffffffffffffffffffffffffffffffff6024351661041e565b3480156101ad57600080fd5b5061011460043573ffffffffffffffffffffffffffffffffffffffff602435166104f3565b60009081526020819052604090206001015473ffffffffffffffffffffffffffffffffffffffff1690565b60009081526020819052604090205473ffffffffffffffffffffffffffffffffffffffff1690565b600083815260208190526040812054849073ffffffffffffffffffffffffffffffffffffffff16331461025757600080fd5b6040805186815260208082018790528251918290038301822073ffffffffffffffffffffffffffffffffffffffff871683529251929450869288927fce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e8292908290030190a350600090815260208190526040902080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff929092169190911790555050565b600082815260208190526040902054829073ffffffffffffffffffffffffffffffffffffffff16331461034357600080fd5b6040805167ffffffffffffffff84168152905184917f1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68919081900360200190a250600091825260208290526040909120600101805467ffffffffffffffff90921674010000000000000000000000000000000000000000027fffffffff0000000000000000ffffffffffffffffffffffffffffffffffffffff909216919091179055565b60009081526020819052604090206001015474010000000000000000000000000000000000000000900467ffffffffffffffff1690565b600082815260208190526040902054829073ffffffffffffffffffffffffffffffffffffffff16331461045057600080fd5b6040805173ffffffffffffffffffffffffffffffffffffffff84168152905184917f335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0919081900360200190a25060009182526020829052604090912060010180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff909216919091179055565b600082815260208190526040902054829073ffffffffffffffffffffffffffffffffffffffff16331461052557600080fd5b6040805173ffffffffffffffffffffffffffffffffffffffff84168152905184917fd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d266919081900360200190a25060009182526020829052604090912080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff9092169190911790555600a165627a7a723058203213a96a5c5e630e44a93f7fa415f3c625e46c7a560debc4dcf02cff9018ee6e0029608060405234801561001057600080fd5b5060405160208061141e833981016040525160008054600160a060020a03909216600160a060020a03199092169190911790556113cc806100526000396000f3006080604052600436106100c45763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301ffc9a781146100c957806310f13a8c146101175780632203ab56146101b557806329cd62ea1461024f5780632dff69411461026d5780633b3b57de1461029757806359d1d43c146102d8578063623195b0146103ab578063691f34311461040b5780637737221314610423578063c3d014d614610481578063c86902331461049c578063d5fa2b00146104cd575b600080fd5b3480156100d557600080fd5b506101037fffffffff00000000000000000000000000000000000000000000000000000000600435166104fe565b604080519115158252519081900360200190f35b34801561012357600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101b395833595369560449491939091019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506107139650505050505050565b005b3480156101c157600080fd5b506101d060043560243561098f565b6040518083815260200180602001828103825283818151815260200191508051906020019080838360005b838110156102135781810151838201526020016101fb565b50505050905090810190601f1680156102405780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b34801561025b57600080fd5b506101b3600435602435604435610a9b565b34801561027957600080fd5b50610285600435610bcb565b60408051918252519081900360200190f35b3480156102a357600080fd5b506102af600435610be1565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b3480156102e457600080fd5b5060408051602060046024803582810135601f8101859004850286018501909652858552610336958335953695604494919390910191908190840183828082843750949750610c099650505050505050565b6040805160208082528351818301528351919283929083019185019080838360005b83811015610370578181015183820152602001610358565b50505050905090810190601f16801561039d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156103b757600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526101b3948235946024803595369594606494920191908190840183828082843750949750610d309650505050505050565b34801561041757600080fd5b50610336600435610e61565b34801561042f57600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101b3958335953695604494919390910191908190840183828082843750949750610f059650505050505050565b34801561048d57600080fd5b506101b360043560243561108b565b3480156104a857600080fd5b506104b460043561119c565b6040805192835260208301919091528051918290030190f35b3480156104d957600080fd5b506101b360043573ffffffffffffffffffffffffffffffffffffffff602435166111b9565b60007fffffffff0000000000000000000000000000000000000000000000000000000082167f3b3b57de00000000000000000000000000000000000000000000000000000000148061059157507fffffffff0000000000000000000000000000000000000000000000000000000082167fd8389dc500000000000000000000000000000000000000000000000000000000145b806105dd57507fffffffff0000000000000000000000000000000000000000000000000000000082167f691f343100000000000000000000000000000000000000000000000000000000145b8061062957507fffffffff0000000000000000000000000000000000000000000000000000000082167f2203ab5600000000000000000000000000000000000000000000000000000000145b8061067557507fffffffff0000000000000000000000000000000000000000000000000000000082167fc869023300000000000000000000000000000000000000000000000000000000145b806106c157507fffffffff0000000000000000000000000000000000000000000000000000000082167f59d1d43c00000000000000000000000000000000000000000000000000000000145b8061070d57507fffffffff0000000000000000000000000000000000000000000000000000000082167f01ffc9a700000000000000000000000000000000000000000000000000000000145b92915050565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810187905290518693339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b15801561078957600080fd5b505af115801561079d573d6000803e3d6000fd5b505050506040513d60208110156107b357600080fd5b505173ffffffffffffffffffffffffffffffffffffffff16146107d557600080fd5b6000848152600160209081526040918290209151855185936005019287929182918401908083835b6020831061083a57805182527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe090920191602091820191016107fd565b51815160209384036101000a6000190180199092169116179052920194855250604051938490038101909320845161087b9591949190910192509050611305565b50826040518082805190602001908083835b602083106108ca57805182527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0909201916020918201910161088d565b51815160209384036101000a60001901801990921691161790526040805192909401829003822081835289518383015289519096508a95507fd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550948a94508392908301919085019080838360005b8381101561094f578181015183820152602001610937565b50505050905090810190601f16801561097c5780820380516001836020036101000a031916815260200191505b509250505060405180910390a350505050565b60008281526001602081905260409091206060905b838311610a8e57828416158015906109dd5750600083815260068201602052604081205460026000196101006001841615020190911604115b15610a8357600083815260068201602090815260409182902080548351601f600260001961010060018616150201909316929092049182018490048402810184019094528084529091830182828015610a775780601f10610a4c57610100808354040283529160200191610a77565b820191906000526020600020905b815481529060010190602001808311610a5a57829003601f168201915b50505050509150610a93565b6002909202916109a4565b600092505b509250929050565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810187905290518693339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b158015610b1157600080fd5b505af1158015610b25573d6000803e3d6000fd5b505050506040513d6020811015610b3b57600080fd5b505173ffffffffffffffffffffffffffffffffffffffff1614610b5d57600080fd5b604080518082018252848152602080820185815260008881526001835284902092516003840155516004909201919091558151858152908101849052815186927f1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46928290030190a250505050565b6000908152600160208190526040909120015490565b60009081526001602052604090205473ffffffffffffffffffffffffffffffffffffffff1690565b600082815260016020908152604091829020915183516060936005019285929182918401908083835b60208310610c6f57805182527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe09092019160209182019101610c32565b518151600019602094850361010090810a820192831692199390931691909117909252949092019687526040805197889003820188208054601f6002600183161590980290950116959095049283018290048202880182019052818752929450925050830182828015610d235780601f10610cf857610100808354040283529160200191610d23565b820191906000526020600020905b815481529060010190602001808311610d0657829003601f168201915b5050505050905092915050565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810187905290518693339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b158015610da657600080fd5b505af1158015610dba573d6000803e3d6000fd5b505050506040513d6020811015610dd057600080fd5b505173ffffffffffffffffffffffffffffffffffffffff1614610df257600080fd5b6000198301831615610e0357600080fd5b600084815260016020908152604080832086845260060182529091208351610e2d92850190611305565b50604051839085907faa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe390600090a350505050565b6000818152600160208181526040928390206002908101805485516000199582161561010002959095011691909104601f81018390048302840183019094528383526060939091830182828015610ef95780601f10610ece57610100808354040283529160200191610ef9565b820191906000526020600020905b815481529060010190602001808311610edc57829003601f168201915b50505050509050919050565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810186905290518593339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b158015610f7b57600080fd5b505af1158015610f8f573d6000803e3d6000fd5b505050506040513d6020811015610fa557600080fd5b505173ffffffffffffffffffffffffffffffffffffffff1614610fc757600080fd5b60008381526001602090815260409091208351610fec92600290920191850190611305565b50604080516020808252845181830152845186937fb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f79387939092839283019185019080838360005b8381101561104c578181015183820152602001611034565b50505050905090810190601f1680156110795780820380516001836020036101000a031916815260200191505b509250505060405180910390a2505050565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810186905290518593339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b15801561110157600080fd5b505af1158015611115573d6000803e3d6000fd5b505050506040513d602081101561112b57600080fd5b505173ffffffffffffffffffffffffffffffffffffffff161461114d57600080fd5b6000838152600160208181526040928390209091018490558151848152915185927f0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc92908290030190a2505050565b600090815260016020526040902060038101546004909101549091565b60008054604080517f02571be30000000000000000000000000000000000000000000000000000000081526004810186905290518593339373ffffffffffffffffffffffffffffffffffffffff16926302571be39260248083019360209383900390910190829087803b15801561122f57600080fd5b505af1158015611243573d6000803e3d6000fd5b505050506040513d602081101561125957600080fd5b505173ffffffffffffffffffffffffffffffffffffffff161461127b57600080fd5b60008381526001602090815260409182902080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff86169081179091558251908152915185927f52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd292908290030190a2505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061134657805160ff1916838001178555611373565b82800160010185558215611373579182015b82811115611373578251825591602001919060010190611358565b5061137f929150611383565b5090565b61139d91905b8082111561137f5760008155600101611389565b905600a165627a7a72305820494d2089cb484863f6172bb6f0ed3b148d6c8eb2a0dd86d330bf08813f99f65d0029a165627a7a72305820df7d6399d923bc8a4fe3d290869ee8614cd2e7d4ac7481c4de85e2df61d183370029"

func Test_DeployENSFactory(t *testing.T) {
	from := getAddress(t)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["data"] = ensFactory
	param[0]["gas"] = "0x300000"

	rpcRes, err := call(t, "eth_sendTransaction", param)
	require.NoError(t, err)

	var hash hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &hash)
	require.NoError(t, err)

	receipt := waitForReceipt(t, hash)
	require.NotNil(t, receipt, "transaction failed")
	require.Equal(t, "0x1", receipt["status"].(string))
}

func getTransactionReceipt(t *testing.T, hash hexutil.Bytes) map[string]interface{} {
	param := []string{hash.String()}
	rpcRes, err := call(t, "eth_getTransactionReceipt", param)
	require.NoError(t, err)

	receipt := make(map[string]interface{})
	err = json.Unmarshal(rpcRes.Result, &receipt)
	require.NoError(t, err)

	return receipt
}

func TestEth_GetTxLogs(t *testing.T) {
	hash := deployTestContract(t)

	time.Sleep(time.Second * 5)

	param := []string{hash.String()}
	rpcRes, err := call(t, "eth_getTxLogs", param)
	require.NoError(t, err)

	logs := new([]*ethtypes.Log)
	err = json.Unmarshal(rpcRes.Result, logs)
	require.NoError(t, err)

	require.Equal(t, 1, len(*logs))
	t.Log((*logs)[0])
	time.Sleep(time.Second)
}

func TestEth_GetFilterChanges_NoTopics(t *testing.T) {
	rpcRes, err := call(t, "eth_blockNumber", []string{})
	require.NoError(t, err)

	var res hexutil.Uint64
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	param := make([]map[string]interface{}, 1)
	param[0] = make(map[string]interface{})
	param[0]["topics"] = []string{}
	param[0]["fromBlock"] = res.String()
	param[0]["toBlock"] = zeroString // latest

	// deploy contract, emitting some event
	deployTestContract(t)

	rpcRes, err = call(t, "eth_newFilter", param)
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// get filter changes
	changesRes, err := call(t, "eth_getFilterChanges", []string{ID.String()})
	require.NoError(t, err)

	var logs []*ethtypes.Log
	err = json.Unmarshal(changesRes.Result, &logs)
	require.NoError(t, err)

	require.Equal(t, 1, len(logs))
	time.Sleep(time.Second)

	//t.Log(logs[0])
	// TODO: why is the tx hash in the log not the same as the tx hash of the transaction?
	//require.Equal(t, logs[0].TxHash, common.BytesToHash(hash))
}

func TestEth_GetFilterChanges_Addresses(t *testing.T) {
	// TODO: need transaction receipts to determine contract deployment address
}

func TestEth_GetFilterChanges_BlockHash(t *testing.T) {
	// TODO: need transaction receipts to determine tx block
}

// hash of Hello event
var helloTopic = "0x775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd738898"

// world parameter in Hello event
var worldTopic = "0x0000000000000000000000000000000000000000000000000000000000000011"

func deployTestContractWithFunction(t *testing.T) hexutil.Bytes {
	// pragma solidity ^0.5.1;

	// contract Test {
	//     event Hello(uint256 indexed world);
	//     event Test(uint256 indexed a, uint256 indexed b);

	//     constructor() public {
	//         emit Hello(17);
	//     }

	//     function test(uint256 a, uint256 b) public {
	//         emit Test(a, b);
	//     }
	// }

	bytecode := "0x608060405234801561001057600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a260c98061004d6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063eb8ac92114602d575b600080fd5b606060048036036040811015604157600080fd5b8101908080359060200190929190803590602001909291905050506062565b005b80827f91916a5e2c96453ddf6b585497262675140eb9f7a774095fb003d93e6dc6921660405160405180910390a3505056fea265627a7a72315820ef746422e676b3ed22147cd771a6f689e7c33ef17bf5cd91921793b5dd01e3e064736f6c63430005110032"

	from := getAddress(t)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["data"] = bytecode

	rpcRes, err := call(t, "eth_sendTransaction", param)
	require.NoError(t, err)

	var hash hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &hash)
	require.NoError(t, err)

	return hash
}

// Tests topics case where there are topics in first two positions
func TestEth_GetFilterChanges_Topics_AB(t *testing.T) {
	time.Sleep(time.Second)

	rpcRes, err := call(t, "eth_blockNumber", []string{})
	require.NoError(t, err)

	var res hexutil.Uint64
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	param := make([]map[string]interface{}, 1)
	param[0] = make(map[string]interface{})
	param[0]["topics"] = []string{helloTopic, worldTopic}
	param[0]["fromBlock"] = res.String()
	param[0]["toBlock"] = zeroString // latest

	deployTestContractWithFunction(t)

	rpcRes, err = call(t, "eth_newFilter", param)
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	// get filter changes
	changesRes, err := call(t, "eth_getFilterChanges", []string{ID.String()})
	require.NoError(t, err)

	var logs []*ethtypes.Log
	err = json.Unmarshal(changesRes.Result, &logs)
	require.NoError(t, err)

	require.Equal(t, 1, len(logs))
	time.Sleep(time.Second * 2)
}

func TestEth_GetFilterChanges_Topics_XB(t *testing.T) {
	rpcRes, err := call(t, "eth_blockNumber", []string{})
	require.NoError(t, err)

	var res hexutil.Uint64
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	param := make([]map[string]interface{}, 1)
	param[0] = make(map[string]interface{})
	param[0]["topics"] = []interface{}{nil, worldTopic}
	param[0]["fromBlock"] = res.String()
	param[0]["toBlock"] = "0x0" // latest

	deployTestContractWithFunction(t)

	rpcRes, err = call(t, "eth_newFilter", param)
	require.NoError(t, err)

	var ID hexutil.Bytes
	err = json.Unmarshal(rpcRes.Result, &ID)
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	// get filter changes
	changesRes, err := call(t, "eth_getFilterChanges", []string{ID.String()})
	require.NoError(t, err)

	var logs []*ethtypes.Log
	err = json.Unmarshal(changesRes.Result, &logs)
	require.NoError(t, err)

	require.Equal(t, 1, len(logs))
	time.Sleep(time.Second)
}

func TestEth_GetFilterChanges_Topics_XXC(t *testing.T) {
	// TODO: call test function, need tx receipts to determine contract address
}

func TestEth_GetLogs_NoLogs(t *testing.T) {
	param := make([]map[string][]string, 1)
	param[0] = make(map[string][]string)
	param[0]["topics"] = []string{}
	_, err := call(t, "eth_getLogs", param)
	require.NoError(t, err)
}

func TestEth_GetLogs_Topics_AB(t *testing.T) {
	rpcRes, err := call(t, "eth_blockNumber", []string{})
	require.NoError(t, err)

	var res hexutil.Uint64
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)

	param := make([]map[string]interface{}, 1)
	param[0] = make(map[string]interface{})
	param[0]["topics"] = []string{helloTopic, worldTopic}
	param[0]["fromBlock"] = res.String()
	param[0]["toBlock"] = zeroString // latest

	deployTestContractWithFunction(t)

	rpcRes, err = call(t, "eth_getLogs", param)
	require.NoError(t, err)

	var logs []*ethtypes.Log
	err = json.Unmarshal(rpcRes.Result, &logs)
	require.NoError(t, err)

	require.Equal(t, 1, len(logs))
}

func TestEth_NewPendingTransactionFilter(t *testing.T) {
	rpcRes, err := call(t, "eth_newPendingTransactionFilter", []string{})
	require.NoError(t, err)

	var code hexutil.Bytes
	err = code.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)
	require.NotNil(t, code)

	for i := 0; i < 5; i++ {
		deployTestContractWithFunction(t)
	}

	time.Sleep(10 * time.Second)

	// get filter changes
	changesRes, err := call(t, "eth_getFilterChanges", []string{code.String()})
	require.NoError(t, err)
	require.NotNil(t, changesRes)

	var txs []*hexutil.Bytes
	err = json.Unmarshal(changesRes.Result, &txs)
	require.NoError(t, err, string(changesRes.Result))

	require.True(t, len(txs) >= 2, "could not get any txs", "changesRes.Result", string(changesRes.Result))

}
