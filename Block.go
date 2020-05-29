package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

var (
	maxnonce int32 = math.MaxInt32
)

type Block struct{
	Version int32
	PreBlockHash []byte
	Merkleroot []byte
	Hash []byte
	Time int32
	Bits int32
	Nonce int32
	Transations []*Transation
	Height int32
}

func (block *Block) serialize() []byte{
	result := bytes.Join(
		[][]byte{
			IntToHex(block.Version),
			block.PreBlockHash,
			block.Merkleroot,
			IntToHex(block.Time),
			IntToHex(block.Bits),
			IntToHex(block.Nonce)},
			[]byte{},
		)
	return result
}

//高效的序列化
func (b *Block) Serialize() []byte{
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(b)
	DoError(err)

	return encoded.Bytes()
}

//反序列化
func DeserializeBlock(d []byte) *Block{
	var block Block

	decode := gob.NewDecoder(bytes.NewReader(d))
	err := decode.Decode(&block)
	DoError(err)

	return &block
}

func (b *Block)String(){
	fmt.Printf("Version: %s\n",strconv.FormatInt(int64(b.Version),10))
	fmt.Printf("PreBlockHash: %x\n",b.PreBlockHash)
	fmt.Printf("MerkleRoot: %x\n",b.Merkleroot)
	fmt.Printf("Hash: %x\n",b.Hash)
	fmt.Printf("Times: %s\n",strconv.FormatInt(int64(b.Time),10))
	fmt.Printf("Bits: %s\n",strconv.FormatInt(int64(b.Bits),10))
	fmt.Printf("Nonce: %s\n",strconv.FormatInt(int64(b.Nonce),10))
}

//18 1B7B74
//计算难度
func CalculateTargetFast(bite []byte) []byte{
	var result []byte
	//第一个字节，计算指数
	exponent := bite[:1]
	fmt.Printf("%x\n",exponent)

	//计算后面的3个系数
	coeffient := bite[1:]
	fmt.Printf("%x\n",coeffient)

	//将字节，他的16进制为“18” 转化为了string“18”
	str := hex.EncodeToString(exponent)	//18
	fmt.Printf("str=%s\n",str)

	//将字符串18转化为10进制的int64 24
	exp,_:=strconv.ParseInt(str,16,8)
	fmt.Printf("%d\n",exp)

	//拼接，计算出目标hash
	result = append(bytes.Repeat([]byte{0x00},32-int(exp)),coeffient...)
	result = append(result,bytes.Repeat([]byte{0x00},32-len(result))...)
	return result
}

//新建一个区块
func NewBlock(trans []*Transation, preBlockHash []byte ,height int32) *Block{
	//初始化区块
	block := &Block{
		2,
		preBlockHash,
		[]byte{},
		[]byte{},
		int32(time.Now().Unix()),
		404454260,
		0,
		trans,
		height,
	}
	pow := NewProofOfWork(block)

	nonce,hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash

	block.String()
	return block
}

//创建创世区块
func NewGensisBlock(trans []*Transation) * Block{
	//初始化区块
	block := &Block{
		2,
		[]byte{},
		[]byte{},
		[]byte{},
		int32(time.Now().Unix()),
		404454260,
		0,
		trans,
		0,
	}
	pow := NewProofOfWork(block)

	nonce,hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash

	block.String()
	return block
}

//创建默克尔树根节点
func (b *Block) createMerkleTreeRoot(txs []*Transation){
	var transHash [][]byte

	for _,tx := range txs{
		transHash = append(transHash,tx.Hash())
	}
	//将区块和交易连接起来了
	mTree := NewMerkleTree(transHash)

	b.Merkleroot = mTree.RootNode.Data
}


