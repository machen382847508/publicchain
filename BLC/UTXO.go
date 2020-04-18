package BLC

import (
	"github.com/boltdb/bolt"
	"log"
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"publicchain/base58"
)

const utxoTableName = "utxoTableName"

type UTXO struct {
	TxHash []byte
	Index int
	Output *TXOutput
}

type TXOutputs struct {
	UTXOS []*UTXO
}

type UTXOSet struct {
	blockchain *Blockchain
}

func (utxoset *UTXOSet)ResetUTXOSet(){
	err := utxoset.blockchain.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))
		if b != nil{
			err := tx.DeleteBucket([]byte(utxoTableName))
			if err != nil{
				log.Panic(err)
			}
		}

		b,_ = tx.CreateBucket([]byte(utxoTableName))
		if b!=nil{
			txOutputsMap := utxoset.blockchain.FindUTXOMap()
			for keyHash,outs := range txOutputsMap{
				txHash,_ := hex.DecodeString(keyHash)
				b.Put(txHash,outs.SerializeTXOutputs())
			}
		}

		return nil
	})
	if err != nil{
		log.Panic(err)
	}
}

func (txoutputs *TXOutputs) SerializeTXOutputs() []byte{
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(txoutputs)
	if err != nil{
		log.Panic(err)
	}
	return result.Bytes()
}

func  DerializeTXOutputs(txoutputsBytes []byte) *TXOutputs{
	var txoutputs TXOutputs

	decoder := gob.NewDecoder(bytes.NewReader(txoutputsBytes))
	err := decoder.Decode(&txoutputs)
	if err != nil{
		log.Panic(err)
	}

	return &txoutputs
}

func (utxoset *UTXOSet)findUTXOForAddress(address string) []*UTXO{
	var utxos []*UTXO

	err := utxoset.blockchain.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))
		c := b.Cursor()

		for k,v := c.First();k != nil;k,v = c.Next(){
			txOutputs := DerializeTXOutputs(v)
			for _,utxo := range txOutputs.UTXOS{
				if utxo.Output.UnLockScriptPubKeyWithAddress(address){
					utxos = append(utxos,utxo)
				}
			}
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return utxos
}

func (utxoset *UTXOSet)GetBalance(address string) int{
	UTXOS := utxoset.findUTXOForAddress(address)
	var amount int
	for _,utxo := range UTXOS{
		amount += utxo.Output.Value
	}

	return amount
}

func (utxoset *UTXOSet)FindUnpackageSpendableUTXOs(from string,txs []*Transaction)[]*UTXO{
	var unUTXO []*UTXO
	spentTXOutputs := make(map[string][]int)

	for _,tx := range txs {
		if tx.IsCoinBase() == false{
			for _,in := range tx.Vin{
				publicKeyHash := base58.Decode(from)
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
			if out.UnLockScriptPubKeyWithAddress(from){
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
	return unUTXO
}

func (utxoset *UTXOSet)FindSpendableUTXOS(from string,amount int,txs []*Transaction)(int,map[string][]int){
	unPackageUTXOS := utxoset.FindUnpackageSpendableUTXOs(from,txs)
	money := 0
	spentableUTXO := make(map[string][]int)
	for _,utxo := range unPackageUTXOS{
		money += utxo.Output.Value
		txHash := hex.EncodeToString(utxo.TxHash)
		spentableUTXO[txHash] = append(spentableUTXO[txHash],utxo.Index)
		if money >= amount{
			return money,spentableUTXO
		}
	}

	utxoset.blockchain.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))
		if b != nil{
			c := b.Cursor()
			UTXOBREAK:
			for k,v := c.First();k != nil;k,v = c.Next(){
				txOutputs := DerializeTXOutputs(v)
				for _,u := range txOutputs.UTXOS{
					money += u.Output.Value
					txHash := hex.EncodeToString(u.TxHash)
					spentableUTXO[txHash] = append(spentableUTXO[txHash],u.Index)
					if money >= amount{
						break UTXOBREAK
					}
				}
			}
		}
		return nil
	})

	if money < amount{
		log.Panic("yu e bu zu.....")
	}

	return money,spentableUTXO
}

func (utxoset *UTXOSet)Update(){
	block := utxoset.blockchain.Iterator().Next()
	var ins []*TXInput
	outsMap := make(map[string]*TXOutputs)

	for _,tx := range block.Transactions{
		for _,in := range tx.Vin{
			ins = append(ins,in)
		}
	}

	for _,tx := range block.Transactions{
		var utxos []*UTXO
		for index,out := range tx.Vout{

			isSpent := false
			for _,in := range ins{
				if in.Vout == index && bytes.Compare(tx.ID,in.Txid) == 0 &&
					bytes.Compare(out.Ripemd160Hash,RipemdHash160(in.PublicKey))==0{
						isSpent = true
						continue
				}
			}
			if isSpent == false{
				utxo := &UTXO{tx.ID,index,out}
				utxos = append(utxos,utxo)
			}
		}
		if len(utxos) > 0{
			txHash := hex.EncodeToString(tx.ID)
			outsMap[txHash] = &TXOutputs{utxos}
		}
	}

	err := utxoset.blockchain.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))
		if b!=nil{
			for _,in := range ins{
				txOutputsBytes := b.Get(in.Txid)
				if len(txOutputsBytes) == 0{
					continue
				}

				txOutputs := DerializeTXOutputs(txOutputsBytes)
				var utxos []*UTXO
				isNeedDelete := false
				for _,utxo := range txOutputs.UTXOS{
					if in.Vout == utxo.Index && bytes.Compare(utxo.Output.Ripemd160Hash,RipemdHash160(in.PublicKey)) == 0{
						isNeedDelete = true
					}else {
						utxos = append(utxos,utxo)
					}
				}
				if isNeedDelete{
					b.Delete([]byte(in.Txid))
					if len(utxos) > 0{
						preTXOutputs := outsMap[hex.EncodeToString(in.Txid)]
						preTXOutputs.UTXOS = append(preTXOutputs.UTXOS,utxos...)
						outsMap[hex.EncodeToString(in.Txid)] = preTXOutputs
					}
				}
			}

			for keyHash,outPuts := range outsMap{
				keyHashBytes,_ := hex.DecodeString(keyHash)
				b.Put(keyHashBytes,outPuts.SerializeTXOutputs())
			}
		}

		return nil
	})

	if err != nil{
		log.Panic(err)
	}

}