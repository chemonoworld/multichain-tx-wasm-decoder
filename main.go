package main

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/gogoproto/jsonpb"
	osmokeepers "github.com/osmosis-labs/osmosis/v21/app/keepers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"strings"
	"syscall/js"
)

func FuncName(M int, durations []int) int {
	fmt.Println("[GO] Log file.go", M, durations)
	return 10
}

func Add(this js.Value, args []js.Value) any {
	if len(args) != 2 {
		fmt.Println("invalid number of args")
		return math.MinInt
	}
	if args[0].Type() != js.TypeNumber || args[1].Type() != js.TypeNumber {
		fmt.Println("both arguments should be numbers")
		return math.MinInt
	}

	return args[0].Int() + args[1].Int()
}

func GetTxMessagesByTxHash(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		fmt.Println("invalid number of args")
		return nil
	}
	if args[0].Type() != js.TypeString {
		fmt.Println("the first argument should be a number")
		return nil
	}

	hash := args[0].String()

	// TODO: fetch tx
	grpcEndpoint := "osmosis-querier.keplr.app:19999"
	msgSize := 1024 * 1024 * 60
	conn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(msgSize)))
	if err != nil {
		fmt.Println("grpc.Dial error", err)
		return nil
	}
	//tmClient := tmservice.NewServiceClient(conn) // for block
	encodingConfig := moduletestutil.MakeTestEncodingConfig(osmokeepers.AppModuleBasics...)
	txDecoder := authtx.NewTxConfig(codec.NewProtoCodec(encodingConfig.InterfaceRegistry), authtx.DefaultSignModes).TxDecoder()
	txClient := tx.NewServiceClient(conn)
	marshaler := jsonpb.Marshaler{}

	tx, err := txClient.GetTx(context.Background(), &tx.GetTxRequest{
		Hash: hash,
	})
	if err != nil {
		fmt.Println("txClient.GetTx error", err)
		return nil
	}

	decoded, err := txDecoder(tx.GetTxResponse().Tx.Value)
	if err != nil {
		fmt.Println("txDecoder error", err)
		return nil
	}

	var bulkData []string
	for msgIdx, msg := range decoded.GetMsgs() {
		msgStr, err := marshaler.MarshalToString(msg)
		if err != nil {
			fmt.Println("marshaler.MarshalToString error", err)
			return nil
		}
		bulkData = append(bulkData, fmt.Sprintf("%d: %s", msgIdx, msgStr))
	}

	return strings.Join(bulkData, "\n\n")
}

func main() {
	fmt.Println("Hello, WebAssembly!")

	// Mount the function on the JavaScript global object.
	js.Global().Set("FuncName", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 {
			fmt.Println("invalid number of args")
			return nil
		}
		if args[0].Type() != js.TypeNumber {
			fmt.Println("the first argument should be a number")
			return nil
		}

		arg := args[1]
		if arg.Type() != js.TypeObject {
			fmt.Println("the second argument should be an array")
			return nil
		}

		durations := make([]int, arg.Length())
		for i := 0; i < len(durations); i++ {
			item := arg.Index(i)
			if item.Type() != js.TypeNumber {
				fmt.Printf("the item at index %d should be a number\n", i)
				return nil
			}
			durations[i] = item.Int()
		}

		// Call the actual func.
		return FuncName(args[0].Int(), durations)
	}))

	js.Global().Set("Add", js.FuncOf(Add))

	txRes := GetTxMessagesByTxHash(js.Value{}, []js.Value{js.ValueOf("D886838B17555FCDC9CB2A804AD87D507BB893B1AC2F7442D143655399B5610F")})
	fmt.Println(txRes)

	// Prevent the program from exiting.
	// Note: the exported func should be released if you don't need it any more,
	// and let the program exit after then. To simplify this demo, this is
	// omitted. See https://pkg.go.dev/syscall/js#Func.Release for more information.
	select {}
}
