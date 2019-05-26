package main

import (
	"testing"
	"time"
)

func TestGenisisBlock(t *testing.T){
	BlockChain = []Block
	i =: time.Now()
	genesisBlock := Block{0,i.String(),0,"",""}
	BlockChain = append(BlockChain, genesisBlock)
	if (BlockChain[0].Index != 0{
		t.Errorf("First block isn't Genisis Block")
	}
}

func TestNodeServerStart(t *testing.T){
	err := godotenv.Load("/home/nik/go/src/blockchain/block.env")
	StartNode()
}

func TestWebServerStart(t *testing.T){
	err := godotenv.Load("/home/nik/go/src/blockchain/block.env")
	log.Fatal(run())
}

func TestGetRequestHandler(t *testing.T){

}

func TestGetResponse(t *testing.T){

}

func TestTCPConnectionHandler(t *testing.T){

}

func TestPostRequestHandler(t *testing.T){

}

func TestCreatNewBlock(t *testing.T){

}

func TestBlockValidation(t *testing.T){

}

func TestBlockChainBroadcast(t *testing.T){

}

func SimulateTCPConnnection(int BPM, string port){
	 
	fmt.Fprintf(conn, 83)
}

func handleConn(conn net.Conn){
	log.Println("Successful connection")
	defer conn.Close()
}