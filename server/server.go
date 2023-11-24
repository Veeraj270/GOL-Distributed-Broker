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
var turnHundred int

var paused bool

var reset bool

var clients []*rpc.Client

/*
func readAliveCounts(width, height int) map[int]int {
	f, err := os.Open("check/alive/" + fmt.Sprintf("%vx%v.csv", width, height))
	util.Check(err)
	reader := csv.NewReader(f)
	table, err := reader.ReadAll()
	util.Check(err)
	alive := make(map[int]int)
	for i, row := range table {
		if i == 0 {
			continue
		}
		completedTurns, err := strconv.Atoi(row[0])
		util.Check(err)
		aliveCount, err := strconv.Atoi(row[1])
		util.Check(err)
		alive[completedTurns] = aliveCount
	}
	return alive
}

func TestAlive(t *testing.T) {
	p := gol.Params{
		Turns:       100000000,
		Threads:     8,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	alive := readAliveCounts(p.ImageWidth, p.ImageHeight)
	events := make(chan gol.Event)
	keyPresses := make(chan rune, 2)
	go gol.Run(p, events, keyPresses)

	implemented := make(chan bool)
	go func() {
		timer := time.After(5 * time.Second)
		select {
		case <-timer:
			t.Fatal("no AliveCellsCount events received in 5 seconds")
		case <-implemented:
			return
		}
	}()

	i := 0
	timer := time.NewTicker(2 * time.Second)
	for {
		<-timer.C
		var expected int
		if turn <= 10000 {
			expected = alive[turn]
		} else if turn%2 == 0 {
			expected = 5565
		} else {
			expected = 5567
		}
		actual := numberOfAliveCells(worldCopy, len(worldCopy), len(worldCopy[0]))
		if expected != actual {
			t.Fatalf("At turn %v expected %v alive cells, got %v instead", turn, expected, actual)
		} else {
			fmt.Printf("--------------------------------------------TURN:%d, CellCount:%d-------------------------------\n", turn, actual)
			if i == 0 {
				implemented <- true
			}
			i++
		}

		if i >= 5 {
			keyPresses <- 'q'
			return
		}
	}
	t.Fatal("not enough AliveCellsCount events received")
}

*/

func createWorldCopy(world [][]uint8) [][]uint8 {
	worldCopy := make([][]uint8, len(world))
	for i := range worldCopy {
		worldCopy[i] = make([]uint8, len(world[i]))
		copy(worldCopy[i], world[i])
	}
	return worldCopy
}

//Removed c DistributorChannels and turns *int as they were only needs for SDL
func parallelCalculateNextState(worldCopy [][]uint8, startY, endY, height, width int) [][]uint8 {

	//fmt.Println("--------NextStateCalculating------------")
	worldSection := make([][]uint8, endY-startY)
	for i := 0; i < (endY - startY); i++ {
		worldSection[i] = make([]uint8, width)
	}

	for j := startY; j < endY; j++ {
		for i := 0; i < width; i++ {
			sum := 0
			var neighbours [8]uint8
			top := j - 1
			bottom := j + 1
			left := i - 1
			right := i + 1
			if top == -1 {
				top = height - 1
			}
			if bottom == height {
				bottom = 0
			}
			if left == -1 {
				left = width - 1
			}
			if right == width {
				right = 0
			}
			neighbours[0] = worldCopy[bottom][left]
			neighbours[1] = worldCopy[bottom][i]
			neighbours[2] = worldCopy[bottom][right]
			neighbours[3] = worldCopy[j][left]
			neighbours[4] = worldCopy[j][right]
			neighbours[5] = worldCopy[top][left]
			neighbours[6] = worldCopy[top][i]
			neighbours[7] = worldCopy[top][right]

			for _, n := range neighbours {
				if n == 255 {
					sum = sum + 1
				}
			}

			if worldCopy[j][i] == 255 {
				if sum < 2 {
					worldSection[j-startY][i] = 0
					//c.events <- CellFlipped{CompletedTurns: *turns, Cell: util.Cell{X: i, Y: j}}
				} else if (sum == 2) || (sum == 3) {
					worldSection[j-startY][i] = 255
				} else if sum > 3 {
					worldSection[j-startY][i] = 0
					//c.events <- CellFlipped{CompletedTurns: *turns, Cell: util.Cell{X: i, Y: j}}
				}
			} else if worldCopy[j][i] == 0 {
				if sum == 3 {
					worldSection[j-startY][i] = 255
					//c.events <- CellFlipped{CompletedTurns: *turns, Cell: util.Cell{X: i, Y: j}}
				} else {
					worldSection[j-startY][i] = 0
				}
			}
		}
	}
	return worldSection
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

	turnHundred = 0

	turn = 0
	worldCopy = createWorldCopy(world)
	height := len(world)
	width := len(world[0])

	chunkSize := height / threads
	remainingChunk := height % threads

	clients = make([]*rpc.Client, threads)
	errs := make([]error, threads)
	address := make([]string, threads)
	address[0] = "50.19.151.156:8040"
	address[1] = "54.208.4.59:8040"
	address[2] = "54.82.254.59:8040"
	address[3] = "54.242.224.35:8040"
	address[4] = "54.89.202.168:8040"
	address[5] = "54.90.104.137:8040"
	address[6] = "54.92.221.161:8040"
	address[7] = "54.162.152.176:8040"

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

		//world = parallelCalculateNextState(worldCopy, 0, height, height, width)
		var bufferedSliceChan = make([]chan [][]uint8, threads)
		//fmt.Println("STATES ABOUT TO BE CALCULATED")
		for k := 0; k < threads; k++ {
			fmt.Println("K=", k, " Threads=", threads)
			if k < threads-remainingChunk {
				Begin := k * chunkSize
				End := (k + 1) * chunkSize
				bufferedSliceChan[k] = make(chan [][]uint8, 1)
				request := stubs.WorkerRequest{
					WorldCopy: worldCopy,
					StartY:    Begin,
					EndY:      End,
					Height:    height,
					Width:     width,
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
					Height:    height,
					Width:     width,
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
					Height:    height,
					Width:     width,
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
	/*
		}  else if p.Threads > 1 {
			chunkSize := height / p.Threads
			remainingChunk := height % p.Threads

			for i := 0; i < turns; i++ {
				select {
				case v := <-c.keypress:
					switch v {
					case 'p':
						c.events <- StateChange{CompletedTurns: turn, NewState: Paused}
						for {
							switch <-c.keypress {
							case 'p':
								c.events <- StateChange{CompletedTurns: turn, NewState: Executing}
								break
							}
							break
						}
					case 's':
						go writeToPgmFile(c, world, height, width, &turn)
						c.ioCommand <- ioCheckIdle
						<-c.ioIdle
					case 'q':
						c.events <- StateChange{CompletedTurns: turn, NewState: Quitting}
						writeToPgmFile(c, world, height, width, &turn)
						c.ioCommand <- ioCheckIdle
						<-c.ioIdle
						os.Exit(0)
					}
				case <-timer.C:
					c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: numberOfAliveCells(world, height, width)}
				default:
					var parallelWorld [][]uint8
					if turns == 0 {
						//skip section
					} else {
						var bufferedSliceChan = make([]chan [][]uint8, p.Threads)

						for k := 0; k < p.Threads; k++ {
							if k < p.Threads-remainingChunk {
								Begin := k * chunkSize
								End := (k + 1) * chunkSize
								bufferedSliceChan[k] = make(chan [][]uint8)
								go worker(c, &turn, Begin, End, height, width, worldCopy, bufferedSliceChan[k])
							} else if k == p.Threads-remainingChunk {
								Begin := k * chunkSize
								End := (k+1)*chunkSize + 1
								bufferedSliceChan[k] = make(chan [][]uint8)
								go worker(c, &turn, Begin, End, height, width, worldCopy, bufferedSliceChan[k])
							} else if k > p.Threads-remainingChunk {
								Begin := (k * chunkSize) + (k - (p.Threads - remainingChunk))
								End := (k+1)*chunkSize + (k + 1 - (p.Threads - remainingChunk))
								bufferedSliceChan[k] = make(chan [][]uint8)
								go worker(c, &turn, Begin, End, height, width, worldCopy, bufferedSliceChan[k])
							}
						}

						for i := 0; i < p.Threads; i++ {
							parallelWorld = append(parallelWorld, <-bufferedSliceChan[i]...)
						}
						worldCopy = parallelWorld
						world = parallelWorld
						turn++
						c.events <- TurnComplete{CompletedTurns: turn}
					}
				}
			}
		}*/

	//Report the final state using FinalTurnCompleteEvent and write bits of world to a PGM file.
	//writeToPgmFile(c, world, height, width, &turn)

	return world
}

type RemoteProcessor struct{}

func (r *RemoteProcessor) CallNumberOfAliveCells(request stubs.CellCountRequest, response *stubs.CellCountResponse) (err error) {
	done := false

	for done != true {
		if sync == true {
			response.Turn = turn
			response.CellCount = numberOfAliveCells(worldCopy, len(worldCopy), len(worldCopy[0]))
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
