package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

//模拟Coinbase挖矿收益值，指定为100
const subsidy=100

type Transation struct{
	ID []byte
	Vin []TXInput
	Vout []TXOutput
}

//所有权转移的过程
type TXInput struct{
	TXid []byte
	Voutindex int
	Signature []byte	//"Bob"，签名可以解锁后一笔交易，让其可以使用
	//后来增加公钥
	Pubkey []byte		//公钥
}

type TXOutput struct{
	Value int
	PubkeyHash []byte	//公钥的Hash，"Bob"，要输出给谁，他和前一笔交易的签名是一致的
}

type TXOutputs struct{
	Outputs []TXOutput
}

func (outs TXOutputs) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(outs)
	DoError(err)
	return encoded.Bytes()
}

func DeserializeOutputs(data []byte) TXOutputs{
	var outputs TXOutputs

	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	DoError(err)

	return outputs
}

//将Transation标准化输出
func (tx Transation) String() string{
	var lines[] string

	lines = append(lines, fmt.Sprintf("———— Transation %x：",tx.ID))
	for i,input := range tx.Vin{
		lines = append(lines, fmt.Sprintf("        Input %d：",i))
		lines = append(lines, fmt.Sprintf("           TXID %x：",input.TXid))
		lines = append(lines, fmt.Sprintf("           voutindex %d：",input.Voutindex))
		lines = append(lines, fmt.Sprintf("           Signature %x：",input.Signature))
	}
	for i,output := range tx.Vout{
		lines = append(lines, fmt.Sprintf("        Output %d：",i))
		lines = append(lines, fmt.Sprintf("           Value %x：",output.Value))
		lines = append(lines, fmt.Sprintf("           PubkeyHash %x：",output.PubkeyHash))
	}
	return strings.Join(lines,"\n")
}

//序列化
func (tx Transation) Serialize() []byte{
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(tx)
	DoError(err)
	return encoded.Bytes()
}

//计算Hash值
func (tx *Transation)Hash() []byte{
	txcopy := *tx
	//将ID中的Hash值清空
	txcopy.ID = []byte{}

	hash := sha256.Sum256(txcopy.Serialize())
	return hash[:]
}

func(out *TXOutput) Lock(addr []byte){
	decodeAddr := Base58Decode(addr)

	pubkeyHash := decodeAddr[1:len(decodeAddr)-4]
	out.PubkeyHash = pubkeyHash
}

//根据金额和地址新建一个输出
func NewTXOutput(value int, address string) *TXOutput{
	txo := &TXOutput{value,nil}
	//txo.PubkeyHash = []byte(address)
	txo.Lock([]byte(address))
	return txo
}

//新建一个第一笔Coinbase挖矿交易
//func NewCoinbaseTX(to string) *Transation{
func NewCoinbaseTX(to,data string) *Transation{
	//所有的输入为空
	//txin := TXInput{[]byte{},-1,nil}
	txin := TXInput{[]byte{},-1,nil,[]byte(data)}
	//该交易中只有Coinbase向矿工奖励的输出
	txout := NewTXOutput(subsidy,to)

	tx := Transation{nil,[]TXInput{txin},[]TXOutput{*txout}}
	tx.ID = tx.Hash()
	return &tx
}

//判断是否可以解锁
//func (out *TXOutput) CanBeUnlockedWith(unlockdata string) bool{
func (out *TXOutput) CanBeUnlockedWith(pubkeyhash []byte) bool{
	//return string(out.PubkeyHash) == unlockdata
	return bytes.Compare(out.PubkeyHash,pubkeyhash)==0
}

//func (in *TXInput) CanUnlockOutputWith(unlockdata string) bool{
func (in *TXInput) CanUnlockOutputWith(unlockdata []byte) bool{
	lockinghash := HashPubKey(in.Pubkey)
	//return string(in.Signature) == unlockdata
	return bytes.Compare(lockinghash,unlockdata)==0
}

//判断是否是第一笔交易
func (tx Transation) IsCoinbase() bool{
	return len(tx.Vin) ==1 && len(tx.Vin[0].TXid)==0 && tx.Vin[0].Voutindex == -1
}

func (tx Transation) Sign(prikey ecdsa.PrivateKey, prevTXs map[string]Transation) {
	if tx.IsCoinbase(){
		return
	}
	//检查过程
	for _,vin := range tx.Vin{
		if prevTXs[hex.EncodeToString(vin.TXid)].ID == nil{
			log.Panic("Error: 交易ID错误")
		}
	}
	txcopy := tx.TrimmedCopy()
	for inID,vin := range txcopy.Vin{
		prevTx := prevTXs[hex.EncodeToString(vin.TXid)]		//前一笔交易的结构体

		txcopy.Vin[inID].Signature = nil
		//这笔交易的这笔输入的引用的前一笔交易的输出的公钥哈希
		txcopy.Vin[inID].Pubkey = prevTx.Vout[vin.Voutindex].PubkeyHash
		txcopy.ID = txcopy.Hash()
		r,s,err := ecdsa.Sign(rand.Reader,&prikey,txcopy.ID)
		DoError(err)
		signature := append(r.Bytes(),s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}
}

func (tx *Transation) TrimmedCopy() Transation {
	var inputs 	[]TXInput
	var outputs []TXOutput

	for _,vin := range tx.Vin{
		//没有管签名和pubkey
		inputs = append(inputs, TXInput{vin.TXid,vin.Voutindex,nil,nil})
	}
	for _,vout := range tx.Vout{
		outputs = append(outputs,TXOutput{vout.Value,vout.PubkeyHash})
	}
	txCopy := Transation{tx.ID,inputs,outputs}
	return txCopy
}

//验证签名是否OK
func (tx *Transation) Verify(prevTXs map[string]Transation) bool {
	if tx.IsCoinbase(){
		return true
	}
	for _,vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.TXid)].ID == nil {
			log.Panic("Error: 交易ID错误")
		}
	}
	txcopy := tx.TrimmedCopy()	//新建一个副本
	curve := elliptic.P256()	//准备椭圆曲线

	//对原始的交易循环，不是txcopy
	for inID,vin := range tx.Vin{
		//第一步：准备当初要签名的数据
		prevTx := prevTXs[hex.EncodeToString(vin.TXid)]		//前一笔交易的结构体
		txcopy.Vin[inID].Signature = nil
		//这笔交易的这笔输入的引用的前一笔交易的输出的公钥哈希
		txcopy.Vin[inID].Pubkey = prevTx.Vout[vin.Voutindex].PubkeyHash
		txcopy.ID = txcopy.Hash()

		r := big.Int{}
		s := big.Int{}
		signlen := len(vin.Signature)	//真实长度还是OK的，只是拷贝中的没有了
		r.SetBytes(vin.Signature[:(signlen/2)])
		s.SetBytes(vin.Signature[(signlen/2):])

		//公钥的椭圆曲线上的x,y坐标
		x := big.Int{}
		y := big.Int{}
		keylen  := len(vin.Pubkey)
		x.SetBytes(vin.Pubkey[:(keylen/2)])
		y.SetBytes(vin.Pubkey[(keylen/2):])

		//第二步：构建一个公钥的结构体
		rowPubkey := ecdsa.PublicKey{curve,&x,&y}

		//第三步：验证
		if ecdsa.Verify(&rowPubkey,txcopy.ID,&r,&s) == false {
			fmt.Printf("rowPubkey: %x\n",rowPubkey)
			fmt.Printf("txcopy.ID: %x\n",txcopy.ID)

			return false
		}
		txcopy.Vin[inID].Pubkey = nil
	}
	//循环结束，都没有错误，返回True
	return true
}

//新建一个转账交易
func NewUTXOTransation(from , to string, amount int, bc* BlockChain) *Transation{
	var inputs []TXInput
	var outputs []TXOutput

	//通过钱包的方式获取公钥
	ws,err := NewWallets()
	DoError(err)
	wallet := ws.GetWallet(from)

	acc, validoutputs := bc.FindSpendableOutputs(HashPubKey(wallet.PublicKey), amount)
	if acc < amount{
		log.Panic("Error:Not enough funds")
	}

	for txid, outs := range validoutputs{
		txID, err := hex.DecodeString(txid)
		DoError(err)
		for _, out := range outs{
			//最开始的交易中是不做签名的
		 	input := TXInput{ txID,out,nil, wallet.PublicKey}
		 	inputs = append(inputs,input)
		}
	}

	outputs = append(outputs,*NewTXOutput(amount,to))

	//将剩余的钱再转给自己
	if acc>amount{
		outputs = append(outputs,*NewTXOutput(acc-amount,from))
	}

	tx := Transation{nil,inputs,outputs}
	tx.ID = tx.Hash()

	//from的私钥进行数据签名
	bc.SignTransation(&tx , wallet.PrivateKey)

	return &tx
}