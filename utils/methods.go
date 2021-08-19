package utils

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin/exported"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"reflect"
	"runtime"
	"strings"
)

type MethodMeta struct {
	Name 	string
	Params 	cbg.CBORUnmarshaler
}

var MethodsMap = map[cid.Cid]map[abi.MethodNum]MethodMeta{}

func init() {
	actors := exported.BuiltinActors()

	for _, actor := range actors {
		exports := actor.Exports()
		methods := make(map[abi.MethodNum]MethodMeta, len(exports))

		methods[builtin.MethodSend] = MethodMeta{
			Name:   "Send",
			Params: abi.Empty,
		}

		for number, export := range exports {
			if export == nil {
				continue
			}

			ev := reflect.ValueOf(export)
			fnName := runtime.FuncForPC(ev.Pointer()).Name()
			methods[abi.MethodNum(number)] = MethodMeta{
				Name:   strings.TrimSuffix(fnName[strings.LastIndexByte(fnName, '.')+1:], "-fm"),
				Params: reflect.New(ev.Type().In(1).Elem()).Interface().(cbg.CBORUnmarshaler),
			}
		}
		MethodsMap[actor.Code()] = methods
	}
}
