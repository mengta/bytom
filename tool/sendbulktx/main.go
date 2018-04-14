package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/tool/sendbulktx/core"
)

var (
	acctNum   int
	btmNum    int
	thdNum    int
	txBtmNum  int
	sendAcct  string
	sendasset string
)

func init() {
	flag.IntVar(&acctNum, "acctNum", 10, "Number of created accounts")
	flag.IntVar(&btmNum, "btmNum", 10000, "Number of btm to send trading accounts")
	flag.IntVar(&thdNum, "thdNum", 5, "goroutine num")
	flag.IntVar(&txBtmNum, "txBtmNum", 10, "Number of transactions btm")
	flag.StringVar(&sendAcct, "sendAcct", "0CHHJNM3G0A02", "who send btm")
	flag.StringVar(&sendasset, "sendasset", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "send asset")

}

func main() {
	flag.Parse()
	controlPrograms := make([]string, acctNum)
	txidChan := make(chan string)

	// create key
	param := []string{"alice", "123"}
	fmt.Println("*****************create key start*****************")
	var xpub pseudohsm.XPub
	resp, b := core.SendReq(core.CreateKey, param)
	if !b {
		resp, b := core.SendReq(core.ListKeys, param)
		if !b {
			os.Exit(1)
		}
		dataList, _ := resp.([]interface{})
		for _, item := range dataList {
			core.RestoreStruct(item, &xpub)
			if strings.EqualFold(xpub.Alias, param[0]) {
				break
			}

		}
	} else {
		core.RestoreStruct(resp, &xpub)
	}
	fmt.Println("*****************create key end*****************")

	fmt.Println("*****************create account start*****************")
	for i := 0; i < acctNum; i++ {
		// create account
		name := fmt.Sprintf("alice%d", i)
		param = []string{name, xpub.XPub.String()}
		_, b = core.SendReq(core.CreateAccount, param)
		// create receiver
		param = []string{name}
		resp, b = core.SendReq(core.CreateReceiver, param)
		if !b {
			os.Exit(1)
		}
		var recv txbuilder.Receiver
		core.RestoreStruct(resp, &recv)
		recvText, _ := recv.ControlProgram.MarshalText()
		controlPrograms[i] = string(recvText)
	}
	fmt.Println("*****************create account end*****************")

	threadTxNum := btmNum / (thdNum * txBtmNum)
	txBtm := fmt.Sprintf("%d", txBtmNum*2000)
	fmt.Println("*****************send tx start*****************")
	// send btm to account
	for i := 0; i < thdNum; i++ {
		go core.Sendbulktx(threadTxNum, txBtm, sendAcct, sendasset, controlPrograms, txidChan)
	}
	var txid string
	fail := 0
	sucess := 0
	//以读写方式打开文件，如果不存在，则创建
	file, error := os.OpenFile("./txid.txt", os.O_RDWR|os.O_CREATE, 0766)
	if error != nil {
		fmt.Println(error)
	}
	for {
		select {
		case txid = <-txidChan:
			if strings.EqualFold(txid, "") {
				fail++
			} else {
				sucess++
				file.WriteString(txid)
				file.WriteString("\n")
			}
		default:
			if (sucess + fail) >= (thdNum * threadTxNum) {
				file.Close()
				os.Exit(0)
			}
			time.Sleep(time.Second * 2)
		}
	}

}
