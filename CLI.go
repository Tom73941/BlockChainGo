package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct {
	bc *BlockChain
}

func (cli *CLI)validataArgs(){
	if len(os.Args)<1{
		println("没有输入参数")
		os.Exit(1)
	}
	fmt.Println(os.Args)
}

func (cli *CLI)addBlock(){
	cli.bc.MineBlock([]*Transation{})
}

//打印区块链到控制台
func (cli *CLI)printChain(){
	cli.bc.printBlockChain()
}

func (cli *CLI)createWallet(){
	wallets,_ := NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	fmt.Printf("Your address : %s\n",address)
}

func (cli *CLI)listAddress(){
	wallets,err:= NewWallets()
	DoError(err)

	addresses := wallets.getAddresses()
	for _,addr := range addresses{
		fmt.Println(addr)
	}
}

func (cli *CLI)send(from,to string, amount int){
	tx := NewUTXOTransation(from,to,amount,cli.bc)

	newBlock := cli.bc.MineBlock([]*Transation{tx})
	set := UTXOSet{cli.bc}
	set.Update(newBlock)

	//通过硬编码的方式getBalance
	//cli.getBalance("1HCJytT5c3aFPCoduiZZxDrhkv4DVKoBTS")
	//cli.getBalance("1BQv99Q51QgiW9n7oxNixemgUuVzbLteX8")
	//cli.getBalance("1BmMQFgUraCZq3CvWkfh6hYoUsryVzXs48")

	fmt.Println("Success!")
}

func (cli *CLI) getBestHeight() {
	lastHeight := cli.bc.GetBestHeight()
	fmt.Println(lastHeight)
}

func (cli *CLI)getBalance(address string){
	balance := 0
	decodeAddr := Base58Decode([]byte(address))
	pubkeyHash := decodeAddr[1:len(decodeAddr)-4]

	//重新构建，使用数据库
	//UTXOs := cli.bc.FindUTXO(pubkeyHash)
	set := UTXOSet{cli.bc}
	UTXOs := set.FindUTXOByPubkeyHash(pubkeyHash)

	for _,out := range UTXOs{
		balance += out.Value
	}
	fmt.Printf("balance of %s : %d\n",address,balance)
}

func (cli *CLI)printUsage(){
	fmt.Println("Usages:")
	fmt.Println("addblock: 新增区块")
	fmt.Println("printchain: 打印区块链")
	fmt.Println("getbalance: 获取指定地址的Balance")
}

func (cli *CLI) Run(){
	cli.validataArgs()

	//准备网络环境
	nodeID :=os.Getenv("NODE_ID")
	if nodeID ==""{
		fmt.Println("NODE_ID is not set.")
		os.Exit(1)
	}

	addBlockCmd 	:= flag.NewFlagSet("addblock",flag.ExitOnError)
	printChainCmd 	:= flag.NewFlagSet("printchain",flag.ExitOnError)
	getBalanceCmd 	:= flag.NewFlagSet("getbalance",flag.ExitOnError)
	sendCmd 		:= flag.NewFlagSet("send",flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet",flag.ExitOnError)
	listAddressCmd 	:= flag.NewFlagSet("listaddress",flag.ExitOnError)
	getBestHeightCmd:= flag.NewFlagSet("getbestheight",flag.ExitOnError)
	//网络命令
	startNodeCmd 	:= flag.NewFlagSet("startnode",flag.ExitOnError)

	getBalanceAddr 	:= getBalanceCmd.String("address","","The address to get Balance of ")
	sendFrom 		:= sendCmd.String("from","","Source wallet address")
	sendTo 			:= sendCmd.String("to","","Destination wallet address")
	sendAmount		:= sendCmd.Int("amount",0,"Amount to send")
	startNodeMinner	:= startNodeCmd.String("minner","","minner address")

	switch os.Args[1]{
	case "startnode":
		err:=startNodeCmd.Parse(os.Args[2:])
		DoError(err)
	case "getbestheight":
		err:=getBestHeightCmd.Parse(os.Args[2:])
		DoError(err)
	case "createwallet":
		err:=createWalletCmd.Parse(os.Args[2:])
		DoError(err)
	case "listaddress":
		err:=listAddressCmd.Parse(os.Args[2:])
		DoError(err)
	case "send":
		err:=sendCmd.Parse(os.Args[2:])
		DoError(err)
	case "getbalance":
		err:=getBalanceCmd.Parse(os.Args[2:])
		DoError(err)
	case "addblock":
		err:=addBlockCmd.Parse(os.Args[2:])
		DoError(err)
	case "printchain":
		err:=printChainCmd.Parse(os.Args[2:])
		DoError(err)
	default:
		cli.printUsage()
		os.Exit(1)
	}
	if addBlockCmd.Parsed(){
		cli.addBlock()
	}
	if printChainCmd.Parsed(){
		cli.printChain()
	}
	if(getBalanceCmd.Parsed()){
		if *getBalanceAddr == ""{
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddr)
	}
	if(sendCmd.Parsed()){
		if *sendFrom == "" || *sendTo=="" || *sendAmount<=0{
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
	if createWalletCmd.Parsed(){
		cli.createWallet()
	}
	if listAddressCmd.Parsed(){
		cli.listAddress()
	}
	if getBestHeightCmd.Parsed(){
		cli.getBestHeight()
	}
	if startNodeCmd.Parsed(){
		nodeID := os.Getenv("NODE_ID")
		if nodeID ==""{
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMinner)
	}
}

func (cli *CLI) startNode(nodeid string, minneraddr string) {
	fmt.Printf("Starting node %s\n",nodeid)
	if len(minneraddr) >0 {
		if ValidateAddress([]byte(minneraddr)){
			fmt.Printf("Minner is on %s\n",minneraddr)
		}else{
			log.Panic("Error Address")
		}
	}
	StartServer(nodeid,minneraddr,cli.bc)
}
