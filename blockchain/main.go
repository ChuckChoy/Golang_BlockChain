package main
//export PATH=$PATH:/home/nik/Public/go/bin
//go run /home/nik/go/src/blockchain main.go

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"bufio"
	"log"
	"net/http"
	"net"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

//structs - typed collections of fields. Usefule for grouping
//data together. Like a half class. Go doesn't provide typical classes
type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}
//current chain    	 
var BlockChain []Block
//this will be for the chain that has yet to be validated
var NewChain []Block

//creating a function that converts a Block info into a hash string
//calculateHash looks to take Object Block typed block variable and returns a string?
func calculateHash(block Block) string {
	// creates a variable that holds the concatenated Block information
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	// instantiates a new has key
	h := sha256.New()
	//Write string to has and stores it? write it to itself?
	h.Write([]byte(record))
	//returns the hashed string to the variable hashed
	hashed := h.Sum(nil)
	//returns the has string as a hash
	return hex.EncodeToString(hashed)
}

//creates the first block in the chain
func createGenesisBlock(){
	t := time.Now()
	//creates all important genesis block. The Chain needs a first block.
	genesisBlock := Block{0,t.String(),0,"",""}
	spew.Dump(genesisBlock)
	//Add block to chain.
	BlockChain = append(BlockChain, genesisBlock)
}

//function to generate a new block that takes a Block object and
//in object and returns a Block object and an error?
func generateBlock(oldBlock Block, BPM int) (Block, error) {
	//create a Block object
	var newBlock Block
	//get timestamp
	t := time.Now()
	//get the number from the last chain block and add one
	newBlock.Index = oldBlock.Index + 1
	//fill timestamp field with timestamp converted to string
	newBlock.Timestamp = t.String()
	//fill bpm field with input int
	newBlock.BPM = BPM
	//fill prevHash field with input block hash
	newBlock.PrevHash = oldBlock.Hash
	//fill current hash field with calculate hash function
	newBlock.Hash = calculateHash(newBlock)
	//return newBlock Block with a nil(null) for the error?
	return newBlock, nil
}

//a function to validate blocks
//When two or more consecutive name parameters share a type you can
//omit the type from all but the last.
//function returns a bool
func isBlockValid(newBlock, oldBlock Block) bool {
	//if the old index plus one doesn't match the new index
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	//if old hashes don't match
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	//if new hashes don't match
	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}
	//if all checsk are passed
	return true
}

//compares chainlength and replaces new chain is longer than the old chain
func replaceChain(newBlocks []Block, bcServer chan []Block) {
	//length of input array vs length of Blockchain array
	if len(newBlocks) > len(BlockChain) {
		//replaces blockchain
		NewChain = newBlocks
		//add blockchain to channel, so we can communicate between go routines.
		bcServer <- NewChain
		log.Println("Chain to Channel")
	}
}

//start webserver
func run(bcServer chan []Block) error {
	// := is a type like var that can only be used in a function
	mux := makeMuxRouter(bcServer)
	//gets port from env file
	httpAddr := os.Getenv("ADDR")
	//log listening... though shouldnt' this be after listen and serve?
	log.Println("WebServer Listening on ", os.Getenv("ADDR"))
	// & is for getting the memory address. Its a pointer
	//these looks like to be server variables
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// semi-colon is for a list of statements
	// this looks to mean the first part  start server and the err variable 
	// will store any returns. The second part is the actual comparison  for 
	// the if statement
	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

//start tcp server
func startNode(){
	//creating a tcp server on port 9000 from the .env file
	server, err := net.Listen("tcp",":" + os.Getenv("ADDR2"))
	if err != nil{
		log.Fatal(err)
	}
	//if successfully launched write message to terminal
	log.Println("TCP server listening on ", os.Getenv("ADDR2"))
	defer server.Close()

	//continuous loop that accepts clients and then passes them to the handler
	//which is on it's own thread.
	for{
		conn, err := server.Accept()
		if err != nil{
			log.Fatal(err)
		}
		//spin handler off on it's own thread
		go handleConn(conn)
	}
}

//handles connections from tcp server.
func handleConn(conn net.Conn){
	//create new Block array to receive chain
	var compareChain []Block
	//convert byte stream into json object
	blockAsJson := json.NewDecoder(conn)
	//fill block array with json object
	blockAsJson.Decode(compareChain)
	//compare length of chains. 
	if (len(compareChain) > len(BlockChain)){
		//send yes and replace blockchain
		conn.Write([]byte("yes"))
		BlockChain = compareChain
	} else {
		//send no, reject blockchain
		conn.Write([]byte("no"))
	}

	//closes connection when done.
	defer conn.Close()
}

// Handles the routing and the actions it will take. REturns an http.Handler
func makeMuxRouter(bcServer chan []Block) http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/",handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		handleWriteBlock(w,r,bcServer)
	}).Methods("POST")
	return muxRouter
}

//Get Handler
// pass in response object and requset object
func handleGetBlockchain(w http.ResponseWriter, r *http.Request){
	//grabs BlockChain variable and converts it into a byte array and error?
	bytes, err := json.MarshalIndent(BlockChain, "", " ")
	//Checks to see if there are any errors in conversion process
	if err != nil{
		//send error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//write byte[] as a string to the resonpse object? also sends object?
	io.WriteString(w, string(bytes))
}

type Message struct{
	BPM int
}

//Write Handler for new blocks with parameters of repsonse and request
func handleWriteBlock(w http.ResponseWriter, r *http.Request, bcServer chan []Block){
	//create a message variable that has a pointer
	var m Message

	//create a variable that equals new decoder with the body of the reqeust as a parameter
	//for some reason NewDecoder is chosen over the Unmarshall function
	decoder := json.NewDecoder(r.Body)

	//Check for Errors in the with decoding the message. If bad return BadRequest message
	//I think the first statement decodes the request body into the location of the m
	//variable using a ponter
	if err := decoder.Decode(&m); err != nil{
		//http.StatusBadRequest equals an integer value. It is a system constant.
		log.Println("failed @ POST handler part1")
		respondWithJSON(w,r,http.StatusBadRequest, r.Body)
		return
	}

	//waits till the resource is done being used?
	//defer tells the program to wait and i think the close is to stop resouce
	//maybe the decoder inherently closes resources(channels) being used?
	defer r.Body.Close()

	//These two variables equal a new generated block using the current blockchain size
	// and the beats per minute from the message. If no errors return an Internal Server
	//Error Message
	newBlock, err := generateBlock(BlockChain[len(BlockChain)-1],m.BPM)
	if err != nil {
		respondWithJSON(w,r,http.StatusInternalServerError,m)
		log.Println("failed @ POST handler part2")
		return
	}
	
	//Validates new Block and existing chain
	if isBlockValid(newBlock, BlockChain[len(BlockChain)-1]){
		//if chain is valid add new block to old chain
		newBlockchain := append(BlockChain, newBlock)
		//replace odl chaint wit new chian
		replaceChain(newBlockchain,bcServer)
		//prints struct in console for debug purposes
		spew.Dump(BlockChain)
	}

	//send user new status and newBlock to be displayed?
	respondWithJSON(w,r,http.StatusCreated,newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}){
	//response and err equal to an encoded formatted payload
	response, err := json.MarshalIndent(payload,""," ")
	//if there is an error return response internal server error
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	//server code
	w.WriteHeader(code)
	//encoded message
	w.Write(response)
}

//I need this function to continually monitor the bcServer channel and
//to send the latest chain to the other nodes.
func broadcastCurrentChain(bcServer chan []Block){
	//wanted this to run on separate routine so as not to block the starting
	//of other functions. go routines can be made anonymously. The parameters
	//of the function go in the func(). then at the end there is a set of parenthesis
	//() where the arguments are put.
		go func() {
		validation := 0
		//hardcoded ports
		ports := []string{"8001","8002","8003","8004","8005"}
		//get chain from server
		currentBC := <- bcServer
		//create a for each loop for all the ports
		for _, v := range ports{
			
			//if the channel isn't empty.
			if(currentBC != nil){
				log.Println("broadcast")
				//create a connection.. thought I'll need an array with ipaddresses
				//had to hardcode for expedience and inexperience reasons.
				conn, _ := net.Dial("tcp", "127.0.0.1:" + v)
				//encode blockchain
				byteblock, _ := json.MarshalIndent(currentBC,""," ")
				//write chain to connection
				io.WriteString(conn,string(byteblock))
				//make a buffer site of 1024
				//buff:= make([]byte,1024)
				rep := 0				
				// decided to make a repeating loop until there was a respose from the node
				// because I couldn't find how else to wait for a response.
				for rep != 1{
					// nodeResponse is equal the the buffer that the connection has filled
					//nodeResponse, _ := conn.Read(buff)
					nodeResponse, _ := bufio.NewReader(conn).ReadString('\n')
					//log.Println("buffer: ")
					//convert byte[] to string
					//responseString := string(nodeResponse);
					if(nodeResponse == "yes"){
						validation++
						log.Println("valid")
						rep = 1
					}
				}
			
			
			}
		}
		//if more than 50% of nodes agree to block
		if(validation >= 3){
			//replace blockchain with node validated blockchain
			BlockChain = currentBC
			log.Println("Chain switched")
			//set node validation count back to zero
			validation = 0
		}
		//if loop finishes and blockchain isn't validated. Get a blockchain from
		//a differnt node

		//reset validation count;
		validation = 0
	}()
}

//loops through the values of passed in ports and launches separate server
//for each port
func createEchoNodes(nodePorts []string){
	//for each address create a listening server. Using the _ as a place holder
	//that doesn't bring up any "not used" errors.
	for _, v := range nodePorts{
		//run each node on a separate go routine
		go LaunchNodes(v);
	}
}

//receive a string 
func LaunchNodes(val string){
	//start server listening on specific port
	server, err := net.Listen("tcp",":" + val)
			if err != nil{
				log.Fatal(err)
			}
			//if successfully launched write message to terminal
			log.Println("Echo server listening on " + val)
			defer server.Close()

			//continuous loop that accepts clients and then passes them to the handler
			//which is on it's own thread.
			for{
				conn, err := server.Accept()
				if err != nil{
					log.Fatal(err)
				}
				
				//nodes will always reply with affirmative to connections
				//because I don't tHink I have time to give them all the correct logic
				conn.Write([]byte("yes"))
				log.Println("Verfication Echo")
				conn.Close()
			}
}

func main(){
	//load environment file
	err := godotenv.Load("/home/nik/go/src/blockchain/block.env")
	//if error then log
	if err != nil{
		// .Fatal exits the OS
		log.Fatal(err)
	}

	//chan called Channels are how different Go routines can communicate
	//Different go routines can write and read to the same channel.
	//this is a channel with type Array of Block.
	bcServer := make( chan []Block)
	//need a channel for receiving and one for sending?
	//bc := make(chan []Block)
	//starts goroutine(read thread) to create the starting block in the chain.
	go createGenesisBlock()

	//since this will be more in the background, I was thinking of having it
	//run on its own thread.
	go startNode()

	//start echonodes for demo purposes
	tcpAddresses := []string{"8001","8002","8003","8004","8005"}
	createEchoNodes(tcpAddresses)

	//start broadcasting chain. If you want to use the same channel for different
	//routines, you have to pass in a reference to that channel.
	broadcastCurrentChain(bcServer)

	//start the webserver
	log.Fatal(run(bcServer))
	
}