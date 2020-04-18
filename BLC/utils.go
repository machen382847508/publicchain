package BLC

import (
	"bytes"
	"encoding/binary"
	"log"
	"encoding/json"
)

func IntToHex(num int64) []byte{
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil{
		log.Panic(err)
	}
	return buff.Bytes()
}

func JsonToArray(jsonString string) []string{
	var sArray []string
	if err := json.Unmarshal([]byte(jsonString), &sArray); err != nil{
		log.Panic(err)
	}

	return sArray
}