package BLC

import (
	"math/big"
	"bytes"
	"math"
	"crypto/sha256"
	"fmt"
)

const targetbits = 17

var (
	maxNonce = math.MaxInt64
)

type ProofOfWork struct {
	block *Block
	target *big.Int
}

func NewProofOfWork(block *Block) *ProofOfWork{
	target := big.NewInt(1)
	target.Lsh(target,uint(256-targetbits))
	pow := &ProofOfWork{block,target}

	return pow
}

func (pow *ProofOfWork) prepareData(nonce int) []byte{
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.HashTransaction(),
			IntToHex(pow.block.Timestamp),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}

func (pow *ProofOfWork) Run()(int, []byte){
	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	fmt.Printf("Start Mining New Block....")
	for nonce < maxNonce{
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x",hash)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1{
			break
		}else {
			nonce ++
		}
	}
	return nonce,hash[:]
}