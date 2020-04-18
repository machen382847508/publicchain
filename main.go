package main

import "publicchain/BLC"

const blocksBucket = "blocks"
func main(){


	cli := BLC.CLI{}
	cli.Run()
}