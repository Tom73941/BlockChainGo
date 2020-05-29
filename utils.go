package main

import (
	"bytes"
	"encoding/binary"
	"log"
)

//将类型转化为字节数组，小端模式
func IntToHex(num int32) []byte{
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.LittleEndian,num)
	DoError(err)
	return buff.Bytes()
}

//将类型转化为字节数组，大端模式
func IntToHex2(num int32) []byte{
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.BigEndian,num)
	DoError(err)
	return buff.Bytes()
}

//字节反转
func ReverseBytes(data []byte){
	for i,j := 0,len(data)-1; i<j; i,j = i+1, j-1{
		data[i],data[j] = data[j], data[i]
	}
}

func DoError(err error) {
	if err!=nil{
		log.Panic(err)
	}
}