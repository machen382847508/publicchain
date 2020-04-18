package BLC

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"publicchain/base58"
	"bytes"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Wallet struct{
	PrivateKey ecdsa.PrivateKey
	PublicKey []byte
}

func NewWallet() *Wallet{
	private, public := newKeyPair()
	wallet := Wallet{private,public}
	return &wallet
}

func newKeyPair()(ecdsa.PrivateKey,[]byte){
	curve := elliptic.P256()
	private,err := ecdsa.GenerateKey(curve,rand.Reader)
	if err != nil{
		log.Panic(err)
	}

	pubKey := append(private.PublicKey.X.Bytes(),private.PublicKey.Y.Bytes()...)

	return *private,pubKey
}

func (w *Wallet) GetAddress() string{
	ripemdHash160 := RipemdHash160(w.PublicKey)
	version_ripemdHash160 := append([]byte{version},ripemdHash160...)
	checkSumBytes := CheckSum(version_ripemdHash160)
	bytes := append(version_ripemdHash160,checkSumBytes...)
	return base58.Encode(bytes)
}

func RipemdHash160(publickey []byte) []byte{
	hash256 := sha256.New()
	hash256.Write(publickey)
	hash := hash256.Sum(nil)

	ripemd160 := ripemd160.New()
	ripemd160.Write(hash)
	return ripemd160.Sum(nil)
}

func CheckSum(payload []byte) []byte{
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:addressChecksumLen]
}

func IsValidForAddress(address string) bool{
	version_public_checksumBytes := base58.Decode(address)
	if len(version_public_checksumBytes)-addressChecksumLen < 0{
		return false
	}
	checkSumbytes := version_public_checksumBytes[len(version_public_checksumBytes)-addressChecksumLen:]

	version_ripemd160 := version_public_checksumBytes[:len(version_public_checksumBytes)-addressChecksumLen]
	checkBytes := CheckSum(version_ripemd160)

	if bytes.Compare(checkSumbytes,checkBytes) == 0{
		return true
	}

	return false
}