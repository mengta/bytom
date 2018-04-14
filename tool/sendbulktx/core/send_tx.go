package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bytom/api"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/util"
)

const (
	CreateKey      = "create_key"
	ListKeys       = "list_keys"
	CreateAccount  = "create_account"
	CreateAsset    = "create_asset"
	CreateReceiver = "CreateReceiver"
	BuildTx        = "build_tx"
	SignTx         = "SignTx"
	SubmitTx       = "submit_tx"
	GetTransaction = "get_transaction"
)

var buildSpendReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_receiver", "asset_id": "%s", "amount": %s, "receiver":{"control_program": "%s"}}
	]}`

var (
	buildType     = ""
	btmGas        = "20000000"
	accountQuorum = 1
)

// RestoreStruct Restore data
func RestoreStruct(data interface{}, out interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if ok != true {
		fmt.Println("invalid type assertion")
		os.Exit(util.ErrLocalParse)
	}

	rawData, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(util.ErrLocalParse)
	}
	json.Unmarshal(rawData, out)
}

// SendReq genetate tx and send data
func SendReq(method string, args []string) (interface{}, bool) {
	var param interface{}
	var methodPath string
	switch method {
	case CreateKey:
		ins := keyIns{Alias: args[0], Password: args[1]}
		param = ins
		methodPath = "/create-key"
		//return ins, true
	case ListKeys:
		methodPath = "/list-keys"
	case CreateAccount:
		ins := account{}
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			fmt.Println("CreateAccount error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		ins.RootXPubs = append(ins.RootXPubs, xpub)
		ins.Quorum = accountQuorum
		ins.Alias = args[0]
		ins.AccessToken = ""
		param = ins
		methodPath = "/create-account"
		//return ins, true
	case CreateAsset:
		ins := asset{}
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			fmt.Println("CreateAsset error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		ins.RootXPubs = append(ins.RootXPubs, xpub)
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.AccessToken = ""
		param = ins
		methodPath = "/create-asset"
		//return ins, true
	case CreateReceiver:
		var ins = Reveive{AccountAlias: args[0]}
		param = ins
		methodPath = "/create-account-receiver"
		//return ins, true
	case BuildTx:
		accountInfo := args[0]
		assetInfo := args[1]
		amount := args[2]
		receiverProgram := args[3]
		buildReqStr := fmt.Sprintf(buildSpendReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, receiverProgram)
		var ins api.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &ins); err != nil {
			fmt.Println("generate build tx is error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		param = ins
		methodPath = "/build-transaction"
	case SignTx:
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			fmt.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		ins := struct {
			Password []string           `json:"password"`
			Txs      txbuilder.Template `json:"transaction"`
		}{Password: []string{""}, Txs: template}
		param = ins
		methodPath = "/sign-transaction"
	case SubmitTx:
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			fmt.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		ins := struct {
			Password string             `json:"password"`
			Txs      txbuilder.Template `json:"transaction"`
		}{Password: "123456", Txs: template}
		param = ins
		methodPath = "/sign-submit-transaction"
	case GetTransaction:
		ins := &struct {
			TxID string `json:"tx_id"`
		}{TxID: args[0]}
		param = ins
		methodPath = "/get-transaction"
	default:
		return "", false
	}
	data, exitCode := util.ClientCall(methodPath, &param)
	if exitCode > util.Success {
		return "", false
	}
	return data, true
}

// Sendbulktx send asset tx
func Sendbulktx(threadTxNum int, txBtmNum string, sendAcct string, sendasset string, controlPrograms []string, txidChan chan string) {
	arrayLen := len(controlPrograms)
	for i := 0; i < threadTxNum; i++ {
		//build tx
		receiver := controlPrograms[i/arrayLen]
		if strings.EqualFold(receiver, "") {
			txidChan <- ""
			continue
		}

		param := []string{sendAcct, sendasset, txBtmNum, receiver}
		resp, b := SendReq(BuildTx, param)
		if !b {
			txidChan <- ""
			continue
		}
		//dataMap, _ := resp.(map[string]interface{})
		rawTemplate, _ := json.Marshal(resp)
		//sign

		// submit
		param = []string{string(rawTemplate)}
		resp, b = SendReq(SubmitTx, param)
		if !b {
			//os.Exit(1)
			fmt.Println("SignSubmitTx fail")
			txidChan <- ""
			continue
		}
		type txId struct {
			Txid string `json:"tx_id"`
		}
		var out txId
		RestoreStruct(resp, &out)
		txidChan <- out.Txid
	}
}
