package main

import (
	"context"
	"fil-assistant/common"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

var globalVar = new(common.UI)

func main() {
	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())
	src, err := fyne.LoadResourceFromPath("./resource/fil.png")
	if err == nil {
		a.SetIcon(src)
	}
	w := a.NewWindow("FileCoin矿工助手 V2.0.0")
	w.SetOnClosed(globalVar.Close)

	globalVar.Init(w)

	tabs := make([]*container.TabItem, 7)
	tabs[0] = container.NewTabItem("私钥加/解密", encryption())
	tabs[1] = container.NewTabItem("签名", sign())
	tabs[2] = container.NewTabItem("转账", send())
	tabs[3] = container.NewTabItem("矿工提现", withdraw())
	tabs[4] = container.NewTabItem("发起更换owner", proposeChangeOwner())
	tabs[5] = container.NewTabItem("确认更换owner", confirmChangeOwner())
	tabs[6] = container.NewTabItem("更换worker", changeWorker())

	w.SetContent(container.NewVBox(Process(), container.NewAppTabs(tabs...)))
	w.Resize(fyne.NewSize(800, 200))
	w.CenterOnScreen()
	w.ShowAndRun()
}

func Process() fyne.CanvasObject {
	ProcessBar := widget.NewProgressBar()
	ProcessBar.Bind(globalVar.Process)
	ProcessBar.Resize(ProcessBar.MinSize())
	return ProcessBar
}

func sign() fyne.CanvasObject {
	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	MsgEntry := widget.NewEntry()
	MsgEntry.PlaceHolder = "签名内容"

	result := widget.NewEntry()
	result.PlaceHolder = "签名结果"
	res := binding.NewString()
	result.Bind(res)

	signer := widget.NewButton("签名", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		sig, err := globalVar.Handler.Sign(strings.TrimSpace(pkEntry.Text), strings.TrimSpace(MsgEntry.Text))
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			res.Set(sig)
		}
	})

	return container.NewVBox(pkEntry, MsgEntry, signer, result)
}

func encryption() fyne.CanvasObject {
	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	address := widget.NewEntry()
	address.PlaceHolder = "加/解密对应地址"
	ad := binding.NewString()
	address.Bind(ad)

	private := widget.NewEntry()
	private.PlaceHolder = "加/解密对应私钥"
	pri := binding.NewString()
	private.Bind(pri)

	encrypt := widget.NewButton("加密", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		addr, newPk, err := globalVar.Handler.Encrypt(strings.TrimSpace(pkEntry.Text))
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			ad.Set(addr)
			pri.Set(newPk)
		}
	})

	decrypt := widget.NewButton("解密", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		addr, rawPk, err := globalVar.Handler.Decrypt(strings.TrimSpace(pkEntry.Text))
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			ad.Set(addr)
			pri.Set(rawPk)
		}
	})

	mid := container.NewGridWithColumns(2, encrypt, decrypt)
	return container.NewVBox(pkEntry, mid, address, private)
}

func send() fyne.CanvasObject {
	toEntry := widget.NewEntry()
	toEntry.PlaceHolder = "收款地址"

	amountEntry := widget.NewEntry()
	amountEntry.PlaceHolder = "金额"

	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	confirm := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if toEntry.Text == "" || amountEntry.Text == "" || pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.Send(context.TODO(), strings.TrimSpace(pkEntry.Text), strings.TrimSpace(toEntry.Text),
			strings.TrimSpace(amountEntry.Text), nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "转账成功")
			globalVar.Process.Set(1)
		}
	})

	bottom := container.NewGridWithColumns(2, amountEntry, confirm)
	return container.NewVBox(pkEntry, toEntry, bottom)
}

func withdraw() fyne.CanvasObject {
	// 初始化输入框
	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	minerEntry := widget.NewEntry()
	minerEntry.PlaceHolder = "矿工号"

	amountEntry := widget.NewEntry()
	amountEntry.PlaceHolder = "金额"

	confirm := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if minerEntry.Text == "" || amountEntry.Text == "" || pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.Withdraw(context.TODO(), strings.TrimSpace(pkEntry.Text),
			strings.TrimSpace(minerEntry.Text), strings.TrimSpace(amountEntry.Text), nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "提现成功")
			globalVar.Process.Set(1)
		}
	})

	top := container.NewGridWithColumns(2, minerEntry, amountEntry)
	return container.NewVBox(pkEntry, top, confirm)
}

func proposeChangeOwner() fyne.CanvasObject {
	minerEntry := widget.NewEntry()
	minerEntry.PlaceHolder = "矿工号"

	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	newOwner := widget.NewEntry()
	newOwner.PlaceHolder = "新owner地址"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if minerEntry.Text == "" || pkEntry.Text == "" || newOwner.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ChangeOwner1(context.TODO(), strings.TrimSpace(pkEntry.Text),
			strings.TrimSpace(newOwner.Text), strings.TrimSpace(minerEntry.Text), nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "发起更换owner完成")
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pkEntry, newOwner, container.NewGridWithColumns(2, minerEntry, submit))
}

func confirmChangeOwner() fyne.CanvasObject {
	minerEntry := widget.NewEntry()
	minerEntry.PlaceHolder = "矿工号"

	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if minerEntry.Text == "" || pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ChangeOwner2(context.TODO(), strings.TrimSpace(pkEntry.Text),
			strings.TrimSpace(minerEntry.Text), nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "确认更换owner完成")
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pkEntry, container.NewGridWithColumns(2, minerEntry, submit))
}

func changeWorker() fyne.CanvasObject {
	minerEntry := widget.NewEntry()
	minerEntry.PlaceHolder = "矿工号"

	pkEntry := widget.NewPasswordEntry()
	pkEntry.PlaceHolder = "私钥"

	workerEntry := widget.NewEntry()
	workerEntry.PlaceHolder = "worker地址"

	controlsEntry := widget.NewMultiLineEntry()
	controlsEntry.PlaceHolder = "controls地址"

	propose := widget.NewButton("提交发起", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if minerEntry.Text == "" || workerEntry.Text == "" || pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		controls := strings.Split(controlsEntry.Text, "\n")
		newControls := make([]string, 0, len(controls))
		for _, control := range controls {
			if newControl := strings.TrimSpace(control); newControl != "" {
				newControls = append(newControls, newControl)
			}
		}
		if len(newControls) == 0 {
			globalVar.Msg(common.Warn, "controls地址输入有误")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ProposeChangeWorker(context.TODO(), strings.TrimSpace(pkEntry.Text),
			strings.TrimSpace(minerEntry.Text), strings.TrimSpace(workerEntry.Text), newControls, nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "完成更换worker第一步")
			globalVar.Process.Set(1)
		}
	})

	confirm := widget.NewButton("提交确认", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if minerEntry.Text == "" || workerEntry.Text == "" || pkEntry.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ConfirmChangeWorker(context.TODO(), strings.TrimSpace(pkEntry.Text),
			strings.TrimSpace(minerEntry.Text), nil)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "完成更换worker第二步")
			globalVar.Process.Set(1)
		}
	})

	bottomRight := container.NewGridWithColumns(2, propose, confirm)
	bottom := container.NewGridWithColumns(2, minerEntry, bottomRight)
	return container.NewVBox(pkEntry, workerEntry, controlsEntry, bottom)
}