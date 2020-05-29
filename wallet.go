package main

import (
	"bytes"
	"crypto/ecdsa"
	"golang.org/x/crypto/ripemd160"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
)

const version = byte(0x00)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey []byte
}

func NewWallet() *Wallet{
	privateKey, publicKey := newKeyPair()
	wallet := Wallet{privateKey,publicKey}

	return &wallet
}

//生成私钥和公钥
func newKeyPair() (ecdsa.PrivateKey,[]byte){
	//生成椭圆曲线,  secp256r1 曲线。 比特币当中的曲线是secp256k1
	curve :=elliptic.P256()
	private,err :=ecdsa.GenerateKey(curve,rand.Reader)
	DoError(err)

	pubkey :=append(private.PublicKey.X.Bytes(),private.PublicKey.Y.Bytes()...)

	return *private,pubkey
}

func HashPubKey(pubkey []byte) []byte{
	pubkeyHash := sha256.Sum256(pubkey)
	PIPEMD160Hasher := ripemd160.New()

	_,err:=	PIPEMD160Hasher.Write(pubkeyHash[:])
	DoError(err)

	publicRIPEMD160 := PIPEMD160Hasher.Sum(nil)
	return publicRIPEMD160
}

//返回检查值
func checksum(payload []byte) []byte{
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	//checksum 是前面的4个字节
	checksum := secondSHA[:4]
	return checksum
}

//产生比特币地址
func (w Wallet)GetAddress() []byte{
	//1、计算pubkeuhash
	pubkeyHash256 := HashPubKey(w.PublicKey)

	//2、计算checksum
	versionPayload := append([]byte{version},pubkeyHash256...)
	check := checksum(versionPayload)
	fullPayload := append(versionPayload,check...)

	//3、base58编码
	address:=Base58Encode(fullPayload)
	//返回地址
	return address
}

func ValidateAddress(address []byte) bool{
	//反编码
	pubkeyHash := Base58Decode([]byte(address))
	//取最后4个字节
	checkSum := pubkeyHash[len(pubkeyHash)-4:]
	//得到中间的publickey
	publickeyHash := pubkeyHash[1:len(pubkeyHash)-4]
	targetCheckSum := checksum(append([]byte{0x00},publickeyHash...))

	return bytes.Compare(checkSum,targetCheckSum)==0
}