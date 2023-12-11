package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var worldCopy [][]uint8
var turn int

var sync bool

var paused bool

var reset bool

var clients []*rpc.Client

func createWorldCopy(world [][]uint8) [][]uint8 {
	worldCopy := make([][]uint8, len(world))
	for i := range worldCopy {
		worldCopy[i] = make([]uint8, len(world[i]))
		copy(worldCopy[i], world[i])
	}
	return worldCopy
}

func calculateAliveCells(world [][]uint8, height int, width int) []util.Cell {
	var newCell []util.Cell
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			if world[j][i] == 255 {
				addedCell := util.Cell{
					X: i,
					Y: j,
				}
				newCell = append(newCell, addedCell)
			}
		}
	}

	return newCell
}

func numberOfAliveCells(world [][]uint8, height, width int) int {
	aliveCells := calculateAliveCells(world, height, width)
	sum := 0
	for range aliveCells {
		sum++
	}
	return sum
}

//Removed c DistributerChannels, p gol.Params as they were only needs for SDL
//Removed threads arg as it was only needed for parallel
func remoteDistributor(world [][]uint8, turns int, threads int) [][]uint8 {

	//fmt.Println("-------------------------------------Remote Distributor Called------------------------------")
	turn = 0
	worldCopy = createWorldCopy(world)
	height := len(world)

	chunkSize := height / threads
	remainingChunk := height % threads

	clients = make([]*rpc.Client, threads)
	errs := make([]error, threads)

	

		address := make([]string, 8)
		address[0] = "3.82.27.85:8040"
		address[1] = "54.204.155.28:8040"
		address[2] = "3.90.143.115:8040"
		address[3] = "34.207.204.117:8040"
		address[4] = "54.80.187.144:8040"
		address[5] = "54.196.68.175:8040"
		address[6] = "54.91.255.100:8040"
		address[7] = "54.80.108.27:8040"

	

	for i := 0; i < threads; i++ {

		//port := 8040 + (i * 10)
		//address := "localhost:" + fmt.Sprint(port)
		fmt.Println(address[i])
		clients[i], errs[i] = rpc.Dial("tcp", address[i])
		if errs[i] != nil {
			fmt.Println("-----------Unable to connect--------------------")
		}
	}

	//fmt.Println("NUMBER OF TURNS:", turns)

	fmt.Println("--------------------------------Turn:", turn, "------------------------------------------------")

	//Timer sends time down channel to notify SDL of the number of alive cells and turns completed every 2 seconds
	//timer := time.NewTicker(2 * time.Second)

	//Execute all turns of the Game of Life and Populate Alive cells.
	//if threads == 1 {
	//fmt.Println("---------------------------ONE THREAD----------------------------------------")
	for i := 0; i < turns; i++ {
		//fmt.Println("FOR LOOP ENTERED")
		//fmt.Println("--------------------------------Turn:", turn, "------------------------------------------------")

		if reset == true {
			reset = false
			return world
		}
		for paused {

		}

		var bufferedSliceChan = make([]chan [][]uint8, threads)
		//fmt.Println("STATES ABOUT TO BE CALCULATED")
		for k := 0; k < threads; k++ {
			//fmt.Println("K=", k, " Threads=", threads)
			if k < threads-remainingChunk {
				Begin := k * chunkSize
				End := (k + 1) * chunkSize
				//fmt.Println("Begin: ", Begin, " End: ", End)
				//fmt.Println("len(worldCopy[0]):", len(worldCopy[0]))
				//fmt.Println("len(worldCopy):", len(worldCopy))
				bufferedSliceChan[k] = make(chan [][]uint8, 1)
				request := stubs.WorkerRequest{
					WorldCopy: worldCopy,
					StartY:    Begin,
					EndY:      End,
				}
				response := new(stubs.WorkerResponse)
				go func(k int, request stubs.WorkerRequest, response *stubs.WorkerResponse, channel chan [][]uint8) {
					err := clients[k].Call(stubs.WorkerCalculate, request, response)
					if err != nil {
						fmt.Println("Error calling Worker Calculate")
					}
					bufferedSliceChan[k] <- response.World
				}(k, request, response, bufferedSliceChan[k])

			} else if k == threads-remainingChunk {
				Begin := k * chunkSize
				End := (k+1)*chunkSize + 1
				bufferedSliceChan[k] = make(chan [][]uint8, 1)
				request := stubs.WorkerRequest{
					WorldCopy: worldCopy,
					StartY:    Begin,
					EndY:      End,
				}
				response := new(stubs.WorkerResponse)
				go func(k int, request stubs.WorkerRequest, response *stubs.WorkerResponse, channel chan [][]uint8) {
					err := clients[k].Call(stubs.WorkerCalculate, request, response)
					if err != nil {
						fmt.Println("Error calling Worker Calculate")
					}
					bufferedSliceChan[k] <- response.World
				}(k, request, response, bufferedSliceChan[k])

			} else if k > threads-remainingChunk {
				Begin := (k * chunkSize) + (k - (threads - remainingChunk))
				End := (k+1)*chunkSize + (k + 1 - (threads - remainingChunk))
				bufferedSliceChan[k] = make(chan [][]uint8, 1)
				request := stubs.WorkerRequest{
					WorldCopy: worldCopy,
					StartY:    Begin,
					EndY:      End,
				}
				response := new(stubs.WorkerResponse)
				go func(k int, request stubs.WorkerRequest, response *stubs.WorkerResponse, channel chan [][]uint8) {
					err := clients[k].Call(stubs.WorkerCalculate, request, response)
					if err != nil {
						fmt.Println("Error calling Worker Calculate")
					}
					bufferedSliceChan[k] <- response.World
				}(k, request, response, bufferedSliceChan[k])

			}
		}

		//fmt.Println("STATES CALCULATED")
		var parallelWorld [][]uint8

		for i := 0; i < threads; i++ {
			parallelWorld = append(parallelWorld, <-bufferedSliceChan[i]...)
		}
		//fmt.Println("DONE")

		//fmt.Println("TURN ADVANCED")

		//sync prevents the cell count being read while the turn and cell count are out of sync
		sync = false
		world = parallelWorld
		worldCopy = createWorldCopy(world)
		turn++
		/*
			if turn%100 == 0 {
				turnHundred++
				//fmt.Println("--------------------HUNDRED TURNS-------------------------------")
			}
		*/
		sync = true

	}

	return world
}

type RemoteProcessor struct{}

func (r *RemoteProcessor) CallNumberOfAliveCells(request stubs.CellCountRequest, response *stubs.CellCountResponse) (err error) {
	done := false

	for done != true {
		if sync == true {
			response.Turn = turn
			if turn == 0 {
				response.CellCount = 0
			} else {
				response.CellCount = numberOfAliveCells(worldCopy, len(worldCopy), len(worldCopy[0]))
			}
			done = true
		}
	}
	fmt.Printf("Reported CellCount: %d, Reported turn: %d\n", response.CellCount, response.Turn)
	return
}

func (r *RemoteProcessor) CallPause(request stubs.PauseReq, response *stubs.PauseResp) (err error) {
	paused = request.Paused
	response.Turn = turn
	return
}

func (r *RemoteProcessor) CallSave(request stubs.SaveReq, response *stubs.SaveResp) (err error) {
	response.World = worldCopy
	response.Turn = turn
	return
}

func (r *RemoteProcessor) CallClose(request stubs.CloseReq, response *stubs.CloseResp) (err error) {
	reset = true
	time.Sleep(1 * time.Second)
	for i, v := range clients {
		err := v.Call(stubs.WorkerClose, stubs.CloseReq{}, new(stubs.CloseResp))
		if err != nil {
			fmt.Println(err)
		}
		err = v.Close()
		if err != nil {
			fmt.Println("Couldn't close rpc connection to worker number", i)
		}
	}
	os.Exit(0)
	return
}

func (r *RemoteProcessor) CallRemoteDistributor(request stubs.Request, response *stubs.Response) (err error) {
	fmt.Println("-------------------------------------RPC For Remote Distributor Called------------------------------")
	reset = true

	time.Sleep(1 * time.Second)
	reset = false
	world := request.World //testing purposes only so i dont have to edit the test loop below
	response.World = remoteDistributor(request.World, request.Turns, request.Threads)
	//fmt.Println("DISTRIBUTOR COMPLETE")

	//fmt.Println(turn)
	test := 0
	for i, _ := range world {
		for i2, _ := range world[i] {
			if world[i][i2] == response.World[i][i2] {
				test++
			}
		}
	}
	if test == len(world)*len(world[0]) {
		//fmt.Println("-----------------------FUCK-------------------------")
	}
	return
}

/*
func connectToWorkers(threads int, clients []*rpc.Client) {
	clients = make([]*rpc.Client, threads)
	errs := make([]error, threads)
	for i := 0; i < threads; i++ {

		port := 8040 + (i * 10)
		address := "localhost:" + fmt.Sprint(port)
		fmt.Println(address)
		clients[i], errs[i] = rpc.Dial("tcp", address)
		if errs[i] != nil {
			fmt.Println("-----------Unable to connect--------------------")
		}

		defer func(client *rpc.Client) {
			err := client.Close()
			if err != nil {
				fmt.Println("-----------Unable to close connection--------------------")
			}
		}(clients[i])

	}
}
*/
func main() {
	pAddr := flag.String("port", ":8030", "Port to listen on")
	flag.Parse()

	reset = false
	paused = false

	listener, _ := net.Listen("tcp", *pAddr)
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("Error closing the listener")
		}
	}(listener)
	err := rpc.Register(&RemoteProcessor{})
	if err != nil {
		fmt.Println("Error registering rpc")
	}

	rpc.Accept(listener)

}
