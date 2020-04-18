package BLC

import (
	"github.com/boltdb/bolt"
	"log"
)

type BlockchainIterator struct {
	CurrentHash []byte
	DB *bolt.DB
}

func (blockchain *Blockchain) Iterator() *BlockchainIterator{
	return &BlockchainIterator{blockchain.Tip,blockchain.DB}
}

func (bi *BlockchainIterator) Next() *Block{
	var block *Block
	err := bi.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		if b != nil{
			currentBlockBytes := b.Get(bi.CurrentHash)
			block = DeserialBlock(currentBlockBytes)
			bi.CurrentHash = block.PrevBlockHash
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}

	return block
}

