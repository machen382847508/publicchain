// Block
package BLC

import (
	"time"
	"strconv"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

type Block struct {
	Timestamp     int64  //区块时间戳
	PrevBlockHash []byte //前区块哈希
	Transactions  []*Transaction
	Hash          []byte //当前区块哈希
	Nonce 		  int
}

//创建新的区块并返回
func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), prevBlockHash, transactions, []byte{},0}

	pow := NewProofOfWork(block)
	nonce,hash := pow.Run()
	block.Hash = hash
	block.Nonce = nonce

	return block
}

func (block *Block)SetHash(){
	timestamp := []byte(strconv.FormatInt(block.Timestamp,2))
	headers := bytes.Join([][]byte{block.PrevBlockHash,block.HashTransaction(),timestamp},[]byte{})
	hash := sha256.Sum256(headers)
	block.Hash = hash[:]
}

func NewGenensisBlock(coinbase *Transaction) *Block{
	return NewBlock([]*Transaction{coinbase},[]byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})
}

func (b *Block)Serialize() []byte{
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)

	if err != nil{
		log.Panic(err)
	}
	return result.Bytes()
}

func (b *Block) HashTransaction() []byte{
	var txHashes [][]byte
	var txHash [32]byte

	for _,tx := range b.Transactions{
		txHashes = append(txHashes,tx.ID)
	}
	txHash = sha256.Sum256(bytes.Join(txHashes,[]byte{}))
	return txHash[:]
}