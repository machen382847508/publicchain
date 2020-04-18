package BLC

import (
	"flag"
	"os"
	"fmt"
	"log"
)

type CLI struct {
	BC *Blockchain
}

func (cli *CLI) validateArgs(){
	if len(os.Args) < 2{
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI)printUsage(){
	fmt.Println("Usage:")
	fmt.Println("\taddresslists -- Print All Wallets Address")
	fmt.Println("\tcreatewallet -- Create a Wallet")
	fmt.Println("\tgetbalance -address ADDRESS -- Get balance of ADDRESS")
	fmt.Println("\tcreateblockchain -address ADDRESS -- Create a blockchain")
	fmt.Println("\tprintchain -- Print all the blocks of the blockchain")
	fmt.Println("\tsend -from FROM -to TO -amount AMOUNT -- Send AMOUNT from FROM TO to")
}

func (cli *CLI)Run(){

	cli.validateArgs()
	addressListsCmd := flag.NewFlagSet("addresslists",flag.ExitOnError)

	createWalletCmd := flag.NewFlagSet("createwallet",flag.ExitOnError)

	createBlockchainCmd := flag.NewFlagSet("createblockchain",flag.ExitOnError)
	genesisAddress := createBlockchainCmd.String("address","","Create Genesis Block,Pack data to the  db")

	printChainCmd := flag.NewFlagSet("printchain",flag.ExitOnError)

	getBalanceCmd := flag.NewFlagSet("getbalance",flag.ExitOnError)
	balanceAddress := getBalanceCmd.String("address","","balance address...")

	sendCmd := flag.NewFlagSet("send",flag.ExitOnError)
	sendFrom := sendCmd.String("from","","source address...")
	sendTo := sendCmd.String("to","","destinate address...")
	sendAmount := sendCmd.String("amount","","transform money...")

	if len(os.Args) > 1{
		switch os.Args[1] {
		case "addresslists":
			err := addressListsCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}

		case "createwallet":
			err := createWalletCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}
		case "createblockchain":
			err := createBlockchainCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}
		case "printchain":
			err := printChainCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}
		case "send":
			err := sendCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}
		case "getbalance":
			err := getBalanceCmd.Parse(os.Args[2:])
			if err != nil{
				log.Panic(err)
			}
		default:
			cli.printUsage()
			os.Exit(1)
		}
	}

	if createWalletCmd.Parsed(){
		cli.createWallet()
	}

	if createBlockchainCmd.Parsed(){
		if IsValidForAddress(*genesisAddress) == false{
			fmt.Println("genesisAddress is invalid......")
			cli.printUsage()
			os.Exit(1)
		}
		cli.createGenesisBlockchain(*genesisAddress)
	}

	if printChainCmd.Parsed(){
		if dbExists() == false{
			fmt.Println("db is not exists....")
			os.Exit(1)
		}
		blockchain := BlockchainObject()
		defer blockchain.DB.Close()
		blockchain.printChain()
		fmt.Println()
	}

	if getBalanceCmd.Parsed(){

		if IsValidForAddress(*balanceAddress) == false{
			fmt.Println("balanceAddress is invalid......")
			cli.printUsage()
			os.Exit(1)
		}

		cli.getBalance(*balanceAddress)
	}

	if addressListsCmd.Parsed(){
		cli.addressLists()
	}

	if sendCmd.Parsed(){
		from := JsonToArray(*sendFrom)
		to := JsonToArray(*sendTo)

		for index,fromAddress := range from{
			if IsValidForAddress(fromAddress) == false || IsValidForAddress(to[index]) == false || *sendAmount == ""{
				fmt.Println("from ,to ,amount is invalid......")
				cli.printUsage()
				os.Exit(1)
			}
		}
		amount := JsonToArray(*sendAmount)

		cli.send(from,to,amount)
	}
}

func (cli *CLI) getBalance(address string){
	blockchain := BlockchainObject()
	defer blockchain.DB.Close()
	utxoSet := &UTXOSet{blockchain}
	amount := utxoSet.GetBalance(address)
	fmt.Printf("%s's has %d token\n ",address,amount)
}

func (cli *CLI) createWallet(){
	wallets,_ := NewWallets()
	wallets.CreateNewWallets()

	fmt.Println(wallets.WalletsMap)
}

func (cli *CLI) send(from,to,amount []string){
	if dbExists() == false{
		fmt.Println("db is not exists.....")
		os.Exit(1)
	}

	blockchain := BlockchainObject()
	defer blockchain.DB.Close()
	blockchain.MineNewBlock(from,to,amount)
	utxoSet := &UTXOSet{blockchain}
	utxoSet.Update()
	fmt.Println()
}

func (cli *CLI) addressLists(){

	fmt.Println("Print All Address:")
	wallets,_ := NewWallets()

	for address := range wallets.WalletsMap{
		fmt.Println("\t"+address)
	}
}

func (cli *CLI) createGenesisBlockchain(address string){
	blockchain := cli.BC.NewBlockchainWithGenesisBlock(address)
	defer blockchain.DB.Close()
	utxoSet := &UTXOSet{blockchain}
	utxoSet.ResetUTXOSet()
	fmt.Println()
}