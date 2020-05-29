package main

import (
	"bolt"
	"encoding/hex"
	"log"
)

type UTXOSet struct{
	Bchain *BlockChain
}

const utxoBucket = "ChainSet"

//重置UTXO数据索引
func (u UTXOSet) ReIndex(){
	db := u.Bchain.db

	//存在，则删除
	bucketName := []byte(utxoBucket)
	err := db.Update(func(tx *bolt.Tx) error{
		err2 := tx.DeleteBucket(bucketName)
		if err2 !=nil && err2 != bolt.ErrBucketNotFound{
			log.Panic(err2)
		}
		//重新创建桶
		_,err3 := tx.CreateBucket(bucketName)
		DoError(err3)
		return nil
	})
	DoError(err)
	//得到所有UTXO
	UTXOs := u.Bchain.FindAllUTXOs()
	//放到数据库中去
	err = db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket(bucketName)
		for txID,outs :=range UTXOs{
			key,err4 := hex.DecodeString(txID)
			DoError(err4)
			err4 = b.Put(key,outs.Serialize())
			DoError(err4)
		}
		return nil
	})
	DoError(err)
}

func (utxos UTXOSet) FindUTXOByPubkeyHash(pubkeyHash []byte) []TXOutput{
	var UTXOs []TXOutput
	db := utxos.Bchain.db
	err := db.View(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k,v := c.First(); k!=nil; k,v=c.Next(){
			outs := DeserializeOutputs(v)
			for _,out := range outs.Outputs{
				//必须能被pubkeyHash解锁，才是我这个地址的UTXO
				if out.CanBeUnlockedWith(pubkeyHash){
					UTXOs = append(UTXOs,out)
				}
			}
		}
		return nil
	})
	DoError(err)
	return UTXOs
}

//当有区块有新的交易的时候，进行区块的更新
func (utxos UTXOSet) Update(block *Block){
	db := utxos.Bchain.db

	err := db.Update(func(tx * bolt.Tx)error{
		b:= tx.Bucket([]byte(utxoBucket))

		for _,tx := range block.Transations{
			if tx.IsCoinbase() == false{
				for _,vin := range tx.Vin{
					updateouts := TXOutputs{}
					outsbyte := b.Get(vin.TXid)
					outs := DeserializeOutputs(outsbyte)

					for outIdx,out := range outs.Outputs{
						if outIdx != vin.Voutindex{
							//新的output列表
							updateouts.Outputs = append(updateouts.Outputs,out)
						}
					}
					//长度为零，则意味着所有的都已花费了
					if len(updateouts.Outputs) ==0 {
						err := b.Delete(vin.TXid)
						DoError(err)
					}else {
						err := b.Put(vin.TXid,updateouts.Serialize())
						DoError(err)
					}

				}
			}
			//新的交易
			newOutputs := TXOutputs{}
			for _,out := range tx.Vout{
				newOutputs.Outputs = append(newOutputs.Outputs,out)
			}
			err := b.Put(tx.ID,newOutputs.Serialize())
			DoError(err)
		}
		return nil
	})
	DoError(err)
}