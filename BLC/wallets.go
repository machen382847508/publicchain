package BLC

import (
	"bytes"
	"encoding/gob"
	"log"
	"io/ioutil"
	"crypto/elliptic"
	"os"
)

const walletFile  =  "Wallets.dat"

type Wallets struct {
	WalletsMap map[string]*Wallet
}

func NewWallets() (*Wallets,error){
	if _,err := os.Stat(walletFile); os.IsNotExist(err){
		wallets := &Wallets{}
		wallets.WalletsMap = make(map[string]*Wallet)
		return wallets, err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil{
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil{
		log.Panic(err)
	}

	return &wallets,nil
}

func (w *Wallets) CreateNewWallets(){
	wallet := NewWallet()
	w.WalletsMap[wallet.GetAddress()] = wallet
	w.SaveWallets()
}

func (w *Wallets) SaveWallets(){
	var content bytes.Buffer
	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(&w)
	if err != nil{
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile,content.Bytes(),0644)
	if err != nil{
		log.Panic(err)
	}
}

func (w *Wallets) LoadFromFile() (*Wallets,error){
	if _,err := os.Stat(walletFile); os.IsNotExist(err){
		wallets := &Wallets{}
		wallets.WalletsMap = make(map[string]*Wallet)
		return wallets, err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil{
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil{
		log.Panic(err)
	}

	return &wallets,nil
}