package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/minchenzz/brc20tool/internal/ord"
	"github.com/minchenzz/brc20tool/pkg/btcapi/mempool"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/joho/godotenv"
)

var gwif string
var (
	gop     string
	gtick   string
	gamount string
	grepeat string
	gsats   string
)

func main() {
	fmt.Println("============ brc20 tool ============")

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	gwif = os.Getenv("PK")
	if gwif == "" {
		fmt.Println("Environment variable not found")
		return
	}

	gop = "mint"
	gtick = "omni"
	gamount = "100"
	grepeat = "2"
	gsats = "2"

	simulate := false
	txid, txids, fee, err := run(simulate)

	fmt.Println("\n[INFO] transactions (if not simulating):")
	fmt.Println("prepare utxo txid:", txid)
	fmt.Println("inscription txids:", txids)
	fmt.Println("fee:", fee)

	// a := app.New()
	// w := a.NewWindow("brc20 tool")
	// w.Resize(fyne.NewSize(800, 600))
	// // w.SetContent(widget.NewLabel("Hello World!"))
	// w.SetContent(makeForm(w))
	// w.ShowAndRun()
}

func makeForm(_w fyne.Window) fyne.CanvasObject {
	pk := widget.NewPasswordEntry()

	op := widget.NewEntry()
	op.SetPlaceHolder("op")

	tick := widget.NewEntry()
	tick.SetPlaceHolder("tick")

	amount := widget.NewEntry()
	amount.SetPlaceHolder("amount")

	fee := widget.NewEntry()
	fee.SetPlaceHolder("sats")
	fee.SetText("20")

	repeat := widget.NewEntry()
	repeat.SetPlaceHolder("repeat")
	repeat.SetText("1")

	fees := widget.NewEntry()
	fees.SetPlaceHolder("fees")

	txid := widget.NewEntry()
	txid.SetPlaceHolder("main txid")

	inscribeTxs := widget.NewEntry()
	inscribeTxs.SetPlaceHolder("inscribe txs")
	inscribeTxs.MultiLine = true

	estimate := widget.NewButton("estimate", func() {
		gwif = pk.Text
		gop = op.Text
		gtick = tick.Text
		gamount = amount.Text
		grepeat = repeat.Text
		gsats = fee.Text

		_, _, fee, err := run(true)
		if err != nil {
			dialog.ShowError(err, _w)
			return
		}
		fees.SetText(strconv.FormatInt(fee, 10))
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "private key", Widget: pk, HintText: "Your wif private key"},
			{Text: "op", Widget: op, HintText: "eg: mint, transfer"},
			{Text: "tick", Widget: tick, HintText: "eg: ordi, SHIB"},
			{Text: "amount", Widget: amount, HintText: "eg: 1, 100000"},
			{Text: "sats", Widget: fee, HintText: "eg: 20, 30"},
			{Text: "repeat", Widget: repeat, HintText: "eg: 1, 5, 10"},
			{Text: "estimate fee", Widget: estimate, HintText: "estimate fee(sats)"},
			{Text: "fee", Widget: fees, HintText: "txs fee"},
			{Text: "txid", Widget: txid, HintText: "main txid"},
			{Text: "inscribe txids", Widget: inscribeTxs, HintText: "inscribe txids"},
		},
		OnSubmit: func() {
			gwif = pk.Text
			gop = op.Text
			gtick = tick.Text
			gamount = amount.Text
			grepeat = repeat.Text
			gsats = fee.Text

			_txid, txids, _, err := run(false)
			if err != nil {
				dialog.ShowError(err, _w)
				return
			}
			txid.SetText(_txid)
			txisstr := strings.Join(txids, "\n")
			inscribeTxs.SetText(txisstr)
		},
	}

	return form
}

func run(forEstimate bool) (txid string, txids []string, fee int64, err error) {
	// ------------------- Log the basic information -------------------
	fmt.Println("\n[INFO] constructing tx...")
	fmt.Println("simulating:", forEstimate)

	// ------------------- Define network parameters -------------------
	netParams := &chaincfg.SigNetParams
	btcApiClient := mempool.NewClient(netParams)
	wifKey, err := btcutil.DecodeWIF(gwif)
	if err != nil {
		return
	}
	fmt.Println("net:", netParams.Name)

	// ------------------- Load the address -------------------
	utxoTaprootAddress, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(wifKey.PrivKey.PubKey())), netParams)
	if err != nil {
		return
	}
	fmt.Println("utxo address:", utxoTaprootAddress.EncodeAddress())

	// --------------------- Collect UTXOs ---------------------
	unspentList, err := btcApiClient.ListUnspent(utxoTaprootAddress)
	if err != nil {
		return
	}
	if len(unspentList) == 0 {
		err = fmt.Errorf("no utxo for %s", utxoTaprootAddress)
		return
	}
	vinAmount := 0
	commitTxOutPointList := make([]*wire.OutPoint, 0)
	commitTxPrivateKeyList := make([]*btcec.PrivateKey, 0)
	utxoValues := make([]string, len(unspentList))
	for i := range unspentList {
		utxoValues[i] = fmt.Sprintf("%d", unspentList[i].Output.Value)
		if unspentList[i].Output.Value < 10000 {
			utxoValues[i] = fmt.Sprintf("%s (unusable)", utxoValues[i])
			continue
		}
		commitTxOutPointList = append(commitTxOutPointList, unspentList[i].Outpoint)
		commitTxPrivateKeyList = append(commitTxPrivateKeyList, wifKey.PrivKey)
		vinAmount += int(unspentList[i].Output.Value)
	}
	fmt.Println("utxo count:", len(unspentList))
	fmt.Println("utxo value list: [", strings.Join(utxoValues, ", "), "]")

	// --------------- Construct the inscription data ---------------
	dataList := make([]ord.InscriptionData, 0)
	ordinalText := fmt.Sprintf(`{"p":"brc-20","op":"%s","tick":"%s","amt":"%s"}`, gop, gtick, gamount)
	mint := ord.InscriptionData{
		ContentType: "text/plain;charset=utf-8",
		Body:        []byte(ordinalText),
		Destination: utxoTaprootAddress.EncodeAddress(),
	}
	count, err := strconv.Atoi(grepeat)
	if err != nil {
		return
	}
	for i := 0; i < count; i++ {
		dataList = append(dataList, mint)
	}
	fmt.Println("inscription text:", ordinalText)
	fmt.Println("repeat mint times: ", len(dataList))

	// --------------- Construct the inscription request ---------------
	txFee, err := strconv.Atoi(gsats)
	if err != nil {
		return
	}
	request := ord.InscriptionRequest{
		CommitTxOutPointList:   commitTxOutPointList,
		CommitTxPrivateKeyList: commitTxPrivateKeyList,
		CommitFeeRate:          int64(txFee),
		FeeRate:                int64(txFee),
		DataList:               dataList,
		SingleRevealTxOnly:     false,
	}

	// ----------------- Prepare the inscription tool -----------------
	fmt.Println("current balance:", float32(vinAmount)/1e8)
	fmt.Print("total spent fee: ")
	tool, err := ord.NewInscriptionToolWithBtcApiClient(netParams, btcApiClient, &request)
	if err != nil {
		fmt.Println("new tool err: ", err)
		return
	}
	fee = tool.CalculateFee()
	fmt.Println("balance after inscription:", (float32(vinAmount)-float32(fee))/1e8)
	if forEstimate {
		return
	}

	// ------------------- Submit the transaction -------------------
	commitTxHash, revealTxHashList, _, _, err := tool.Inscribe()
	if err != nil {
		err = fmt.Errorf("send tx errr, %v", err)
		return
	}
	txid = commitTxHash.String()
	for i := range revealTxHashList {
		txids = append(txids, revealTxHashList[i].String())
	}
	return
}
