package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const targetBits = 16

type ProofOfWork struct{
	block *Block
	target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork{
	target := big.NewInt(1)
	target.Lsh(target,uint(256-targetBits))
	fmt.Printf("%x\n",target.Bytes())
	pow := &ProofOfWork{b,target}
	return pow
}

//准备数据
func (pow * ProofOfWork) prepareData(nonce int32) []byte{
	data := bytes.Join(
		[][]byte{
			IntToHex(pow.block.Version),
			pow.block.PreBlockHash,
			pow.block.Merkleroot,
			IntToHex(pow.block.Time),
			IntToHex(pow.block.Bits),
			IntToHex(nonce)},
		[]byte{},
	)
	return data
}

//挖矿
func (pow *ProofOfWork) Run() (int32,[]byte){
	var nonce int32
	var firsthash, secondhash [32]byte
	var currentHash big.Int
	//一直计算到最大值
	for nonce=0;nonce<maxnonce;nonce++{
		data := pow.prepareData(nonce)
		//Double Hash
		firsthash = sha256.Sum256(data)
		secondhash= sha256.Sum256(firsthash[:])
		//fmt.Printf("%x\n",secondhash)
		currentHash.SetBytes(secondhash[:])

		//当前Hash小于目标值Hash，找到了
		if currentHash.Cmp(pow.target) == -1{
			break
		}
	}
	return nonce,secondhash[:]
}

//验证工作量
func (pow *ProofOfWork) Validate() bool{
	var hashInt big.Int
	data:=pow.prepareData(pow.block.Nonce)

	firstHash := sha256.Sum256(data)
	secondHash:= sha256.Sum256(firstHash[:])

	hashInt.SetBytes(secondHash[:])
	isValid := hashInt.Cmp(pow.target)==-1

	return isValid
}