package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
)

//钱包集合文件
const walletFile = "wallet.dat"
//钱包集合
type Wallets struct {
	WalletStore map[string]*Wallet
}

func (ws *Wallets) CreateWallet() string{
	wallet := NewWallet()

	address := fmt.Sprintf("%s",wallet.GetAddress())
	ws.WalletStore[address] = wallet

	return address
}

//根据地址取到钱包
func (ws *Wallets) GetWallet(addr string) Wallet{
	return *ws.WalletStore[addr]
}

//取钱包中所有的地址
func (ws *Wallets) getAddresses() []string{
	var addresses []string
	for addr,_ := range ws.WalletStore{
		addresses = append(addresses,addr)
	}
	return addresses
}

func (ws *Wallets) SaveToFile(){
	var content bytes.Buffer

	//注册椭圆曲线函数，因为结构体重包含有接口，所以必须要事先指定是什么接口实例，
	//否则不知道编译连接什么接口
	gob.Register(elliptic.P256())
	//将结构体序列化
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	DoError(err)

	err = ioutil.WriteFile(walletFile,content.Bytes(),0777)
	DoError(err)
}

func (ws *Wallets) LoadFromFile() error {
	if _,err := os.Stat(walletFile); os.IsNotExist(err){
		return err
	}
	fileContent,err := ioutil.ReadFile(walletFile)
	DoError(err)
	var wallets Wallets

	//注册椭圆曲线函数，因为结构体重包含有接口，所以必须要事先指定是什么接口实例，
	//否则不知道编译连接什么接口
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	DoError(err)
	ws.WalletStore = wallets.WalletStore
	return err
}

//创建钱包集合
func NewWallets() (*Wallets,error){
	wallets := Wallets{}
	wallets.WalletStore = make(map[string]*Wallet)
	err:=wallets.LoadFromFile()

	return &wallets,err
}
