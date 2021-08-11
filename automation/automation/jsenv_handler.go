package automation

import (
	"context"
	"fmt"

	"github.com/cortezaproject/corteza-server/pkg/jsenv"
)

type (
	jsenvHandler struct {
		reg queueHandlerRegistry
	}
)

func JsenvHandler(reg queueHandlerRegistry) *jsenvHandler {
	h := &jsenvHandler{
		reg: reg,
	}

	h.register()
	return h
}

func (h jsenvHandler) execute(ctx context.Context, args *jsenvExecuteArgs) (res *jsenvExecuteResults, err error) {
	res = &jsenvExecuteResults{}

	if !args.hasSource {
		err = fmt.Errorf("could not process payload, function missing")
		return
	}

	if !args.hasScope {
		err = fmt.Errorf("could not process payload, scope missing")
		return
	}

	// call jsenv, feed it function and expect a result
	tr := jsenv.NewTransformer(jsenv.LoaderJS, jsenv.TargetNoop)
	vm := jsenv.New(tr)

	fn, err := vm.RegisterFunction(args.Source)

	if err != nil {
		err = fmt.Errorf("could not register jsenv function: %s", err)
		return
	}

	out, err := fn.Exec(vm.New(args.scopeString))

	if err != nil {
		err = fmt.Errorf("could not exec jsenv function: %s", err)
		return
	}

	switch vv := out.(type) {

	// this one should go out once the ResultAny
	// is mainly used
	case uint64, int64:
		res.ResultInt = int64(vv.(uint64))

	// this one should go out once the ResultAny
	// is mainly used
	case string:
		res.ResultString = string(vv)

	default:
		res.ResultAny = vv
	}

	return
}
