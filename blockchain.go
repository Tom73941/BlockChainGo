package main

import (
	"bolt"
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
)

const dbFile = "blockchain.db"
const blockBucket = "blocks"
const genesisData = "Tom73941's Blockchain"

type BlockChain struct{
	tip []byte	//最近的一个区块的hash值
	db *bolt.DB
}

type BlockChainIterator struct{
	currentHash []byte	//当前的Hash
	db *bolt.DB
}

//遍历BlockChain
func (bc *BlockChain) iterator() *BlockChainIterator{
	bci := &BlockChainIterator{bc.tip,bc.db}
	return bci
}

func (i* BlockChainIterator) Next() *Block{
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		deblock := b.Get(i.currentHash)
		block = DeserializeBlock(deblock)
		return nil
	})
	DoError(err)
	i.currentHash = block.PreBlockHash
	return block
}

//打印BlockChain
func (bc *BlockChain) printBlockChain(){
	bci := bc.iterator()
	for {
		block := bci.Next()
		block.String()
		fmt.Println()
		//一直到创世区块
		if len(block.PreBlockHash)==0 {
			break
		}
	}
}

//在链上添加一个区块
func (bc *BlockChain) AddBlock(block *Block) {
	err := bc.db.Update(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		//看Hash是否存在，存在返回nil
		blockIndb := b.Get(block.Hash)
		if blockIndb!=nil {//说明已经在链上了
			return nil
		}
		//添加到区块链中
		blockData := block.Serialize()
		err1:=b.Put(block.Hash,blockData)
		DoError(err1)

		//处理“l”，最高的区块，也是最后的区块
		lastHash := b.Get([]byte("l"))
		lastBlockdata := b.Get(lastHash)
		lastblock := DeserializeBlock(lastBlockdata)

		if block.Height > lastblock.Height{
			//将结果更新到数据库中标记为“l”的值
			err1 = b.Put([]byte("l"),block.Hash)
			DoError(err1)
			bc.tip = block.Hash
		}

		return nil
	})
	DoError(err)
}

//在区块链上挖矿一个Block
func (bc *BlockChain) MineBlock(trans []*Transation) *Block {
	var lastHash []byte
	var lastHeight int32

	//第一步，先验证前面的交易是否有效
	for _,tx := range trans{
		if bc.VirifyTransation(tx) != true{
			log.Panic("EROOR: INVALID Transation")
		}else{
			fmt.Println("Verify Success!")
		}
	}

	//第二步：读取区块链
	err := bc.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		lastHash = b.Get([]byte("l"))
		blockdata := b.Get(lastHash)
		block := DeserializeBlock(blockdata)
		lastHeight = block.Height
		return nil
	})
	DoError(err)

	//第三步，开始挖矿，得到一个有效的区块，高度加1
	newBlock := NewBlock(trans, lastHash, lastHeight+1)

	//第四步，放到数据库中去
	bc.db.Update(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		err:=b.Put(newBlock.Hash,newBlock.Serialize())
		DoError(err)
		//将结果更新到数据库中标记为“l”的值
		err = b.Put([]byte("l"),newBlock.Hash)
		DoError(err)
		bc.tip = newBlock.Hash
		return nil
	})
	return newBlock
}

//新建一个BlockChain
func NewBolckChain(address string) *BlockChain{
	var tip []byte
	db,err := bolt.Open(dbFile,0600,nil)
	DoError(err)
	err = db.Update(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))

		//不存在，则创建新的区块
		if b==nil{
			fmt.Println("区块链不存在，创建一个新的区块链")

			trans := NewCoinbaseTX(address,genesisData)
			genesis := NewGensisBlock([]*Transation{trans})
			b,err := tx.CreateBucket([]byte(blockBucket))
			DoError(err)
			//将结果放到数据库中去
			err = b.Put(genesis.Hash,genesis.Serialize())
			DoError(err)
			err = b.Put([]byte("l"),genesis.Hash)
			DoError(err)
			tip = genesis.Hash
		}else{
			tip = b.Get([]byte("l"))
		}
		return nil
	})
	DoError(err)
	bc := BlockChain{tip,db}

	//关联UTXO数据库
	set := UTXOSet{&bc}
	set.ReIndex()

	return &bc
}

//func (bc *BlockChain) FindUnSpentTX(address string) []Transation{
func (bc *BlockChain) FindUnSpentTX(pubkeyhash []byte) []Transation{
	var unspentTXs []Transation			//所有未花费的交易

	//存储已经花费的交易
	spendTXOs := make(map[string][]int)	//string 交易的Hash值 ->[]int 存储已花费的交易序号

	bci := bc.iterator()

	//遍历区块链中所有的区块
	for{
		//迭代器，从区块链的最后一个遍历到最开始的一个
		block := bci.Next()

		//第二层循环，遍历每一个区块的所有交易
		for _,tx := range block.Transations{
			txID := hex.EncodeToString(tx.ID)

			output:
			//第三层循环，遍历每一个交易中的所有输出
			for outIdx, out := range tx.Vout{
				//判断，spendTXOs[txID] !=nil，则意味着有交易被花费了
				if spendTXOs[txID] !=nil{
					//第四层循环，遍历spendTXOs中所有的交易序号
					for _,spentOut:= range spendTXOs[txID]{
						//spentOut == outIdx意味着，存储的spentOut是被花费的
						if spentOut == outIdx{
							continue output
						}
					}
				}
				//如果这个地址与当前判断者的地址匹配，则，意味着这笔交易未花费
				if out.CanBeUnlockedWith(pubkeyhash){
					//添加到未花费交易切片中
					unspentTXs = append(unspentTXs, *tx)
				}
			}
			//因为Coinbase没有输入，所以跳过
			if tx.IsCoinbase() == false{
				//再次遍历所有的输入
				for _,in := range tx.Vin{
					//与判断者一致的地址，存起来
					if in.CanUnlockOutputWith(pubkeyhash){
						//已经花费的交易
						inTxId := hex.EncodeToString(in.TXid)	//哪一笔交易被花费了，ID
						spendTXOs[inTxId] = append(spendTXOs[inTxId],in.Voutindex)
					}
				}
			}
		}

		//到了第一个区块，结束循环
		if len(block.PreBlockHash) == 0{
			break
		}
	}
	fmt.Println(unspentTXs)
	return unspentTXs
}

//查找指定地址的所有未花费交易
func (bc *BlockChain) FindUTXO(pubkeyhash []byte) []TXOutput{
	var UTXOs []TXOutput
	unspendTrans := bc.FindUnSpentTX(pubkeyhash)
	for _,tx := range unspendTrans{
		for _,out := range tx.Vout{
			//可以解锁就是未花费的交易
			if out.CanBeUnlockedWith(pubkeyhash){
				UTXOs = append(UTXOs,out)
			}
		}
	}
	return UTXOs
}

//查找所有未花费交易
func (bc *BlockChain) FindAllUTXOs() map[string]TXOutputs{
	UTXOs := make(map[string]TXOutputs)
	spentTXs := make(map[string][]int)

	bci := bc.iterator()
	for{
		block := bci.Next()
		for _,tx := range block.Transations{
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx,out:= range tx.Vout{
				if spentTXs[txID] !=nil{
					for _,spendOutIds:= range spentTXs[txID]{
						if spendOutIds == outIdx{
							//代表交易是已经花费过了
							continue Outputs
						}
					}
				}
				outs:= UTXOs[txID]
				outs.Outputs = append(outs.Outputs,out)
				UTXOs[txID] = outs
			}
			//取所有的输入
			if tx.IsCoinbase()==false{
				for _,in := range tx.Vin{
					inTXID := hex.EncodeToString(in.TXid)
					spentTXs[inTXID] = append(spentTXs[inTXID],in.Voutindex)
				}
			}
		}
		if len(block.PreBlockHash)==0{
			break
		}
	}
	return UTXOs
}

func (bc *BlockChain) FindSpendableOutputs(pubkeyhash []byte, amount int) (int, map[string][]int){
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnSpentTX(pubkeyhash)
	accumulated :=0

	Work:
	//遍历所有的指定地址的所有未花费输出
	for _,tx := range unspentTXs{
		txID := hex.EncodeToString(tx.ID)

		for outIdx,out := range tx.Vout{
			if out.CanBeUnlockedWith(pubkeyhash) && accumulated < amount{
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID],outIdx)

				if(accumulated >= amount){
					break Work
				}
			}
		}
	}
	return accumulated, unspentOutputs
}

func (bc *BlockChain) FindTransationByID(ID []byte)(Transation,error){
	bci := bc.iterator()

	for{
		block := bci.Next()
		for _,tx := range block.Transations{
			if bytes.Compare(tx.ID,ID)==0{
				//找到对应的ID，返回交易
				return *tx,nil
			}
		}
		//遍历到第一个区块，跳出循环
		if(len(block.PreBlockHash)==0){
			break
		}
	}
	return Transation{}, errors.New("Tramsation is not found.")
}

//from的私钥进行数据签名
func (bc *BlockChain) SignTransation(tx *Transation ,prikey ecdsa.PrivateKey){
	prevTXs := make(map[string]Transation)
	//循环遍历所有交易的输入
	for _,vin := range tx.Vin{
		prevTX, err := bc.FindTransationByID(vin.TXid)
		DoError(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	//对所有的输入引用的前一笔交易进行签名
	tx.Sign(prikey,prevTXs)
}

//验证签名
func (bc *BlockChain)VirifyTransation(tx *Transation) bool{
	prevTXs := make(map[string]Transation)
	//循环遍历所有交易的输入
	for _,vin := range tx.Vin{
		prevTX, err := bc.FindTransationByID(vin.TXid)
		DoError(err)
		//存在，则放入到prevTXs
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func (bc *BlockChain) GetBestHeight() int32{
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		lasthash := b.Get([]byte("l"))
		blockdata:= b.Get(lasthash)
		lastBlock = *DeserializeBlock(blockdata)
		return nil
	})
	DoError(err)
	return lastBlock.Height
}

func (bc *BlockChain) GetBlocksHash() [][]byte {
	var blocks [][]byte
	bci := bc.iterator()
	for{
		block := bci.Next()
		blocks = append(blocks, block.Hash)

		if len(block.PreBlockHash)==0{
			break
		}
	}

	return blocks
}

func (bc *BlockChain) GetBlock(blockHash []byte) (Block,error) {
	var block Block

	err := bc.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		blockData := b.Get(blockHash)

		//没有找到区块
		if blockData == nil{
			return errors.New("Block is not found!")
		}

		block = *DeserializeBlock(blockData)

		return nil
	})

	return block,err
}
