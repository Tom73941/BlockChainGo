package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
)
const nodeVersion =0x00
const commandLength = 12

var nodeAddress string

type Version struct{
	Version int
	BestHeight int32
	AddrFrom string
}

type getblocksAddr struct{
	AddrFrom string
}

type inv struct{
	AddrFrom string
	Type	string
	Items	[][]byte
}

type getdata struct{
	AddrFrom string
	Type	string
	ID		[]byte
}
type blocksend struct{
	AddrFrom string
	Block	 []byte
}

//存储本地链中已有的所有block HASH
var blockInTransit [][]byte

//种子节点
var knownNodes = []string {"localhost:3000"}

func (ver *Version) String(){
	fmt.Printf("Version : %x\n", ver.Version)
	fmt.Printf("BestHeight : %d\n", ver.BestHeight)
	fmt.Printf("AddrFrom : %s\n", ver.AddrFrom)
}

func StartServer(nodeID, mineAddress string, bc *BlockChain){
	nodeAddress = fmt.Sprintf("localhost:%s",nodeID)
	ln,err := net.Listen("tcp",nodeAddress)
	DoError(err)
	//延迟关闭，使用完的时候关闭
	defer ln.Close()

	//bc := NewBolckChain("1HCJytT5c3aFPCoduiZZxDrhkv4DVKoBTS")
	if nodeAddress != knownNodes[0]{
		SendVersion(knownNodes[0],bc)
	}

	//监听循环，始终进行
	for{
		conn,err2 := ln.Accept()
		DoError(err2)
		//启动协程
		go HandleConection(conn,bc)
	}
}

func SendVersion(addr string, bc *BlockChain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(Version{nodeVersion,bestHeight, nodeAddress})
	request := append(commandToBytes("version"),payload...)

	sendData(addr,request)
}

func HandleConection(conn net.Conn, bc *BlockChain) {
	request,err := ioutil.ReadAll(conn)
	DoError(err)
	//获取命令
	command := bytesToCommand(request[:commandLength])
	switch command{
	case "version":
		fmt.Printf("收到Version %s。\n",command)
		handleVersion(request,bc)
	case "getblocks":
		handleGetBlock(request,bc)
	case "inv":
		handleInv(request,bc)
	case "getdata":
		handleGetData(request,bc)
	case "block":
		handleBlock(request,bc)
	}
}

func handleBlock(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload blocksend

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	DoError(err)

	blockdata := payload.Block
	block := DeserializeBlock(blockdata)

	//添加一个区块
	bc.AddBlock(block)
	fmt.Printf("Receive a Block!")

	//
	if len(blockInTransit)>0 {
		blockHash := blockInTransit[0]
		sendGetData(payload.AddrFrom,"block",blockHash)
		blockInTransit = blockInTransit[1:]
	}else{
		//更新UTXO
		set := UTXOSet{bc}
		set.ReIndex()
	}
}

func handleGetData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	DoError(err)

	if  payload.Type == "block" {
		block,err:=bc.GetBlock([]byte(payload.ID))
		DoError(err)

		sendBlock(payload.AddrFrom,&block)
	}
}

func sendBlock(addr string, block *Block) {
	data := blocksend{nodeAddress,block.Serialize()}
	payload := gobEncode(data)

	request := append(commandToBytes("block"),payload...)
	sendData(addr,request)
}

func handleInv(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	DoError(err)

	fmt.Printf("Receive inventory %d, %s",len(payload.Items),payload.Type)
	//for _,b:=range payload.Items{
	//	fmt.Printf("\n%x\n",b)
	//}

	if  payload.Type == "block"{
		blockInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom,"block",blockHash)

		newInTransit := [][]byte{}

		for _, b := range blockInTransit{
			if bytes.Compare(b, blockHash) !=0 {
				newInTransit = append(newInTransit,b)
			}
		}
		blockInTransit = newInTransit
	}
}

func sendGetData(addr string, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress,kind,id})

	request := append(commandToBytes("getdata"),payload...)
	sendData(addr,request)
}

func handleGetBlock(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getblocksAddr
	//取指令后面的数据
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	DoError(err)

	//存储Hash值
	blshash := bc.GetBlocksHash()
	sendInv(payload.AddrFrom,"block",blshash)

}

func sendInv(addr string, kind string, items [][]byte) {
	inventory := inv{nodeAddress,kind,items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"),payload...)

	sendData(addr, request)
}

func handleVersion(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload Version
	//取指令后面的数据
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	DoError(err)

	payload.String()
	myBestHeight := bc.GetBestHeight()

	//获取外部的高度
	foreignerBestHeight := payload.BestHeight
	//需要从外部获取节点
	if myBestHeight < foreignerBestHeight{
		sendGetBlock(payload.AddrFrom)
	}else{
		SendVersion(payload.AddrFrom,bc)
	}

	//把节点添加到已知节点列表中去
	if !nodeIsKnow(payload.AddrFrom){
		knownNodes = append(knownNodes,payload.AddrFrom)
	}
}

func sendGetBlock(addr string) {
	payload := gobEncode(getblocksAddr{nodeAddress})
	request := append(commandToBytes("getblocks"),payload...)

	sendData(addr, request)
}

func nodeIsKnow(addr string) bool {
	for _,node := range knownNodes{
		if node ==addr{
			return true
		}
	}
	return false
}

func sendData(addr string, data []byte) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("%s is no available",addr)

		var updateNodes []string
		for _,node := range knownNodes{
			if node !=addr{
				updateNodes = append(updateNodes,node)
			}
		}
		knownNodes = updateNodes
	}
	defer conn.Close()

	//开始传递数据
	_,err = io.Copy(conn, bytes.NewReader(data))
	DoError(err)
}


func commandToBytes(command string) []byte{
	var rtbytes [commandLength]byte
	for i,c := range command{
		rtbytes[i] = byte(c)
	}
	return rtbytes[:]
}

func bytesToCommand(bytes []byte) string{
	var command []byte

	for _,b := range bytes{
		if b != 0x00{
			command = append(command,b)
		}
	}
	return fmt.Sprintf("%s",command)
}

func gobEncode(data interface{}) []byte{
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	DoError(err)
	return buff.Bytes()
}
