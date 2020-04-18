package BLC

import (
	"encoding/gob"
	"bytes"
	"log"
	"github.com/boltdb/bolt"
	"math/big"
	"os"
	"fmt"
	"time"
	"strconv"
	"encoding/hex"
	"publicchain/base58"
	"crypto/ecdsa"
)

const dbFile = "blockchain.db"
const blockBucket = "blocks"

type Blockchain struct {
	Tip []byte
	DB *bolt.DB
}


func (blockchain *Blockchain)NewBlockchainWithGenesisBlock(address string)*Blockchain{

	if dbExists(){
		fmt.Println("db have already exists")
		os.Exit(1)
	}

	db,err := bolt.Open(dbFile,0600,nil)
	if err != nil{
		log.Panic(err)
	}
	var blockHash []byte
	err = db.Update(func(tx *bolt.Tx) error {

		b,err := tx.CreateBucket([]byte(blockBucket))
		if err != nil{
			log.Panic(err)
		}
		if b != nil{
			genesisBlock := NewGenensisBlock(NewCoinbaseTx(address))

			err := b.Put(genesisBlock.Hash,genesisBlock.Serialize())
			if err != nil{
				log.Panic(err)
			}

			err = b.Put([]byte("l"),genesisBlock.Hash)
			if err != nil{
				log.Panic(err)
			}
			blockHash = genesisBlock.Hash
			
		}
		return nil
	})

	if err!=nil{
		log.Panic(err)
	}

	return &Blockchain{blockHash,db}
}

func DeserialBlock(d []byte) *Block{
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))

	err := decoder.Decode(&block)
	if err != nil{
		log.Panic(err)
	}

	return &block
}

func (blockchain *Blockchain) AddBlock(transactions []*Transaction){
	newBlock := NewBlock(transactions,blockchain.Tip)

	err := blockchain.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		err := b.Put(newBlock.Hash,newBlock.Serialize())
		if err != nil{
			log.Panic(err)
		}
		err = b.Put([]byte("l"),newBlock.Hash)
		if err != nil{
			log.Panic(err)
		}

		blockchain.Tip = newBlock.Hash
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
}

//find current user utxo
func (blockchain *Blockchain)FindUnspentTransactions(address string,txs []*Transaction) []*UTXO{
	var unUTXO []*UTXO
	spentTXOutputs := make(map[string][]int)

	for _,tx := range txs {
		if tx.IsCoinBase() == false{
			for _,in := range tx.Vin{
				publicKeyHash := base58.Decode(address)
				ripe160Hash := publicKeyHash[1:len(publicKeyHash)-addressChecksumLen]
				if in.UnlockRipemd160Hash(ripe160Hash){
					key := hex.EncodeToString(in.Txid)
					spentTXOutputs[key] = append(spentTXOutputs[key],in.Vout)
				}
			}
		}
	}

	for _,tx := range txs {
		work1:
		for index,out := range tx.Vout{
			if out.UnLockScriptPubKeyWithAddress(address){
				if len(spentTXOutputs) == 0{
					utxo := &UTXO{tx.ID,index,out}
					unUTXO = append(unUTXO,utxo)
				}else{
					for hash,indexArray := range spentTXOutputs{
						txHashStr := hex.EncodeToString(tx.ID)
						if hash == txHashStr{
							var isUnspentUTXO bool

							for _,outIndex := range indexArray{
								if index == outIndex{
									isUnspentUTXO = true
									continue work1
								}

								if isUnspentUTXO == false{
									utxo := &UTXO{tx.ID,index,out}
									unUTXO = append(unUTXO,utxo)
								}
							}
						}else {
							utxo := &UTXO{tx.ID,index,out}
							unUTXO = append(unUTXO,utxo)
						}
					}
				}
			}
		}
	}


	bi := blockchain.Iterator()
	for {
		block := bi.Next()

		for i := len(block.Transactions)-1; i>=0; i--{
			tx := block.Transactions[i]
			if tx.IsCoinBase() == false{
				for _,in := range tx.Vin{
					publicKeyHash := base58.Decode(address)
					ripe160Hash := publicKeyHash[1:len(publicKeyHash)-addressChecksumLen]
					if in.UnlockRipemd160Hash(ripe160Hash){
						key := hex.EncodeToString(in.Txid)
						spentTXOutputs[key] = append(spentTXOutputs[key],in.Vout)
					}
				}
			}

			work:
			for index,out := range tx.Vout{
				if out.UnLockScriptPubKeyWithAddress(address){
					if spentTXOutputs != nil{
						if len(spentTXOutputs) != 0{
							var isSpentUTXO bool
							for txHash,indexArray := range spentTXOutputs{
								for _,i := range indexArray{
									if index == i && txHash == hex.EncodeToString(tx.ID){
										isSpentUTXO = true
										continue work
									}
								}
							}

							if isSpentUTXO == false{
								utxo := &UTXO{tx.ID,index,out}
								unUTXO = append(unUTXO,utxo)
							}
						}else {
							utxo := &UTXO{tx.ID,index,out}
							unUTXO = append(unUTXO,utxo)
						}
					}
				}
			}
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if hashInt.Cmp(big.NewInt(0)) == 0{
			break
		}
	}
	return unUTXO
}

func (bc *Blockchain) GetBalance(address string) int{
	utxos := bc.FindUnspentTransactions(address,[]*Transaction{})
	var amount int
	for _,utxo := range utxos{
		amount = amount + utxo.Output.Value
	}
	return amount
}

func (bc *Blockchain) MineNewBlock(from,to,amount []string){

	utxoSet := &UTXOSet{bc}
	var txs []*Transaction

	for index,address := range from {
		value,_ := strconv.Atoi(amount[index])
		tx := NewSimpleTransaction(address,to[index],value,utxoSet,txs)
		txs = append(txs,tx)
	}

	//reward
	tx := NewCoinbaseTx(from[0])
	txs = append(txs,tx)

	var block *Block

	bc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		if b!= nil{
			hash := b.Get([]byte("l"))
			blockBytes := b.Get(hash)
			block = DeserialBlock(blockBytes)
		}
		return nil
	})

	var _txs []*Transaction

	for _,tx := range txs{
		if bc.VerifyTransaction(tx,_txs) != true{
			log.Panic("Signature is error...")
		}

		_txs = append(_txs,tx)
	}

	block = NewBlock(txs,block.Hash)

	bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		if b!= nil{
			b.Put(block.Hash,block.Serialize())
			b.Put([]byte("l"),block.Hash)
			bc.Tip = block.Hash
		}
		return nil
	})
}

func dbExists() bool{
	if _, err := os.Stat(dbFile); os.IsNotExist(err){
		return false
	}

	return true
}

func (blc *Blockchain) printChain(){

	blockchainIterator := blc.Iterator()

	for {
		block := blockchainIterator.Next()
		fmt.Printf("PrevBlockHash: %x\n",block.PrevBlockHash)
		fmt.Printf("Timestamp: %s\n",time.Unix(block.Timestamp,0).Format("2006-01-02 03:04:05 PM"))
		fmt.Printf("Hash: %x\n",block.Hash)
		fmt.Printf("Nonce: %d\n",block.Nonce)
		fmt.Println("Transactions:")

		for _,tx := range block.Transactions{
			fmt.Printf("\tTXID: %x\n",tx.ID)
		}
		fmt.Println()

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if big.NewInt(0).Cmp(&hashInt) == 0{
			break
		}
	}
}

func BlockchainObject() *Blockchain{
	db,err := bolt.Open(dbFile,0600,nil)

	if err != nil{
		log.Panic(err)
	}

	var hash []byte
	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blockBucket))
		if b != nil{
			hash = b.Get([]byte("l"))

		}
		return nil
	})

	return &Blockchain{hash,db}
}

func (blockchain *Blockchain) FindSpendableUTXOs(from string,amount int,txs []*Transaction)(int,map[string][]int){
	utxos := blockchain.FindUnspentTransactions(from,txs)
	var value int
	spendAbleUTXO := make(map[string][]int)
	for _,utxo := range utxos{
		value = value + utxo.Output.Value
		hash := hex.EncodeToString(utxo.TxHash)
		spendAbleUTXO[hash] = append(spendAbleUTXO[hash],utxo.Index)

		if value >= amount{
			break
		}
	}
	if value < amount {
		fmt.Printf("%s 's fund is not enough\n",from)
		os.Exit(1)
	}

	return value, spendAbleUTXO
}

func (blockchain *Blockchain)SignTransaction(tx *Transaction,private ecdsa.PrivateKey,txs []*Transaction){
	if tx.IsCoinBase(){
		return
	}

	prevTxs := make(map[string]Transaction)
	for _,vin := range tx.Vin{
		prevTx, err := blockchain.FindTransaction(vin.Txid,txs)
		if err != nil{
			log.Panic(err)
		}
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(private,prevTxs)
}

func (blockchain *Blockchain)FindTransaction(ID []byte,txs []*Transaction)(Transaction,error){

	for _,tx := range txs {
		if bytes.Compare(tx.ID,ID) == 0{
			return *tx,nil
		}
	}

	bci := blockchain.Iterator()
	for{
		block := bci.Next()

		for _,tx := range block.Transactions{
			if bytes.Compare(tx.ID,ID) == 0{
				return *tx,nil
			}
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if big.NewInt(0).Cmp(&hashInt) == 0{
			break
		}
	}

	// return Transaction{},errors.New("Transaction is not found")
	return Transaction{},nil
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction,txs []*Transaction) bool{
	prevTXs := make(map[string]Transaction)

	for _,vin := range tx.Vin{
		prevTx,err := bc.FindTransaction(vin.Txid,txs)
		if err != nil{
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTXs)
}

func (bc *Blockchain)FindUTXOMap() map[string]*TXOutputs{
	bci := bc.Iterator()

	spentableUTXOsMap := make(map[string][]*TXInput)

	utxoMaps := make(map[string]*TXOutputs)

	for {
		block := bci.Next()

		for i := len(block.Transactions)-1; i>=0 ; i--{
			txOutputs := &TXOutputs{[]*UTXO{}}
			tx := block.Transactions[i]

			if tx.IsCoinBase() == false{
				for _,txInput := range tx.Vin{
					txHash := hex.EncodeToString(txInput.Txid)
					spentableUTXOsMap[txHash]  = append(spentableUTXOsMap[txHash],txInput)
				}
			}

			txHash := hex.EncodeToString(tx.ID)

			WorkOutLoop:
			for index,out := range tx.Vout{
				txInputs := spentableUTXOsMap[txHash]
				if len(txInputs)>0{
					isSpent := false

					for _,in := range txInputs{

						outPublicKey := out.Ripemd160Hash
						inPublicKey := in.PublicKey

						if bytes.Compare(outPublicKey,RipemdHash160(inPublicKey)) == 0{
							if index == in.Vout{
								isSpent = true
								continue WorkOutLoop
							}
						}
					}
					if isSpent == false{
						utxo := &UTXO{tx.ID,index,out}
						txOutputs.UTXOS = append(txOutputs.UTXOS,utxo)
					}
				}else {
					utxo := &UTXO{tx.ID,index,out}
					txOutputs.UTXOS = append(txOutputs.UTXOS,utxo)
				}
			}

			utxoMaps[txHash] = txOutputs
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)
		if hashInt.Cmp(big.NewInt(0)) == 0{
			break
		}
	}
	return utxoMaps
}