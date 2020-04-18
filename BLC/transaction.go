package BLC

import (
	"bytes"
	"encoding/gob"
	"log"
	"crypto/sha256"
	"encoding/hex"
	"publicchain/base58"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/elliptic"
	"math/big"
	"time"
)

const subsidy  =  10
type Transaction struct {
	ID []byte
	Vin []*TXInput
	Vout []*TXOutput
}

type TXInput struct {
	Txid []byte
	Vout int
	Signature []byte
	PublicKey []byte
}

type TXOutput struct {
	Value int
	Ripemd160Hash []byte
}



func NewCoinbaseTx(address string) *Transaction{

	txin := &TXInput{[]byte{},-1,nil,[]byte{}}
	txout := NewTxOutput(subsidy,address)
	tx := &Transaction{nil,[]*TXInput{txin},[]*TXOutput{txout}}
	tx.SetID()
	return tx
}

func (tx *Transaction)SetID(){
	var result bytes.Buffer

	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(tx)
	if err != nil{
		log.Panic(err)
	}

	resultBytes := bytes.Join([][]byte{IntToHex(time.Now().Unix()),result.Bytes()},[]byte{})

	hash := sha256.Sum256(resultBytes)
	tx.ID = hash[:]
}

func (tx *Transaction)IsCoinBase() bool{
	return len(tx.Vin) == 1 && tx.Vin[0].Vout == -1 && len(tx.Vin[0].Txid) == 0
}

func (in *TXInput) UnlockRipemd160Hash(Ripemd160Hash []byte) bool{

	PublicKey := RipemdHash160(in.PublicKey)
	return bytes.Compare(PublicKey,Ripemd160Hash) == 0
}

func (out *TXOutput) UnLockScriptPubKeyWithAddress(address string) bool{
	pubKeyHash := base58.Decode(address)
	hash160 := pubKeyHash[1:len(pubKeyHash)-addressChecksumLen]
	return bytes.Compare(out.Ripemd160Hash,hash160) == 0
}

func NewSimpleTransaction(from string,to string,amount int,utxoset *UTXOSet,txs []*Transaction) *Transaction{
	wallets,_ := NewWallets()
	wallet := wallets.WalletsMap[from]

	money,spendableUTXOdic := utxoset.FindSpendableUTXOS(from,amount,txs)

	var txInputs []*TXInput
	var txOutputs []*TXOutput

	for txHash,indexArray := range spendableUTXOdic{
		txHashBytes,_ := hex.DecodeString(txHash)
		for _,index := range indexArray{
			txin := &TXInput{txHashBytes,index,nil,wallet.PublicKey}
			txInputs = append(txInputs, txin)
		}
	}

	txout := NewTxOutput(amount,to)
	txOutputs = append(txOutputs,txout)

	txout = NewTxOutput(money - amount,from)
	txOutputs = append(txOutputs,txout)

	tx := &Transaction{nil,txInputs,txOutputs}
	tx.SetID()
	//sign
	utxoset.blockchain.SignTransaction(tx, wallet.PrivateKey, txs)

	return tx
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction){
	if tx.IsCoinBase(){
		return
	}

	for _,vin := range tx.Vin{
		if prevTxs[hex.EncodeToString(vin.Txid)].ID == nil{
			log.Panic("Error: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin{
		prevTx := prevTxs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PublicKey = prevTx.Vout[vin.Vout].Ripemd160Hash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PublicKey = nil

		r,s,err := ecdsa.Sign(rand.Reader,&privKey,txCopy.ID)
		if err != nil{
			log.Panic(err)
		}

		signature := append(r.Bytes(),s.Bytes()...)
		tx.Vin[inID].Signature = signature
	}
}


func (out *TXOutput) Lock(address string){
	pubKeyHash := base58.Decode(address)
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-addressChecksumLen]
	out.Ripemd160Hash = pubKeyHash
}

func NewTxOutput(value int,address string)*TXOutput{
	txoutput := &TXOutput{value,nil}
	txoutput.Lock(address)
	return txoutput
}

func (tx *Transaction) TrimmedCopy() Transaction{
	var inputs []*TXInput
	var outputs []*TXOutput

	for _,vin := range tx.Vin{
		inputs = append(inputs,&TXInput{vin.Txid,vin.Vout,nil,nil})
	}

	for _,vout := range tx.Vout{
		outputs = append(outputs,&TXOutput{vout.Value,vout.Ripemd160Hash})
	}

	txCopy := Transaction{tx.ID,inputs,outputs}

	return txCopy
}

func (tx Transaction) Serialize()[]byte{
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil{
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte{
	txCopy := tx
	txCopy.ID = []byte{}
	hash := sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Verify(txmap map[string]Transaction)bool{
	if tx.IsCoinBase(){
		return true
	}

	for _,vin := range tx.Vin{
		if txmap[hex.EncodeToString(vin.Txid)].ID == nil{
			log.Panic("Error: previous Transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID,vin := range tx.Vin{
		prevTx := txmap[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PublicKey = prevTx.Vout[vin.Vout].Ripemd160Hash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PublicKey = nil

		r:=big.Int{}
		s:=big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen/2)])
		s.SetBytes(vin.Signature[(sigLen/2):])

		x:=big.Int{}
		y:=big.Int{}
		keyLen := len(vin.PublicKey)
		x.SetBytes(vin.PublicKey[:(keyLen/2)])
		y.SetBytes(vin.PublicKey[(keyLen/2):])

		rawPubKey := ecdsa.PublicKey{curve,&x,&y}
		if ecdsa.Verify(&rawPubKey,txCopy.ID,&r,&s) == false{
			return false
		}
	}

	return true
}