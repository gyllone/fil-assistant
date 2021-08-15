package common

import (
	"context"
	"encoding/base64"
	"fil-assistant/utils"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/subchen/go-trylock"
)

const (
	Info = iota
	Warn
	Error
)

type UI struct {
	Window 				fyne.Window
	Process 			binding.Float
	Handler 			*Handler
	Locker 				trylock.TryLocker
}

func (u *UI) Msg(level int, text string) {
	switch level {
	case 0:
		dialog.NewInformation("通知", text, u.Window).Show()
	case 1:
		dialog.NewInformation("警告", text, u.Window).Show()
	case 2:
		diag := dialog.NewInformation("失败", text, u.Window)
		diag.SetOnClosed(u.Window.Close)
		diag.Show()
	}
}

func (u *UI) Init(w fyne.Window) {
	u.Window = w
	u.Locker = trylock.New()
	u.Process = binding.NewFloat()

	cfg, err := utils.ReadConfig("./config.toml")
	if err != nil {
		u.Msg(Error, fmt.Sprintf("read config.toml error: %s", err))
		return
	}

	var aesKey []byte
	if len(cfg.AESKey) != 0 {
		aesKey, err = base64.StdEncoding.DecodeString(cfg.AESKey)
		if err != nil {
			u.Msg(Error, fmt.Sprintf("AES key %s decode error: %s", cfg.AESKey, err))
			return
		} else if len(aesKey) != 32 {
			u.Msg(Error, fmt.Sprintf("AES key length %d is invalid", len(aesKey)))
			return
		}
	}

	maxFee, err := types.ParseFIL(cfg.MaxFee)
	if err != nil {
		u.Msg(Error, err.Error())
		return
	}
	gasFeeCap, err := types.BigFromString(cfg.GasFeeCap)
	if err != nil {
		u.Msg(Error, err.Error())
		return
	}

	u.Handler, err = newHandler(context.TODO(), cfg.EndPoint, cfg.ApiToken, abi.TokenAmount(maxFee), gasFeeCap,
		cfg.Confidence, aesKey, u.Process.Set)
	if err != nil {
		u.Msg(Error, fmt.Sprintf("initialization failed: %s", err))
	}
}

func (u *UI) Close() {
	if u.Handler != nil {
		u.Handler.Close()
	}
}
