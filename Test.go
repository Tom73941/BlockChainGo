package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

//测试新的序列化
func TestNewSerialize(){
	//初始化区块
	block := &Block{
		2,
		[]byte{},
		[]byte{},
		[]byte{},
		1418755780,
		404454260,
		0,
		[]*Transation{},
		0,
	}
	deBlock := DeserializeBlock(block.Serialize())
	deBlock.String()
}

//测试POW
func TestPow(){
	//初始化区块
	block := &Block{
		2,
		[]byte{},
		[]byte{},
		[]byte{},
		1418755780,
		404454260,
		0,
		[]*Transation{},
		0,
	}
	pow := NewProofOfWork(block)

	nonce,_ := pow.Run()
	block.Nonce = nonce

	fmt.Printf("POW:",pow.Validate())
}

//测试创建默克尔树
func TestCreateMerkleTreeRoot(){
	//初始化区块
	block := &Block{
		2,
		[]byte{},
		[]byte{},
		[]byte{},
		1418755780,
		404454260,
		0,
		[]*Transation{},
		0,
	}

	txin := TXInput{[]byte{},-1,nil,nil}
	txout:= TXOutput{subsidy,[]byte("First")}
	tx := Transation{nil,[]TXInput{txin},[]TXOutput{txout}}

	txin2 := TXInput{[]byte{},-1,nil,nil}
	txout2:= TXOutput{subsidy,[]byte("Second")}
	tx2 := Transation{nil,[]TXInput{txin2},[]TXOutput{txout2}}

	var trans []*Transation
	trans = append(trans,&tx,&tx2)

	block.createMerkleTreeRoot(trans)
	fmt.Printf("%x\n",block.Merkleroot)
}

//模拟挖矿
func BTCMine(){
	//前一区块Hash
	prev,_ := hex.DecodeString("000000000000000016145aa12fa7e81a304c38aec3d7c5208f1d33b587f966a6")
	ReverseBytes(prev)
	//默克尔根
	merkleroot,_ := hex.DecodeString("3a4f410269fcc4c7885770bc8841ce6781f15dd304ae5d2770fc93a21dbd70d7")
	ReverseBytes(merkleroot)
	block := Block{2,
		prev,
		merkleroot,
		[]byte{},
		1418755780,
		404454260,
		0,
		[]*Transation{},
		0,
	}

	//计算目标Hash
	var target, currentHash big.Int
	targethash := CalculateTargetFast(IntToHex2(block.Bits))
	target.SetBytes(targethash)

	//block.nonce = 1865996500
	//一直计算到最大值
	for block.Nonce<maxnonce{
		data := block.serialize()
		firsthash := sha256.Sum256(data)
		secondhash:= sha256.Sum256(firsthash[:])
		ReverseBytes(secondhash[:])
		fmt.Printf("nonce:%d, currentHash: %x\n",block.Nonce, secondhash)
		currentHash.SetBytes(secondhash[:])

		//当前Hash小于目标值Hash，找到了
		if currentHash.Cmp(&target) == -1{
			break
		} else {
			block.Nonce++
		}
	}
}

//比特币序列化
func BTCSerialize(){
	//BTC Block #334599
	//PreHash:	000000000000000016145aa12fa7e81a304c38aec3d7c5208f1d33b587f966a6
	//Hash：	00000000000000000a1f57cd656e5522b7bac263aa33fc98c583ad68de309603
	//Merkle:	3a4f410269fcc4c7885770bc8841ce6781f15dd304ae5d2770fc93a21dbd70d7
	//Version：	2
	//Bits		404,454,260
	//nonce		1,865,996,595
	//time		2014-12-17 02:49	//有个疑问，老师的视频中的时间是2014-12-16 18:49:40
	//			日期代表的就是1970-01-01的时间，25569 *365*24*60*60

	//版本
	var version int32 = 2
	fmt.Printf("%x\n",IntToHex(version))
	//前一个Hash
	prev,_ := hex.DecodeString("000000000000000016145aa12fa7e81a304c38aec3d7c5208f1d33b587f966a6")
	ReverseBytes(prev)
	fmt.Printf("%x\n",prev)
	//默克尔根
	merkleroot,_ := hex.DecodeString("3a4f410269fcc4c7885770bc8841ce6781f15dd304ae5d2770fc93a21dbd70d7")
	ReverseBytes(merkleroot)
	fmt.Printf("%x\n",merkleroot)
	//交易时间
	var time int32 = 1418755780
	fmt.Printf("%x\n",IntToHex(time))
	//交易难度
	var bits int32 = 404454260
	fmt.Printf("%x\n",IntToHex(bits))
	//随机数
	var nonce int32 = 1865996595
	fmt.Printf("%x\n",IntToHex(nonce))
	//拼接
	result := bytes.Join([][]byte{IntToHex(version),prev,merkleroot,IntToHex(time),IntToHex(bits),IntToHex(nonce)},[]byte{})
	fmt.Printf("%x\n",result)

	//Double Hash
	firsthash := sha256.Sum256(result)
	resulthash:= sha256.Sum256(firsthash[:])
	ReverseBytes(resulthash[:])
	fmt.Printf("%x\n",resulthash)
}

func TestBoltDB(){
	blockbhain := NewBolckChain("1HCJytT5c3aFPCoduiZZxDrhkv4DVKoBTS")
	blockbhain.MineBlock([]*Transation{})
	blockbhain.MineBlock([]*Transation{})
	blockbhain.printBlockChain()
}

func TestCLI(){
	bc := NewBolckChain("1HCJytT5c3aFPCoduiZZxDrhkv4DVKoBTS")

	cli :=CLI{bc}
	cli.Run()
}

func TestWallet(){
	wallet :=NewWallet()

	fmt.Printf("私钥：%x\n",wallet.PrivateKey.D.Bytes())
	fmt.Printf("公钥：%x\n",wallet.PublicKey)
	fmt.Printf("地址：%x\n",wallet.GetAddress())
	addr,_ := hex.DecodeString("314d695932734d566d66764638576e507137733475707a617778466d517262524851")
	fmt.Printf("验证：%d\n",ValidateAddress(addr))

	//私钥：310052935e50914bd4ad9552229f61057ff5ac908ce46993fb1e7bf95b505cc0
	//公钥：18d6d6212d47bf61496eb3c09174a04a22ae73531ab8f72e2355288bb90176c4aed7a8100a628de793bf7d8d0197b5ac1330e1a0827a4413aefc681e6fedd5bd
	//地址：314d74354379514b4b4d4859466639564b47486150767a545a31324c6a6b4e6d684a
	//验证：%!d(bool=true)

	//1BQv99Q51QgiW9n7oxNixemgUuVzbLteX8
	//1HCJytT5c3aFPCoduiZZxDrhkv4DVKoBTS
	//1BmMQFgUraCZq3CvWkfh6hYoUsryVzXs48
}
