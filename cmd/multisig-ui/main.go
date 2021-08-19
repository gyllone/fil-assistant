package main

import (
	"context"
	"fil-assistant/common"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"os"
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
	w := a.NewWindow("FileCoin多重签名助手 V1.0.0")
	w.SetOnClosed(globalVar.Close)

	globalVar.Init(w)

	tabs := make([]*container.TabItem, 5)
	tabs[0] = container.NewTabItem("创建多签账户", createMsig())
	tabs[1] = container.NewTabItem("发起通用提案", generalProposals())
	tabs[2] = container.NewTabItem("发起矿工提案", miningProposals())
	tabs[3] = container.NewTabItem("赞成/反对提案", approveOrCancel())
	tabs[4] = container.NewTabItem("查询待定提案", getProposals())

	w.SetContent(container.NewVBox(Process(), container.NewAppTabs(tabs...)))
	w.Resize(fyne.NewSize(800, 0))
	w.CenterOnScreen()
	w.ShowAndRun()
}

func Process() fyne.CanvasObject {
	ProcessBar := widget.NewProgressBar()
	ProcessBar.Bind(globalVar.Process)
	ProcessBar.Resize(ProcessBar.MinSize())
	return ProcessBar
}

func generalProposals() fyne.CanvasObject {
	tabs := make([]*container.TabItem, 6)
	tabs[0] = container.NewTabItem("添加signer", addSigner())
	tabs[1] = container.NewTabItem("移除signer", removeSigner())
	tabs[2] = container.NewTabItem("交换signer", swapSigner())
	tabs[3] = container.NewTabItem("更改投票阈值", updateThreshold())
	tabs[4] = container.NewTabItem("锁仓", lockBalance())
	tabs[5] = container.NewTabItem("转账", send())

	return container.NewAppTabs(tabs...)
}

func miningProposals() fyne.CanvasObject {
	tabs := make([]*container.TabItem, 4)
	tabs[0] = container.NewTabItem("发起更换owner", proposeChangeOwner())
	tabs[1] = container.NewTabItem("确认更换owner", confirmChangeOwner())
	tabs[2] = container.NewTabItem("挖矿提现", withdraw())
	tabs[3] = container.NewTabItem("更换worker地址", changeWorker())

	return container.NewAppTabs(tabs...)
}

func createMsig() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	signers := widget.NewMultiLineEntry()
	signers.PlaceHolder = "signers地址"

	threshold := widget.NewEntry()
	threshold.PlaceHolder = "投票阈值"

	duration := widget.NewEntry()
	duration.PlaceHolder = "锁定期限"

	initAmount := widget.NewEntry()
	initAmount.PlaceHolder = "转账金额"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || threshold.Text == "" || duration.Text == "" || initAmount.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		signers := strings.Split(signers.Text, "\n")
		newSigners := make([]string, 0, len(signers))
		for _, signer := range signers {
			if newSigner := strings.TrimSpace(signer); newSigner != "" {
				newSigners = append(newSigners, newSigner)
			}
		}
		if len(newSigners) == 0 {
			globalVar.Msg(common.Warn, "signers地址输入有误")
			return
		}

		globalVar.Process.Set(0)

		id, err := globalVar.Handler.CreateMultisig(context.TODO(), newSigners, strings.TrimSpace(pk.Text),
			strings.TrimSpace(threshold.Text), strings.TrimSpace(duration.Text), strings.TrimSpace(initAmount.Text))
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("多签账号已生成: %s", id))
			globalVar.Process.Set(1)
		}
	})

	mid := container.NewGridWithColumns(2, threshold, duration)
	bottom := container.NewGridWithColumns(2, initAmount, submit)
	return container.NewVBox(pk, signers, mid, bottom)
}

func addSigner() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	adding := widget.NewEntry()
	adding.PlaceHolder = "signer地址"

	increase := widget.NewCheck("增加投票阈值", nil)

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || adding.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeAddSigner(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(adding.Text), increase.Checked, proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	right := container.NewGridWithColumns(2, increase, submit)
	return container.NewVBox(pk, adding, container.NewGridWithColumns(2, msig, right))
}

func removeSigner() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	removing := widget.NewEntry()
	removing.PlaceHolder = "signer地址"

	decrease := widget.NewCheck("减少投票阈值", nil)

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || removing.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeRemoveSigner(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(removing.Text), decrease.Checked, proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	right := container.NewGridWithColumns(2, decrease, submit)
	return container.NewVBox(pk, removing, container.NewGridWithColumns(2, msig, right))
}

func swapSigner() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	oldSigner := widget.NewEntry()
	oldSigner.PlaceHolder = "旧signer地址"

	newSigner := widget.NewEntry()
	newSigner.PlaceHolder = "新signer地址"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || oldSigner.Text == "" || newSigner.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeSwapSigner(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(oldSigner.Text), strings.TrimSpace(newSigner.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, oldSigner, newSigner, container.NewGridWithColumns(2, msig, submit))
}

func updateThreshold() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	threshold := widget.NewEntry()
	threshold.PlaceHolder = "投票阈值"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || threshold.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeChangeThreshold(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(threshold.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, container.NewGridWithColumns(2, msig, threshold), submit)
}

func lockBalance() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	start := widget.NewEntry()
	start.PlaceHolder = "启动区块高度"

	duration := widget.NewEntry()
	duration.PlaceHolder = "锁定期限"

	amount := widget.NewEntry()
	amount.PlaceHolder = "金额"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || start.Text == "" || duration.Text == "" || amount.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeLockBalance(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(start.Text), strings.TrimSpace(duration.Text), strings.TrimSpace(amount.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	mid := container.NewGridWithColumns(2, start, duration)
	bottom := container.NewGridWithColumns(2, msig, amount)
	return container.NewVBox(pk, mid, bottom, submit)
}

func send() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	to := widget.NewEntry()
	to.PlaceHolder = "收款地址"

	amount := widget.NewEntry()
	amount.PlaceHolder = "金额"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || to.Text == "" || amount.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.Send(context.TODO(), strings.TrimSpace(pk.Text), strings.TrimSpace(to.Text),
			strings.TrimSpace(amount.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, to, container.NewGridWithColumns(2, msig, submit))
}

func proposeChangeOwner() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	newOwner := widget.NewEntry()
	newOwner.PlaceHolder = "新owner地址"

	minerID := widget.NewEntry()
	minerID.PlaceHolder = "矿工号"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		globalVar.Process.Set(0)

		if pk.Text == "" || msig.Text == "" || newOwner.Text == "" || minerID.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ChangeOwner1(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(newOwner.Text), strings.TrimSpace(minerID.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, newOwner, container.NewGridWithColumns(2, msig, minerID), submit)
}

func confirmChangeOwner() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	minerID := widget.NewEntry()
	minerID.PlaceHolder = "矿工号"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || minerID.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ChangeOwner2(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(minerID.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, container.NewGridWithColumns(3, msig, minerID), submit)
}

func withdraw() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	minerID := widget.NewEntry()
	minerID.PlaceHolder = "矿工号"

	amount := widget.NewEntry()
	amount.PlaceHolder = "金额"

	submit := widget.NewButton("提交", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || minerID.Text != "" || amount.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.Withdraw(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(minerID.Text), strings.TrimSpace(amount.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	mid := container.NewGridWithColumns(2, minerID, amount)
	bottom := container.NewGridWithColumns(2, msig, submit)
	return container.NewVBox(pk, mid, bottom)
}

func changeWorker() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	minerID := widget.NewEntry()
	minerID.PlaceHolder = "矿工号"

	worker := widget.NewEntry()
	worker.PlaceHolder = "worker地址"

	controls := widget.NewMultiLineEntry()
	controls.PlaceHolder = "controls地址"

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

		if pk.Text == "" || msig.Text == "" || minerID.Text != "" || worker.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		controls := strings.Split(controls.Text, "\n")
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

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ProposeChangeWorker(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(minerID.Text), strings.TrimSpace(worker.Text), newControls, proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	confirm := widget.NewButton("提交确认", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || minerID.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		proposal := &common.Proposal{ Msig: strings.TrimSpace(msig.Text) }
		err := globalVar.Handler.ConfirmChangeWorker(context.TODO(), strings.TrimSpace(pk.Text),
			strings.TrimSpace(minerID.Text), proposal)
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, fmt.Sprintf("提案号已生成: %s", proposal.TxnID))
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(pk, worker, controls, container.NewGridWithColumns(4, msig, minerID, propose, confirm))
}

func approveOrCancel() fyne.CanvasObject {
	pk := widget.NewPasswordEntry()
	pk.PlaceHolder = "私钥"

	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	txID := widget.NewEntry()
	txID.PlaceHolder = "提案号"

	approve := widget.NewButton("赞成提案", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || txID.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ApproveOrCancel(context.TODO(), strings.TrimSpace(pk.Text), true,
			common.Proposal{ Msig: strings.TrimSpace(msig.Text) })
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "赞成提案完成")
			globalVar.Process.Set(1)
		}
	})
	aSrc, err := fyne.LoadResourceFromPath("./resource/approve.png")
	if err == nil {
		approve.SetIcon(aSrc)
	}

	reject := widget.NewButton("反对提案", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if pk.Text == "" || msig.Text == "" || txID.Text != "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		err := globalVar.Handler.ApproveOrCancel(context.TODO(), strings.TrimSpace(pk.Text), false,
			common.Proposal{ Msig: strings.TrimSpace(msig.Text) })
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "反对提案完成")
			globalVar.Process.Set(1)
		}
	})
	rSrc, err := fyne.LoadResourceFromPath("./resource/reject.png")
	if err == nil {
		reject.SetIcon(rSrc)
	}

	mid := container.NewGridWithColumns(2, msig, txID)
	bottom := container.NewGridWithColumns(2, approve, reject)
	return container.NewVBox(pk, mid, bottom)
}

func getProposals() fyne.CanvasObject {
	msig := widget.NewEntry()
	msig.PlaceHolder = "多签账号"

	query := widget.NewButton("查询", func() {
		if !globalVar.Locker.TryLock(0) {
			globalVar.Msg(common.Warn, "请稍后再试")
			return
		}
		defer globalVar.Locker.Unlock()

		if globalVar.Handler == nil {
			globalVar.Msg(common.Error, "初始化异常")
			return
		}

		if msig.Text == "" {
			globalVar.Msg(common.Warn, "输入为空")
			return
		}

		globalVar.Process.Set(0)

		str, err := globalVar.Handler.GetPendingProposals(context.TODO(), strings.TrimSpace(msig.Text))
		if err != nil {
			globalVar.Msg(common.Warn, err.Error())
			return
		}

		if err = os.WriteFile("./待定提案.txt", str, 0666); err != nil {
			globalVar.Msg(common.Warn, err.Error())
		} else {
			globalVar.Msg(common.Info, "待定提案.txt 已生成")
			globalVar.Process.Set(1)
		}
	})

	return container.NewVBox(container.NewGridWithColumns(2, msig, query), layout.NewSpacer())
}